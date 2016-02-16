package iris

//This file just exposes the server and it's router & middlewares
import (
	"net/http"
	"reflect"
	"strings"
)

var (
	avalaibleMethodsStr = strings.Join(HTTPMethods.ANY, ",")
	mainIris            *Server
)

//only one init to the whole package
func init() {
	//Context.go
	contextType = reflect.TypeOf(Context{})
	//Renderer.go
	rendererType = reflect.TypeOf(Renderer{})
	//TemplateCache.go
	templatesDirectory = getCurrentDir()

	mainIris = nil //I don't want to store in the memory a New() Iris because user maybe wants to use the form of api := Iris.New(); api.Get... instead of Iris.Get..
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
/*func (this *Iris) Route(path string, handler HTTPHandler) *Route {

	return this.server.Router.Route(path, handler)
}*/

//path string, handler HTTPHandler OR
//any struct implements the custom Iris Handler interface.
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

func (this *Server) RegisterHandler(irisHandler Handler) (*Route, error) {
	return this.Router.RegisterHandler(irisHandler)
}

/////////////////////////////////
//for standalone instance of iris
/////////////////////////////////

func check() {
	if mainIris == nil {
		mainIris = New()
	}
}

/* ServeHTTP, use as middleware only in already http server. */
func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	check()
	mainIris.ServeHTTP(res, req)
}

func Listen(fullHostOrPort interface{}) error {
	check()
	return mainIris.Listen(fullHostOrPort)
}

func Use(handler MiddlewareHandler) *Server {
	check()
	mainIris.Router.Use(handler)
	return mainIris
}

func UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Server {
	check()
	mainIris.Router.UseFunc(handlerFunc)
	return mainIris
}

func UseHandler(handler http.Handler) *Server {
	check()
	mainIris.Router.UseHandler(handler)
	return mainIris
}

func Handle(params ...interface{}) *Route {
	check()
	return mainIris.Handle(params...)

}

func Get(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Get(path, handler)
}

func Post(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Post(path, handler)
}

func Put(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Put(path, handler)
}

func Delete(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Delete(path, handler)
}

func Connect(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Connect(path, handler)
}

func Head(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Head(path, handler)
}

func Options(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Options(path, handler)
}

func Patch(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Patch(path, handler)
}

func Trace(path string, handler HTTPHandler) *Route {
	check()
	return mainIris.Trace(path, handler)
}

func RegisterHandler(irisHandler Handler) (*Route, error) {
	check()
	return mainIris.RegisterHandler(irisHandler)
}

func Close() { mainIris.Close() }
