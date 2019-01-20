// black-box testing
package router_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"

	"github.com/kataras/iris/httptest"
)

var defaultErrHandler = func(ctx context.Context) {
	text := http.StatusText(ctx.GetStatusCode())
	ctx.WriteString(text)
}

func TestOnAnyErrorCode(t *testing.T) {
	app := iris.New()
	app.Configure(iris.WithFireMethodNotAllowed)

	buff := &bytes.Buffer{}
	expectedPrintBeforeExecuteErr := "printed before error"

	// with a middleware
	app.OnAnyErrorCode(func(ctx context.Context) {
		buff.WriteString(expectedPrintBeforeExecuteErr)
		ctx.Next()
	}, defaultErrHandler)

	expectedFoundResponse := "found"
	app.Get("/found", func(ctx context.Context) {
		ctx.WriteString(expectedFoundResponse)
	})

	app.Get("/406", func(ctx context.Context) {
		ctx.Record()
		ctx.WriteString("this should not be sent, only status text will be sent")
		ctx.WriteString("the handler can handle 'rollback' of the text when error code fired because of the recorder")
		ctx.StatusCode(iris.StatusNotAcceptable)
	})

	e := httptest.New(t, app)

	e.GET("/found").Expect().Status(iris.StatusOK).
		Body().Equal(expectedFoundResponse)

	e.GET("/notfound").Expect().Status(iris.StatusNotFound).
		Body().Equal(http.StatusText(iris.StatusNotFound))

	checkAndClearBuf(t, buff, expectedPrintBeforeExecuteErr)

	e.POST("/found").Expect().Status(iris.StatusMethodNotAllowed).
		Body().Equal(http.StatusText(iris.StatusMethodNotAllowed))

	checkAndClearBuf(t, buff, expectedPrintBeforeExecuteErr)

	e.GET("/406").Expect().Status(iris.StatusNotAcceptable).
		Body().Equal(http.StatusText(iris.StatusNotAcceptable))

	checkAndClearBuf(t, buff, expectedPrintBeforeExecuteErr)

}

func checkAndClearBuf(t *testing.T, buff *bytes.Buffer, expected string) {
	if got, expected := buff.String(), expected; got != expected {
		t.Fatalf("expected middleware to run before the error handler, expected %s but got %s", expected, got)
	}

	buff.Reset()
}
