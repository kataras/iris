package main

import (
	"sync"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

// Owner is our application structure, it contains the methods or fields we need,
// think it as the owner of our *Context.
type Owner struct {
	// define here the fields that are global
	// and shared to all clients.
	sessionsManager *sessions.Sessions
}

// this package-level variable "application" will be used inside context to communicate with our global Application.
var owner = &Owner{
	sessionsManager: sessions.New(sessions.Config{Cookie: "mysessioncookie"}),
}

// Context is our custom context.
// Let's implement a context which will give us access
// to the client's Session with a trivial `ctx.Session()` call.
type Context struct {
	iris.Context
	session *sessions.Session
}

// Session returns the current client's session.
func (ctx *Context) Session() *sessions.Session {
	// this help us if we call `Session()` multiple times in the same handler
	if ctx.session == nil {
		// start a new session if not created before.
		ctx.session = owner.sessionsManager.Start(ctx.Context)
	}

	return ctx.session
}

// Bold will send a bold text to the client.
func (ctx *Context) Bold(text string) {
	ctx.HTML("<b>" + text + "</b>")
}

var contextPool = sync.Pool{New: func() interface{} {
	return &Context{}
}}

func acquire(original iris.Context) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.Context = original // set the context to the original one in order to have access to iris's implementation.
	ctx.session = nil      // reset the session
	return ctx
}

func release(ctx *Context) {
	contextPool.Put(ctx)
}

// Handler will convert our handler of func(*Context) to an iris Handler,
// in order to be compatible with the HTTP API.
func Handler(h func(*Context)) iris.Handler {
	return func(original iris.Context) {
		ctx := acquire(original)
		h(ctx)
		release(ctx)
	}
}

func newApp() *iris.Application {
	app := iris.New()

	// Work as you did before, the only difference
	// is that the original context.Handler should be wrapped with our custom
	// `Handler` function.
	app.Get("/", Handler(func(ctx *Context) {
		ctx.Bold("Hello from our *Context")
	}))

	app.Post("/set", Handler(func(ctx *Context) {
		nameFieldValue := ctx.FormValue("name")
		ctx.Session().Set("name", nameFieldValue)
		ctx.Writef("set session = " + nameFieldValue)
	}))

	app.Get("/get", Handler(func(ctx *Context) {
		name := ctx.Session().GetString("name")
		ctx.Writef(name)
	}))

	return app
}

func main() {
	app := newApp()

	// GET: http://localhost:8080
	// POST: http://localhost:8080/set
	// GET: http://localhost:8080/get
	app.Run(iris.Addr(":8080"))
}
