package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/recovery"
	"github.com/kataras/iris/v14/middleware/sessions"
	"github.com/kataras/iris/v14/rest/errors" // gives the ability to register controller methods that return an error too.
)

type Service interface {
	Print(name string) string
}

type serviceImpl struct{}

func (s *serviceImpl) Print(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

type controller struct {
	service Service
}

func newController(service Service) *controller {
	return &controller{service: service}
}

func (c *controller) Routing(r *errors.Router) {
	registerRoutes(r, c.service)
}

func TestRoutingController(t *testing.T) {
	app := iris.New()
	opts := errors.NewOptions()
	opts.MapErrors(errors.Unauthenticated, ErrDomainLevelError)
	app.Use(opts.Handler())

	app.RegisterDependency(new(serviceImpl))
	app.Router("/", newController)

	e := httptest.New(t, app)
	testMethods(e)
}

func TestControllerStructMethodAndErrMethod(t *testing.T) {
	app := newApp(false)

	e := httptest.New(t, app, httptest.Domain /* for sessions */)
	e.GET("/basic").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello from basicController\nTotal visits from you: 1")
	e.GET("/basic/other").Expect().Status(httptest.StatusInternalServerError).JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.Internal,
			Status:        500,
		},
		Message: "server error",
		Detail:  "an unexpected error occurred while processing the request",
	})
	e.GET("/basic/sub").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello from basicSubController.\nRead-only visits count: 1")
	e.POST("/basic/sub").Expect().Status(httptest.StatusOK).Body().IsEqual("Incremented")
	e.GET("/basic/sub").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello from basicSubController.\nRead-only visits count: 2")
}

var ( // samples
	hashKey  = []byte("D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x") // 64 bytes length.
	blockKey = []byte("VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!")                                 // 32 bytes length.
)

type sessionData struct {
	Count int
}

func newApp(debug bool) *iris.Application {
	app := iris.New()
	app.Use(recovery.Handler)

	if debug {
		app.Logger().SetLevel("debug")

		// Register middlewares to run under the /basic path prefix.
		ac := accesslog.File("./basic_access.log")
		defer ac.Close()
		app.Use(ac.Handler)
	}

	sessionOptions := sessions.NewOptions(hashKey, blockKey)
	sessionDB := sessions.Single(sessions.NewCookieDatabase[sessionData](sessionOptions), "mysession")

	// Register dependencies which will be binding to the controller(s).
	app.RegisterDependency(
		&prefixedLogger{prefix: "DEV", isDebug: debug},
		sessionDB,
	)
	// GET: http://localhost:8080/basic
	// GET: http://localhost:8080/basic/other
	// GET: http://localhost:8080/basic/other2
	app.Router("/basic", NewBasicController)

	// GET: http://localhost:8080/basic/sub
	app.Router("/basic/sub", newBasicSubController)

	return app
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
	prefix  string
	isDebug bool
}

func (s *prefixedLogger) Log(msg string) {
	if !s.isDebug {
		return
	}

	fmt.Printf("%s: %s\n", s.prefix, msg)
}

type BasicController struct {
	// filled up by constructor.
	logger    LoggerService
	sessionDB *sessions.SingleDatabase[sessionData]
}

func NewBasicController(logger LoggerService, sessionDB *sessions.SingleDatabase[sessionData]) *BasicController {
	return &BasicController{logger: logger, sessionDB: sessionDB}
}

func (c *BasicController) Routing(r iris.Router) {
	// When the controller is registered and completes the Routing(Router) method,
	// then this is an indication that the controller's methods are totally controlled by this method,
	// and any other automatic route registration is disabled.
	r.Get("/", errors.FuncHandler(c.Get))

	r.Get("/other /other2", errors.FuncHandler(c.Other))
}

func (c *BasicController) Get(ctx iris.Context) error {
	session, _ := c.sessionDB.Get(ctx)
	fields := accesslog.GetFields(ctx)

	session.Data.Count++
	if err := session.Commit(ctx); err != nil {
		return err
	}

	count := session.Data.Count
	fields.Set("count", count)

	body := fmt.Sprintf("Hello from basicController\nTotal visits from you: %d", count)
	c.logger.Log(body)

	_, err := ctx.WriteString(body)
	return err
}

func (c *BasicController) Other(ctx iris.Context) error {
	return fmt.Errorf("custom error")
}

type basicSubController struct {
	SessionDB *sessions.SingleDatabase[sessionData]
}

func newBasicSubController(sessionDB *sessions.SingleDatabase[sessionData]) *basicSubController {
	return &basicSubController{SessionDB: sessionDB}
}

// No need to manually register the Get(iris.Context) or Get(iris.Context) error methods through the `Routing` method,
// such simple methods are resolved automatically.
func (c *basicSubController) Get(ctx iris.Context) error {
	session, _ := c.SessionDB.Get(ctx)
	count := session.Data.Count
	if count == 0 {
		count = 1
	}

	ctx.Writef("Hello from basicSubController.\nRead-only visits count: %d", count)

	return nil
}

func (c *basicSubController) Post(ctx iris.Context) {
	session, _ := c.SessionDB.Get(ctx)
	session.Data.Count++

	session.Commit(ctx)
	ctx.WriteString("Incremented")
}
