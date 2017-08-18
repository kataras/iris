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

func (t *testControllerPersistence) Get() {
	t.Ctx.WriteString(t.Data)
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
func (t *testControllerBeginAndEndRequestFunc) BeginRequest(ctx context.Context) {
	t.Controller.BeginRequest(ctx)
	t.Username = ctx.Params().Get("username")
	// or t.Params.Get("username") because the
	// t.Ctx == ctx and is being initialized at the t.Controller.BeginRequest.
}

// called after every method (Get() or Post()).
func (t *testControllerBeginAndEndRequestFunc) EndRequest(ctx context.Context) {
	ctx.Writef("done") // append "done" to the response
	t.Controller.EndRequest(ctx)
}

func (t *testControllerBeginAndEndRequestFunc) Get() {
	t.Ctx.Writef(t.Username)
}

func (t *testControllerBeginAndEndRequestFunc) Post() {
	t.Ctx.Writef(t.Username)
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

type Model struct {
	Username string
}

type testControllerModel struct {
	mvc.Controller

	TestModel  Model `iris:"model" name:"myModel"`
	TestModel2 Model `iris:"model"`
}

func (t *testControllerModel) Get() {
	username := t.Ctx.Params().Get("username")
	t.TestModel = Model{Username: username}
	t.TestModel2 = Model{Username: username + "2"}
}

func (t *testControllerModel) EndRequest(ctx context.Context) {
	// t.Ctx == ctx

	m, ok := t.Ctx.GetViewData()["myModel"]
	if !ok {
		t.Ctx.Writef("fail TestModel load and set")
		return
	}

	model, ok := m.(Model)

	if !ok {
		t.Ctx.Writef("fail to override the TestModel name by the tag")
		return
	}

	// test without custom name tag, should have the field's nae.
	m, ok = t.Ctx.GetViewData()["TestModel2"]
	if !ok {
		t.Ctx.Writef("fail TestModel2 load and set")
		return
	}

	model2, ok := m.(Model)

	if !ok {
		t.Ctx.Writef("fail to override the TestModel2 name by the tag")
		return
	}

	// models are being rendered via the View at ViewData but
	// we just test it here, so print it back.
	t.Ctx.Writef(model.Username + model2.Username)

	t.Controller.EndRequest(ctx)
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
