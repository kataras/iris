package server

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/kataras/gapi/router"
)

///TODO: na mhn ksexasw sto use na valw an 9elei mono sigekrimea methods opws px GET,POST,PUT,DELETE ktlp
type Middleware func(http.Handler) http.Handler

type HttpServer struct {
	Options     *HttpServerConfig
	Router      *router.HttpRouter
	middlewares []Middleware
	isRunning   bool
}

func NewHttpServer() *HttpServer {
	_server := new(HttpServer)
	_server.Options = DefaultHttpConfig()

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

func (this *HttpServer) SetRouter(_router *router.HttpRouter) *HttpServer {
	this.Router = _router
	return this
}

func (this *HttpServer) Start() {
	this.isRunning = true
	http.ListenAndServe(this.Options.Host+strconv.Itoa(this.Options.Port), this)
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

func (this *HttpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//var route = this.Router.Routes[req.URL.Path]
	var route = this.Router.Find(req)
	if route == nil {
		///TODO: return custom, defined by developer not found handler.
		http.NotFound(res, req)
	} else if route.Method == req.Method {
		//if this.Router.Match(req.URL.Path, route) {

		//}
		var last http.Handler = http.HandlerFunc(route.Handler)
		for i := len(this.middlewares) - 1; i >= 0; i-- {
			last = this.middlewares[i](last)
		}
		last.ServeHTTP(res, req)

	} else {
		http.Error(res, "Error 405  Method Not Allowed", 405)
	}
}

func (this *HttpServer) Use(handlers ...Middleware) *HttpServer {
	if len(handlers) > 0 {
		this.middlewares = append(this.middlewares, handlers...)
	}

	return this
}
