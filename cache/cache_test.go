package cache_test

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kataras/iris/v12/cache"
	"github.com/kataras/iris/v12/cache/client"
	"github.com/kataras/iris/v12/cache/client/rule"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"

	"github.com/iris-contrib/httpexpect/v2"
	"github.com/kataras/iris/v12/httptest"
)

var (
	cacheDuration   = 2 * time.Second
	expectedBodyStr = "Imagine it as a big message to achieve x20 response performance!"
)

type testError struct {
	expected int
	got      uint32
}

func (h *testError) Error() string {
	return fmt.Sprintf("expected the main handler to be executed %d times instead of %d", h.expected, h.got)
}

func runTest(e *httpexpect.Expect, path string, counterPtr *uint32, expectedBodyStr string, nocache string) error {
	e.GET(path).Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
	time.Sleep(cacheDuration / 5) // lets wait for a while, cache should be saved and ready
	e.GET(path).Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
	counter := atomic.LoadUint32(counterPtr)
	if counter > 1 {
		// n should be 1 because it doesn't changed after the first call
		return &testError{1, counter}
	}
	time.Sleep(cacheDuration)

	// cache should be cleared now
	e.GET(path).Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
	time.Sleep(cacheDuration / 5)
	// let's call again , the cache should be saved
	e.GET(path).Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
	counter = atomic.LoadUint32(counterPtr)
	if counter != 2 {
		return &testError{2, counter}
	}

	// we have cache response saved for the path, we have some time more here, but here
	// we will make the requestS with some of the deniers options
	e.GET(path).WithHeader("max-age", "0").Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
	e.GET(path).WithHeader("Authorization", "basic or anything").Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
	counter = atomic.LoadUint32(counterPtr)
	if counter != 4 {
		return &testError{4, counter}
	}

	if nocache != "" {
		// test the NoCache, first sleep to pass the cache expiration,
		// second add to the cache with a valid request and response
		// third, do it with the "/nocache" path (static for now, pure test design) given by the consumer
		time.Sleep(cacheDuration)

		// cache should be cleared now, this should work because we are not in the "nocache" path
		e.GET("/").Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr) // counter = 5
		time.Sleep(cacheDuration / 5)

		// let's call the "nocache", the expiration is not passed so but the "nocache"
		// route's path has the cache.NoCache so it should be not cached and the counter should be ++
		e.GET(nocache).Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr) // counter should be 6
		counter = atomic.LoadUint32(counterPtr)
		if counter != 6 { // 4 before, 5 with the first call to store the cache, and six with the no cache, again original handler executation
			return &testError{6, counter}
		}

		// let's call again the path the expiration is not passed so  it should be cached
		e.GET(path).Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
		counter = atomic.LoadUint32(counterPtr)
		if counter != 6 {
			return &testError{6, counter}
		}

		// but now check for the No
	}

	return nil
}

func TestClientNoCache(t *testing.T) {
	app := iris.New()
	var n uint32

	app.Get("/", cache.Handler(cacheDuration), func(ctx *context.Context) {
		atomic.AddUint32(&n, 1)
		ctx.Write([]byte(expectedBodyStr))
	})

	app.Get("/nocache", cache.Handler(cacheDuration), func(ctx *context.Context) {
		client.NoCache(ctx) // <----
		atomic.AddUint32(&n, 1)
		ctx.Write([]byte(expectedBodyStr))
	})

	e := httptest.New(t, app)
	if err := runTest(e, "/", &n, expectedBodyStr, "/nocache"); err != nil {
		t.Fatalf(t.Name()+": %v", err)
	}
}

func TestCache(t *testing.T) {
	app := iris.New()
	var n uint32

	app.Use(cache.Handler(cacheDuration))

	app.Get("/", func(ctx *context.Context) {
		atomic.AddUint32(&n, 1)
		ctx.Write([]byte(expectedBodyStr))
	})

	var (
		n2               uint32
		expectedBodyStr2 = "This is the other"
	)

	app.Get("/other", func(ctx *context.Context) {
		atomic.AddUint32(&n2, 1)
		ctx.Write([]byte(expectedBodyStr2))
	})

	e := httptest.New(t, app)
	if err := runTest(e, "/", &n, expectedBodyStr, ""); err != nil {
		t.Fatalf(t.Name()+": %v", err)
	}

	if err := runTest(e, "/other", &n2, expectedBodyStr2, ""); err != nil {
		t.Fatalf(t.Name()+" other: %v", err)
	}
}

// This works but we have issue on golog.SetLevel and get golog.Level on httptest.New
// when tests are running in parallel and the loggers are used.
// // TODO: Fix it on golog repository or here, we'll see.
// func TestCacheHandlerParallel(t *testing.T) {
// 	t.Parallel()
// 	TestCache(t)
// }

func TestCacheValidator(t *testing.T) {
	app := iris.New()
	var n uint32

	h := func(ctx *context.Context) {
		atomic.AddUint32(&n, 1)
		ctx.Write([]byte(expectedBodyStr))
	}

	validCache := cache.Handler(cacheDuration)
	app.Get("/", validCache, h)

	managedCache := cache.Cache(cache.MaxAge(cacheDuration))
	managedCache.AddRule(rule.Validator([]rule.PreValidator{
		func(ctx *context.Context) bool {
			// should always invalid for cache, don't bother to go to try to get or set cache
			return ctx.Request().URL.Path != "/invalid"
		},
	}, nil))

	managedCache2 := cache.Cache(cache.MaxAge(cacheDuration))
	managedCache2.AddRule(rule.Validator(nil,
		[]rule.PostValidator{
			func(ctx *context.Context) bool {
				// it's passed the Claim and now Valid checks if the response contains a header of "DONT"
				return ctx.ResponseWriter().Header().Get("DONT") == ""
			},
		},
	))

	app.Get("/valid", validCache, h)

	app.Get("/invalid", managedCache.ServeHTTP, h)
	app.Get("/invalid2", managedCache2.ServeHTTP, func(ctx *context.Context) {
		atomic.AddUint32(&n, 1)
		ctx.Header("DONT", "DO not cache that response even if it was claimed")
		ctx.Write([]byte(expectedBodyStr))
	})

	e := httptest.New(t, app)

	// execute from cache the next time
	e.GET("/valid").Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
	time.Sleep(cacheDuration / 5) // lets wait for a while, cache should be saved and ready
	e.GET("/valid").Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)
	counter := atomic.LoadUint32(&n)
	if counter > 1 {
		// n should be 1 because it doesn't changed after the first call
		t.Fatalf("%s: %v", t.Name(), &testError{1, counter})
	}
	// don't execute from cache, execute the original, counter should ++ here
	e.GET("/invalid").Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr)  // counter = 2
	e.GET("/invalid2").Expect().Status(http.StatusOK).Body().IsEqual(expectedBodyStr) // counter = 3

	counter = atomic.LoadUint32(&n)
	if counter != 3 {
		// n should be 1 because it doesn't changed after the first call
		t.Fatalf("%s: %v", t.Name(), &testError{3, counter})
	}
}
