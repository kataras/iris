// +build go1.9

package iris

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/host"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/mvc"
)

type (
	// Context is the midle-man server's "object" for the clients.
	//
	// A New context is being acquired from a sync.Pool on each connection.
	// The Context is the most important thing on the iris's http flow.
	//
	// Developers send responses to the client's request through a Context.
	// Developers get request information from the client's request by a Context.
	Context = context.Context
	// A Handler responds to an HTTP request.
	// It writes reply headers and data to the Context.ResponseWriter() and then return.
	// Returning signals that the request is finished;
	// it is not valid to use the Context after or concurrently with the completion of the Handler call.
	//
	// Depending on the HTTP client software, HTTP protocol version,
	// and any intermediaries between the client and the iris server,
	// it may not be possible to read from the Context.Request().Body after writing to the context.ResponseWriter().
	// Cautious handlers should read the Context.Request().Body first, and then reply.
	//
	// Except for reading the body, handlers should not modify the provided Context.
	//
	// If Handler panics, the server (the caller of Handler) assumes that the effect of the panic was isolated to the active request.
	// It recovers the panic, logs a stack trace to the server error log, and hangs up the connection.
	Handler = context.Handler
	// A Map is a shortcut of the map[string]interface{}.
	Map = context.Map

	// Supervisor is a shortcut of the `host#Supervisor`.
	// Used to add supervisor configurators on common Runners
	// without the need of importing the `core/host` package.
	Supervisor = host.Supervisor

	// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
	// Party could also be named as 'Join' or 'Node' or 'Group' , Party chosen because it is fun.
	//
	// Look the `core/router#APIBuilder` for its implementation.
	//
	// A shortcut for the `core/router#Party`, useful when `PartyFunc` is being used.
	Party = router.Party

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
	// All fields that are tagged with iris:"persistence"`
	// are being persistence and kept between the different requests,
	// meaning that these data will not be reset-ed on each new request,
	// they will be the same for all requests.
	//
	// An Example Controller can be:
	//
	// type IndexController struct {
	// 	iris.Controller
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
	// Another example with persistence data:
	//
	// type UserController struct {
	// 	iris.Controller
	//
	// 	CreatedAt time.Time `iris:"persistence"`
	// 	Title     string    `iris:"persistence"`
	// 	DB        *DB       `iris:"persistence"`
	// }
	//
	// // Get serves using the User controller when HTTP Method is "GET".
	// func (c *UserController) Get() {
	// 	c.Tmpl = "user/index.html"
	// 	c.Data["title"] = c.Title
	// 	c.Data["username"] = "kataras " + c.Params.Get("userid")
	// 	c.Data["connstring"] = c.DB.Connstring
	// 	c.Data["uptime"] = time.Now().Sub(c.CreatedAt).Seconds()
	// }
	//
	// Usage: app.Controller("/user/{id:int}", &UserController{
	// 	CreatedAt: time.Now(),
	// 	Title: "User page",
	// 	DB: yourDB,
	// })
	//
	// Look `core/router#APIBuilder#Controller` method too.
	//
	// A shortcut for the `mvc#Controller`,
	// useful when `app.Controller` method is being used.
	//
	// A Controller can be declared by importing
	// the "github.com/kataras/iris/mvc"
	// package for machines that have not installed go1.9 yet.
	Controller = mvc.Controller
	// SessionController is a simple `Controller` implementation
	// which requires a binded session manager in order to give
	// direct access to the current client's session via its `Session` field.
	SessionController = mvc.SessionController
	// C is the lightweight BaseController type as an alternative of the `Controller` struct type.
	// It contains only the Name of the controller and the Context, it's the best option
	// to balance the performance cost reflection uses
	// if your controller uses the new func output values dispatcher feature;
	// func(c *ExampleController) Get() string |
	// (string, string) |
	// (string, int) |
	// int |
	// (int, string |
	// (string, error) |
	// error |
	// (int, error) |
	// (customStruct, error) |
	// customStruct |
	// (customStruct, int) |
	// (customStruct, string) |
	// Result or (Result, error)
	// where Get is an HTTP Method func.
	//
	// Look `core/router#APIBuilder#Controller` method too.
	//
	// A shortcut for the `mvc#C`,
	// useful when `app.Controller` method is being used.
	//
	// A C controller can be declared by importing
	// the "github.com/kataras/iris/mvc" as well.
	C = mvc.C
	// Response completes the `mvc/activator/methodfunc.Result` interface.
	// It's being used as an alternative return value which
	// wraps the status code, the content type, a content as bytes or as string
	// and an error, it's smart enough to complete the request and send the correct response to the client.
	//
	// A shortcut for the `mvc#Response`,
	// useful when return values from method functions, i.e
	// GetHelloworld() iris.Response { iris.Response{ Text:"Hello World!", Code: 200 }}
	Response = mvc.Response
	// View completes the `mvc/activator/methodfunc.Result` interface.
	// It's being used as an alternative return value which
	// wraps the template file name, layout, (any) view data, status code and error.
	// It's smart enough to complete the request and send the correct response to the client.
	//
	// A shortcut for the `mvc#View`,
	// useful when return values from method functions, i.e
	// GetUser() iris.View { iris.View{ Name:"user.html", Data: currentUser } }
	View = mvc.View
	// Result is a response dispatcher.
	// All types that complete this interface
	// can be returned as values from the method functions.
	// A shortcut for the `mvc#Result` which is a shortcut for `mvc/activator/methodfunc#Result`,
	// useful when return values from method functions, i.e
	// GetUser() iris.Result { iris.Response{} or a custom iris.Result }
	// Can be also used for the TryResult function.
	Result = mvc.Result
)

// Try is a shortcut for the function `mvc.Try` result.
// See more at `mvc#Try` documentation.
var Try = mvc.Try
