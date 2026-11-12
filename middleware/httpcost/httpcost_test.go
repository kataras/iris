package httpcost_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/httpcost"
)

// New registers a handler that measures the cost of everything after it and
// stores the resulting *Metric on the Context, where Get finds it.
func TestNewStoresMetric(t *testing.T) {
	var (
		found  bool
		metric *httpcost.Metric
	)

	app := iris.New()
	// The metric is stored after the rest of the chain returned, so read it on
	// the way back out of a handler registered before the middleware.
	app.Use(func(ctx iris.Context) {
		ctx.Next()
		metric, found = httpcost.Get(ctx)
	})
	app.Use(httpcost.New(httpcost.DefaultOptions))
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("payload")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("payload")

	if !found {
		t.Fatal("httpcost.Get: no metric on the context")
	}

	// Duration is not asserted: a handler this cheap can finish inside the
	// platform's clock granularity and legitimately measure 0.
	if metric.Duration < 0 {
		t.Errorf("metric.Duration = %v, want >= 0", metric.Duration)
	}

	if metric.NetworkUsed != uint64(len("payload")) {
		t.Errorf("metric.NetworkUsed = %d, want %d", metric.NetworkUsed, len("payload"))
	}

	if time.Time(metric.StartTime).IsZero() {
		t.Error("metric.StartTime is zero, want the time the measurement began")
	}
}

// Without the middleware there is nothing on the Context.
func TestGetWithoutMiddleware(t *testing.T) {
	var found bool

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		_, found = httpcost.Get(ctx)
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK)

	if found {
		t.Fatal("httpcost.Get returned a metric without the middleware registered")
	}
}
