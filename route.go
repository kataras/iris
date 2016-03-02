package iris

import (
	"net/http"
	"strings"
)

// Route contains its middleware, handler, pattern , it's path string, http methods and a template cache
// Used to determinate which handler on which path must call
// Used on router.go
type Route struct {

	//Middleware
	MiddlewareSupporter
	//mu            sync.RWMutex
	methods    []string
	pathPrefix string // this is to make a little faster the match, before regexp Match runs, it's the path before the first path parameter :
	//the pathPrefix is with the last / if parameters exists.
	parts []string //stores the string path AFTER the pathPrefix, without the pathPrefix. no need to that but no problem also.
	//test
	fullpath string
	//
	handler    Handler
	isReady    bool
	templates  *TemplateCache //this is passed to the Renderer
	httpErrors *HTTPErrors    //the only need of this is to pass into the Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
}

// newRoute creates, from a path string, handler and optional http methods and returns a new route pointer
func newRoute(registedPath string, handler Handler) *Route {
	r := &Route{handler: handler}
	hasPathParameters := false
	firstPathParamIndex := strings.Index(registedPath, ParameterStart)
	if firstPathParamIndex != -1 {
		r.pathPrefix = registedPath[:firstPathParamIndex] ///api/users  to /api/users/
		hasPathParameters = true
	} else {
		//check for only for* , here no path parameters registed.
		firstPathParamIndex = strings.Index(registedPath, MatchEverything)

		if firstPathParamIndex != -1 {
			if firstPathParamIndex <= 1 { // set to '*' to pathPrefix if no prefix exists except the slash / if any [Get("/*",..) or Get("*",...]
				r.pathPrefix = MatchEverything
			} else {
				r.pathPrefix = registedPath[:firstPathParamIndex+1]
			}

		} else {
			//else no path parameter or match everything symbol so use the whole path as prefix it will be faster at the check for static routes too!
			r.pathPrefix = registedPath
		}

	}

	if hasPathParameters {
		r.parts = strings.Split(registedPath[len(r.pathPrefix):], "/")

	}
	r.fullpath = registedPath
	return r
}

// containsMethod determinates if this route contains a http method
func (r *Route) containsMethod(method string) bool {
	for _, m := range r.methods {
		if m == method {
			return true
		}
	}

	return false
}

// Methods adds methods to its registed http methods
func (r *Route) Methods(methods ...string) *Route {
	if r.methods == nil {
		r.methods = make([]string, 0)
	}
	r.methods = append(r.methods, methods...)
	return r
}

// Method SETS a method to its registed http methods, overrides the previous methods registed (if any)
func (r *Route) Method(method string) *Route {
	r.methods = []string{HTTPMethods.GET}
	return r
}

var removePunctuation = func(r rune) rune {
	if strings.ContainsRune("/", r) {
		return -1
	} else {
		return r
	}
}

func (r *Route) getRegistedPath() string {
	if r.parts != nil {
		return r.pathPrefix + strings.Join(r.parts, "/")
	}
	return r.pathPrefix
}

//var routeBuff = make([]byte,40)
// Match determinates if this route match with the request, returns bool as first value and PathParameters as second value, if any
func (r *Route) match(urlPath string) bool {
	//an to kanw me prefix sto router adi gia ta methods
	//tote ta 2 prwta if == MatchEv... auto 9a elenxete sto router kai to r.pathPrefix == urlPath  9a einai autonoito ara dn 9a xreiastei.
	//test:
	//return true, nil

	//for tests
	//testUrl := "/repos/owner/repo/git/blobs/sha"
	//if urlPath == testUrl {
	//	println("match this route ? ")
	//	println("route's .prefix: ", r.pathPrefix)

	//}
	//end for tests
	if r.pathPrefix == MatchEverything {
		return true
	}
	//println(r.pathPrefix + " and urlPath: " + urlPath)
	if r.pathPrefix == urlPath {
		//it's route without path parameters or * symbol, and if the request url has prefix of it  and it's the same as the whole preffix which is the path itself returns true without checking for regexp pattern
		//so it's just a path without named parameters
		return true

	} else if r.parts != nil {
		//	if urlPath == testUrl {
		//		println("route has parts ")
		//		println("route's len(parts) : ", len(r.parts))
		//
		//	}
		//it's not a static route it has parameters,and the url has the right prefix [taken from router's map] so check for it
		//reqParts := strings.Split(urlPath[len(r.pathPrefix):], "/")

		//it's even slower than split ...
		//s := strings.Map(removePunctuation, urlPath[len(r.pathPrefix):])
		//reqParts := strings.Fields(s)
		partsLen := len(r.parts)
		reqPartsLen := 1
		s := urlPath[len(r.pathPrefix):]
		for i := 0; i < len(s); i++ {
			if s[i] == SlashByte {
				reqPartsLen++
			}
		}

		//reqPartsLen := strings.Count(urlPath[len(r.pathPrefix):], "/") + 1 //auto apodiktike polu pio fast twra mono ta parameters mas menoun
		//reqParts := strings.Split(urlPath[len(r.pathPrefix):], "/")
		//reqParts := strings.SplitN(urlPath[len(r.pathPrefix):], "/", partsLen+1) // +1 because if it's more than partsLen we want to return false and no match to the closest route.

		//reqParts := re.FindAllString(urlPath[len(r.pathPrefix):], partsLen+1)
		if reqPartsLen != partsLen {
			//if urlPath == testUrl && len(r.parts) == 5 {

			//println("req parts ", len(reqParts), " are not the same as parts len ", partsLen)
			//println("route's path: " + r.getRegistedPath() + " req path: " + urlPath)
			//	println("routes parts: ")
			//	for _, _ppart := range r.parts {
			//		println(_ppart)
			//	}

			//	println("req parts: ")

			//for _, _ppart := range reqParts {
			//		println(_ppart)
			//	}
			//}
			return false // request has not the correct number of passed parameters to the route.
		}
		//edw ta allocs pernoun fwtia kai ta nanoseconds sto mellon na to ftiaksw. otan telewisw to pws a9 ginonte match ta routes..
		//reqParts := strings.Split(urlPath[len(r.pathPrefix):], "/")

		return true

	} else {
		return false
	}

}

// Template creates (if not exists) and returns the template cache for this route
func (r *Route) Template() *TemplateCache {
	if r.templates == nil {
		r.templates = NewTemplateCache()
	}
	return r.templates
}

// prepare prepares the route's handler , places it to the last middleware , handler acts like a middleware too.
// Runs once before the first ServeHTTP
func (r *Route) prepare() {
	//r.mu.Lock()
	//look why on router ->HandleFunc defer r.mu.Unlock()
	//but wait... do we need locking here?
	if r.handler != nil {
		convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
			r.handler.run(r, res, req)
			next(res, req)
		})

		r.Use(convertedMiddleware)
	}

	//here if no methods are defined at all, then use GET by default.
	if r.methods == nil {
		r.methods = []string{HTTPMethods.GET}
	}

	r.isReady = true
}

// ServeHTTP serves this route and it's middleware
func (r *Route) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if r.isReady == false && r.handler != nil {
		r.prepare()
	}
	r.middleware.ServeHTTP(res, req)
}
