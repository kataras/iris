package iris_test

import (
	"fmt"
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/httptest"
)

// most tests lives inside context_test.go:Transactions, there lives the response writer's full and coblex tests
func TestResponseWriterBeforeFlush(t *testing.T) {
	app := iris.New()
	app.Adapt(newTestNativeRouter())

	body := "my body"
	beforeFlushBody := "body appeneded or setted before callback"

	app.Get("/", func(ctx *iris.Context) {
		w := ctx.ResponseWriter

		w.SetBeforeFlush(func() {
			w.WriteString(beforeFlushBody)
		})

		w.WriteString(body)
	})

	// recorder can change the status code after write too
	// it can also be changed everywhere inside the context's lifetime
	app.Get("/recorder", func(ctx *iris.Context) {
		w := ctx.Recorder()

		w.SetBeforeFlush(func() {
			w.SetBodyString(beforeFlushBody)
			w.WriteHeader(iris.StatusForbidden)
		})

		w.WriteHeader(iris.StatusOK)
		w.WriteString(body)
	})

	e := httptest.New(app, t)

	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(body + beforeFlushBody)
	e.GET("/recorder").Expect().Status(iris.StatusForbidden).Body().Equal(beforeFlushBody)
}

func TestResponseWriterToRecorderMiddleware(t *testing.T) {
	app := iris.New()
	app.Adapt(newTestNativeRouter())

	beforeFlushBody := "body appeneded or setted before callback"
	app.UseGlobal(iris.Recorder)

	app.Get("/", func(ctx *iris.Context) {
		w := ctx.Recorder()

		w.SetBeforeFlush(func() {
			w.SetBodyString(beforeFlushBody)
			w.WriteHeader(iris.StatusForbidden)
		})

		w.WriteHeader(iris.StatusOK)
		w.WriteString("this will not be sent at all because of SetBodyString")
	})

	e := httptest.New(app, t)

	e.GET("/").Expect().Status(iris.StatusForbidden).Body().Equal(beforeFlushBody)
}

func TestResponseRecorderStatusCodeContentTypeBody(t *testing.T) {
	app := iris.New()
	app.Adapt(newTestNativeRouter())

	firstStatusCode := iris.StatusOK
	contentType := "text/html; charset=" + app.Config.Charset
	firstBodyPart := "first"
	secondBodyPart := "second"
	prependedBody := "zero"
	expectedBody := prependedBody + firstBodyPart + secondBodyPart

	app.Use(iris.Recorder)
	// recorder's status code can change if needed by a middleware or the last handler.
	app.UseFunc(func(ctx *iris.Context) {
		ctx.SetStatusCode(firstStatusCode)
		ctx.Next()
	})

	app.UseFunc(func(ctx *iris.Context) {
		ctx.SetContentType(contentType)
		ctx.Next()
	})

	app.UseFunc(func(ctx *iris.Context) {
		// set a body ( we will append it later, only with response recorder we can set append or remove a body or a part of it*)
		ctx.WriteString(firstBodyPart)
		ctx.Next()
	})

	app.UseFunc(func(ctx *iris.Context) {
		ctx.WriteString(secondBodyPart)
		ctx.Next()
	})

	app.Get("/", func(ctx *iris.Context) {
		previousStatusCode := ctx.StatusCode()
		if previousStatusCode != firstStatusCode {
			t.Fatalf("Previous status code should be %d but got %d", firstStatusCode, previousStatusCode)
		}

		previousContentType := ctx.ContentType()
		if previousContentType != contentType {
			t.Fatalf("First content type should be %s but got %d", contentType, previousContentType)
		}
		// change the status code, this will tested later on (httptest)
		ctx.SetStatusCode(iris.StatusForbidden)
		prevBody := string(ctx.Recorder().Body())
		if prevBody != firstBodyPart+secondBodyPart {
			t.Fatalf("Previous body (first handler + second handler's writes) expected to be: %s but got: %s", firstBodyPart+secondBodyPart, prevBody)
		}
		// test it on httptest later on
		ctx.Recorder().SetBodyString(prependedBody + prevBody)
	})

	e := httptest.New(app, t)

	et := e.GET("/").Expect().Status(iris.StatusForbidden)
	et.Header("Content-Type").Equal(contentType)
	et.Body().Equal(expectedBody)
}

func ExampleResponseWriter_WriteHeader() {
	app := iris.New()
	app.Adapt(newTestNativeRouter())

	expectedOutput := "Hey"
	app.Get("/", func(ctx *iris.Context) {

		// here
		for i := 0; i < 10; i++ {
			ctx.ResponseWriter.WriteHeader(iris.StatusOK)
		}

		ctx.Writef(expectedOutput)

		// here
		fmt.Println(expectedOutput)

		// here
		for i := 0; i < 10; i++ {
			ctx.SetStatusCode(iris.StatusOK)
		}
	})

	e := httptest.New(app, nil)
	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(expectedOutput)
	// here it shouldn't log an error that status code write multiple times (by the net/http package.)

	// Output:
	// Hey
}
