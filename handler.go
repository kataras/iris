package iris

import (
	"fmt"
	"net/http"
)

// Annotated is the interface which is used of the structed-routes and be passed to the Router's Handle,
// struct implements this Handler MUST have a function which has this form:
//
// Handle(ctx *Context)
/*Example

  import (
  	"github.com/kataras/iris"
  )

  type UserHandler struct {
  	iris.Annotated `get:"/api/users/:userId"`
  }

  func (u *UserHandler) Handle(ctx *iris.Context) {

  	defer ctx.Close()
  	var userId, _ = ctx.ParamInt("userId")

  	println(userId)

  }

  ...

  api:= iris.New()
  registerError := api.Handle(&UserHandler{})

*/
///TODO: to Annotated na to kanw Handler me Serve
type Annotated interface {
	//must implement func Handle() with a *Context as parameter
}

// the main Iris Handler interface.
type Handler interface {
	Serve(ctx *Context)
}
type HandlerFunc func(*Context)

func (h HandlerFunc) Serve(ctx *Context) {
	h(ctx)
}

// Static is just a function which returns a HandlerFunc with the standar http's fileserver's handler
// It is not a middleware, it just returns a HandlerFunc to use anywhere we want
func Static(SystemPath string, PathToStrip ...string) HandlerFunc {
	//runs only once to start the file server
	path := http.Dir(SystemPath)
	underlineFileserver := http.FileServer(path)
	if PathToStrip != nil && len(PathToStrip) == 1 {
		underlineFileserver = http.StripPrefix(PathToStrip[0], underlineFileserver)
	}

	return ToHandlerFunc(underlineFileserver.ServeHTTP)

}

func ToHandler(handler interface{}) Handler {
	switch handler.(type) {
	case http.Handler:
		return HandlerFunc((func(c *Context) {
			handler.(http.Handler).ServeHTTP(c.ResponseWriter, c.Request)
		}))

	case func(http.ResponseWriter, *http.Request):
		return HandlerFunc((func(c *Context) {
			handler.(func(http.ResponseWriter, *http.Request))(c.ResponseWriter, c.Request)
		}))
	default:
		panic(fmt.Sprintf("Error on Iris.ToHandler: handler is not func(*Context) either an object which implements the iris.Handler interface\n It seems to be a  %T Point to: %v:", handler, handler))
	}
	return nil
}

func ToHandlerFunc(handler interface{}) HandlerFunc {
	return ToHandler(handler).Serve
}
