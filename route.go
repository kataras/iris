package iris

import (
	"net/http"
	"strings"
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

type routeType uint8

const (
	isStatic routeType = iota
	hasParams
	hasWildcard
)

// Route contains its middleware, handler, pattern , it's path string, http methods and a template cache
// Used to determinate which handler on which path must call
// Used on router.go
type Route struct {
	//GET, POST, PUT, DELETE, CONNECT, HEAD, PATCH, OPTIONS, TRACE bool //tried with []string, very slow, tried with map[string]bool gives 10k executions but +20k bytes, with this approact we have to code more but only 1k byte to space and make it 2.2 times faster than before!
	//Middleware
	MiddlewareSupporter
	//mu            sync.RWMutex
	paramsLength        uint8
	pathParts           []*Part
	partsLen            uint8
	rType               routeType
	lastStaticPartIndex uint8  // when( how much slashes we have before) the dynamic path, the start of the dynamic path in words of slashes /
	pathPrefix          string // this is to make a little faster the match, before regexp Match runs, it's the path before the first path parameter :
	//the pathPrefix is with the last / if parameters exists.
	parts []string //stores the string path AFTER the pathPrefix, without the pathPrefix. no need to that but no problem also.
	//if parts != nil means that this route has no params
	fullpath string // need only on parameters.Params(...)
	//fullparts   []string
	handler    Handler
	templates  *TemplateCache //this is passed to the Renderer
	httpErrors *HTTPErrors    //the only need of this is to pass into the Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
	isReady    bool
}

// newRoute creates, from a path string, handler and optional http methods and returns a new route pointer
func newRoute(registedPath string, handler Handler) *Route {
	r := &Route{handler: handler, pathParts: make([]*Part, 0)}

	r.fullpath = registedPath
	r.processPath()
	return r
}

func (r *Route) processPath() {
	var part *Part
	splitted := strings.Split(r.fullpath, "/")
	r.partsLen = uint8(len(splitted) - 1) // dont count the first / splitted item

	for idx, val := range splitted {
		if val == "" {
			continue
		}
		part = &Part{}
		letter := val[0]

		if idx == len(splitted)-1 {
			//we are at the last part
			part.isLast = true
			if letter == MatchEverythingByte {
				//println(r.fullpath + " has wildcard and it's part has")
				//we have finish it with *
				part.isMatchEverything = true
				r.rType = hasWildcard

			}
		}
		if letter != ParameterStartByte {

		} else {
			part.isParam = true
			val = val[1:] //drop the :
			r.rType = hasParams
			r.paramsLength++
		}

		part.Prefix = letter
		part.Value = val

		part.Position = idx

		if !part.isParam && !part.isMatchEverything {
			part.isStatic = true
			if r.rType != hasParams && r.rType != hasWildcard { // it's the last static path.
				r.lastStaticPartIndex = uint8(idx)
				//println(r.lastStaticPartIndex, "for ", r.fullpath)
			}

		}

		//fmt.Printf("%s : Part value '%s' %v \n\n", r.fullpath, part.Value, part)
		r.pathParts = append(r.pathParts, part)

	}
	//if len(r.pathParts) > 0 {
	//r.pathPrefix = "/" + r.pathParts[0].Value

	if r.rType != hasWildcard && r.rType != hasParams {
		r.rType = isStatic

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
			//check for first slash
			endPrefixIndex = strings.IndexByte(r.fullpath, SlashByte)
			if endPrefixIndex != -1 {
				r.pathPrefix = r.fullpath[:endPrefixIndex]
			} else {
				//we don't have ending slash ? then it is the whole r.fullpath
				r.pathPrefix = r.fullpath
			}
		}
	}

	//}

}

func (r *Route) Match(urlPath string) bool {
	if r.rType == isStatic {
		return urlPath == r.fullpath
	} else if len(urlPath) < len(r.pathPrefix) {

		return false
	}
	reqPath := urlPath[len(r.pathPrefix):] //we start from there to make it faster
	rest := reqPath
	var pathIndex = r.lastStaticPartIndex
	var part *Part
	var endSlash int
	var reqPart string
	for pathIndex < r.partsLen {

		endSlash = 1
		for endSlash < len(rest) && rest[endSlash] != '/' {
			endSlash++
		}
		if endSlash > len(rest) {

			return false
		}
		if r.lastStaticPartIndex == r.partsLen-1 {
			reqPart = rest[0:endSlash] // the forward slash was inside prefix, then here we don't have slash, if lastStatic part index is near to the end, then thats means this: /api/users/1
		} else {
			reqPart = rest[1:endSlash] //remove the forward slash
		}

		if len(reqPart) == 0 { // if the reqPart is "" it means that the requested url is SMALLER than the registed
			return false
		}

		rest = rest[endSlash:]

		part = r.pathParts[pathIndex] //edw argei alla dn kserw gt.. siga ti kanei
		pathIndex++

		if part.isStatic {
			if part.Value != reqPart {
				return false //fast return
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
			//TODO: save the parameters and continue
			if part.isLast {
				//it's the last registed part
				if len(rest) > 0 {
					//but the request path is bigger than this
					return false
				}
				return true

			}
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
