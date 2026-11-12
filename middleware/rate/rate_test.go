package rate_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/rate"
)

// A limiter of 1 req/sec with burst 1 must admit the first request and
// reject an immediate second one with 429 Too Many Requests.
func TestLimitRejectsWhenExceeded(t *testing.T) {
	app := iris.New()
	app.Use(rate.Limit(1, 1, rate.Options{}))
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
	e.GET("/").Expect().Status(httptest.StatusTooManyRequests)
}

// A generous limit admits sequential requests within its budget.
func TestLimitAllowsWithinBudget(t *testing.T) {
	app := iris.New()
	app.Use(rate.Limit(100, 10, rate.Options{}))
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)

	for range 5 {
		e.GET("/").Expect().Status(httptest.StatusOK)
	}
}

// Stop must end the background purge goroutine (it blocks until the
// goroutine exits) and stay safe to call more than once.
func TestNewStopEndsPurger(t *testing.T) {
	limiters := []*rate.Limiter{
		rate.New(1, 1, rate.Options{}), // default purger.
		rate.New(1, 1, rate.Options{PurgeEvery: time.Millisecond, PurgeMaxLifetime: time.Millisecond}),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, l := range limiters {
			l.Stop()
			l.Stop() // idempotent.
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Stop did not end the purge goroutine")
	}
}

// A purged client is forgotten: the next request from the same identifier
// gets a brand new full burst. With a refill rate of 1 token per 100 seconds,
// a fresh 200 shortly after a 429 can only come from the purger.
func TestPurgeEveryStillFires(t *testing.T) {
	l := rate.New(0.01, 1, rate.Options{PurgeEvery: 20 * time.Millisecond, PurgeMaxLifetime: 10 * time.Millisecond})
	defer l.Stop()

	app := iris.New()
	app.Use(l.Handler())
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK)
	e.GET("/").Expect().Status(httptest.StatusTooManyRequests)

	// Every rejected poll refreshes the client's last-seen time, so sleep
	// longer than maxLifetime plus the purge interval between attempts.
	deadline := time.Now().Add(10 * time.Second)
	for {
		time.Sleep(50 * time.Millisecond)

		if e.GET("/").Expect().Raw().StatusCode == httptest.StatusOK {
			break // the client was purged and re-admitted with a full burst.
		}

		if time.Now().After(deadline) {
			t.Fatal("the purger never removed the idle client")
		}
	}
}

// rate.Inf allows everything.
func TestLimitInf(t *testing.T) {
	app := iris.New()
	app.Use(rate.Limit(rate.Inf, 0, rate.Options{}))
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)

	for range 5 {
		e.GET("/").Expect().Status(httptest.StatusOK)
	}
}

// Options.ExceedHandler replaces the default 429 and Options.ClientData fills
// the Client's Data field, both read through the Options struct rather than a
// functional option.
func TestOptionsExceedHandlerAndClientData(t *testing.T) {
	app := iris.New()
	app.Use(rate.Limit(1, 1, rate.Options{
		ExceedHandler: func(ctx iris.Context) {
			ctx.StopWithText(iris.StatusServiceUnavailable, "slow down")
		},
		ClientData: func(ctx iris.Context) any {
			return "data-" + ctx.Header("X-Client")
		},
	}))
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString(rate.Get(ctx).Data.(string))
	})

	e := httptest.New(t, app)

	e.GET("/").WithHeader("X-Client", "alice").Expect().
		Status(httptest.StatusOK).Body().IsEqual("data-alice")
	e.GET("/").WithHeader("X-Client", "alice").Expect().
		Status(httptest.StatusServiceUnavailable).Body().IsEqual("slow down")
}

// PurgeEvery and PurgeMaxLifetime are independent. Setting one alone used to
// discard it silently, because the custom purger was only installed when BOTH
// were set: rate.Options{PurgeEvery: time.Minute} produced the stock purger,
// with the stock interval, and nothing said so. Each half now falls back on its
// own.
func TestPurgeOptionsHalvesAreIndependent(t *testing.T) {
	// Both unset: the stock interval and the rate-derived lifetime.
	// With limit 1 and burst 1 the derived lifetime is clamped up to the
	// one-minute floor, so a client idle for a second survives and one idle for
	// two hours does not.
	stock := rate.New(1, 1, rate.Options{})
	defer stock.Stop()

	if got := stock.PurgeInterval(); got != rate.DefaultPurgeInterval {
		t.Errorf("zero Options: interval = %v, want %v", got, rate.DefaultPurgeInterval)
	}
	if stock.ShouldPurgeAfter(time.Second) {
		t.Error("zero Options: a client idle for a second was purged")
	}
	if !stock.ShouldPurgeAfter(2 * time.Hour) {
		t.Error("zero Options: a client idle for two hours was not purged")
	}

	// PurgeEvery alone: the interval is honored, the lifetime stays derived.
	everyOnly := rate.New(1, 1, rate.Options{PurgeEvery: 3 * time.Second})
	defer everyOnly.Stop()

	if got := everyOnly.PurgeInterval(); got != 3*time.Second {
		t.Errorf("PurgeEvery alone: interval = %v, want 3s", got)
	}
	if everyOnly.ShouldPurgeAfter(time.Second) {
		t.Error("PurgeEvery alone: a client idle for a second was purged; the derived lifetime was lost")
	}
	if !everyOnly.ShouldPurgeAfter(2 * time.Hour) {
		t.Error("PurgeEvery alone: a client idle for two hours was not purged")
	}

	// PurgeMaxLifetime alone: the lifetime is honored, the interval stays stock.
	lifetimeOnly := rate.New(1, 1, rate.Options{PurgeMaxLifetime: 10 * time.Millisecond})
	defer lifetimeOnly.Stop()

	if got := lifetimeOnly.PurgeInterval(); got != rate.DefaultPurgeInterval {
		t.Errorf("PurgeMaxLifetime alone: interval = %v, want the default %v", got, rate.DefaultPurgeInterval)
	}
	if !lifetimeOnly.ShouldPurgeAfter(50 * time.Millisecond) {
		t.Error("PurgeMaxLifetime alone: a client idle past the configured lifetime was not purged")
	}
	if lifetimeOnly.ShouldPurgeAfter(time.Millisecond) {
		t.Error("PurgeMaxLifetime alone: a client idle for less than the configured lifetime was purged")
	}
}

// The same, for the rate.Inf edge: defaultMaxLifetime cannot derive a lifetime
// from an infinite rate, so it falls back to the one-minute floor rather than to
// zero, which would evict every client on the first tick.
func TestPurgeMaxLifetimeFallsBackToTheFloorForInf(t *testing.T) {
	l := rate.New(rate.Inf, 0, rate.Options{PurgeEvery: 10 * time.Millisecond})
	defer l.Stop()

	if got := l.PurgeInterval(); got != 10*time.Millisecond {
		t.Fatalf("interval = %v, want 10ms", got)
	}

	// rate.Inf makes defaultMaxLifetime fall back to the one-minute floor, so
	// the tick alone must not evict a fresh client.
	if l.ShouldPurgeAfter(rate.MinPurgeLifetime / 2) {
		t.Error("a client idle for less than the floor lifetime was purged")
	}
}
