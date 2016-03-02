package iris

import (
	"net/http"
	"reflect"
	"strings"
)

// Annotated is the interface which is used of the structed-routes and be passed to the Router's Handle,
// struct implements this Handler MUST have a function which has the form one of them:
//
// Handle(ctx *Context, renderer *Renderer)
// Handle(res http.ResponseWriter, req *http.Request)
// Handle(ctx *Context)
// Handle(renderer *Renderer)
/*Example

  import (
  	"github.com/kataras/iris"
  )

  type UserHandler struct {
  	iris.Annotated `get:"/api/users/{userId(int)}"`
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
  registerError := api.Handle(&UserHandler{})

*/
//or AnnotatedRoute  but its too big 'iris.AnnotatedRoute' 'iris.Annotated' is better but not the best, f* my eng vocabulary?
type Annotated interface {
	//must implement func Handle(
	//FullHandlerFunc or
	//RendereredHandlerFunc or
	//ContextedHandlerFunc or
	//TypicalHandlerFunc)
}

//}

// HTTPHandler is the function which is passed a second parameter/argument to the API methods (Get,Post...)
// It has got one the following forms:
//
// 1. http.ResponseWriter, *http.Request
// 2. *iris.Context
// 3. *iris.Renderer
// 4. *iris.Context, *iris.Renderer
type HTTPHandler interface{}

//
type Handler interface {
	run(r *Route, res http.ResponseWriter, req *http.Request)
}

type FullHandlerFunc func(*Context, *Renderer)

func (h FullHandlerFunc) run(r *Route, res http.ResponseWriter, req *http.Request) {
	ctx := newContext(res, req, r.httpErrors)
	ctx.Params = Params(r, req.URL.Path)
	renderer := newRenderer(res)

	if r.templates != nil {
		renderer.templateCache = r.templates
	}

	h(ctx, renderer)
}

type RendereredHandlerFunc func(*Renderer)

func (h RendereredHandlerFunc) run(r *Route, res http.ResponseWriter, req *http.Request) {
	renderer := newRenderer(res)
	if r.templates != nil {
		renderer.templateCache = r.templates
	}
	h(renderer)
}

type ContextedHandlerFunc func(*Context)

func (h ContextedHandlerFunc) run(r *Route, res http.ResponseWriter, req *http.Request) {
	ctx := newContext(res, req, r.httpErrors)
	ctx.Params = Params(r, req.URL.Path)
	h(ctx)
}

//=HandlerFunc
type TypicalHandlerFunc func(http.ResponseWriter, *http.Request)

func (h TypicalHandlerFunc) run(r *Route, res http.ResponseWriter, req *http.Request) {
	h(res, req)
}

// Static receives a path and returns an http.Handler which is handling the static files
// The FileServer receives a directory and serves all it's children folders and files too
// This maybe not the safest way to do it but we are ok for now.
// When/if at the future I want more lower level & safier approach I can use ServeFile, ServeContent or much 'lower level', this :
// http.ServeContent(res, req, "thefile", time.Now(), bytes.NewReader(data))
/*var Static = func(dirpath string) http.Handler {
	path := http.Dir(dirpath)
	fs := http.FileServer(path)
	return fs
}*/

type staticServer struct {
	directory    string
	fileServer   http.Handler
	finalHandler http.Handler
}

func (s staticServer) run(r *Route, res http.ResponseWriter, req *http.Request) {
	//example: iris.Get("/public/*",iris.Static("./static/files/") we need to strip the public
	//we have access to the route's registed path via this run func, because of that we don't return just the simple http.Handler
	if s.finalHandler == nil {
		pathToStrip := r.pathPrefix[:strings.LastIndex(r.pathPrefix, "/")+1]
		if len(pathToStrip) > 2 { // 2 because of slashes '/'public'/'
			//[the stripPrefix makes some checks but I want the users of iris to use this format /public/* and no public/*]
			s.finalHandler = http.StripPrefix(pathToStrip, s.fileServer)
		} else {
			s.finalHandler = s.fileServer
		}

	}
	s.finalHandler.ServeHTTP(res, req)
}

// Static used as middleware to make a static file serving route
// Static receives a directory/path of the filesystem and returns a handler which can be used inside a route
func Static(dirpath string) staticServer {
	path := http.Dir(dirpath)
	fs := http.FileServer(path)
	staticHandlerServer := staticServer{dirpath, fs, nil}
	return staticHandlerServer
}

func HandlerFunc(handler interface{}) Handler {
	return convertToHandler(handler)
}

func convertToHandler(handler interface{}) Handler {
	switch handler.(type) {
	case Handler:
		//it's already handler?
		return handler.(Handler)
	case http.Handler:
		//it's a http.Handler which this implements the TypicalHandler (res,req) so its a TypicalHandlerFunc
		return TypicalHandlerFunc(handler.(http.Handler).ServeHTTP)
	case func(http.ResponseWriter, *http.Request):
		return TypicalHandlerFunc(handler.(func(http.ResponseWriter, *http.Request)))
	case func(*Context):
		return ContextedHandlerFunc(handler.(func(*Context)))
	case func(*Renderer):
		return RendereredHandlerFunc(handler.(func(*Renderer)))
	case func(*Context, *Renderer):
		return FullHandlerFunc(handler.(func(*Context, *Renderer)))
	default:
		panic("Error on Iris -> convertToHandler handler is not TypicalHandlerFunc or ContextedHandlerFunc or RendereredHandlerFunc or FullHandlerFunc")
		return nil
		//return Handler(handler.(Handler))

	}
}

//type MiddlewareHandlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) it is on middleware.go

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
