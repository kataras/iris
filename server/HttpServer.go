package server

import (
	"fmt"
	. "github.com/kataras/gapi/router"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

///TODO: na mhn ksexasw sto use na valw an 9elei mono sigekrimea methods opws px GET,POST,PUT,DELETE ktlp
type HttpServer struct {
	Options     *HttpServerConfig
	middlewares map[string]http.Handler
	isRunning   bool
	mux         *http.ServeMux
}

func NewHttpServer() *HttpServer {
	_server := new(HttpServer)
	_server.Options = DefaultHttpConfig()
	_server.mux = http.NewServeMux()
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

func (this *HttpServer) Router(router interface{}) *HttpServer {
	var obj = reflect.ValueOf(&router)
	switch obj.Elem().Interface().(type) {
	case *HttpRouter:
		this.Options.Router = router.(*HttpRouter)
	case *HttpRouterBuilder:
		this.Options.Router = router.(*HttpRouterBuilder).Build()
	default:
		fmt.Println("Please use gapi.NewRouter().If('/path').Then(handle) \nOr pass the gapi.router.NewHttpRouter() ")

	}

	return this
}

func (this *HttpServer) Start() {

	//http.Handle("/", loggerMiddleware.Log(testHandler()))//new(SecondHandler)))
	this.initialize()
	this.isRunning = true
	//OR for custom handle errors, css files and e.t.c	this.mux.Handle("/", this.Options.Router.Middleware())

	if this.Options.Router != nil && this.Options.Router.Routes != nil {
		for _key, _val := range this.Options.Router.Routes {
			//fmt.Println("setting route : ",_key)
			this.mux.Handle(_key, _val)
		}
	}
	//, this.Options.Router.Middleware()
	fmt.Println("Server is running at ", this.Options.Host+":"+strconv.Itoa(this.Options.Port))
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

func (this *HttpServer) Use(urlPath string, handler http.Handler) *HttpServer {
	if urlPath == "" {
		urlPath = "/"
	}
	this.middlewares[urlPath] = handler
	if this.isRunning {
		this.handle(urlPath, handler)
	}
	return this
}
