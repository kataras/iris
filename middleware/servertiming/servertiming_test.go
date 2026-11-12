package servertiming_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/servertiming"
)

func TestServerTimingHandler(t *testing.T) {
	app := iris.New()
	app.Use(servertiming.Handler)

	app.Get("/", func(ctx iris.Context) {
		time.Sleep(100 * time.Millisecond) // Simulate some work
		ctx.WriteString("Hello, World!")
	})

	e := httptest.New(t, app)
	resp := e.GET("/").Expect().Status(iris.StatusOK)
	resp.Body().IsEqual("Hello, World!")
	resp.Header("Server-Timing").NotEmpty()
}

func TestServerTimingStartEnd(t *testing.T) {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		servertiming.Start(ctx, "db", "Database processing")
		time.Sleep(50 * time.Millisecond) // Simulate database work
		servertiming.End(ctx, "db")

		servertiming.Start(ctx, "cache", "Cache processing")
		time.Sleep(30 * time.Millisecond) // Simulate cache work
		servertiming.End(ctx, "cache")

		// t.Logf("final Server-Timing header value: %s", ctx.ResponseHeader("Server-Timing"))
		ctx.WriteString("Hello, World!")
	})

	e := httptest.New(t, app)
	resp := e.GET("/").Expect().Status(iris.StatusOK)
	resp.Body().IsEqual("Hello, World!")
	serverTimingHeader := resp.Raw().Header["Server-Timing"]
	// t.Log(serverTimingHeader)

	if !containsMetric(serverTimingHeader, "db") {
		t.Errorf("Expected Server-Timing header to contain 'db' metric")
	}
	if !containsMetric(serverTimingHeader, "cache") {
		t.Errorf("Expected Server-Timing header to contain 'cache' metric")
	}
}

func containsMetric(headerValues []string, metric string) bool {
	return slices.ContainsFunc(headerValues, func(v string) bool {
		return strings.Contains(v, metric)
	})
}
