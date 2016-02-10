package gapi

//This file just exposes the server and it's router & middlewares
import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

var (
	avalaibleMethodsStr = strings.Join(HTTPMethods.ALL, ",")
)

func NewRouter() *HTTPRouter {
	return NewHTTPRouter()
}

func NewServer() *HTTPServer {
	return NewHTTPServer()
}

type Gapi struct {
	server *HTTPServer
}

func New() *Gapi {
	theServer := NewServer()
	theServer.SetRouter(NewRouter())
	return &Gapi{server: theServer}
}

/* ServeHTTP, use as middleware only in already http server. */
func (this *Gapi) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	this.server.ServeHTTP(res, req)
}

/* STANDALONE SERVER */

func (this *Gapi) Listen(fullHostOrPort interface{}) *HTTPServer {
	this.server.Listen(fullHostOrPort)
	return this.server
}

/* GLOBAL MIDDLEWARE(S) */

func (this *Gapi) Use(handler MiddlewareHandler) *Gapi {
	this.server.Router.Use(handler)
	return this
}

func (this *Gapi) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Gapi {
	this.server.Router.UseFunc(handlerFunc)
	return this
}

func (this *Gapi) UseHandler(handler http.Handler) *Gapi {
	this.server.Router.UseHandler(handler)
	return this
}

/* ROUTER */
func (this *Gapi) Route(path string, handler HTTPHandler) *HTTPRoute {

	return this.server.Router.Route(path, handler)
}
//same as Gapi.Route
func (this *Gapi) Handle(path string, handler HTTPHandler) *HTTPRoute {
	return this.Route(path, handler)
}

func (this *Gapi) Get(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.GET)
}

func (this *Gapi) Post(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.POST)
}

func (this *Gapi) Put(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.PUT)
}

func (this *Gapi) Delete(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.DELETE)
}

func (this *Gapi) Connect(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.CONNECT)
}

func (this *Gapi) Head(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.HEAD)
}

func (this *Gapi) Options(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.OPTIONS)
}

func (this *Gapi) Patch(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.PATCH)
}

func (this *Gapi) Trace(path string, handler HTTPHandler) *HTTPRoute {
	return this.server.Router.Route(path, handler, HTTPMethods.TRACE)
}

func (this *Gapi) RegisterHandler(gapiHandler Handler) error {

	var methods []string
	var path string
	var err error = errors.New("")

	val := reflect.ValueOf(gapiHandler).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Name == "Handler" {
			tag := typeField.Tag

			idx := strings.Index(string(tag), ":")

			tagName := string(tag[:idx])
			tagValue, unqerr := strconv.Unquote(string(tag[idx+1:]))

			if unqerr != nil {
				err = errors.New(err.Error() + "\ngapi.RegisterHandler: Error on getting path: " + unqerr.Error())
				continue
			}
			
			path = tagValue

			if strings.Index(tagName, ",") != -1 {
				//has multi methods seperate by commas

				if !strings.Contains(avalaibleMethodsStr, tagName) {
					//wrong methods passed
					err = errors.New(err.Error() + "\ngapi.RegisterHandler: Wrong methods passed to Handler -> " + tagName)
					continue
				}

				methods = strings.Split(tagName, ",")
				err = nil
				break
			} else {
				//it is single 'GET','POST' .... method
				methods = []string{tagName}
				err = nil
				break

			}


		}

	}
	
	if err == nil {
		this.server.Router.Route(path, gapiHandler.Handle, methods...)
	}

	return err
}
