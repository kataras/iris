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
	Get(path string, handler interface{}) *Route
	Post(path string, handler interface{}) *Route
	Put(path string, handler interface{}) *Route
	Delete(path string, handler interface{}) *Route
	Connect(path string, handler interface{}) *Route
	Head(path string, handler interface{}) *Route
	Options(path string, handler interface{}) *Route
	Patch(path string, handler interface{}) *Route
	Trace(path string, handler interface{}) *Route
	Any(path string, handler interface{}) *Route
}

//the IRouter is IRouteRegisted and a routes serving service.
type IRouter interface {
	IMiddlewareSupporter
	IRouterMethods
	IPartyHoster
	HandleAnnotated(irisHandler Annotated) (*Route, error)
	Handle(params ...interface{}) *Route
	HandleFunc(path string, handler Handler, method string) *Route
	Errors() *HTTPErrors //at the main Router struct this is managed by the MiddlewareSupporter
	// ServeHTTP finds and serves a route by it's request
	// If no route found, it sends an http status 404
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// Router is the router , one router per server.
// Router contains the global middleware, the routes and a Mutex for lock and unlock on route prepare
type Router struct {
	//no routes map[string]map[string][]*Route // key = path prefix, value a map which key = method and the vaulue an array of the routes starts with that prefix and method
	//routes map[string][]*Route // key = path prefix, value an array of the routes starts with that prefix
	MiddlewareSupporter
	station    *Station
	trees      Trees
	cache      *IRouterCache
	httpErrors *HTTPErrors //the only reason of this is to pass into the route, which it need it to  passed it to Context, in order to  developer get the ability to perfom emit errors (eg NotFound) directly from context
}

// NewRouter creates and returns an empty Router
func NewRouter(station *Station) *Router {
	return &Router{station: station, trees: make(Trees, 0), httpErrors: DefaultHTTPErrors()}
}

func (r *Router) SetErrors(httperr *HTTPErrors) {
	r.httpErrors = httperr
}

func (r *Router) Errors() *HTTPErrors {
	return r.httpErrors
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//expose common methods as public on the Router, and the  Server struct, also as global used from global iris server.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// HandleFunc registers and returns a route with a path string, a handler and optinally methods as parameters
// registedPath is the relative url path
// handler is the iris.Handler which you can pass anything you want via iris.HandlerFunc(func(res,req){})... or just use func(c iris.Context),func(r iris.Renderer), func(c Context,r Renderer) or func(res http.ResponseWriter, req *http.Request)
// method is the last parameter, pass the http method ex: "GET","POST".. iris.HTTPMethods.PUT, or empty string to match all methods
func (r *Router) HandleFunc(registedPath string, handler Handler, method string) *Route {
	//r.mu.Lock()
	//defer is 5 times slower only some nanosecconds difference but let's make it faster unlock it at the end of the function manually  or not?
	//defer r.mu.Unlock()
	//but wait... do we need locking here?

	var route *Route
	if registedPath == "" {
		registedPath = "/"
	}

	if handler != nil {
		route = newRoute(registedPath, handler)

		if len(r.middlewareHandlers) > 0 {
			//if global middlewares are registed then push them to this route.
			route.middlewareHandlers = r.middlewareHandlers
		}

		r.trees.addRoute(method, route)

	}
	route.httpErrors = r.httpErrors
	return route
}

// HandleAnnotated registers a route handler using a Struct
// implements Handle() function and has iris.Annotated anonymous property
// which it's metadata has the form of
// `method:"path" template:"file.html"` and returns the route and an error if any occurs
func (r *Router) HandleAnnotated(irisHandler Annotated) (*Route, error) {
	//r.mu.Lock()
	//defer r.mu.Unlock()
	var route *Route
	var method string
	var path string
	var handleFunc reflect.Value
	var template string
	var templateIsGLob = false
	var errMessage = ""
	val := reflect.ValueOf(irisHandler).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Anonymous && typeField.Name == "Annotated" {
			tags := strings.Split(strings.TrimSpace(string(typeField.Tag)), " ")
			//we can have two keys, one is the tag starts with the method (GET,POST: "/user/api/{userId(int)}")
			//and the other if exists is the OPTIONAL TEMPLATE/TEMPLATE-GLOB: "file.html"

			//check for Template first because on the method we break and return error if no method found , for now.
			if len(tags) > 1 {
				secondTag := tags[1]

				templateIdx := strings.Index(string(secondTag), ":")

				templateTagName := strings.ToUpper(string(secondTag[:templateIdx]))

				//check if it's regex pattern

				if templateTagName == "TEMPLATE-GLOB" {
					templateIsGLob = true
				}

				temlateTagValue, templateUnqerr := strconv.Unquote(string(secondTag[templateIdx+1:]))

				if templateUnqerr != nil {
					//err = errors.New(err.Error() + "\niris.RegisterHandler: Error on getting template: " + templateUnqerr.Error())
					errMessage = errMessage + "\niris.HandleAnnotated: Error on getting template: " + templateUnqerr.Error()

					continue
				}

				template = temlateTagValue
			}

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
				errMessage = errMessage + "\niris.HandleAnnotated: Wrong methods passed to Handler -> " + tagName
				continue
			}
			//it is single 'GET','POST' .... method
			method = tagName

		} else {
			errMessage = "\nError on Iris on HandleAnnotated: Struct passed but it doesn't have an anonymous property of type iris.Annotated, please refer to docs\n"
		}

	}

	if errMessage == "" {

		//now check/get the Handle method from the irisHandler 'obj'.
		handleFunc = reflect.ValueOf(irisHandler).MethodByName("Handle")
		if !handleFunc.IsValid() {
			errMessage = "Missing Handle function inside iris.Annotated"
		}

		if errMessage == "" {
			route = r.HandleFunc(path, convertToHandler(handleFunc.Interface()), method)
			//check if template string has stored by the tag ( look before this block )

			if template != "" {
				if templateIsGLob {
					route.Template().SetGlob(template)
				} else {
					route.Template().Add(template)
				}

			}
		}

	}

	var err error = nil
	if errMessage != "" {
		err = errors.New(errMessage)
	}

	return route, err
}

// Handle registers a route to the server's router, pass a struct -implements iris.Annotated as parameter
// Or pass just a http.Handler or TypicalHandlerFunc or ContextedHandlerFunc or  RendereredHandlerFunc or ContextedRendererHandlerFunc or already an iris.Handler
func (r *Router) Handle(params ...interface{}) *Route {
	paramsLen := len(params)
	if paramsLen == 0 {
		panic("No arguments given to the Handle function, please refer to docs")
	}

	if reflect.TypeOf(params[0]).Kind() == reflect.String {
		//means first parameter is the path, wich means it is a simple path with handler -> HandleFunc and method
		// means: http.Handler or TypicalHandlerFunc or ContextedHandlerFunc or  RendereredHandlerFunc or ContextedRendererHandlerFunc or already an iris.Handler
		return r.HandleFunc(params[0].(string), convertToHandler(params[1]), params[2].(string))
	} else {
		//means it's a struct which implements the iris.Annotated and have a Handle func inside it -> handleAnnotated
		route, err := r.HandleAnnotated(params[0].(Annotated))
		if err != nil {
			panic(err.Error())
		}
		return route
	}
}

///////////////////
//global middleware
///////////////////

// Use registers a a custom handler, with next, as a global middleware
func (r *Router) Use(handler MiddlewareHandler) {
	r.MiddlewareSupporter.Use(handler)
	//IF this is called after the routes
	if len(r.trees) > 0 {
		for _, _tree := range r.trees {
			for _, _branch := range _tree {
				for _, route := range _branch.routes {
					route.Use(handler)
				}

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
func (r *Router) Get(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.GET)
}

// Post registers a route for the Post http method
func (r *Router) Post(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.POST)
}

// Put registers a route for the Put http method
func (r *Router) Put(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.PUT)
}

// Delete registers a route for the Delete http method
func (r *Router) Delete(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.DELETE)
}

// Connect registers a route for the Connect http method
func (r *Router) Connect(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.CONNECT)
}

// Head registers a route for the Head http method
func (r *Router) Head(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.HEAD)
}

// Options registers a route for the Options http method
func (r *Router) Options(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.OPTIONS)
}

// Patch registers a route for the Patch http method
func (r *Router) Patch(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.PATCH)
}

// Trace registers a route for the Trace http method
func (r *Router) Trace(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), HTTPMethods.TRACE)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (r *Router) Any(path string, handler interface{}) *Route {
	return r.HandleFunc(path, convertToHandler(handler), "")
}

// ServeHTTP finds and serves a route by it's request
// If no route found, it sends an http status 404
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	var _branch *branch
	var _route *Route
	var _method = req.Method

search:
	{
		isHead := _method == "HEAD"
		_tree := r.trees[_method]
		if _tree != nil {
			for i := 0; i < len(_tree); i++ {
				_branch = _tree[i]
				if len(req.URL.Path) < len(_branch.prefix) {
					continue
				}
				hasPrefix := req.URL.Path[0:len(_branch.prefix)] == _branch.prefix
				//println("check url prefix: ", req.URL.Path[0:len(_branch.prefix)]+" with node's:  ", _branch.prefix)
				if hasPrefix {
					for j := 0; j < len(_branch.routes); j++ {
						_route = _branch.routes[j]
						if !_route.Verify(req.URL.Path) {
							continue

						}
						_route.ServeHTTP(res, req)
						return

					}
					//if the prefix was found, so we are 100% at the correct node, because I make it to end with slashes
					//so no need to check other prefixes any more, just return 404 and exit from here.
					//println(req.URL.Path, " NOT found")

					//if prefix found on head but no route no route found, then search to the GET tree also
					if isHead {
						_method = HTTPMethods.GET
						goto search
					}
					r.httpErrors.NotFound(res)
					return

				}

			}
		} else if isHead { //if no any branches with routes found for the HEAD then try to search on GET tree
			_method = HTTPMethods.GET
			goto search
		}
	}
	//nothing found, usualy if _tree == nil and branches for this method not found
	//println(req.URL.Path, " NOT found")
	r.httpErrors.NotFound(res)
}
