package httpcache

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"

	"github.com/redis/go-redis/v9"
)

// A store that is down fails open: the request is served uncached. The store's
// failure must stay out of the response all the same. It used to be stored on
// the Context, where an error handler renders it for the request, so a route
// that answered a bare 403 next sent the store's failure text as its message.
func TestHandlerStoreFailureStaysOffTheResponse(t *testing.T) {
	// Nothing listens on port 1, and the short dial timeout keeps a platform
	// that retries a refused connection from slowing the test down.
	const storeAddr = "127.0.0.1:1"

	c := &Cache{
		Options: &Options{TTL: time.Minute, CommandTimeout: time.Second},
		db: redis.NewClient(&redis.Options{
			Addr:        storeAddr,
			DialTimeout: 300 * time.Millisecond,
			MaxRetries:  -1,
		}),
	}
	t.Cleanup(func() { c.Close() })

	var (
		logs   bytes.Buffer
		public []string // errors an outer handler finds on the Context, readable by a client.
	)

	app := iris.New()
	app.Logger().SetOutput(&logs)

	app.UseRouter(func(ctx iris.Context) {
		ctx.Next()

		if isPublic, err := ctx.ErrPublic(); err != nil && isPublic {
			public = append(public, err.Error())
		}
	})
	app.UseRouter(errors.Handler)

	app.Get("/live", c.Handler(), func(ctx iris.Context) {
		ctx.WriteString("live")
	})
	app.Get("/forbidden", c.Handler(), func(ctx iris.Context) {
		ctx.StopWithStatus(iris.StatusForbidden)
	})

	e := httptest.New(t, app, httptest.LogLevel("error"))

	// Fail open, on the read and on the write that follows a 200.
	e.GET("/live").Expect().Status(httptest.StatusOK).Body().IsEqual("live")

	// A failure the route answers itself reads as that failure.
	res := e.GET("/forbidden").Expect().Status(httptest.StatusForbidden)
	if body := res.Body().Raw(); strings.Contains(body, "127.0.0.1") || strings.Contains(body, "dial") {
		t.Fatalf("the store failure text reached the client: %s", body)
	}
	res.JSON().Object().HasValue("message", "Forbidden")

	if len(public) != 0 {
		t.Fatalf("expected no store error readable by a client on the Context, got %q", public)
	}

	if !strings.Contains(logs.String(), "httpcache: the store failed") {
		t.Fatalf("expected the store failure to be logged, got: %q", logs.String())
	}
}
