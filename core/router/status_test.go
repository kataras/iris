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

var defaultErrHandler = func(ctx *context.Context) {
	text := http.StatusText(ctx.GetStatusCode())
	ctx.WriteString(text)
}

func TestOnAnyErrorCode(t *testing.T) {
	app := iris.New()
	app.Configure(iris.WithFireMethodNotAllowed)

	buff := &bytes.Buffer{}
	expectedPrintBeforeExecuteErr := "printed before error"

	// with a middleware
	app.OnAnyErrorCode(func(ctx *context.Context) {
		buff.WriteString(expectedPrintBeforeExecuteErr)
		ctx.Next()
	}, defaultErrHandler)

	expectedFoundResponse := "found"
	app.Get("/found", func(ctx *context.Context) {
		ctx.WriteString(expectedFoundResponse)
	})

	expected407 := "this should be sent, we manage the response response by ourselves"
	app.Get("/407", func(ctx *context.Context) {
		ctx.Record()
		ctx.WriteString(expected407)
		ctx.StatusCode(iris.StatusProxyAuthRequired)
	})

	e := httptest.New(t, app)

	e.GET("/found").Expect().Status(iris.StatusOK).
		Body().IsEqual(expectedFoundResponse)

	e.GET("/notfound").Expect().Status(iris.StatusNotFound).
		Body().IsEqual(http.StatusText(iris.StatusNotFound))

	checkAndClearBuf(t, buff, expectedPrintBeforeExecuteErr)

	e.POST("/found").Expect().Status(iris.StatusMethodNotAllowed).
		Body().IsEqual(http.StatusText(iris.StatusMethodNotAllowed))

	checkAndClearBuf(t, buff, expectedPrintBeforeExecuteErr)

	e.GET("/407").Expect().Status(iris.StatusProxyAuthRequired).
		Body().IsEqual(expected407)

	// Test Configuration.ResetOnFireErrorCode.
	app2 := iris.New()
	app2.Configure(iris.WithResetOnFireErrorCode)

	app2.OnAnyErrorCode(func(ctx *context.Context) {
		buff.WriteString(expectedPrintBeforeExecuteErr)
		ctx.Next()
	}, defaultErrHandler)

	app2.Get("/406", func(ctx *context.Context) {
		ctx.Record()
		ctx.WriteString("this should not be sent, only status text will be sent")
		ctx.WriteString("the handler can handle 'rollback' of the text when error code fired because of the recorder")
		ctx.StatusCode(iris.StatusNotAcceptable)
	})

	httptest.New(t, app2).GET("/406").Expect().Status(iris.StatusNotAcceptable).
		Body().IsEqual(http.StatusText(iris.StatusNotAcceptable))

	checkAndClearBuf(t, buff, expectedPrintBeforeExecuteErr)
}

func checkAndClearBuf(t *testing.T, buff *bytes.Buffer, expected string) {
	t.Helper()

	if got := buff.String(); got != expected {
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

	usersResponse := "users: method allowed"
	users := app.Party("/users")
	users.OnErrorCode(iris.StatusMethodNotAllowed, func(ctx iris.Context) {
		ctx.WriteString(usersResponse)
	})
	users.Get("/", h)
	write400 := func(ctx iris.Context) { ctx.StatusCode(iris.StatusBadRequest) }
	// test setting the error code from a handler.
	users.Get("/badrequest", write400)

	usersuserResponse := "users:user: method allowed"
	user := users.Party("/{id:int}")
	user.OnErrorCode(iris.StatusMethodNotAllowed, func(ctx iris.Context) {
		ctx.WriteString(usersuserResponse)
	})
	usersuserNotFoundResponse := "users:user: not found"
	user.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.WriteString(usersuserNotFoundResponse)
	})
	user.Get("/", h)
	user.Get("/ab/badrequest", write400)

	friends := users.Party("/friends")
	friends.Get("/{id:int}", h)

	e := httptest.New(t, app)

	e.GET("/notfound").Expect().Status(iris.StatusNotFound).Body().IsEqual(globalNotFoundResponse)
	e.POST("/path").Expect().Status(iris.StatusMethodNotAllowed).Body().IsEqual(globalMethodNotAllowedResponse)
	e.GET("/path").Expect().Status(iris.StatusOK).Body().IsEqual("/path")

	e.POST("/users").Expect().Status(iris.StatusMethodNotAllowed).
		Body().IsEqual(usersResponse)

	e.POST("/users/42").Expect().Status(iris.StatusMethodNotAllowed).
		Body().IsEqual(usersuserResponse)

	e.GET("/users/42").Expect().Status(iris.StatusOK).
		Body().IsEqual("/users/42")
	e.GET("/users/ab").Expect().Status(iris.StatusNotFound).Body().IsEqual(usersuserNotFoundResponse)
	// inherit the parent.
	e.GET("/users/42/friends/dsa").Expect().Status(iris.StatusNotFound).Body().IsEqual(usersuserNotFoundResponse)

	// if not registered to the party, then the root is taking action.
	e.GET("/users/42/ab/badrequest").Expect().Status(iris.StatusBadRequest).Body().IsEqual(http.StatusText(iris.StatusBadRequest))

	// if not registered to the party, and not in root, then just write the status text (fallback behavior)
	e.GET("/users/badrequest").Expect().Status(iris.StatusBadRequest).
		Body().IsEqual(http.StatusText(iris.StatusBadRequest))
}
