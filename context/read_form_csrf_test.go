package context_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
)

type csrfFormNoToken struct {
	Name string `form:"name"`
	Age  int    `form:"age"`
}

// The schema decoder splits a key on dots, so "csrf.token" binds to a field
// tagged "token" inside a field tagged "csrf". A flat `form:"csrf.token"` tag
// never matches the key.
type csrfFormWithToken struct {
	Name string `form:"name"`
	CSRF struct {
		Token string `form:"token"`
	} `form:"csrf"`
}

// newReadFormCSRFApp writes "ok name=... age=..." on success, and on failure
// "err path=<IsErrPath> csrf=<IsErrPathCSRFToken>: <error text>".
func newReadFormCSRFApp() *iris.Application {
	app := iris.New()

	app.Post("/plain", func(ctx iris.Context) {
		var f csrfFormNoToken
		if err := ctx.ReadForm(&f); err != nil {
			ctx.Writef("err path=%v csrf=%v: %v", iris.IsErrPath(err), context.IsErrPathCSRFToken(err), err)
			return
		}
		ctx.Writef("ok name=%s age=%d", f.Name, f.Age)
	})

	app.Post("/token", func(ctx iris.Context) {
		var f csrfFormWithToken
		if err := ctx.ReadForm(&f); err != nil {
			ctx.Writef("err: %v", err)
			return
		}
		ctx.Writef("ok name=%s token=%s", f.Name, f.CSRF.Token)
	})

	return app
}

func TestReadFormCSRFTokenAloneIsDropped(t *testing.T) {
	e := httptest.New(t, newReadFormCSRFApp())

	e.POST("/plain").
		WithFormField(context.CSRFTokenFormKey, "abc").
		WithFormField("name", "kataras").
		WithFormField("age", "30").
		Expect().Status(httptest.StatusOK).Body().IsEqual("ok name=kataras age=30")
}

func TestReadFormCSRFTokenKeepsConversionError(t *testing.T) {
	e := httptest.New(t, newReadFormCSRFApp())

	body := e.POST("/plain").
		WithFormField(context.CSRFTokenFormKey, "abc").
		WithFormField("name", "kataras").
		WithFormField("age", "thirty").
		Expect().Status(httptest.StatusOK).Body()

	body.HasPrefix("err path=false csrf=false: ")
	body.Contains(`"age"`)
	body.NotContains(context.CSRFTokenFormKey)
}

func TestReadFormCSRFTokenKeepsOtherUnknownKey(t *testing.T) {
	e := httptest.New(t, newReadFormCSRFApp())

	body := e.POST("/plain").
		WithFormField(context.CSRFTokenFormKey, "abc").
		WithFormField("name", "kataras").
		WithFormField("nickname", "makis").
		Expect().Status(httptest.StatusOK).Body()

	body.HasPrefix("err path=true csrf=false: ")
	body.Contains(`"nickname"`)
	body.NotContains(context.CSRFTokenFormKey)
}

func TestReadFormWithoutCSRFTokenUnchanged(t *testing.T) {
	e := httptest.New(t, newReadFormCSRFApp())

	e.POST("/plain").
		WithFormField("name", "kataras").
		WithFormField("age", "thirty").
		Expect().Status(httptest.StatusOK).Body().
		IsEqual(`err path=false csrf=false: schema: error converting value for "age"`)

	e.POST("/plain").
		WithFormField("name", "kataras").
		WithFormField("nickname", "makis").
		Expect().Status(httptest.StatusOK).Body().
		IsEqual(`err path=true csrf=false: schema: invalid path "nickname"`)
}

func TestReadFormBindsCSRFTokenField(t *testing.T) {
	e := httptest.New(t, newReadFormCSRFApp())

	e.POST("/token").
		WithFormField(context.CSRFTokenFormKey, "abc").
		WithFormField("name", "kataras").
		Expect().Status(httptest.StatusOK).Body().IsEqual("ok name=kataras token=abc")
}
