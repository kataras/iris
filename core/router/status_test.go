// black-box testing
package router_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"

	"github.com/kataras/iris/v12/httptest"
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
	t.Helper()

	if got, expected := buff.String(), expected; got != expected {
		t.Fatalf("expected middleware to run before the error handler, expected: '%s' but got: '%s'", expected, got)
	}

	buff.Reset()
}

func TestPartyOnErrorCode(t *testing.T) {
	app := iris.New()
	app.Configure(iris.WithFireMethodNotAllowed)

	globalNotFoundResponse := "custom not found"
	app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.WriteString(globalNotFoundResponse)
	})

	globalMethodNotAllowedResponse := "global: method not allowed"
	app.OnErrorCode(iris.StatusMethodNotAllowed, func(ctx iris.Context) {
		ctx.WriteString(globalMethodNotAllowedResponse)
	})

	app.Get("/path", h)

	h := func(ctx iris.Context) { ctx.WriteString(ctx.Path()) }
	usersResponse := "users: method allowed"

	users := app.Party("/users")
	users.OnErrorCode(iris.StatusMethodNotAllowed, func(ctx iris.Context) {
		ctx.WriteString(usersResponse)
	})
	users.Get("/", h)
	// test setting the error code from a handler.
	users.Get("/badrequest", func(ctx iris.Context) { ctx.StatusCode(iris.StatusBadRequest) })

	usersuserResponse := "users:user: method allowed"
	user := users.Party("/{id:int}")
	user.OnErrorCode(iris.StatusMethodNotAllowed, func(ctx iris.Context) {
		ctx.WriteString(usersuserResponse)
	})
	// usersuserNotFoundResponse := "users:user: not found"
	// user.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
	// 	ctx.WriteString(usersuserNotFoundResponse)
	// })
	user.Get("/", h)

	e := httptest.New(t, app)

	e.GET("/notfound").Expect().Status(iris.StatusNotFound).Body().Equal(globalNotFoundResponse)
	e.POST("/path").Expect().Status(iris.StatusMethodNotAllowed).Body().Equal(globalMethodNotAllowedResponse)
	e.GET("/path").Expect().Status(iris.StatusOK).Body().Equal("/path")

	e.POST("/users").Expect().Status(iris.StatusMethodNotAllowed).
		Body().Equal(usersResponse)

	e.POST("/users/42").Expect().Status(iris.StatusMethodNotAllowed).
		Body().Equal(usersuserResponse)

	e.GET("/users/42").Expect().Status(iris.StatusOK).
		Body().Equal("/users/42")
	// e.GET("/users/ab").Expect().Status(iris.StatusNotFound).Body().Equal(usersuserNotFoundResponse)

	// if not registered to the party, then the root is taking action.
	e.GET("/users/ab/cd").Expect().Status(iris.StatusNotFound).Body().Equal(globalNotFoundResponse)

	// if not registered to the party, and not in root, then just write the status text (fallback behavior)
	e.GET("/users/badrequest").Expect().Status(iris.StatusBadRequest).
		Body().Equal(http.StatusText(iris.StatusBadRequest))
}
