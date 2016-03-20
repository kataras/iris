// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
)

const (
	// ParameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	ParameterStartByte = byte(':')
	// SlashByte is just a byte of '/' rune/char
	SlashByte = byte('/')
	// MatchEverythingByte is just a byte of '*" rune/char
	MatchEverythingByte = byte('*')
)

// IRouterMethods is the interface for method routing
type IRouterMethods interface {
	Get(path string, handlersFn ...HandlerFunc) IRoute
	Post(path string, handlersFn ...HandlerFunc) IRoute
	Put(path string, handlersFn ...HandlerFunc) IRoute
	Delete(path string, handlersFn ...HandlerFunc) IRoute
	Connect(path string, handlersFn ...HandlerFunc) IRoute
	Head(path string, handlersFn ...HandlerFunc) IRoute
	Options(path string, handlersFn ...HandlerFunc) IRoute
	Patch(path string, handlersFn ...HandlerFunc) IRoute
	Trace(path string, handlersFn ...HandlerFunc) IRoute
	Any(path string, handlersFn ...HandlerFunc) IRoute
}

// IRouter is the interface of which any Iris router must implement
type IRouter interface {
	IMiddlewareSupporter
	IRouterMethods
	IPartyHoster
	HandleAnnotated(Handler) (IRoute, error)
	Handle(string, string, ...Handler) IRoute
	HandleFunc(string, string, ...HandlerFunc) IRoute
	Errors() IHTTPErrors //at the main Router struct this is managed by the MiddlewareSupporter
	// ServeHTTP finds and serves a route by it's request
	// If no route found, it sends an http status 404
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// Router is the router , one router per server.
// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
type Router struct {
	MiddlewareSupporter
	station    *Station
	garden     Garden
	httpErrors IHTTPErrors //the only reason of this is to pass into the route, which it need it to  passed it to Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
}

// NewRouter creates and returns an empty Router
func NewRouter(station IStation) *Router {
	return &Router{station: station.(*Station), garden: make(Garden, 0, len(HTTPMethods.ANY)), httpErrors: DefaultHTTPErrors()}
}

// SetErrors sets a HTTPErrors object to the router
func (r *Router) SetErrors(httperr IHTTPErrors) {
	r.httpErrors = httperr
}

// Errors get the HTTPErrors from the router
func (r *Router) Errors() IHTTPErrors {
	return r.httpErrors
}

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

func htmlEscape(s string) string {
	return htmlReplacer.Replace(s)
}

/////////////////////////////////
//expose common methods as public
/////////////////////////////////

// Handle registers a route to the server's router
func (r *Router) Handle(method string, registedPath string, handlers ...Handler) IRoute {
	if registedPath == "" {
		registedPath = "/"
	}
	if len(handlers) == 0 {
		panic("Iris.Handle: zero handler to " + method + ":" + registedPath)
	}

	if len(r.Middleware) > 0 {
		//if global middlewares are registed then push them to this route.
		handlers = append(r.Middleware, handlers...)
	}

	route := NewRoute(registedPath, handlers)

	r.station.GetPluginContainer().DoPreHandle(method, route)

	r.garden = r.garden.Plant(method, route).(Garden)

	r.station.GetPluginContainer().DoPostHandle(method, route)

	return route
}

// HandleFunc registers and returns a route with a method string, path string and a handler
// registedPath is the relative url path
// handler is the iris.Handler which you can pass anything you want via iris.ToHandlerFunc(func(res,req){})... or just use func(c *iris.Context)
func (r *Router) HandleFunc(method string, registedPath string, handlersFn ...HandlerFunc) IRoute {
	return r.Handle(method, registedPath, ConvertToHandlers(handlersFn)...)
}

// HandleAnnotated registers a route handler using a Struct implements iris.Handler (as anonymous property)
// which it's metadata has the form of
// `method:"path"` and returns the route and an error if any occurs
// handler is passed by func(urstruct MyStruct) Serve(ctx *Context) {}
func (r *Router) HandleAnnotated(irisHandler Handler) (IRoute, error) {
	var route IRoute
	var method string
	var path string
	var errMessage = ""
	val := reflect.ValueOf(irisHandler).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Anonymous && typeField.Name == "Handler" {
			tags := strings.Split(strings.TrimSpace(string(typeField.Tag)), " ")
			firstTag := tags[0]

			idx := strings.Index(string(firstTag), ":")

			tagName := strings.ToUpper(string(firstTag[:idx]))
			tagValue, unqerr := strconv.Unquote(string(firstTag[idx+1:]))

			if unqerr != nil {
				errMessage = errMessage + "\niris.HandleAnnotated: Error on getting path: " + unqerr.Error()
				continue
			}

			path = tagValue
			avalaibleMethodsStr := strings.Join(HTTPMethods.ANY, ",")

			if !strings.Contains(avalaibleMethodsStr, tagName) {
				//wrong method passed
				errMessage = errMessage + "\niris.HandleAnnotated: Wrong method passed to the anonymous property iris.Handler -> " + tagName
				continue
			}

			method = tagName

		} else {
			errMessage = "\nError on Iris.HandleAnnotated: Struct passed but it doesn't have an anonymous property of type iris.Hanndler, please refer to docs\n"
		}

	}

	if errMessage == "" {
		route = r.Handle(method, path, irisHandler)
	}

	var err error
	if errMessage != "" {
		err = errors.New(errMessage)
	}

	return route, err
}

///////////////////
//global middleware
///////////////////

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party chosen because it has more fun
func (r *Router) Party(rootPath string) IParty {
	return NewParty(rootPath, r)
}

///////////////////////////////
//expose some methods as public
///////////////////////////////

// Get registers a route for the Get http method
func (r *Router) Get(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.GET, path, handlersFn...)
}

// Post registers a route for the Post http method
func (r *Router) Post(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.POST, path, handlersFn...)
}

// Put registers a route for the Put http method
func (r *Router) Put(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.PUT, path, handlersFn...)
}

// Delete registers a route for the Delete http method
func (r *Router) Delete(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.DELETE, path, handlersFn...)
}

// Connect registers a route for the Connect http method
func (r *Router) Connect(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.CONNECT, path, handlersFn...)
}

// Head registers a route for the Head http method
func (r *Router) Head(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.HEAD, path, handlersFn...)
}

// Options registers a route for the Options http method
func (r *Router) Options(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.OPTIONS, path, handlersFn...)
}

// Patch registers a route for the Patch http method
func (r *Router) Patch(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.PATCH, path, handlersFn...)
}

// Trace registers a route for the Trace http method
func (r *Router) Trace(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc(HTTPMethods.TRACE, path, handlersFn...)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (r *Router) Any(path string, handlersFn ...HandlerFunc) IRoute {
	return r.HandleFunc("", path, handlersFn...)
}

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := r.station.GetPool().Get().(*Context)
	ctx.memoryResponseWriter.New(res)
	ctx.Request = req
	ctx.New()

	//defer r.station.pool.Put(ctx)
	// defer is too slow it adds 10k nanoseconds to the benchmarks...so I will wrap the below to a function
	r.processRequest(ctx)

	r.station.GetPool().Put(ctx)

}

//we use that to the router_memory also
//returns true if it actually find serve something
func (r *Router) processRequest(ctx *Context) bool {
	reqPath := ctx.Request.URL.Path
	method := ctx.Request.Method
	gLen := len(r.garden)
	for i := 0; i < gLen; i++ {
		if r.garden[i].Method == method {

			middleware, params, mustRedirect := r.garden[i].Node.GetBranch(reqPath, ctx.Params) // pass the parameters here for 0 allocation
			if middleware != nil {
				ctx.Params = params
				ctx.middleware = middleware
				ctx.Do()
				return true
			} else if mustRedirect && r.station.options.PathCorrection {
				pathLen := len(reqPath)
				//first of all checks if it's the index only slash /
				if pathLen <= 1 {
					reqPath = "/"
					//check if the req path ends with slash
				} else if reqPath[pathLen-1] == '/' {
					reqPath = reqPath[:pathLen-1] //remove the last /
				} else {
					//it has path prefix, it doesn't ends with / and it hasn't be found, then just add the slash
					reqPath = reqPath + "/"
				}
				ctx.Request.URL.Path = reqPath
				urlToRedirect := ctx.Request.URL.String()
				if u, err := url.Parse(urlToRedirect); err == nil {

					if u.Scheme == "" && u.Host == "" {
						//The http://yourserver is done automatically by all browsers today
						//so just clean the path
						trailing := strings.HasSuffix(urlToRedirect, "/")
						urlToRedirect = path.Clean(urlToRedirect)
						//check after clean if we had a slash but after we don't, we have to do that otherwise we will get forever redirects if path is /home but the registed is /home/
						if trailing && !strings.HasSuffix(urlToRedirect, "/") {
							urlToRedirect += "/"
						}

					}

					ctx.ResponseWriter.Header().Set("Location", urlToRedirect)
					ctx.ResponseWriter.WriteHeader(http.StatusMovedPermanently)

					// RFC2616 recommends that a short note "SHOULD" be included in the
					// response because older user agents may not understand 301/307.
					// Shouldn't send the response for POST or HEAD; that leaves GET.
					if method == HTTPMethods.GET {
						note := "<a href=\"" + htmlEscape(urlToRedirect) + "\">Moved Permanently</a>.\n"
						ctx.Write(note)
					}
					return false
				}
			}
		}
	}
	ctx.NotFound()
	return false
}

var _ IRouter = &Router{}
