package iris

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

///TODO: fix the path if no ending with '/' ? or it must be not ending with '/' but handle requests with last '/' redirect to non '/' ? I will think about it.

type IRouterMethods interface {
	Get(path string, handler HandlerFunc) *Route
	Post(path string, handler HandlerFunc) *Route
	Put(path string, handler HandlerFunc) *Route
	Delete(path string, handler HandlerFunc) *Route
	Connect(path string, handler HandlerFunc) *Route
	Head(path string, handler HandlerFunc) *Route
	Options(path string, handler HandlerFunc) *Route
	Patch(path string, handler HandlerFunc) *Route
	Trace(path string, handler HandlerFunc) *Route
	Any(path string, handler HandlerFunc) *Route
}

type IRouter interface {
	IMiddlewareSupporter
	IRouterMethods
	IPartyHoster
	HandleAnnotated(Handler) (*Route, error)
	Handle(string, string, Handler) *Route
	HandleFunc(string, string, HandlerFunc) *Route
	Errors() *HTTPErrors //at the main Router struct this is managed by the MiddlewareSupporter
	// ServeHTTP finds and serves a route by it's request
	// If no route found, it sends an http status 404
	Build()
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// Router is the router , one router per server.
// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
type Router struct {
	//no routes map[string]map[string][]*Route // key = path prefix, value a map which key = method and the vaulue an array of the routes starts with that prefix and method
	//routes map[string][]*Route // key = path prefix, value an array of the routes starts with that prefix
	MiddlewareSupporter
	station    *Station
	tempTrees  trees
	garden     Garden
	httpErrors *HTTPErrors //the only reason of this is to pass into the route, which it need it to  passed it to Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
}

// NewRouter creates and returns an empty Router
func NewRouter(station *Station) *Router {
	return &Router{station: station, tempTrees: make(trees, 0), garden: make(Garden, len(HTTPMethods.ANY)), httpErrors: DefaultHTTPErrors()}
}

func (r *Router) SetErrors(httperr *HTTPErrors) {
	r.httpErrors = httperr
}

func (r *Router) Errors() *HTTPErrors {
	return r.httpErrors
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//expose common methods as public on the Router, also as global used from global iris server.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Handle registers a route to the server's router
func (r *Router) Handle(method string, registedPath string, handler Handler) *Route {
	if registedPath == "" {
		registedPath = "/"
	}
	if handler == nil {
		panic("Iris.Handle: nil passed as handler!")
	}
	route := newRoute(registedPath, handler)

	if len(r.middlewareHandlers) > 0 {
		//if global middlewares are registed then push them to this route.
		route.middlewareHandlers = r.middlewareHandlers
	}

	r.station.pluginContainer.doPreHandle(method, route)

	r.tempTrees.addRoute(method, route)

	r.station.pluginContainer.doPostHandle(method, route)

	return route
}

// HandleFunc registers and returns a route with a path string, a handler and optinally methods as parameters
// registedPath is the relative url path
// handler is the iris.Handler which you can pass anything you want via iris.HandlerFunc(func(res,req){})... or just use func(c iris.Context),func(r iris.Renderer), func(c Context,r Renderer) or func(res http.ResponseWriter, req *http.Request)
// method is the last parameter, pass the http method ex: "GET","POST".. iris.HTTPMethods.PUT, or empty string to match all methods
func (r *Router) HandleFunc(method string, registedPath string, handler HandlerFunc) *Route {
	return r.Handle(method, registedPath, handler)
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
			//has multi methods seperate by commas

			if !strings.Contains(avalaibleMethodsStr, tagName) {
				//wrong methods passed
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

	var err error = nil
	if errMessage != "" {
		err = errors.New(errMessage)
	}

	return route, err
}

///////////////////
//global middleware
///////////////////

// Use registers a a custom handler, with next, as a global middleware
func (r *Router) Use(handler MiddlewareHandler) {
	r.MiddlewareSupporter.Use(handler)
	//IF this is called after the routes
	if len(r.tempTrees) > 0 {
		for _, _tree := range r.tempTrees {
			for _, _route := range _tree {
				_route.Use(handler)
			}
		}
	}

}

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party choosen because it has more fun
func (r *Router) Party(rootPath string) IParty {
	return newParty(rootPath, r)
}

///////////////////////////////
//expose some methods as public
///////////////////////////////

// Get registers a route for the Get http method
func (r *Router) Get(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.GET, path, handler)
}

// Post registers a route for the Post http method
func (r *Router) Post(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.POST, path, handler)
}

// Put registers a route for the Put http method
func (r *Router) Put(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.PUT, path, handler)
}

// Delete registers a route for the Delete http method
func (r *Router) Delete(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.DELETE, path, handler)
}

// Connect registers a route for the Connect http method
func (r *Router) Connect(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.CONNECT, path, handler)
}

// Head registers a route for the Head http method
func (r *Router) Head(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.HEAD, path, handler)
}

// Options registers a route for the Options http method
func (r *Router) Options(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.OPTIONS, path, handler)
}

// Patch registers a route for the Patch http method
func (r *Router) Patch(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.PATCH, path, handler)
}

// Trace registers a route for the Trace http method
func (r *Router) Trace(path string, handler HandlerFunc) *Route {
	return r.HandleFunc(HTTPMethods.TRACE, path, handler)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (r *Router) Any(path string, handler HandlerFunc) *Route {
	return r.HandleFunc("", path, handler)
}

// Build prepares the routes before Serve
// is beeing called one time before .Listen from the Station
func (r *Router) Build() {
	//prepare the temp routes firsts
	for method, _ := range r.tempTrees {
		for i := 0; i < len(r.tempTrees[method]); i++ {
			r.tempTrees[method][i].prepare()
		}
	}

	//and plant them to the radix tree
	r.garden.plant(r.tempTrees)
	//and clear the trees?
	r.tempTrees = nil
}

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var reqPath = req.URL.Path
	var method = req.Method
	ctx := r.station.pool.Get().(*Context)
	ctx.Request = req
	ctx.ResponseWriter = res
	ctx.Params = ctx.Params[0:0]
	_root := r.garden[method]
	if _root != nil {

		handler, params, _ := _root.getValue(reqPath, ctx.Params) // pass the parameters here for 0 allocation
		if handler != nil {
			ctx.Params = params
			ctx.Renderer.responseWriter = ctx.ResponseWriter
			handler.Serve(ctx)
			r.station.pool.Put(ctx)
			return
		}

	}
	r.httpErrors.NotFound(res)

}
