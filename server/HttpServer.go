package server

import (
	"github.com/kataras/gapi/middleware"
	"github.com/kataras/gapi/router"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

///TODO: na mhn ksexasw sto use na valw an 9elei mono sigekrimea methods opws px GET,POST,PUT,DELETE ktlp
type HttpServer struct {
	Options     *HttpServerConfig
	middlewares map[string]*middleware.Middleware
	isRunning   bool
	mux         *http.ServeMux
}

func NewHttpServer() *HttpServer {
	_server := new(HttpServer)
	_server.Options = DefaultHttpConfig()
	_server.mux = http.DefaultServeMux
	_server.middlewares = make(map[string]*middleware.Middleware)
	return _server
}

//options

func (this *HttpServer) Host(host string) *HttpServer {
	this.Options.Host = host
	return this
}

func (this *HttpServer) Port(port int) *HttpServer {
	this.Options.Port = port
	return this
}

func (this *HttpServer) Router(_router interface{}) *HttpServer {
	var obj = reflect.ValueOf(&_router)
	switch obj.Elem().Interface().(type) {
	case *router.HttpRouter:
		this.Options.Router = _router.(*router.HttpRouter)
	case *router.HttpRouterBuilder:
		this.Options.Router = _router.(*router.HttpRouterBuilder).Build()
	default:
		panic("Please use gapi.NewRouter().If('/path').Then(handle) \nOr pass the gapi.router.NewHttpRouter() ")

	}

	return this
}

func (this *HttpServer) Start() {
	this.initialize()
	this.isRunning = true

	//OR for custom handle errors, css files and e.t.c	this.mux.Handle("/", this.Options.Router.Middleware())

	if this.Options.Router != nil && this.Options.Router.Routes != nil {

		this.mux.Handle("/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			var route = this.Options.Router.Routes[req.URL.Path]

			if route == nil {
				///TODO: return custom, defined by developer not found handler.
				http.NotFound(res, req)
			} else if route.Method == req.Method {
				if _middleware := this.middlewares[req.URL.Path]; _middleware != nil && _middleware.Method == route.Method{
					var last http.Handler = http.HandlerFunc(route.Handler)
					for i := len(_middleware.Handlers) - 1; i >= 0; i-- {
						last = _middleware.Handlers[i](last)
					}
					last.ServeHTTP(res, req)

				} else {
					route.Handler(res, req)
				}

			} else {
				http.Error(res, "Error 405  Method Not Allowed", 405)
			}
		}))
	}

	//fmt.Println("Server is running at ", this.Options.Host+":"+strconv.Itoa(this.Options.Port))
	//this.Mux or ?, this.Options.Router.Middleware()

	http.ListenAndServe(this.Options.Host+strconv.Itoa(this.Options.Port), this.mux)

}

func (this *HttpServer) Listen(fullHostOrPort interface{}) {

	switch reflect.ValueOf(fullHostOrPort).Interface().(type) {
	case string:
		options := strings.Split(fullHostOrPort.(string), ":")

		if strings.TrimSpace(options[0]) != "" {
			this.Options.Host = options[0]
		}

		if len(options) > 1 {
			this.Options.Port, _ = strconv.Atoi(options[1])
		}
	default:
		this.Options.Port = fullHostOrPort.(int)
	}

	this.Start()

}

func (this *HttpServer) initialize() {

}

func (this *HttpServer) handle(urlPath string, handler http.Handler) *HttpServer {
	http.Handle(urlPath, handler)
	return this
}

func (this *HttpServer) Unuse(urlPath string) *HttpServer {
	this.middlewares[urlPath] = nil //remove handler
	return this
}

func (this *HttpServer) Use(urlPath string, handlers ...middleware.Handler) *HttpServer {
	if urlPath == "" {
		urlPath = "*" //Future: All registed routes
	}
	this.middlewares[urlPath] = &middleware.Middleware{Method: router.HttpMethods.GET, Handlers: handlers}
	return this
}

//Future
func (this *HttpServer) UseGlobal(handlers ...middleware.Handler) *HttpServer {
	//Means to all paths, not only registed routes but all paths the user will try to request from server, this is good for logging purpose.
	this.middlewares["**"] = &middleware.Middleware{Method: router.HttpMethods.GET, Handlers: handlers}
	return this

}
