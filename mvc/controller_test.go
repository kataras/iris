// black-box testing
package mvc_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/httptest"

	. "github.com/kataras/iris/mvc"
)

type testController struct {
	Ctx context.Context
}

var writeMethod = func(ctx context.Context) {
	ctx.Writef(ctx.Method())
}

func (c *testController) Get() {
	writeMethod(c.Ctx)
}
func (c *testController) Post() {
	writeMethod(c.Ctx)
}
func (c *testController) Put() {
	writeMethod(c.Ctx)
}
func (c *testController) Delete() {
	writeMethod(c.Ctx)
}
func (c *testController) Connect() {
	writeMethod(c.Ctx)
}
func (c *testController) Head() {
	writeMethod(c.Ctx)
}
func (c *testController) Patch() {
	writeMethod(c.Ctx)
}
func (c *testController) Options() {
	writeMethod(c.Ctx)
}
func (c *testController) Trace() {
	writeMethod(c.Ctx)
}

type (
	testControllerAll struct{ Ctx context.Context }
	testControllerAny struct{ Ctx context.Context } // exactly the same as All.
)

func (c *testControllerAll) All() {
	writeMethod(c.Ctx)
}

func (c *testControllerAny) Any() {
	writeMethod(c.Ctx)
}

func TestControllerMethodFuncs(t *testing.T) {
	app := iris.New()

	New(app).Handle(new(testController))
	New(app.Party("/all")).Handle(new(testControllerAll))
	New(app.Party("/any")).Handle(new(testControllerAny))

	e := httptest.New(t, app)
	for _, method := range router.AllMethods {

		e.Request(method, "/").Expect().Status(iris.StatusOK).
			Body().Equal(method)

		e.Request(method, "/all").Expect().Status(iris.StatusOK).
			Body().Equal(method)

		e.Request(method, "/any").Expect().Status(iris.StatusOK).
			Body().Equal(method)
	}
}

type testControllerBeginAndEndRequestFunc struct {
	Ctx context.Context

	Username string
}

// called before of every method (Get() or Post()).
//
// useful when more than one methods using the
// same request values or context's function calls.
func (c *testControllerBeginAndEndRequestFunc) BeginRequest(ctx context.Context) {
	c.Username = ctx.Params().Get("username")
}

// called after every method (Get() or Post()).
func (c *testControllerBeginAndEndRequestFunc) EndRequest(ctx context.Context) {
	ctx.Writef("done") // append "done" to the response
}

func (c *testControllerBeginAndEndRequestFunc) Get() {
	c.Ctx.Writef(c.Username)
}

func (c *testControllerBeginAndEndRequestFunc) Post() {
	c.Ctx.Writef(c.Username)
}

func TestControllerBeginAndEndRequestFunc(t *testing.T) {
	app := iris.New()
	New(app.Party("/profile/{username}")).
		Handle(new(testControllerBeginAndEndRequestFunc))

	e := httptest.New(t, app)
	usernames := []string{
		"kataras",
		"makis",
		"efi",
		"rg",
		"bill",
		"whoisyourdaddy",
	}
	doneResponse := "done"

	for _, username := range usernames {
		e.GET("/profile/" + username).Expect().Status(iris.StatusOK).
			Body().Equal(username + doneResponse)
		e.POST("/profile/" + username).Expect().Status(iris.StatusOK).
			Body().Equal(username + doneResponse)
	}
}

func TestControllerBeginAndEndRequestFuncBindMiddleware(t *testing.T) {
	app := iris.New()
	usernames := map[string]bool{
		"kataras":        true,
		"makis":          false,
		"efi":            true,
		"rg":             false,
		"bill":           true,
		"whoisyourdaddy": false,
	}
	middlewareCheck := func(ctx context.Context) {
		for username, allow := range usernames {
			if ctx.Params().Get("username") == username && allow {
				ctx.Next()
				return
			}
		}

		ctx.StatusCode(iris.StatusForbidden)
		ctx.Writef("forbidden")
	}

	app.PartyFunc("/profile/{username}", func(r iris.Party) {
		r.Use(middlewareCheck)
		New(r).Handle(new(testControllerBeginAndEndRequestFunc))
	})

	e := httptest.New(t, app)

	doneResponse := "done"

	for username, allow := range usernames {
		getEx := e.GET("/profile/" + username).Expect()
		if allow {
			getEx.Status(iris.StatusOK).
				Body().Equal(username + doneResponse)
		} else {
			getEx.Status(iris.StatusForbidden).Body().Equal("forbidden")
		}

		postEx := e.POST("/profile/" + username).Expect()
		if allow {
			postEx.Status(iris.StatusOK).
				Body().Equal(username + doneResponse)
		} else {
			postEx.Status(iris.StatusForbidden).Body().Equal("forbidden")
		}
	}
}

type Model struct {
	Username string
}

type testControllerEndRequestAwareness struct {
	Ctx context.Context
}

func (c *testControllerEndRequestAwareness) Get() {
	username := c.Ctx.Params().Get("username")
	c.Ctx.Values().Set(c.Ctx.Application().ConfigurationReadOnly().GetViewDataContextKey(),
		map[string]interface{}{
			"TestModel": Model{Username: username},
			"myModel":   Model{Username: username + "2"},
		})
}

func writeModels(ctx context.Context, names ...string) {
	if expected, got := len(names), len(ctx.GetViewData()); expected != got {
		ctx.Writef("expected view data length: %d but got: %d for names: %s", expected, got, names)
		return
	}

	for _, name := range names {

		m, ok := ctx.GetViewData()[name]
		if !ok {
			ctx.Writef("fail load and set the %s", name)
			return
		}

		model, ok := m.(Model)
		if !ok {
			ctx.Writef("fail to override the %s' name by the tag", name)
			return
		}

		ctx.Writef(model.Username)
	}
}

func (c *testControllerEndRequestAwareness) BeginRequest(ctx context.Context) {}
func (c *testControllerEndRequestAwareness) EndRequest(ctx context.Context) {
	writeModels(ctx, "TestModel", "myModel")
}

func TestControllerEndRequestAwareness(t *testing.T) {
	app := iris.New()
	New(app.Party("/era/{username}")).Handle(new(testControllerEndRequestAwareness))

	e := httptest.New(t, app)
	usernames := []string{
		"kataras",
		"makis",
	}

	for _, username := range usernames {
		e.GET("/era/" + username).Expect().Status(iris.StatusOK).
			Body().Equal(username + username + "2")
	}
}

type testBindType struct {
	title string
}

type testControllerBindStruct struct {
	Ctx context.Context

	//  should start with upper letter of course
	TitlePointer *testBindType // should have the value of the "myTitlePtr" on test
	TitleValue   testBindType  // should have the value of the "myTitleV" on test
	Other        string        // just another type to check the field collection, should be empty
}

func (t *testControllerBindStruct) Get() {
	t.Ctx.Writef(t.TitlePointer.title + t.TitleValue.title + t.Other)
}

// test if context can be binded to the controller's function
// without need to declare it to a struct if not needed.
func (t *testControllerBindStruct) GetCtx(ctx iris.Context) {
	ctx.StatusCode(iris.StatusContinue)
}

type testControllerBindDeep struct {
	testControllerBindStruct
}

func (t *testControllerBindDeep) BeforeActivation(b BeforeActivation) {
	b.Dependencies().Add(func(ctx iris.Context) (v testCustomStruct, err error) {
		err = ctx.ReadJSON(&v)
		return
	})
}

func (t *testControllerBindDeep) Get() {
	// 	t.testControllerBindStruct.Get()
	t.Ctx.Writef(t.TitlePointer.title + t.TitleValue.title + t.Other)
}

func (t *testControllerBindDeep) Post(v testCustomStruct) string {
	return v.Name
}

func TestControllerDependencies(t *testing.T) {
	app := iris.New()
	// app.Logger().SetLevel("debug")

	t1, t2 := "my pointer title", "val title"
	// test bind pointer to pointer of the correct type
	myTitlePtr := &testBindType{title: t1}
	// test bind value to value of the correct type
	myTitleV := testBindType{title: t2}
	m := New(app)
	m.Register(myTitlePtr, myTitleV)
	m.Handle(new(testControllerBindStruct))
	m.Clone(app.Party("/deep")).Handle(new(testControllerBindDeep))

	e := httptest.New(t, app)
	expected := t1 + t2
	e.GET("/").Expect().Status(iris.StatusOK).
		Body().Equal(expected)
	e.GET("/ctx").Expect().Status(iris.StatusContinue)

	e.GET("/deep").Expect().Status(iris.StatusOK).
		Body().Equal(expected)

	e.POST("/deep").WithJSON(iris.Map{"name": "kataras"}).Expect().Status(iris.StatusOK).
		Body().Equal("kataras")

	e.POST("/deep").Expect().Status(iris.StatusBadRequest).
		Body().Equal("unexpected end of JSON input")
}

type testCtrl0 struct {
	testCtrl00
}

func (c *testCtrl0) Get() string {
	return c.Ctx.Params().Get("username")
}

func (c *testCtrl0) EndRequest(ctx context.Context) {
	if c.TitlePointer == nil {
		ctx.Writef("\nTitlePointer is nil!\n")
	} else {
		ctx.Writef(c.TitlePointer.title)
	}

	//should be the same as `.testCtrl000.testCtrl0000.EndRequest(ctx)`
	c.testCtrl00.EndRequest(ctx)
}

type testCtrl00 struct {
	Ctx context.Context

	testCtrl000
}

type testCtrl000 struct {
	testCtrl0000

	TitlePointer *testBindType
}

type testCtrl0000 struct {
}

func (c *testCtrl0000) BeginRequest(ctx context.Context) {}
func (c *testCtrl0000) EndRequest(ctx context.Context) {
	ctx.Writef("finish")
}

func TestControllerInsideControllerRecursively(t *testing.T) {
	var (
		username = "gerasimos"
		title    = "mytitle"
		expected = username + title + "finish"
	)

	app := iris.New()
	m := New(app.Party("/user/{username}"))
	m.Register(&testBindType{title: title})
	m.Handle(new(testCtrl0))

	e := httptest.New(t, app)
	e.GET("/user/" + username).Expect().
		Status(iris.StatusOK).Body().Equal(expected)
}

type testControllerRelPathFromFunc struct{}

func (c *testControllerRelPathFromFunc) BeginRequest(ctx context.Context) {}
func (c *testControllerRelPathFromFunc) EndRequest(ctx context.Context) {
	ctx.Writef("%s:%s", ctx.Method(), ctx.Path())
}

func (c *testControllerRelPathFromFunc) Get()                         {}
func (c *testControllerRelPathFromFunc) GetBy(uint64)                 {}
func (c *testControllerRelPathFromFunc) GetUint8RatioBy(uint8)        {}
func (c *testControllerRelPathFromFunc) GetInt64RatioBy(int64)        {}
func (c *testControllerRelPathFromFunc) GetAnythingByWildcard(string) {}

func (c *testControllerRelPathFromFunc) GetLogin()  {}
func (c *testControllerRelPathFromFunc) PostLogin() {}

func (c *testControllerRelPathFromFunc) GetAdminLogin() {}

func (c *testControllerRelPathFromFunc) PutSomethingIntoThis() {}

func (c *testControllerRelPathFromFunc) GetSomethingBy(bool) {}

func (c *testControllerRelPathFromFunc) GetSomethingByBy(string, int) {}

func (c *testControllerRelPathFromFunc) GetSomethingNewBy(string, int)      {} // two input arguments, one By which is the latest word.
func (c *testControllerRelPathFromFunc) GetSomethingByElseThisBy(bool, int) {} // two input arguments

func TestControllerRelPathFromFunc(t *testing.T) {
	app := iris.New()
	New(app).Handle(new(testControllerRelPathFromFunc))

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/")

	e.GET("/18446744073709551615").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/18446744073709551615")
	e.GET("/uint8/ratio/255").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/uint8/ratio/255")
	e.GET("/uint8/ratio/256").Expect().Status(iris.StatusNotFound)
	e.GET("/int64/ratio/-42").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/int64/ratio/-42")
	e.GET("/something/true").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/something/true")
	e.GET("/something/false").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/something/false")
	e.GET("/something/truee").Expect().Status(iris.StatusNotFound)
	e.GET("/something/falsee").Expect().Status(iris.StatusNotFound)
	e.GET("/something/kataras/42").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/something/kataras/42")
	e.GET("/something/new/kataras/42").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/something/new/kataras/42")
	e.GET("/something/true/else/this/42").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/something/true/else/this/42")

	e.GET("/login").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/login")
	e.POST("/login").Expect().Status(iris.StatusOK).
		Body().Equal("POST:/login")
	e.GET("/admin/login").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/admin/login")
	e.PUT("/something/into/this").Expect().Status(iris.StatusOK).
		Body().Equal("PUT:/something/into/this")
	e.GET("/42").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/42")
	e.GET("/anything/here").Expect().Status(iris.StatusOK).
		Body().Equal("GET:/anything/here")

}

type testControllerActivateListener struct {
	TitlePointer *testBindType
}

func (c *testControllerActivateListener) BeforeActivation(b BeforeActivation) {
	b.Dependencies().AddOnce(&testBindType{title: "default title"})
}

func (c *testControllerActivateListener) Get() string {
	return c.TitlePointer.title
}

func TestControllerActivateListener(t *testing.T) {
	app := iris.New()
	New(app).Handle(new(testControllerActivateListener))
	m := New(app)
	m.Register(&testBindType{
		title: "my title",
	})
	m.Party("/manual").Handle(new(testControllerActivateListener))
	// or
	m.Party("/manual2").Handle(&testControllerActivateListener{
		TitlePointer: &testBindType{
			title: "my title",
		},
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).
		Body().Equal("default title")
	e.GET("/manual").Expect().Status(iris.StatusOK).
		Body().Equal("my title")
	e.GET("/manual2").Expect().Status(iris.StatusOK).
		Body().Equal("my title")
}

type testControllerNotCreateNewDueManuallySettingAllFields struct {
	T *testing.T

	TitlePointer *testBindType
}

func (c *testControllerNotCreateNewDueManuallySettingAllFields) AfterActivation(a AfterActivation) {
	if n := a.DependenciesReadOnly().Len(); n != 2 {
		c.T.Fatalf(`expecting 2 dependency, the 'T' and the 'TitlePointer' that we manually insert
			and the fields total length is 2 so it will not create a new controller on each request
			however the dependencies are available here
			although the struct injector is being ignored when
			creating the controller's handlers because we set it to invalidate state at "newControllerActivator"
			--  got dependencies length: %d`, n)
	}

	if !a.Singleton() {
		c.T.Fatalf(`this controller should be tagged as Singleton. It shouldn't be tagged used as request scoped(create new instances on each request),
		 it doesn't contain any dynamic value or dependencies that should be binded via the iris mvc engine`)
	}
}

func (c *testControllerNotCreateNewDueManuallySettingAllFields) Get() string {
	return c.TitlePointer.title
}

func TestControllerNotCreateNewDueManuallySettingAllFields(t *testing.T) {
	app := iris.New()
	New(app).Handle(&testControllerNotCreateNewDueManuallySettingAllFields{
		T: t,
		TitlePointer: &testBindType{
			title: "my title",
		},
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).
		Body().Equal("my title")
}
