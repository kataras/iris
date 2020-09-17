package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/kataras/iris/v12/sessions"

	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	basic := app.Party("/basic")
	{
		// Register middlewares to run under the /basic path prefix.
		ac := accesslog.File("./basic_access.log")
		defer ac.Close()

		basic.UseRouter(ac.Handler)
		basic.UseRouter(recover.New())

		mvc.Configure(basic, basicMVC)
	}

	app.Listen(":8080")
}

func basicMVC(app *mvc.Application) {
	// Disable verbose logging of controllers for this and its children mvc apps
	// when the log level is "debug":
	app.SetControllersNoLog(true)

	// You can still register middlewares at MVC apps of course.
	// The app.Router returns the Party that this MVC
	// was registered on.
	// app.Router.UseRouter/Use/....

	// Register dependencies which will be binding to the controller(s),
	// can be either a function which accepts an iris.Context and returns a single value (dynamic binding)
	// or a static struct value (service).
	app.Register(
		sessions.New(sessions.Config{}).Start,
		&prefixedLogger{prefix: "DEV"},
		accesslog.GetFields, // Set custom fields through a controller or controller's methods.
	)

	// GET: http://localhost:8080/basic
	// GET: http://localhost:8080/basic/custom
	// GET: http://localhost:8080/basic/custom2
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
	Logger    LoggerService     // the static logger service attached to this app.
	Session   *sessions.Session // current HTTP session.
	LogFields *accesslog.Fields // accesslog middleware custom fields.
}

func (c *basicController) BeforeActivation(b mvc.BeforeActivation) {
	b.HandleMany("GET", "/custom /custom2", "Custom")
}

func (c *basicController) AfterActivation(a mvc.AfterActivation) {
	if a.Singleton() {
		panic("basicController should be stateless, a request-scoped, we have a 'Session' which depends on the context.")
	}
}

func (c *basicController) Get() string {
	count := c.Session.Increment("count", 1)
	c.LogFields.Set("count", count)

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
