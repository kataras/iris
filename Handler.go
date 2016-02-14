package gapi

import (
	"reflect"
)

/*Example

import (
	"github.com/kataras/gapi"
)

type UserHandler struct {
	gapi.Handler `GET:"/api/users/{userId(int)}"`
}

func (u *UserHandler) Handle(ctx *gapi.Context) {
//or
//Handle(ctx *Context, renderer *Renderer)
//Handle(res http.ResponseWriter, req *http.Request)
//Handle(renderer *Renderer)

	defer ctx.Close()
	var userId, _ = ctx.ParamInt("userId")

	println(userId)

}

...

api:= gapi.New()
registerError := api.RegisterHandler(new(UserHandler))

*/

type Handler interface {
	/*must provide one of those: 
	Handle(ctx *Context, renderer *Renderer)
	Handle(res http.ResponseWriter, req *http.Request)
	Handle(ctx *Context)
	Handle(renderer *Renderer)
	
	golang does'nt provide a way for overloading methods, and I can't find a quick solution right now for this
	//...interface doesn't work it gives me a runtime panic exception*
	because of this I will get the Handle method via reflect inside the gapi.go -> RegisterHandler
	this will runs before the server gets up only once, so I don't think this is a problem for now.
	Ofc using reflection too much is not a good idea...
	*/
}
//
type HTTPHandler interface{} //func(...interface{}) //[]reflect.Value

//4 possibilities
//1. http.ResponseWriter, *http.Request
//2. *gapi.Context
//3. *gapi.Renderer
//4. *gapi.Context, *gapi.Renderer

//check the first parameter only for Context
//check if the handler needs a Context , has the first parameter as type of *Context
//use in NewHTTPRoute inside HTTPRoute.go
func hasContextParam(handlerType reflect.Type) bool {
	//if the handler doesn't take arguments, false
	if handlerType.NumIn() == 0 {
		return false
	}

	//if the first argument is not a pointer, false
	p1 := handlerType.In(0)
	if p1.Kind() != reflect.Ptr {
		return false
	}
	//but if the first argument is a context, true
	if p1.Elem() == contextType {
		return true
	}

	return false
}

//check the first parameter only for Renderer
func hasRendererParam(handlerType reflect.Type) bool {
	//if the handler doesn't take arguments, false
	if handlerType.NumIn() == 0 {
		return false
	}

	//if the first argument is not a pointer, false
	p1 := handlerType.In(0)
	if p1.Kind() != reflect.Ptr {
		return false
	}
	//but if the first argument is a renderer, true
	if p1.Elem() == rendererType {
		return true
	}

	return false
}

func hasContextAndRenderer(handlerType reflect.Type) bool {

	//first check if we have pass 2 arguments
	if handlerType.NumIn() < 2 {
		return false
	}

	
	firstParamIsContext := hasContextParam(handlerType)
	
	//the first argument/parameter is always context if exists otherwise it's only Renderer or ResponseWriter,Request.
	if firstParamIsContext == false {
		return false
	}
	
	p2 := handlerType.In(1)
	if p2.Kind() != reflect.Ptr {
		return false
	}
	//but if the first argument is a context, true
	if p2.Elem() == rendererType {
		return true
	}
	
	return false
}