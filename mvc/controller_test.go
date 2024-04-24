// black-box testing
package mvc_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/httptest"

	. "github.com/kataras/iris/v12/mvc"
)

type testController struct {
	Ctx *context.Context
}

var writeMethod = func(ctx *context.Context) {
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
	testControllerAll struct{ Ctx *context.Context }
	testControllerAny struct{ Ctx *context.Context } // exactly the same as All.
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
			Body().IsEqual(method)

		e.Request(method, "/all").Expect().Status(iris.StatusOK).
			Body().IsEqual(method)

		e.Request(method, "/any").Expect().Status(iris.StatusOK).
			Body().IsEqual(method)
	}
}

type testControllerBeginAndEndRequestFunc struct {
	Ctx *context.Context

	Username string
}

// called before of every method (Get() or Post()).
//
// useful when more than one methods using the
// same request values or context's function calls.
func (c *testControllerBeginAndEndRequestFunc) BeginRequest(ctx *context.Context) {
	c.Username = ctx.Params().Get("username")
}

// called after every method (Get() or Post()).
func (c *testControllerBeginAndEndRequestFunc) EndRequest(ctx *context.Context) {
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
			Body().IsEqual(username + doneResponse)
		e.POST("/profile/" + username).Expect().Status(iris.StatusOK).
			Body().IsEqual(username + doneResponse)
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
	middlewareCheck := func(ctx *context.Context) {
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
				Body().IsEqual(username + doneResponse)
		} else {
			getEx.Status(iris.StatusForbidden).Body().IsEqual("forbidden")
		}

		postEx := e.POST("/profile/" + username).Expect()
		if allow {
			postEx.Status(iris.StatusOK).
				Body().IsEqual(username + doneResponse)
		} else {
			postEx.Status(iris.StatusForbidden).Body().IsEqual("forbidden")
		}
	}
}

type Model struct {
	Username string
}

type testControllerEndRequestAwareness struct {
	Ctx *context.Context
}

func (c *testControllerEndRequestAwareness) Get() {
	username := c.Ctx.Params().Get("username")
	c.Ctx.Values().Set(c.Ctx.Application().ConfigurationReadOnly().GetViewDataContextKey(),
		map[string]interface{}{
			"TestModel": Model{Username: username},
			"myModel":   Model{Username: username + "2"},
		})
}

func writeModels(ctx *context.Context, names ...string) {
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

func (c *testControllerEndRequestAwareness) BeginRequest(ctx *context.Context) {}
func (c *testControllerEndRequestAwareness) EndRequest(ctx *context.Context) {
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
			Body().IsEqual(username + username + "2")
	}
}

type testBindType struct {
	title string
}

type testControllerBindStruct struct {
	Ctx *context.Context

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
	b.Dependencies().Register(func(ctx iris.Context) (v testCustomStruct, err error) {
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
		Body().IsEqual(expected)
	e.GET("/ctx").Expect().Status(iris.StatusContinue)

	e.GET("/deep").Expect().Status(iris.StatusOK).
		Body().IsEqual(expected)

	e.POST("/deep").WithJSON(iris.Map{"name": "kataras"}).Expect().Status(iris.StatusOK).
		Body().IsEqual("kataras")

	e.POST("/deep").Expect().Status(iris.StatusBadRequest).
		Body().IsEqual("EOF")
}

type testCtrl0 struct {
	testCtrl00
}

func (c *testCtrl0) Get() string {
	return c.Ctx.Params().Get("username")
}

func (c *testCtrl0) EndRequest(ctx *context.Context) {
	if c.TitlePointer == nil {
		ctx.Writef("\nTitlePointer is nil!\n")
	} else {
		ctx.Writef(c.TitlePointer.title)
	}

	// should be the same as `.testCtrl000.testCtrl0000.EndRequest(ctx)`
	c.testCtrl00.EndRequest(ctx)
}

type testCtrl00 struct {
	Ctx *context.Context

	testCtrl000
}

type testCtrl000 struct {
	testCtrl0000

	TitlePointer *testBindType
}

type testCtrl0000 struct {
}

func (c *testCtrl0000) BeginRequest(ctx *context.Context) {}
func (c *testCtrl0000) EndRequest(ctx *context.Context) {
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
		Status(iris.StatusOK).Body().IsEqual(expected)
}

type testControllerRelPathFromFunc struct{}

func (c *testControllerRelPathFromFunc) BeginRequest(ctx *context.Context) {}
func (c *testControllerRelPathFromFunc) EndRequest(ctx *context.Context) {
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

func (c *testControllerRelPathFromFunc) GetLocationX()      {}
func (c *testControllerRelPathFromFunc) GetLocationXY()     {}
func (c *testControllerRelPathFromFunc) GetLocationZBy(int) {}

func TestControllerRelPathFromFunc(t *testing.T) {
	app := iris.New()
	New(app).Handle(new(testControllerRelPathFromFunc))

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/")

	e.GET("/18446744073709551615").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/18446744073709551615")
	e.GET("/uint8/ratio/255").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/uint8/ratio/255")
	e.GET("/uint8/ratio/256").Expect().Status(iris.StatusNotFound)
	e.GET("/int64/ratio/-42").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/int64/ratio/-42")
	e.GET("/something/true").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/something/true")
	e.GET("/something/false").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/something/false")
	e.GET("/something/truee").Expect().Status(iris.StatusNotFound)
	e.GET("/something/falsee").Expect().Status(iris.StatusNotFound)
	e.GET("/something/kataras/42").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/something/kataras/42")
	e.GET("/something/new/kataras/42").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/something/new/kataras/42")
	e.GET("/something/true/else/this/42").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/something/true/else/this/42")

	e.GET("/login").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/login")
	e.POST("/login").Expect().Status(iris.StatusOK).
		Body().IsEqual("POST:/login")
	e.GET("/admin/login").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/admin/login")
	e.PUT("/something/into/this").Expect().Status(iris.StatusOK).
		Body().IsEqual("PUT:/something/into/this")
	e.GET("/42").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/42")
	e.GET("/anything/here").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/anything/here")

	e.GET("/location/x").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/location/x")
	e.GET("/location/x/y").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/location/x/y")
	e.GET("/location/z/42").Expect().Status(iris.StatusOK).
		Body().IsEqual("GET:/location/z/42")
}

type testControllerActivateListener struct {
	TitlePointer *testBindType
}

func (c *testControllerActivateListener) BeforeActivation(b BeforeActivation) {
	b.Dependencies().Register(&testBindType{title: "overrides the dependency but not the field"}) // overrides the `Register` previous calls.

	// b.Handle("POST", "/me/tos-read", "MeTOSRead")
	// b.Handle("GET", "/me/tos-read", "MeTOSRead")
	// OR:
	b.HandleMany("GET POST", "/me/tos-read", "MeTOSRead")
}

func (c *testControllerActivateListener) Get() string {
	return c.TitlePointer.title
}

func (c *testControllerActivateListener) MeTOSRead() string {
	return "MeTOSRead"
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
			title: "my manual title",
		},
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusOK).
		Body().IsEqual("overrides the dependency but not the field")
	e.GET("/me/tos-read").Expect().Status(iris.StatusOK).
		Body().IsEqual("MeTOSRead")
	e.POST("/me/tos-read").Expect().Status(iris.StatusOK).
		Body().IsEqual("MeTOSRead")

	e.GET("/manual").Expect().Status(iris.StatusOK).
		Body().IsEqual("overrides the dependency but not the field")
	e.GET("/manual2").Expect().Status(iris.StatusOK).
		Body().IsEqual("my manual title")
}

type testControllerNotCreateNewDueManuallySettingAllFields struct {
	T *testing.T

	TitlePointer *testBindType
}

func (c *testControllerNotCreateNewDueManuallySettingAllFields) AfterActivation(a AfterActivation) {
	if n := len(a.DependenciesReadOnly()) - len(hero.BuiltinDependencies) - 1; /* Application */ n != 1 {
		c.T.Fatalf(`expecting 1 dependency;
- the 'T' and the 'TitlePointer' are manually binded (nonzero fields on initilization)
- controller has no more than these two fields, it's a singleton
- however, the dependencies length here should be 1 because the injector's options handler dependencies contains the controller's value dependency itself
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
		Body().IsEqual("my title")
}

type testControllerRequestScopedDependencies struct {
	MyContext    *testMyContext
	CustomStruct *testCustomStruct
}

func (c *testControllerRequestScopedDependencies) Get() *testCustomStruct {
	return c.CustomStruct
}

func (c *testControllerRequestScopedDependencies) GetCustomContext() string {
	return c.MyContext.OtherField
}

func newRequestDep1(ctx *context.Context) *testCustomStruct {
	return &testCustomStruct{
		Name: ctx.URLParam("name"),
		Age:  ctx.URLParamIntDefault("age", 0),
	}
}

type testMyContext struct {
	Context    *context.Context
	OtherField string
}

func newRequestDep2(ctx *context.Context) *testMyContext {
	return &testMyContext{
		Context:    ctx,
		OtherField: "test",
	}
}

func TestControllerRequestScopedDependencies(t *testing.T) {
	app := iris.New()
	m := New(app)
	m.Register(newRequestDep1)
	m.Register(newRequestDep2)
	m.Handle(new(testControllerRequestScopedDependencies))

	e := httptest.New(t, app)
	e.GET("/").WithQuery("name", "kataras").WithQuery("age", 27).
		Expect().Status(httptest.StatusOK).JSON().IsEqual(&testCustomStruct{
		Name: "kataras",
		Age:  27,
	})
	e.GET("/custom/context").Expect().Status(httptest.StatusOK).Body().IsEqual("test")
}

type (
	testServiceDoSomething struct{}

	TestControllerAsDeepDep struct {
		Ctx     iris.Context
		Service *testServiceDoSomething
	}

	FooController struct {
		TestControllerAsDeepDep
	}

	BarController struct {
		FooController
	}

	FinalController struct {
		BarController
	}
)

func (s *testServiceDoSomething) DoSomething(ctx iris.Context) {
	ctx.WriteString("foo bar")
}

func (c *FinalController) GetSomething() {
	c.Service.DoSomething(c.Ctx)
}

func TestControllersInsideControllerDeep(t *testing.T) {
	app := iris.New()
	m := New(app)
	m.Register(new(testServiceDoSomething))
	m.Handle(new(FinalController))

	e := httptest.New(t, app)
	e.GET("/something").Expect().Status(httptest.StatusOK).Body().IsEqual("foo bar")
}

type testApplicationDependency struct {
	App *Application
}

func (c *testApplicationDependency) Get() string {
	return c.App.Name
}

func TestApplicationDependency(t *testing.T) {
	app := iris.New()
	m := New(app).SetName("app1")
	m.Handle(new(testApplicationDependency))

	m2 := m.Clone(app.Party("/other")).SetName("app2")
	m2.Handle(new(testApplicationDependency))

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("app1")
	e.GET("/other").Expect().Status(httptest.StatusOK).Body().IsEqual("app2")
}

type testControllerMethodHandlerBindStruct struct{}

type bindStructData struct {
	Name string `json:"name" url:"name"`
}

func (*testControllerMethodHandlerBindStruct) Any(data bindStructData) bindStructData {
	return data
}

func (*testControllerMethodHandlerBindStruct) PostBySlice(id uint64, manyData []bindStructData) []bindStructData {
	return manyData
}

type dataSlice []bindStructData

func (*testControllerMethodHandlerBindStruct) PostBySlicetype(id uint64, manyData dataSlice) dataSlice {
	return manyData
}

type dataSlicePtr []*bindStructData

func (*testControllerMethodHandlerBindStruct) PostBySlicetypeptr(id uint64, manyData dataSlicePtr) dataSlicePtr {
	return manyData
}

func TestControllerMethodHandlerBindStruct(t *testing.T) {
	app := iris.New()

	m := New(app.Party("/data"))
	m.HandleError(func(ctx iris.Context, err error) {
		t.Fatalf("Path: %s, Error: %v", ctx.Path(), err)
	})

	m.Handle(new(testControllerMethodHandlerBindStruct))

	data := bindStructData{Name: "kataras"}
	manyData := []bindStructData{data, {"john doe"}}

	e := httptest.New(t, app)
	e.GET("/data").WithQueryObject(data).Expect().Status(httptest.StatusOK).JSON().IsEqual(data)
	e.PATCH("/data").WithJSON(data).Expect().Status(httptest.StatusOK).JSON().IsEqual(data)
	e.POST("/data/42/slice").WithJSON(manyData).Expect().Status(httptest.StatusOK).JSON().IsEqual(manyData)
	e.POST("/data/42/slicetype").WithJSON(manyData).Expect().Status(httptest.StatusOK).JSON().IsEqual(manyData)
	e.POST("/data/42/slicetypeptr").WithJSON(manyData).Expect().Status(httptest.StatusOK).JSON().IsEqual(manyData)
	// more tests inside the hero package itself.
}

func TestErrorHandlerContinue(t *testing.T) {
	app := iris.New()
	m := New(app)
	m.Handle(new(testControllerErrorHandlerContinue))
	m.Handle(new(testControllerFieldErrorHandlerContinue))
	e := httptest.New(t, app)

	for _, path := range []string{"/test", "/test/field"} {
		e.POST(path).WithMultipart().
			WithFormField("username", "makis").
			WithFormField("age", "27").
			WithFormField("unknown", "continue").
			Expect().Status(httptest.StatusOK).Body().IsEqual("makis is 27 years old\n")
	}
}

type testControllerErrorHandlerContinue struct{}

type registerForm struct {
	Username string `form:"username"`
	Age      int    `form:"age"`
}

func (c *testControllerErrorHandlerContinue) HandleError(ctx iris.Context, err error) {
	if iris.IsErrPath(err) {
		return // continue.
	}

	ctx.StopWithError(iris.StatusBadRequest, err)
}

func (c *testControllerErrorHandlerContinue) PostTest(form registerForm) string {
	return fmt.Sprintf("%s is %d years old\n", form.Username, form.Age)
}

type testControllerFieldErrorHandlerContinue struct {
	Form *registerForm
}

func (c *testControllerFieldErrorHandlerContinue) HandleError(ctx iris.Context, err error) {
	if iris.IsErrPath(err) {
		return // continue.
	}

	ctx.StopWithError(iris.StatusBadRequest, err)
}

func (c *testControllerFieldErrorHandlerContinue) PostTestField() string {
	return fmt.Sprintf("%s is %d years old\n", c.Form.Username, c.Form.Age)
}
