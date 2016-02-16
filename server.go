package iris

import (
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var once sync.Once

type Server struct {
	Config      *ServerConfig
	Router      *Router
	listenerTCP net.Listener
	isRunning   bool
}

func NewServer() *Server {
	_server := new(Server)
	_server.Config = DefaultServerConfig()
	_server.Router = NewRouter()
	return _server
}

//options/config

func (this *Server) Host(host string) *Server {
	this.Config.Host = host
	return this
}

func (this *Server) Port(port int) *Server {
	this.Config.Port = port
	return this
}

func (this *Server) SetRouter(_router *Router) *Server {
	this.Router = _router
	return this
}

func (this *Server) Start() error {

	mux := http.NewServeMux()
	mux.Handle("/", this)

	//return http.ListenAndServe(this.Config.Host+strconv.Itoa(this.Config.Port), mux)
	fullAddr := this.Config.Host + ":" + strconv.Itoa(this.Config.Port)
	listener, err := net.Listen("tcp", fullAddr)

	if err != nil {
		panic("Cannot run the server [problem with tcp listener on host:port]: " + fullAddr)
	}

	this.listenerTCP = listener //we need this because I think that we have to 'have' a Close() method on the server instance
	err = http.Serve(this.listenerTCP, mux)
	this.listenerTCP.Close()
	this.isRunning = true
	return err
}

func (this *Server) Close() {
	if this.isRunning && this.listenerTCP != nil {
		this.listenerTCP.Close()
		this.isRunning = false
	}
}

func (this *Server) Listen(fullHostOrPort interface{}) error {

	switch reflect.ValueOf(fullHostOrPort).Interface().(type) {
	case string:
		config := strings.Split(fullHostOrPort.(string), ":")

		if strings.TrimSpace(config[0]) != "" {
			this.Config.Host = config[0]
		}

		if len(config) > 1 {
			this.Config.Port, _ = strconv.Atoi(config[1])
		}
	default:
		this.Config.Port = fullHostOrPort.(int)
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
