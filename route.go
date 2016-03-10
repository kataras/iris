package iris

import (
	"net/http"
	"strings"
)

const (
	ParameterStartByte  = byte(':')
	SlashByte           = byte('/')
	MatchEverythingByte = byte('*')
)

type Part struct {
	Position          int    // the position of this Part by the total slashes, starts from 1. Position = 2 means that this Part is  starting at the second slash of the registedPath, maybe not useful here.
	Prefix            byte   //the first character
	Value             string //the whole word
	isStatic          bool
	isParam           bool // if it is path parameter
	isLast            bool // if it is the last of the registed path
	isMatchEverything bool //if this is true then the isLast = true and the isParam = false
}

// Route contains its middleware, handler, pattern , it's path string, http methods and a template cache
// Used to determinate which handler on which path must call
// Used on router.go
type Route struct {
	//GET, POST, PUT, DELETE, CONNECT, HEAD, PATCH, OPTIONS, TRACE bool //tried with []string, very slow, tried with map[string]bool gives 10k executions but +20k bytes, with this approact we have to code more but only 1k byte to space and make it 2.2 times faster than before!
	//Middleware
	MiddlewareSupporter
	//mu            sync.RWMutex
	paramsLength        uint8
	pathParts           []Part
	partsLen            uint8
	isStatic            bool
	lastStaticPartIndex uint8  // when( how much slashes we have before) the dynamic path, the start of the dynamic path in words of slashes /
	pathPrefix          string // this is to make a little faster the match, before regexp Match runs, it's the path before the first path parameter :
	//the pathPrefix is with the last / if parameters exists.
	//the priority used on the Routes sort.Interface implementation
	//less is more important than a bigger number
	// TODO:
	//priority int
	fullpath string // need only on parameters.Params(...)
	//fullparts   []string
	handler    Handler
	templates  *TemplateCache //this is passed to the Renderer
	httpErrors *HTTPErrors    //the only need of this is to pass into the Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
	isReady    bool
}

// newRoute creates, from a path string, handler and optional http methods and returns a new route pointer
func newRoute(registedPath string, handler Handler) *Route {
	r := &Route{handler: handler, pathParts: make([]Part, 0)}

	r.fullpath = registedPath
	r.processPath()
	return r
}

func (r *Route) processPath() {
	var part Part
	var hasParams bool
	var hasWildcard bool
	splitted := strings.Split(r.fullpath, "/")
	r.partsLen = uint8(len(splitted) - 1) // dont count the first / splitted item

	for idx, val := range splitted {
		if val == "" {
			continue
		}
		part = Part{}
		letter := val[0]

		if idx == len(splitted)-1 {
			//we are at the last part
			part.isLast = true
			if letter == MatchEverythingByte {
				//println(r.fullpath + " has wildcard and it's part has")
				//we have finish it with *
				part.isMatchEverything = true
				hasWildcard = true

			}
		}
		if letter != ParameterStartByte {

		} else {
			part.isParam = true
			val = val[1:] //drop the :
			hasParams = true
			r.paramsLength++
		}

		part.Prefix = letter
		part.Value = val

		part.Position = idx

		if !part.isParam && !part.isMatchEverything {
			part.isStatic = true
			if !hasParams && !hasWildcard { // it's the last static path.
				r.lastStaticPartIndex = uint8(idx)
				//println(r.lastStaticPartIndex, "for ", r.fullpath)
			}

		}

		//fmt.Printf("%s : Part value '%s' %v \n\n", r.fullpath, part.Value, part)
		r.pathParts = append(r.pathParts, part)

	}
	//if len(r.pathParts) > 0 {
	//r.pathPrefix = "/" + r.pathParts[0].Value

	if !hasParams && !hasWildcard {
		r.isStatic = true
	}

	//find the prefix which is the path which ends on the first :,* if exists otherwise the first /
	//we don't care about performance in this method, because of this is a little shit.
	endPrefixIndex := strings.IndexByte(r.fullpath, ParameterStartByte)

	if endPrefixIndex != -1 {
		r.pathPrefix = r.fullpath[:endPrefixIndex]

	} else {
		//check for *
		endPrefixIndex = strings.IndexByte(r.fullpath, MatchEverythingByte)
		if endPrefixIndex != -1 {
			r.pathPrefix = r.fullpath[:endPrefixIndex]
		} else {
			//check for the last slash
			endPrefixIndex = strings.LastIndexByte(r.fullpath, SlashByte)
			if endPrefixIndex != -1 {
				r.pathPrefix = r.fullpath[:endPrefixIndex]
			} else {
				//we don't have ending slash ? then it is the whole r.fullpath
				r.pathPrefix = r.fullpath
			}
		}
	}

	//1.check if pathprefix is empty ( it's empty when we have just '/' as fullpath) so make it '/'
	//2. check if it's not ending with '/', ( it is not ending with '/' when the next part is parameter or *)

	lastIndexOfSlash := strings.LastIndexByte(r.pathPrefix, SlashByte)
	//if r.pathPrefix[len(r.pathPrefix)-1:][0] != SlashByte {
	if lastIndexOfSlash != len(r.pathPrefix)-1 || r.pathPrefix == "" {
		r.pathPrefix += "/"
	}

	//}

	//for path prefix result :
	//all routes which has only one static part the pathPrefix is just a slash, so all routes with one static part like /users ,/home will be at the same treenode prefix '/'
	//else the route prefix is when the first ':', or '*' or last slash '/' index found
	//on each handlefunc on the router the Routes collections is sorted with the priority of the biggest
	//path prefix to the smaller: first longest path prefix.
	//this is done with each register route because we don't have a mechanism yet that we can understand
	//when the developer stop routing, we could make it at .Listen but because Iris can run as
	//just a handler with ServeHTTP this is can't be done on .Listen.
}

// Verify checks if this route is matching with the urlPath parameter
//
// Returns true if matched, otherwise false
func (r *Route) Verify(urlPath string) bool {
	if r.isStatic {
		return urlPath == r.fullpath
	} else if len(urlPath) < len(r.pathPrefix) {

		return false
	}
	var pathIndex = r.lastStaticPartIndex
	var part Part
	var endSlash int
	var reqPart string
	//var params PathParameters = nil
	//var paramsBuff bytes.Buffer
	var rest string
	reqPath := urlPath[len(r.pathPrefix):] //we start from there to make it faster
	rest = reqPath
	for pathIndex < r.partsLen {

		endSlash = 1
		for endSlash < len(rest) && rest[endSlash] != '/' {
			endSlash++
		}
		if endSlash > len(rest) {

			return false
		}
		reqPart = rest[0:endSlash]

		if len(reqPart) == 0 { // if the reqPart is "" it means that the requested url is SMALLER than the registed
			return false
		}

		part = r.pathParts[pathIndex] //edw argei alla dn kserw gt.. siga ti kanei
		pathIndex++

		if pathIndex == 0 || pathIndex >= r.partsLen || len(rest) <= endSlash {
			rest = rest[endSlash:]
		} else {
			//if this is the not first, and it is safe to concat, forget the forward slash because it was from the prefix/or/and static part
			//but also checks if this is not the end of the url because if it is then we will have error on +1
			//it is used to take the correct parameter if any otherwise we will have
			//the first parameter with no forward slash
			//but the others begins with a forward slash
			rest = rest[endSlash+1:]
		}

		if part.isStatic {
			if part.Value != reqPart {
				return false
			} else {
				if part.isLast {
					//it's the last registed part
					if len(rest) > 0 {
						//but the request path is bigger than this
						return false
					}
					return true

				}

				continue

			}
		} else if part.isParam {
			//stfu that, too much memory allocations because it searches to the params until false or true
			// i will do the excactly thing I am doing here at the context handler if registed as handler
			//if params == nil {
			//	params = PathParameters{}
			//}
			//TODO: save the parameters and continue
			//params.Set(part.Value, reqPart) TOO MUCH MEM ALLOCATIONS I HAVE TO FIND A WAY FOR ROUTES THAT DONT MATCH DONT COME HERE..xmm
			//println("setting parameter: ", part.Value, " = ", reqPart)
			//paramsBuff.WriteString(part.Value)
			//paramsBuff.WriteRune('=')
			//paramsBuff.WriteString(reqPart)
			if part.isLast {
				//it's the last registed part
				if len(rest) > 0 {
					//but the request path is bigger than this
					return false
				}

				return true

			}

			//paramsBuff.WriteRune(',')
			continue

		} else if part.isMatchEverything {
			// just return true it is matching everything after that, be care when I make the params here return params too because wildcard can have params before the *
			return true
		}

	}
	return true
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
// MUST REMOVE IT SOME DAY AND MAKE MIDDLEWARE MORE LIGHTER
func (r *Route) prepare() {
	//r.mu.Lock()
	//look why on router ->HandleFunc defer r.mu.Unlock()
	//but wait... do we need locking here?

	convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		r.handler.run(r, res, req)
		next(res, req)
	})

	r.Use(convertedMiddleware)
	r.isReady = true

}

// ServeHTTP serves this route and it's middleware
func (r *Route) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if r.middlewareHandlers != nil {
		if !r.isReady {
			r.prepare()
		}
		r.middleware.ServeHTTP(res, req)
	} else {
		r.handler.run(r, res, req)
	}
}
