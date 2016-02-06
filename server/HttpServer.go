package server

import (
	"github.com/kataras/gapi/router"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var once sync.Once

type HttpServer struct {
	Options   *HttpServerConfig
	Router    *router.HttpRouter
	isRunning bool
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

///TODO: na kanw kai ta global middleware kai routes, auto 9a ginete me to '*'
func (this *HttpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//var route = this.Router.Routes[req.URL.Path]

	var route, errCode = this.Router.Find(req)

	if errCode > 0 {
		switch errCode {
		case 405:
			http.Error(res, "Error 405  Method Not Allowed", 405)

		default:
			http.NotFound(res, req)
		}
	} else {
		/*var last http.Handler = http.HandlerFunc(route.Handler)
		for i := len(this.middlewares) - 1; i >= 0; i-- {
			last = this.middlewares[i](last)
		}
		last.ServeHTTP(res, req)*/

		//this.middleware.ServeHTTP(res,req)
		//and after middlewares executed, run
		//edw omws to next an dn kaleite tote auto to route
		//kanei execute alla to 9ema einai na min kanei
		//an kapio middleware den to pei
		//me auta p ekana ws twra mono metaksu tous ta middleware
		//apofasizoun an 9a ginei next i oxi sto epomeno middleware
		//oxi sto route omws..
		//xmm na to dw...
		//route.Handler(res,req)
		route.ServeHTTP(res, req)
	}

}
