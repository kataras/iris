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
	mainGapi            *Gapi
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


type Gapi struct {
	server *Server
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

func (this *Gapi) Listen(fullHostOrPort interface{}) error {
	return this.server.Listen(fullHostOrPort)
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
/*func (this *Gapi) Route(path string, handler HTTPHandler) *Route {

	return this.server.Router.Route(path, handler)
}*/

//path string, handler HTTPHandler OR
//any struct implements the custom gapi Handler interface.
func (this *Gapi) Handle(params ...interface{}) *Route {
		//poor, but means path, custom HTTPhandler
	if len(params) == 2 {
		return this.server.Router.Handle(params[0].(string), params[1].(HTTPHandler))
	} else {
		route, err := this.RegisterHandler(params[0].(Handler))

		if err != nil {
			panic(err.Error())
		}

		return route

	}
}

func (this *Gapi) Get(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.GET)
}

func (this *Gapi) Post(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.POST)
}

func (this *Gapi) Put(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.PUT)
}

func (this *Gapi) Delete(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.DELETE)
}

func (this *Gapi) Connect(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.CONNECT)
}

func (this *Gapi) Head(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.HEAD)
}

func (this *Gapi) Options(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.OPTIONS)
}

func (this *Gapi) Patch(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.PATCH)
}

func (this *Gapi) Trace(path string, handler HTTPHandler) *Route {
	return this.server.Router.Handle(path, handler, HTTPMethods.TRACE)
}

func (this *Gapi) RegisterHandler(gapiHandler Handler) (*Route, error) {
	var route *Route
	var methods []string
	var path string
	var handleFunc reflect.Value
	var template string
	var templateIsGLob bool = false
	var err error = errors.New("")
	val := reflect.ValueOf(gapiHandler).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Name == "Handler" {
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
					err = errors.New(err.Error() + "\ngapi.RegisterHandler: Error on getting template: " + templateUnqerr.Error())
					continue
				}

				template = temlateTagValue
			}

			firstTag := tags[0]

			idx := strings.Index(string(firstTag), ":")

			tagName := strings.ToUpper(string(firstTag[:idx]))
			tagValue, unqerr := strconv.Unquote(string(firstTag[idx+1:]))

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
		//route = this.server.Router.Route(path, gapiHandler.Handle, methods...)

		//now check/get the Handle method from the gapiHandler 'obj'.
		handleFunc = reflect.ValueOf(gapiHandler).MethodByName("Handle")

		if !handleFunc.IsValid() {
			err = errors.New("Missing Handle function inside gapi.Handler")
		}

		if err == nil {
			route = this.server.Router.Handle(path, handleFunc.Interface(), methods...)
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

	return route, err
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
	mainGapi.server.ServeHTTP(res, req)
}


func Listen(fullHostOrPort interface{}) error {
	check()
	return mainGapi.server.Listen(fullHostOrPort)
}

func Use(handler MiddlewareHandler) *Gapi {
	check()
	mainGapi.server.Router.Use(handler)
	return mainGapi
}

func UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) *Gapi {
	check()
	mainGapi.server.Router.UseFunc(handlerFunc)
	return mainGapi
}

func UseHandler(handler http.Handler) *Gapi {
	check()
	mainGapi.server.Router.UseHandler(handler)
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
