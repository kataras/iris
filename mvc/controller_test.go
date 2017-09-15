// black-box testing
package mvc_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/mvc"

	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/httptest"
)

type testController struct {
	mvc.Controller
}

var writeMethod = func(c mvc.Controller) {
	c.Ctx.Writef(c.Ctx.Method())
}

func (c *testController) Get() {
	writeMethod(c.Controller)
}
func (c *testController) Post() {
	writeMethod(c.Controller)
}
func (c *testController) Put() {
	writeMethod(c.Controller)
}
func (c *testController) Delete() {
	writeMethod(c.Controller)
}
func (c *testController) Connect() {
	writeMethod(c.Controller)
}
func (c *testController) Head() {
	writeMethod(c.Controller)
}
func (c *testController) Patch() {
	writeMethod(c.Controller)
}
func (c *testController) Options() {
	writeMethod(c.Controller)
}
func (c *testController) Trace() {
	writeMethod(c.Controller)
}

type (
	testControllerAll struct{ mvc.Controller }
	testControllerAny struct{ mvc.Controller } // exactly the same as All
)

func (c *testControllerAll) All() {
	writeMethod(c.Controller)
}

func (c *testControllerAny) Any() {
	writeMethod(c.Controller)
}

func TestControllerMethodFuncs(t *testing.T) {
	app := iris.New()
	app.Controller("/", new(testController))
	app.Controller("/all", new(testControllerAll))
	app.Controller("/any", new(testControllerAny))

	e := httptest.New(t, app)
	for _, method := range router.AllMethods {

		e.Request(method, "/").Expect().Status(httptest.StatusOK).
			Body().Equal(method)

		e.Request(method, "/all").Expect().Status(httptest.StatusOK).
			Body().Equal(method)

		e.Request(method, "/any").Expect().Status(httptest.StatusOK).
			Body().Equal(method)
	}
}

func TestControllerMethodAndPathHandleMany(t *testing.T) {
	app := iris.New()
	app.Controller("/ /path1 /path2 /path3", new(testController))

	e := httptest.New(t, app)
	for _, method := range router.AllMethods {

		e.Request(method, "/").Expect().Status(httptest.StatusOK).
			Body().Equal(method)

		e.Request(method, "/path1").Expect().Status(httptest.StatusOK).
			Body().Equal(method)

		e.Request(method, "/path2").Expect().Status(httptest.StatusOK).
			Body().Equal(method)
	}
}

type testControllerPersistence struct {
	mvc.Controller
	Data string `iris:"persistence"`
}

func (c *testControllerPersistence) Get() {
	c.Ctx.WriteString(c.Data)
}

func TestControllerPersistenceFields(t *testing.T) {
	data := "this remains the same for all requests"
	app := iris.New()
	app.Controller("/", &testControllerPersistence{Data: data})
	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal(data)
}

type testControllerBeginAndEndRequestFunc struct {
	mvc.Controller

	Username string
}

// called before of every method (Get() or Post()).
//
// useful when more than one methods using the
// same request values or context's function calls.
func (c *testControllerBeginAndEndRequestFunc) BeginRequest(ctx context.Context) {
	c.Controller.BeginRequest(ctx)
	c.Username = ctx.Params().Get("username")
	// or t.Params.Get("username") because the
	// t.Ctx == ctx and is being initialized at the t.Controller.BeginRequest.
}

// called after every method (Get() or Post()).
func (c *testControllerBeginAndEndRequestFunc) EndRequest(ctx context.Context) {
	ctx.Writef("done") // append "done" to the response
	c.Controller.EndRequest(ctx)
}

func (c *testControllerBeginAndEndRequestFunc) Get() {
	c.Ctx.Writef(c.Username)
}

func (c *testControllerBeginAndEndRequestFunc) Post() {
	c.Ctx.Writef(c.Username)
}

func TestControllerBeginAndEndRequestFunc(t *testing.T) {
	app := iris.New()
	app.Controller("/profile/{username}", new(testControllerBeginAndEndRequestFunc))

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
		e.GET("/profile/" + username).Expect().Status(httptest.StatusOK).
			Body().Equal(username + doneResponse)
		e.POST("/profile/" + username).Expect().Status(httptest.StatusOK).
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

		ctx.StatusCode(httptest.StatusForbidden)
		ctx.Writef("forbidden")
	}

	app.Controller("/profile/{username}", new(testControllerBeginAndEndRequestFunc), middlewareCheck)

	e := httptest.New(t, app)

	doneResponse := "done"

	for username, allow := range usernames {
		getEx := e.GET("/profile/" + username).Expect()
		if allow {
			getEx.Status(httptest.StatusOK).
				Body().Equal(username + doneResponse)
		} else {
			getEx.Status(httptest.StatusForbidden).Body().Equal("forbidden")
		}

		postEx := e.POST("/profile/" + username).Expect()
		if allow {
			postEx.Status(httptest.StatusOK).
				Body().Equal(username + doneResponse)
		} else {
			postEx.Status(httptest.StatusForbidden).Body().Equal("forbidden")
		}
	}
}

type Model struct {
	Username string
}

type testControllerModel struct {
	mvc.Controller

	TestModel  Model `iris:"model" name:"myModel"`
	TestModel2 Model `iris:"model"`
}

func (c *testControllerModel) Get() {
	username := c.Ctx.Params().Get("username")
	c.TestModel = Model{Username: username}
	c.TestModel2 = Model{Username: username + "2"}
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

func (c *testControllerModel) EndRequest(ctx context.Context) {
	writeModels(ctx, "myModel", "TestModel2")
	c.Controller.EndRequest(ctx)
}
func TestControllerModel(t *testing.T) {
	app := iris.New()
	app.Controller("/model/{username}", new(testControllerModel))

	e := httptest.New(t, app)
	usernames := []string{
		"kataras",
		"makis",
	}

	for _, username := range usernames {
		e.GET("/model/" + username).Expect().Status(httptest.StatusOK).
			Body().Equal(username + username + "2")
	}
}

type testBindType struct {
	title string
}

type testControllerBindStruct struct {
	mvc.Controller
	//  should start with upper letter of course
	TitlePointer *testBindType // should have the value of the "myTitlePtr" on test
	TitleValue   testBindType  // should have the value of the "myTitleV" on test
	Other        string        // just another type to check the field collection, should be empty
}

func (t *testControllerBindStruct) Get() {
	t.Ctx.Writef(t.TitlePointer.title + t.TitleValue.title + t.Other)
}

type testControllerBindDeep struct {
	testControllerBindStruct
}

func (t *testControllerBindDeep) Get() {
	// 	t.testControllerBindStruct.Get()
	t.Ctx.Writef(t.TitlePointer.title + t.TitleValue.title + t.Other)
}
func TestControllerBind(t *testing.T) {
	app := iris.New()

	t1, t2 := "my pointer title", "val title"
	// test bind pointer to pointer of the correct type
	myTitlePtr := &testBindType{title: t1}
	// test bind value to value of the correct type
	myTitleV := testBindType{title: t2}

	app.Controller("/", new(testControllerBindStruct), myTitlePtr, myTitleV)
	app.Controller("/deep", new(testControllerBindDeep), myTitlePtr, myTitleV)

	e := httptest.New(t, app)
	expected := t1 + t2
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal(expected)
	e.GET("/deep").Expect().Status(httptest.StatusOK).
		Body().Equal(expected)
}

type (
	UserController            struct{ mvc.Controller }
	Profile                   struct{ mvc.Controller }
	UserProfilePostController struct{ mvc.Controller }
)

func writeRelatives(c mvc.Controller) {
	c.Ctx.JSON(context.Map{
		"RelPath":  c.RelPath(),
		"TmplPath": c.RelTmpl(),
	})
}
func (c *UserController) Get() {
	writeRelatives(c.Controller)
}

func (c *Profile) Get() {
	writeRelatives(c.Controller)
}

func (c *UserProfilePostController) Get() {
	writeRelatives(c.Controller)
}

func TestControllerRelPathAndRelTmpl(t *testing.T) {
	app := iris.New()
	var tests = map[string]context.Map{
		// UserController
		"/user":    {"RelPath": "/", "TmplPath": "user/"},
		"/user/42": {"RelPath": "/42", "TmplPath": "user/"},
		"/user/me": {"RelPath": "/me", "TmplPath": "user/"},
		// Profile (without Controller suffix, should work as expected)
		"/profile":    {"RelPath": "/", "TmplPath": "profile/"},
		"/profile/42": {"RelPath": "/42", "TmplPath": "profile/"},
		"/profile/me": {"RelPath": "/me", "TmplPath": "profile/"},
		// UserProfilePost
		"/user/profile/post":      {"RelPath": "/", "TmplPath": "user/profile/post/"},
		"/user/profile/post/42":   {"RelPath": "/42", "TmplPath": "user/profile/post/"},
		"/user/profile/post/mine": {"RelPath": "/mine", "TmplPath": "user/profile/post/"},
	}

	app.Controller("/user /user/me /user/{id}",
		new(UserController))

	app.Controller("/profile /profile/me /profile/{id}",
		new(Profile))

	app.Controller("/user/profile/post /user/profile/post/mine /user/profile/post/{id}",
		new(UserProfilePostController))

	e := httptest.New(t, app)
	for path, tt := range tests {
		e.GET(path).Expect().Status(httptest.StatusOK).JSON().Equal(tt)
	}
}

type testCtrl0 struct {
	testCtrl00
}

func (c *testCtrl0) Get() {
	username := c.Params.Get("username")
	c.Model = Model{Username: username}
}

func (c *testCtrl0) EndRequest(ctx context.Context) {
	writeModels(ctx, "myModel")

	if c.TitlePointer == nil {
		ctx.Writef("\nTitlePointer is nil!\n")
	} else {
		ctx.Writef(c.TitlePointer.title)
	}

	//should be the same as `.testCtrl000.testCtrl0000.EndRequest(ctx)`
	c.testCtrl00.EndRequest(ctx)
}

type testCtrl00 struct {
	testCtrl000

	Model Model `iris:"model" name:"myModel"`
}

type testCtrl000 struct {
	testCtrl0000

	TitlePointer *testBindType
}

type testCtrl0000 struct {
	mvc.Controller
}

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

	app.Controller("/user/{username}", new(testCtrl0),
		&testBindType{title: title})

	e := httptest.New(t, app)
	e.GET("/user/" + username).Expect().
		Status(httptest.StatusOK).Body().Equal(expected)
}

type testControllerRelPathFromFunc struct{ mvc.Controller }

func (c *testControllerRelPathFromFunc) EndRequest(ctx context.Context) {
	ctx.Writef("%s:%s", ctx.Method(), ctx.Path())
	c.Controller.EndRequest(ctx)
}

func (c *testControllerRelPathFromFunc) Get()                 {}
func (c *testControllerRelPathFromFunc) GetBy(int64)          {}
func (c *testControllerRelPathFromFunc) GetByWildcard(string) {}

func (c *testControllerRelPathFromFunc) GetLogin()  {}
func (c *testControllerRelPathFromFunc) PostLogin() {}

func (c *testControllerRelPathFromFunc) GetAdminLogin() {}

func (c *testControllerRelPathFromFunc) PutSomethingIntoThis()              {}
func (c *testControllerRelPathFromFunc) GetSomethingBy(bool)                {}
func (c *testControllerRelPathFromFunc) GetSomethingByElseThisBy(bool, int) {} // two input arguments

func TestControllerRelPathFromFunc(t *testing.T) {
	app := iris.New()
	app.Controller("/", new(testControllerRelPathFromFunc))

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/")

	e.GET("/42").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/42")
	e.GET("/something/true").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/something/true")
	e.GET("/something/false").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/something/false")
	e.GET("/something/truee").Expect().Status(httptest.StatusNotFound)
	e.GET("/something/falsee").Expect().Status(httptest.StatusNotFound)
	e.GET("/something/true/else/this/42").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/something/true/else/this/42")

	e.GET("/login").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/login")
	e.POST("/login").Expect().Status(httptest.StatusOK).
		Body().Equal("POST:/login")
	e.GET("/admin/login").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/admin/login")
	e.PUT("/something/into/this").Expect().Status(httptest.StatusOK).
		Body().Equal("PUT:/something/into/this")
	e.GET("/42").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/42")
	e.GET("/anything/here").Expect().Status(httptest.StatusOK).
		Body().Equal("GET:/anything/here")
}
