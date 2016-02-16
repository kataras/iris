package gapi

//This file just exposes the server and it's router & middlewares
import (
	"net/http"
	"reflect"
	"strings"
)

var (
	avalaibleMethodsStr = strings.Join(HTTPMethods.ANY, ",")
	mainGapi            *Server
)

//only one init to the whole package
func init() {
	//Context.go
	contextType = reflect.TypeOf(Context{})
	//Renderer.go
	rendererType = reflect.TypeOf(Renderer{})
	//TemplateCache.go
	templatesDirectory = getCurrentDir()

	mainGapi = nil //I don't want to store in the memory a New() gapi because user maybe wants to use the form of api := gapi.New(); api.Get... instead of gapi.Get..
}

func New() *Server {
	return NewServer()
}

/* GLOBAL MIDDLEWARE */

func (this *Server) Use(handler MiddlewareHandler) *Server {
	this.Router.Use(handler)
	return this
}

func (this *Server) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Server {
	this.Router.UseFunc(handlerFunc)
	return this
}

func (this *Server) UseHandler(handler http.Handler) *Server {
	this.Router.UseHandler(handler)
	return this
}

/* ROUTER */
/*func (this *Gapi) Route(path string, handler HTTPHandler) *Route {

	return this.server.Router.Route(path, handler)
}*/

//path string, handler HTTPHandler OR
//any struct implements the custom gapi Handler interface.
func (this *Server) Handle(params ...interface{}) *Route {
	//poor, but means path, custom HTTPhandler
	if len(params) == 2 {
		return this.Router.Handle(params[0].(string), params[1].(HTTPHandler))
	} else {
		route, err := this.RegisterHandler(params[0].(Handler))

		if err != nil {
			panic(err.Error())
		}

		return route

	}
}

func (this *Server) Get(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.GET)
}

func (this *Server) Post(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.POST)
}

func (this *Server) Put(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.PUT)
}

func (this *Server) Delete(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.DELETE)
}

func (this *Server) Connect(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.CONNECT)
}

func (this *Server) Head(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.HEAD)
}

func (this *Server) Options(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.OPTIONS)
}

func (this *Server) Patch(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.PATCH)
}

func (this *Server) Trace(path string, handler HTTPHandler) *Route {
	return this.Router.Handle(path, handler, HTTPMethods.TRACE)
}

func (this *Server) RegisterHandler(gapiHandler Handler) (*Route, error) {
	return this.Router.RegisterHandler(gapiHandler)
}

/////////////////////////////////
//for standalone instance of gapi
/////////////////////////////////

func check() {
	if mainGapi == nil {
		mainGapi = New()
	}
}

/* ServeHTTP, use as middleware only in already http server. */
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	check()
	mainGapi.ServeHTTP(res, req)
}

func Listen(fullHostOrPort interface{}) error {
	check()
	return mainGapi.Listen(fullHostOrPort)
}

func Use(handler MiddlewareHandler) *Server {
	check()
	mainGapi.Router.Use(handler)
	return mainGapi
}

func UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Server {
	check()
	mainGapi.Router.UseFunc(handlerFunc)
	return mainGapi
}

func UseHandler(handler http.Handler) *Server {
	check()
	mainGapi.Router.UseHandler(handler)
	return mainGapi
}

func Handle(params ...interface{}) *Route {
	check()
	return mainGapi.Handle(params...)

}

func Get(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Get(path, handler)
}

func Post(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Post(path, handler)
}

func Put(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Put(path, handler)
}

func Delete(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Delete(path, handler)
}

func Connect(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Connect(path, handler)
}

func Head(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Head(path, handler)
}

func Options(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Options(path, handler)
}

func Patch(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Patch(path, handler)
}

func Trace(path string, handler HTTPHandler) *Route {
	check()
	return mainGapi.Trace(path, handler)
}

func RegisterHandler(gapiHandler Handler) (*Route, error) {
	check()
	return mainGapi.RegisterHandler(gapiHandler)
}

func Close() { mainGapi.Close() }
