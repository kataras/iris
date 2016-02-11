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
	defer ctx.Close()
	var userId, _ = ctx.ParamInt("userId")

	println(userId)

}

...

api:= gapi.New()
registerError := api.RegisterHandler(new(UserHandler))

*/

type Handler interface {
	Handle(ctx *Context)
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
