package mvc

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/memstore"
	"github.com/kataras/iris/mvc/activator"
)

// Controller is the base controller for the high level controllers instances.
//
// This base controller is used as an alternative way of building
// APIs, the controller can register all type of http methods.
//
// Keep note that controllers are bit slow
// because of the reflection use however it's as fast as possible because
// it does preparation before the serve-time handler but still
// remains slower than the low-level handlers
// such as `Handle, Get, Post, Put, Delete, Connect, Head, Trace, Patch`.
//
//
// All fields that are tagged with iris:"persistence"` or binded
// are being persistence and kept the same between the different requests.
//
// An Example Controller can be:
//
// type IndexController struct {
// 	Controller
// }
//
// func (c *IndexController) Get() {
// 	c.Tmpl = "index.html"
// 	c.Data["title"] = "Index page"
// 	c.Data["message"] = "Hello world!"
// }
//
// Usage: app.Controller("/", new(IndexController))
//
//
// Another example with bind:
//
// type UserController struct {
// 	mvc.Controller
//
// 	DB        *DB
// 	CreatedAt time.Time
// }
//
// // Get serves using the User controller when HTTP Method is "GET".
// func (c *UserController) Get() {
// 	c.Tmpl = "user/index.html"
// 	c.Data["title"] = "User Page"
// 	c.Data["username"] = "kataras " + c.Params.Get("userid")
// 	c.Data["connstring"] = c.DB.Connstring
// 	c.Data["uptime"] = time.Now().Sub(c.CreatedAt).Seconds()
// }
//
// Usage: app.Controller("/user/{id:int}", new(UserController), db, time.Now())
//
// Look `core/router/APIBuilder#Controller` method too.
type Controller struct {
	// path and path params.
	Path   string
	Params *context.RequestParams

	// some info read and write,
	// can be already set-ed by previous handlers as well.
	Status int
	Values *memstore.Store

	// view read and write,
	// can be already set-ed by previous handlers as well.
	Layout string
	Tmpl   string
	Data   map[string]interface{}

	// give access to the request context itself.
	Ctx context.Context
}

// BeginRequest starts the main controller
// it initialize the Ctx and other fields.
//
// End-Developer can ovverride it but it still MUST be called.
func (c *Controller) BeginRequest(ctx context.Context) {
	// path and path params
	c.Path = ctx.Path()
	c.Params = ctx.Params()
	// response status code
	c.Status = ctx.GetStatusCode()
	// share values
	c.Values = ctx.Values()
	// view
	c.Data = make(map[string]interface{}, 0)
	// context itself
	c.Ctx = ctx
}

// EndRequest is the final method which will be executed
// before response sent.
//
// It checks for the fields and calls the necessary context's
// methods to modify the response to the client.
//
// End-Developer can ovveride it but still should be called at the end.
func (c *Controller) EndRequest(ctx context.Context) {
	if path := c.Path; path != "" && path != ctx.Path() {
		// then redirect
		ctx.Redirect(path)
		return
	}

	if status := c.Status; status > 0 && status != ctx.GetStatusCode() {
		ctx.StatusCode(status)
	}

	if view := c.Tmpl; view != "" {
		if layout := c.Layout; layout != "" {
			ctx.ViewLayout(layout)
		}
		if data := c.Data; data != nil {
			for k, v := range data {
				ctx.ViewData(k, v)
			}
		}
		ctx.View(view)
	}
}

var _ activator.BaseController = &Controller{}
