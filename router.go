package iris

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// IRouterMethods is the interface for method routing
type IRouterMethods interface {
	Get(path string, handlersFn ...HandlerFunc) *Route
	Post(path string, handlersFn ...HandlerFunc) *Route
	Put(path string, handlersFn ...HandlerFunc) *Route
	Delete(path string, handlersFn ...HandlerFunc) *Route
	Connect(path string, handlersFn ...HandlerFunc) *Route
	Head(path string, handlersFn ...HandlerFunc) *Route
	Options(path string, handlersFn ...HandlerFunc) *Route
	Patch(path string, handlersFn ...HandlerFunc) *Route
	Trace(path string, handlersFn ...HandlerFunc) *Route
	Any(path string, handlersFn ...HandlerFunc) *Route
}

// IRouter is the interface of which any Iris router must implement
type IRouter interface {
	IMiddlewareSupporter
	IRouterMethods
	IPartyHoster
	HandleAnnotated(Handler) (*Route, error)
	Handle(string, string, ...Handler) *Route
	HandleFunc(string, string, ...HandlerFunc) *Route
	Errors() *HTTPErrors //at the main Router struct this is managed by the MiddlewareSupporter
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
	httpErrors *HTTPErrors //the only reason of this is to pass into the route, which it need it to  passed it to Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
}

// NewRouter creates and returns an empty Router
func NewRouter(station *Station) *Router {
	return &Router{station: station, garden: make(Garden, len(HTTPMethods.ANY)), httpErrors: DefaultHTTPErrors()}
}

// SetErrors sets a HTTPErrors object to the router
func (r *Router) SetErrors(httperr *HTTPErrors) {
	r.httpErrors = httperr
}

// Errors get the HTTPErrors from the router
func (r *Router) Errors() *HTTPErrors {
	return r.httpErrors
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//expose common methods as public on the Router, also as global used from global iris server.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Handle registers a route to the server's router
func (r *Router) Handle(method string, registedPath string, handlers ...Handler) *Route {
	if registedPath == "" {
		registedPath = "/"
	}
	if len(handlers) == 0 {
		panic("Iris.Handle: zero handler to " + method + ":" + registedPath)
	}

	if len(r.middleware) > 0 {
		//if global middlewares are registed then push them to this route.
		handlers = append(r.middleware, handlers...)
	}

	route := newRoute(registedPath, handlers)

	r.station.pluginContainer.doPreHandle(method, route)

	r.garden.plant(method, route)

	r.station.pluginContainer.doPostHandle(method, route)

	return route
}

// HandleFunc registers and returns a route with a method string, path string and a handler
// registedPath is the relative url path
// handler is the iris.Handler which you can pass anything you want via iris.ToHandlerFunc(func(res,req){})... or just use func(c *iris.Context)
func (r *Router) HandleFunc(method string, registedPath string, handlersFn ...HandlerFunc) *Route {
	return r.Handle(method, registedPath, convertToHandlers(handlersFn)...)
}

// HandleAnnotated registers a route handler using a Struct implements iris.Handler (as anonymous property)
// which it's metadata has the form of
// `method:"path"` and returns the route and an error if any occurs
// handler is passed by func(urstruct MyStruct) Serve(ctx *Context) {}
func (r *Router) HandleAnnotated(irisHandler Handler) (*Route, error) {
	var route *Route
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
// Party can also be named as 'Join' or 'Node' or 'Group' , Party choosen because it has more fun
func (r *Router) Party(rootPath string) IParty {
	return newParty(rootPath, r)
}

///////////////////////////////
//expose some methods as public
///////////////////////////////

// Get registers a route for the Get http method
func (r *Router) Get(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.GET, path, handlersFn...)
}

// Post registers a route for the Post http method
func (r *Router) Post(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.POST, path, handlersFn...)
}

// Put registers a route for the Put http method
func (r *Router) Put(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.PUT, path, handlersFn...)
}

// Delete registers a route for the Delete http method
func (r *Router) Delete(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.DELETE, path, handlersFn...)
}

// Connect registers a route for the Connect http method
func (r *Router) Connect(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.CONNECT, path, handlersFn...)
}

// Head registers a route for the Head http method
func (r *Router) Head(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.HEAD, path, handlersFn...)
}

// Options registers a route for the Options http method
func (r *Router) Options(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.OPTIONS, path, handlersFn...)
}

// Patch registers a route for the Patch http method
func (r *Router) Patch(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.PATCH, path, handlersFn...)
}

// Trace registers a route for the Trace http method
func (r *Router) Trace(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.TRACE, path, handlersFn...)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (r *Router) Any(path string, handlersFn ...HandlerFunc) *Route {
	return r.HandleFunc("", path, handlersFn...)
}

//we use that to the router_memory also
func (r *Router) poolContextFor(res http.ResponseWriter, req *http.Request) *Context {
	ctx := r.station.pool.Get().(*Context)
	ctx.clear()

	ctx.Request = req
	ctx.ResponseWriter = res
	return ctx
}

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	ctx := r.poolContextFor(res, req)
	//defer r.station.pool.Put(ctx)
	// defer is too slow it adds 10k nanoseconds to the benchmarks...so I will wrap the below to a function
	r.processRequest(ctx)

	r.station.pool.Put(ctx)

}

//we use that to the router_memory also
//returns true if it actualy find serve something
func (r *Router) processRequest(ctx *Context) bool {
	_root := r.garden[ctx.Request.Method]
	if _root != nil {

		middleware, params, _ := _root.getValue(ctx.Request.URL.Path, ctx.Params) // pass the parameters here for 0 allocation
		if middleware != nil {
			ctx.Params = params
			ctx.middleware = middleware
			///TODO: fix this shit
			ctx.Renderer.responseWriter = ctx.ResponseWriter
			ctx.do()

			return true
		}

	}
	ctx.NotFound()
	return false
}
