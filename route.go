package iris

import (
	"net/http"
	"strings"
)

// Route contains its middleware, handler, pattern , it's path string, http methods and a template cache
// Used to determinate which handler on which path must call
// Used on router.go
type Route struct {
	GET, POST, PUT, DELETE, CONNECT, HEAD, PATCH, OPTIONS, TRACE bool //tried with []string, very slow, tried with map[string]bool gives 10k executions but +20k bytes, with this approact we have to code more but only 1k byte to space and make it 2.2 times faster than before!
	//Middleware
	MiddlewareSupporter
	//mu            sync.RWMutex

	pathPrefix string // this is to make a little faster the match, before regexp Match runs, it's the path before the first path parameter :
	//the pathPrefix is with the last / if parameters exists.
	parts       []string //stores the string path AFTER the pathPrefix, without the pathPrefix. no need to that but no problem also.
	fullpath    string   // need only on parameters.Params(...)
	handler     Handler
	isReady     bool
	templates   *TemplateCache //this is passed to the Renderer
	httpErrors  *HTTPErrors    //the only need of this is to pass into the Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
	hasWildcard bool
}

// newRoute creates, from a path string, handler and optional http methods and returns a new route pointer
func newRoute(registedPath string, handler Handler) *Route {
	r := &Route{handler: handler}

	hasPathParameters := false
	firstPathParamIndex := strings.IndexByte(registedPath, ParameterStartByte)
	if firstPathParamIndex != -1 {
		r.pathPrefix = registedPath[:firstPathParamIndex]
		hasPathParameters = true

		if strings.HasSuffix(registedPath, MatchEverything) {
			r.hasWildcard = true
		}

	} else {
		//check for only for* , here no path parameters registed.
		firstPathParamIndex = strings.IndexByte(registedPath, MatchEverythingByte)

		if firstPathParamIndex != -1 {
			if firstPathParamIndex <= 1 { // set to '*' to pathPrefix if no prefix exists except the slash / if any [Get("/*",..) or Get("*",...]
				//has no prefix just *
				r.pathPrefix = MatchEverything
				r.hasWildcard = true
			} else { //if firstPathParamIndex == len(registedPath)-1 { // it's the last
				//has some prefix and sufix of *
				r.pathPrefix = registedPath[:firstPathParamIndex] //+1
				r.hasWildcard = true
			}

		} else {
			//else no path parameter or match everything symbol so use the whole path as prefix it will be faster at the check for static routes too!
			r.pathPrefix = registedPath
		}

	}

	if hasPathParameters || r.hasWildcard {
		r.parts = strings.Split(registedPath[len(r.pathPrefix):], "/")
		r.fullpath = registedPath //we need this only to take Params so set it if has path parameters.
	}

	return r
}

// containsMethod determinates if this route contains a http method
func (r *Route) containsMethod(method string) bool {
	/*for _, m := range r.methods {
		if m == method {
			return true
		}
	}*/
	switch method {
	case "GET":
		return r.GET
	case "POST":
		return r.POST
	case "PUT":
		return r.PUT
	case "DELETE":
		return r.DELETE
	case "CONNECT":
		return r.CONNECT
	case "HEAD":
		return r.HEAD
	case "PATCH":
		return r.PATCH
	case "OPTIONS":
		return r.OPTIONS
	case "TRACE":
		return r.TRACE

	}
	return false
}

// Methods adds methods to its registed http methods
func (r *Route) Methods(methods ...string) *Route {
	//if r.methods == nil {
	//	r.methods = make([]string, 0)
	//}
	//r.methods = append(r.methods, methods...)
	for i := 0; i < len(methods); i++ {
		switch methods[i] {
		case "GET":
			r.GET = true
			break
		case "POST":
			r.POST = true
			break
		case "PUT":
			r.PUT = true
			break
		case "DELETE":
			r.DELETE = true
			break
		case "CONNECT":
			r.CONNECT = true
			break
		case "HEAD":
			r.HEAD = true
			break
		case "PATCH":
			r.PATCH = true
			break
		case "OPTIONS":
			r.OPTIONS = true
			break
		case "TRACE":
			r.TRACE = true
			break

		}
	}
	return r
}

// Method SETS a method to its registed http methods, overrides the previous methods registed (if any)
func (r *Route) Method(method string) *Route {
	switch method {
	case "GET":
		r.GET = true
		break
	case "POST":
		r.POST = true
		break
	case "PUT":
		r.PUT = true
		break
	case "DELETE":
		r.DELETE = true
		break
	case "CONNECT":
		r.CONNECT = true
		break
	case "HEAD":
		r.HEAD = true
		break
	case "PATCH":
		r.PATCH = true
		break
	case "OPTIONS":
		r.OPTIONS = true
		break
	case "TRACE":
		r.TRACE = true
		break

	}
	return r
}

// match determinates if this route match with the request, returns bool as first value and PathParameters as second value, if any
func (r *Route) match(urlPath string) bool {
	if r.pathPrefix == MatchEverything {
		return true
	}
	if r.pathPrefix == urlPath {
		//it's route without path parameters or * symbol, and if the request url has prefix of it  and it's the same as the whole preffix which is the path itself returns true without checking for regexp pattern
		//so it's just a path without named parameters
		return true

		//kapws kai to sufix na vlepw an den einai parameter, an einai idio kai meta na sunexizei sto path parameters.
	} else if r.parts != nil {
		partsLen := len(r.parts)
		// count the slashes after the prefix, we start from one because we will have at least one slash.
		reqPartsLen := 1
		s := urlPath[len(r.pathPrefix):]
		for i := 0; i < len(s); i++ {
			if s[i] == SlashByte {
				reqPartsLen++
			}
		}

		//if request has more parts than the route, but the route has finish with * symbol then it's wildcard
		//maybe it's a little confusing , why dont u use just reqPartsLen < partsLen return false ? it doesnt work this way :)
		if reqPartsLen >= partsLen && r.hasWildcard { // r.parts[partsLen-1][0] == MatchEverythingByte { // >= and no != because we check for matchEveryting *
			return true
		} else if reqPartsLen != partsLen {
			return false
		}

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

	r.isReady = true
}

// ServeHTTP serves this route and it's middleware
func (r *Route) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if r.isReady == false && r.handler != nil {
		r.prepare()
	}
	r.middleware.ServeHTTP(res, req)
}
