package hero_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12"
	. "github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/httptest"
)

// dynamic func
type testUserStruct struct {
	ID       int64  `json:"id" form:"id" url:"id"`
	Username string `json:"username" form:"username" url:"username"`
}

func testBinderFunc(ctx iris.Context) testUserStruct {
	id, _ := ctx.Params().GetInt64("id")
	username := ctx.Params().Get("username")
	return testUserStruct{
		ID:       id,
		Username: username,
	}
}

// service
type (
	// these testService and testServiceImpl could be in lowercase, unexported
	// but the `Say` method should be exported however we have those exported
	// because of the controller handler test.
	testService interface {
		Say(string) string
	}
	testServiceImpl struct {
		prefix string
	}
)

func (s *testServiceImpl) Say(message string) string {
	return s.prefix + " " + message
}

var (
	// binders, as user-defined
	testBinderFuncUserStruct = testBinderFunc
	testBinderService        = &testServiceImpl{prefix: "say"}
	testBinderFuncParam      = func(ctx iris.Context) string {
		return ctx.Params().Get("param")
	}

	// consumers
	// a context as first input arg, which is not needed to be binded manually,
	// and a user struct which is binded to the input arg by the #1 func(ctx) any binder.
	testConsumeUserHandler = func(ctx iris.Context, user testUserStruct) {
		ctx.JSON(user)
	}

	// just one input arg, the service which is binded by the #2 service binder.
	testConsumeServiceHandler = func(service testService) string {
		return service.Say("something")
	}
	// just one input arg, a standar string which is binded by the #3 func(ctx) any binder.
	testConsumeParamHandler = func(myParam string) string {
		return "param is: " + myParam
	}
)

func TestHandler(t *testing.T) {
	Register(testBinderFuncUserStruct)
	Register(testBinderService)
	Register(testBinderFuncParam)
	var (
		h1 = Handler(testConsumeUserHandler)
		h2 = Handler(testConsumeServiceHandler)
		h3 = Handler(testConsumeParamHandler)
	)

	testAppWithHeroHandlers(t, h1, h2, h3)
}

func testAppWithHeroHandlers(t *testing.T, h1, h2, h3 iris.Handler) {
	app := iris.New()
	app.Get("/{id:int64}/{username:string}", h1)
	app.Get("/service", h2)
	app.Get("/param/{param:string}", h3)

	expectedUser := testUserStruct{
		ID:       42,
		Username: "kataras",
	}

	e := httptest.New(t, app)
	// 1
	e.GET(fmt.Sprintf("/%d/%s", expectedUser.ID, expectedUser.Username)).Expect().Status(httptest.StatusOK).
		JSON().IsEqual(expectedUser)
	// 2
	e.GET("/service").Expect().Status(httptest.StatusOK).
		Body().IsEqual("say something")
	// 3
	e.GET("/param/the_param_value").Expect().Status(httptest.StatusOK).
		Body().IsEqual("param is: the_param_value")
}

// TestBindFunctionAsFunctionInputArgument tests to bind
// a whole dynamic function based on the current context
// as an input argument in the hero handler's function.
func TestBindFunctionAsFunctionInputArgument(t *testing.T) {
	app := iris.New()
	postsBinder := func(ctx iris.Context) func(string) string {
		return ctx.PostValue // or FormValue, the same here.
	}

	h := New(postsBinder).Handler(func(get func(string) string) string {
		// send the `ctx.PostValue/FormValue("username")` value
		// to the client.
		return get("username")
	})

	app.Post("/", h)

	e := httptest.New(t, app)

	expectedUsername := "kataras"
	e.POST("/").WithFormField("username", expectedUsername).
		Expect().Status(iris.StatusOK).Body().IsEqual(expectedUsername)
}

func TestPayloadBinding(t *testing.T) {
	h := New()

	ptrHandler := h.Handler(func(input *testUserStruct /* ptr */) string {
		return input.Username
	})

	valHandler := h.Handler(func(input testUserStruct) string {
		return input.Username
	})

	h.GetErrorHandler = func(iris.Context) ErrorHandler {
		return ErrorHandlerFunc(func(ctx iris.Context, err error) {
			if iris.IsErrPath(err) {
				return // continue.
			}

			ctx.StopWithError(iris.StatusBadRequest, err)
		})
	}

	app := iris.New()
	app.Get("/", ptrHandler)
	app.Post("/", ptrHandler)
	app.Post("/2", valHandler)

	e := httptest.New(t, app)

	// JSON
	e.POST("/").WithJSON(iris.Map{"username": "makis"}).Expect().Status(httptest.StatusOK).Body().IsEqual("makis")
	e.POST("/2").WithJSON(iris.Map{"username": "kataras"}).Expect().Status(httptest.StatusOK).Body().IsEqual("kataras")

	// FORM (url-encoded)
	e.POST("/").WithFormField("username", "makis").Expect().Status(httptest.StatusOK).Body().IsEqual("makis")
	// FORM (multipart)
	e.POST("/").WithMultipart().WithFormField("username", "makis").Expect().Status(httptest.StatusOK).Body().IsEqual("makis")
	// FORM: test ErrorHandler skip the ErrPath.
	e.POST("/").WithMultipart().WithFormField("username", "makis").WithFormField("unknown", "continue").
		Expect().Status(httptest.StatusOK).Body().IsEqual("makis")

	// POST URL query.
	e.POST("/").WithQuery("username", "makis").Expect().Status(httptest.StatusOK).Body().IsEqual("makis")
	// GET URL query.
	e.GET("/").WithQuery("username", "makis").Expect().Status(httptest.StatusOK).Body().IsEqual("makis")
}

/* Author's notes:
If aksed or required by my company, make the following test to pass but think downsides of code complexity and performance-cost
before begin the implementation of it.
- Dependencies without depending on other values can be named "root-level dependencies"
- Dependencies could be linked (a new .DependsOn?) to a "root-level dependency"(or by theirs same-level deps too?) with much
  more control if "root-level dependencies" are named, e.g.:
	b.Register("db", &myDBImpl{})
	b.Register("user_dep", func(db myDB) User{...}).DependsOn("db")
	b.Handler(func(user User) error{...})
	b.Handler(func(ctx iris.Context, reuseDB myDB) {...})
Why linked over automatically? Because more than one dependency can implement the same input and
end-user does not care about ordering the registered ones.
Link with `DependsOn` SHOULD be optional, if exists then limit the available dependencies,
`DependsOn` SHOULD accept comma-separated values, e.g. "db, otherdep" and SHOULD also work
by calling it multiple times i.e `Depends("db").DependsOn("otherdep")`.
Handlers should also be able to explicitly limit the list of
their available dependencies per-handler, a `.DependsOn` feature SHOULD exist there too.

Also, note that with the new implementation a `*hero.Input` value can be accepted on dynamic dependencies,
that value contains an `Options.Dependencies` field which lists all the registered dependencies,
so, in theory, end-developers could achieve same results by hand-code(inside the dependency's function body).

26 Feb 2020. Gerasimos Maropoulos
______________________________________________

29 Feb 2020. It's done.
*/

type testMessage struct {
	Body string
}

type myMap map[string]*testMessage

func TestDependentDependencies(t *testing.T) {
	b := New()
	b.Register(&testServiceImpl{prefix: "prefix:"})
	b.Register(func(service testService) testMessage {
		return testMessage{Body: service.Say("it is a deep") + " dependency"}
	})
	b.Register(myMap{"test": &testMessage{Body: "value"}})
	var (
		h1 = b.Handler(func(msg testMessage) string {
			return msg.Body
		})
		h2 = b.Handler(func(reuse testService) string {
			return reuse.Say("message")
		})
		h3 = b.Handler(func(m myMap) string {
			return m["test"].Body
		})
	)

	app := iris.New()
	app.Get("/h1", h1)
	app.Get("/h2", h2)
	app.Get("/h3", h3)

	e := httptest.New(t, app)
	e.GET("/h1").Expect().Status(httptest.StatusOK).Body().IsEqual("prefix: it is a deep dependency")
	e.GET("/h2").Expect().Status(httptest.StatusOK).Body().IsEqual("prefix: message")
	e.GET("/h3").Expect().Status(httptest.StatusOK).Body().IsEqual("value")
}

func TestHandlerPathParams(t *testing.T) {
	// See white box `TestPathParams` test too.
	// All cases should pass.
	app := iris.New()
	handler := func(id uint64) string {
		return fmt.Sprintf("%d", id)
	}

	app.Party("/users").ConfigureContainer(func(api *iris.APIContainer) {
		api.Get("/{id:uint64}", handler)
	})

	app.Party("/editors/{id:uint64}").ConfigureContainer(func(api *iris.APIContainer) {
		api.Get("/", handler)
	})

	// should receive the last one, as we expected only one useful for MVC (there is a similar test there too).
	app.ConfigureContainer().Get("/{ownerID:uint64}/book/{booKID:uint64}", handler)

	e := httptest.New(t, app)

	for _, testReq := range []*httptest.Request{
		e.GET("/users/42"),
		e.GET("/editors/42"),
		e.GET("/1/book/42"),
	} {
		testReq.Expect().Status(httptest.StatusOK).Body().IsEqual("42")
	}
}

func TestRegisterDependenciesFromContext(t *testing.T) {
	// Tests serve-time struct dependencies through a common Iris middleware.
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.RegisterDependency(testUserStruct{Username: "kataras"})
		ctx.Next()
	})
	app.Use(func(ctx iris.Context) {
		ctx.RegisterDependency(&testServiceImpl{prefix: "say"})
		ctx.Next()
	})

	app.ConfigureContainer(func(api *iris.APIContainer) {
		api.Get("/", func(u testUserStruct) string {
			return u.Username
		})

		api.Get("/service", func(s *testServiceImpl) string {
			return s.Say("hello")
		})

		// Note: we are not allowed to pass the service as an interface here
		// because the container will, correctly, panic because it will expect
		// a dependency to be registered before server ran.
		api.Get("/both", func(s *testServiceImpl, u testUserStruct) string {
			return s.Say(u.Username)
		})

		api.Get("/non", func() string {
			return "nothing"
		})
	})

	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("kataras")
	e.GET("/service").Expect().Status(httptest.StatusOK).Body().IsEqual("say hello")
	e.GET("/both").Expect().Status(httptest.StatusOK).Body().IsEqual("say kataras")
	e.GET("/non").Expect().Status(httptest.StatusOK).Body().IsEqual("nothing")
}
