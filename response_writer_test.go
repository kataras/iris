package iris_test

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"testing"
)

// most tests lives inside context_test.go:Transactions, there lives the response writer's full and coblex tests
func TestResponseWriterBeforeFlush(t *testing.T) {
	api := iris.New()
	body := "my body"
	beforeFlushBody := "body appeneded or setted before callback"

	api.Get("/", func(ctx *iris.Context) {
		w := ctx.ResponseWriter

		w.SetBeforeFlush(func() {
			w.WriteString(beforeFlushBody)
		})

		w.WriteString(body)
	})

	// recorder can change the status code after write too
	// it can also be changed everywhere inside the context's lifetime
	api.Get("/recorder", func(ctx *iris.Context) {
		w := ctx.Recorder()

		w.SetBeforeFlush(func() {
			w.SetBodyString(beforeFlushBody)
			w.WriteHeader(iris.StatusForbidden)
		})

		w.WriteHeader(iris.StatusOK)
		w.WriteString(body)
	})

	e := httptest.New(api, t)

	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(body + beforeFlushBody)
	e.GET("/recorder").Expect().Status(iris.StatusForbidden).Body().Equal(beforeFlushBody)
}

func TestResponseWriterToRecorderMiddleware(t *testing.T) {
	api := iris.New()
	beforeFlushBody := "body appeneded or setted before callback"
	api.UseGlobal(iris.Recorder)

	api.Get("/", func(ctx *iris.Context) {
		w := ctx.Recorder()

		w.SetBeforeFlush(func() {
			w.SetBodyString(beforeFlushBody)
			w.WriteHeader(iris.StatusForbidden)
		})

		w.WriteHeader(iris.StatusOK)
		w.WriteString("this will not be sent at all because of SetBodyString")
	})

	e := httptest.New(api, t)

	e.GET("/").Expect().Status(iris.StatusForbidden).Body().Equal(beforeFlushBody)
}
