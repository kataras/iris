package main

import (
	"fmt"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"

	"github.com/kataras/iris/mvc"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")
	mvc.Configure(app.Party("/basic"), basicMVC)

	app.Run(iris.Addr(":8080"))
}

func basicMVC(app *mvc.Application) {
	// You can use normal middlewares at MVC apps of course.
	app.Router.Use(func(ctx iris.Context) {
		ctx.Application().Logger().Infof("Path: %s", ctx.Path())
		ctx.Next()
	})

	// Register dependencies which will be binding to the controller(s),
	// can be either a function which accepts an iris.Context and returns a single value (dynamic binding)
	// or a static struct value (service).
	app.Register(
		sessions.New(sessions.Config{}).Start,
		&prefixedLogger{prefix: "DEV"},
	)

	// GET: http://localhost:8080/basic
	// GET: http://localhost:8080/basic/custom
	app.Handle(new(basicController))

	// All dependencies of the parent *mvc.Application
	// are cloned to this new child,
	// thefore it has access to the same session as well.
	// GET: http://localhost:8080/basic/sub
	app.Party("/sub").
		Handle(new(basicSubController))
}

// If controller's fields (or even its functions) expecting an interface
// but a struct value is binded then it will check
// if that struct value implements
// the interface and if true then it will add this to the
// available bindings, as expected, before the server ran of course,
// remember? Iris always uses the best possible way to reduce load
// on serving web resources.

type LoggerService interface {
	Log(string)
}

type prefixedLogger struct {
	prefix string
}

func (s *prefixedLogger) Log(msg string) {
	fmt.Printf("%s: %s\n", s.prefix, msg)
}

type basicController struct {
	Logger LoggerService

	Session *sessions.Session
}

func (c *basicController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/custom", "Custom")
}

func (c *basicController) AfterActivation(a mvc.AfterActivation) {
	if a.Singleton() {
		panic("basicController should be stateless, a request-scoped, we have a 'Session' which depends on the context.")
	}
}

func (c *basicController) Get() string {
	count := c.Session.Increment("count", 1)

	body := fmt.Sprintf("Hello from basicController\nTotal visits from you: %d", count)
	c.Logger.Log(body)
	return body
}

func (c *basicController) Custom() string {
	return "custom"
}

type basicSubController struct {
	Session *sessions.Session
}

func (c *basicSubController) Get() string {
	count := c.Session.GetIntDefault("count", 1)
	return fmt.Sprintf("Hello from basicSubController.\nRead-only visits count: %d", count)
}
