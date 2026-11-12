package handler_test

import (
	"fmt"
	"testing"

	"github.com/kataras/blocks"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/view"
)

type customRenderer struct {
	msg string
}

func (r *customRenderer) Render(ctx iris.Context) error {
	return ctx.HTML(fmt.Sprintf("<h1>Hello, %s!</h1>", r.msg))
}

func TestRenderer(t *testing.T) {
	mfs := blocks.NewMemoryFileSystem()
	err := mfs.ParseTemplate("index.html", []byte(`<h1>Hello, {{.Name}}!</h1>`), nil)
	if err != nil {
		t.Fatal(err)
	}
	viewEngine := blocks.New(mfs)
	if err := viewEngine.Load(); err != nil {
		t.Fatal(err)
	}

	app := iris.New()
	app.UseRouter(view.Handler(viewEngine))

	custRenderer := &customRenderer{msg: "world"}
	h := handler.ResponseHandler(custRenderer)
	app.Get("/", h)

	viewHandler := handler.ResponseHandler(handler.View().Name("index.html").Data(iris.Map{"Name": "world"}))
	app.Get("/view", viewHandler)

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).HasContentType("text/html", "utf-8").Body().IsEqual("<h1>Hello, world!</h1>")
	e.GET("/view").Expect().Status(httptest.StatusOK).HasContentType("text/html", "utf-8").Body().IsEqual("<h1>Hello, world!</h1>")
}
