package main

import (
	"fmt"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"

	"github.com/kataras/iris/mvc"
)

// TODO: It's not here but this file is what I'll see before the commit in order to delete it:
// Think a way to simplify the router cycle, I did create it to support any type of router
// but as I see nobody wants to override the iris router's behavior(I'm not speaking about wrapper, this will stay of course because it's useful on security-critical middlewares) because it's the best by far.
// Therefore I should reduce some "freedom of change" for the shake of code maintanability in the core/router files: handler.go | router.go and single change on APIBuilder's field.
func main() {
	app := iris.New()
	mvc.New(app.Party("/todo")).Configure(TodoApp)
	// no let's have a clear "mvc" package without any conversions and type aliases,
	// it's one extra import path for a whole new world, it worths it.
	//
	// app.UseMVC(app.Party("/todo")).Configure(func(app *iris.MVCApplication))

	app.Run(iris.Addr(":8080"))
}

func TodoApp(app *mvc.Application) {
	// You can use normal middlewares at MVC apps of course.
	app.Router.Use(func(ctx iris.Context) {
		ctx.Application().Logger().Infof("Path: %s", ctx.Path())
		ctx.Next()
	})

	// Add dependencies which will be binding to the controller(s),
	// can be either a function which accepts an iris.Context and returns a single value (dynamic binding)
	// or a static struct value (service).
	app.AddDependencies(
		mvc.Session(sessions.New(sessions.Config{})),
		&prefixedLogger{prefix: "DEV"},
	)

	app.Register(new(TodoController))

	// All dependencies of the parent *mvc.Application
	// are cloned to that new child, thefore it has access to the same session as well.
	app.NewChild(app.Router.Party("/sub")).
		Register(new(TodoSubController))
}

// If controller's fields (or even its functions) expecting an interface
// but a struct value is binded then it will check if that struct value implements
// the interface and if true then it will bind it as expected.

type LoggerService interface {
	Log(string)
}

type prefixedLogger struct {
	prefix string
}

func (s *prefixedLogger) Log(msg string) {
	fmt.Printf("%s: %s\n", s.prefix, msg)
}

type TodoController struct {
	Logger LoggerService

	Session *sessions.Session
}

func (c *TodoController) Get() string {
	count := c.Session.Increment("count", 1)

	body := fmt.Sprintf("Hello from TodoController\nTotal visits from you: %d", count)
	c.Logger.Log(body)
	return body
}

type TodoSubController struct {
	Session *sessions.Session
}

func (c *TodoSubController) Get() string {
	count, _ := c.Session.GetIntDefault("count", 1)
	return fmt.Sprintf("Hello from TodoSubController.\nRead-only visits count: %d", count)
}
