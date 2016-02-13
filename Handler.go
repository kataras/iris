package gapi

import (
	//"reflect"
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

/*
type GapiHandler struct {
	Handler
	methodRoutes []func(ctx *Context) string //string -> returns the path of the handler
}

func (h *GapiHandler) Handle(ctx *Context) {
	if h.methodRoutes == nil {
		h.methodRoutes = make([]func(ctx *Context) string,0)
		//get the correct functions interfaces on the first call only
		val := reflect.ValueOf(h).Elem()
		for i := 0; i < val.NumMethod(); i++ {
			method := val.Method(i)
			if method.Type().Name() != "methodRoutes" {
				methodInterface := method.Call([]reflect.Value{})[0].Interface()
				h.methodRoutes = append(h.methodRoutes,methodInterface.(func(ctx *Context) string))
			}

		}
	}

}*/
