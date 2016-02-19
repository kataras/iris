package iris

import (
	"reflect"
)

// Handler is the interface which is used of the structed-routes and be passed to the Router's RegisterHandler,
// struct implements this Handler MUST have a function which has the form one of them:
//
// Handle(ctx *Context, renderer *Renderer)
// Handle(res http.ResponseWriter, req *http.Request)
// Handle(ctx *Context)
// Handle(renderer *Renderer)
type Handler interface {
	/*Example

	  import (
	  	"github.com/kataras/iris"
	  )

	  type UserHandler struct {
	  	iris.Handler `get:"/api/users/{userId(int)}"`
	  }

	  func (u *UserHandler) Handle(ctx *iris.Context) {
	  //or
	  //Handle(ctx *Context, renderer *Renderer)
	  //Handle(res http.ResponseWriter, req *http.Request)
	  //Handle(renderer *Renderer)

	  	defer ctx.Close()
	  	var userId, _ = ctx.ParamInt("userId")

	  	println(userId)

	  }

	  ...

	  api:= iris.New()
	  registerError := api.RegisterHandler(new(UserHandler))

	*/
}

// HTTPHandler is the function which is passed a second parameter/argument to the API methods (Get,Post...)
// It has got one the following forms:
//
// 1. http.ResponseWriter, *http.Request
// 2. *iris.Context
// 3. *iris.Renderer
// 4. *iris.Context, *iris.Renderer
type HTTPHandler interface{}

// check the first parameter, true if it wants only a *Context
// check if the handler needs a Context , has the first parameter as type of *Context
// it's usefuly in NewRoute inside route.go
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

// check the first parameter, true if it wants only a *Renderer
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

// check if two parameters, true if it wants *Context following by a *Renderer
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
