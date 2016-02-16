package gapi

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var once sync.Once

type Server struct {
	Options   *ServerConfig
	Router    *Router
	isRunning bool
}

func NewServer() *Server {
	_server := new(Server)
	_server.Options = DefaultHttpConfig()

	return _server
}

//options

func (this *Server) Host(host string) *Server {
	this.Options.Host = host
	return this
}

func (this *Server) Port(port int) *Server {
	this.Options.Port = port
	return this
}

func (this *Server) SetRouter(_router *Router) *Server {
	this.Router = _router
	return this
}

func (this *Server) Start() error {
	this.isRunning = true
	mux := http.NewServeMux()
	mux.Handle("/", this)
	
	return http.ListenAndServe(this.Options.Host+strconv.Itoa(this.Options.Port), mux)
}

func (this *Server) Listen(fullHostOrPort interface{}) error {

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

	return this.Start()

}

///TODO: na kanw kai ta global middleware kai routes, auto 9a ginete me to '*'
func (this *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
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


