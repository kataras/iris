package iris

import (
	"fmt"
	"net/http"
	"strings"
)

// Annotated is the interface which is used of the structed-routes and be passed to the Router's Handle,
// struct implements this Handler MUST have a function which has the form one of them:
//
// Handle(ctx *Context)
// Handle(res http.ResponseWriter, req *http.Request)
/*Example

  import (
  	"github.com/kataras/iris"
  )

  type UserHandler struct {
  	iris.Annotated `get:"/api/users/:userId"`
  }

  func (u *UserHandler) Handle(ctx *iris.Context) {
  //or
  //Handle(res http.ResponseWriter, req *http.Request)

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
	//must implement func Handle() with supported parameters:
	//ctx *Context or
	//res http.ResponseWriter, req *http.Request
}

// HTTPHandler is the function which is passed a second parameter/argument to the API methods (Get,Post...)
// It has got one the following forms:
//
// 1. http.ResponseWriter, *http.Request
// 2. http.Handler
// 3. *iris.Context
type HTTPHandler interface{}

//
type Handler interface {
	run(r *Route, res http.ResponseWriter, req *http.Request)
}

var _renderer = &Renderer{}

func GetRenderer(res http.ResponseWriter) *Renderer {
	_renderer.responseWriter = res
	return _renderer
}

func ResetRenderer() {
	_renderer.responseWriter = nil
}

type ContextedHandlerFunc func(*Context)

func (h ContextedHandlerFunc) run(r *Route, res http.ResponseWriter, req *http.Request) {
	/*ctx := GetContextPointer(res, req, r.httpErrors)
	s := TryGetParameters(r, req.URL.Path)
	t := make(PathParameters, len(s), (cap(s)+1)*2)
	copy(t, s)
	ctx.Params = t
	h(ctx)

	ResetContext()*/

	ctx := r.station.pool.Get().(*Context)

	ctx.ResponseWriter = res
	ctx.Request = req
	ctx.route = r
	if int(r.paramsLength) > len(ctx.Params) {
		ctx.Params = append(ctx.Params, ctx.Params...) //double the cap of the params
	}
	ctx.Params = ctx.Params[0:r.paramsLength]
	SetParametersTo(ctx, req.URL.Path)
	ctx.Renderer = GetRenderer(res)
	if ctx.Renderer.templates == nil {
		ctx.Renderer.templates = r.station.htmlTemplates
	}

	h(ctx)
	r.station.pool.Put(ctx)

}

type TypicalHandlerFunc func(http.ResponseWriter, *http.Request)

func (h TypicalHandlerFunc) run(r *Route, res http.ResponseWriter, req *http.Request) {
	h(res, req)
}

type staticServer struct {
	directory    string
	fileServer   http.Handler
	finalHandler http.Handler
}

func (s staticServer) run(r *Route, res http.ResponseWriter, req *http.Request) {
	//example: iris.Get("/public/*",iris.Static("./static/files/") we need to strip the public
	//we have access to the route's registed path via this run func, because of that we don't return just the simple http.Handler
	if s.finalHandler == nil {
		pathToStrip := r.PathPrefix[:strings.LastIndex(r.PathPrefix, "/")+1]
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
	if handler == nil {
		panic("Error on Iris -> convertToHandler handler is nil")
	}
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
	default:
		panic(fmt.Sprintf("Error on Iris -> convertToHandler handler is not TypicalHandlerFunc or ContextedHandlerFunc, is %T Point to: %v:", handler, handler))
	}
}
