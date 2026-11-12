package httpcache_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
)

// Must is New's panicking companion. With Disable set no store is contacted, so
// this exercises the constructor without a live DragonflyDB/Redis: it returns a
// usable Cache whose Handler is nil, which the framework skips.
func TestMustDisabled(t *testing.T) {
	cache := httpcache.Must(httpcache.Options{Disable: true})

	if cache == nil {
		t.Fatal("Must returned a nil Cache")
	}

	if handler := cache.Handler(); handler != nil {
		t.Fatal("a disabled Cache returned a non-nil Handler")
	}

	app := iris.New()
	app.Get("/", cache.Handler(), func(ctx iris.Context) {
		ctx.WriteString("live")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("live")
}

// New reports a missing address instead of panicking; Must turns that same
// error into a panic.
func TestNewAndMustOnBadOptions(t *testing.T) {
	if _, err := httpcache.New(httpcache.Options{}); err == nil {
		t.Fatal("New accepted Options with no Addr")
	}

	defer func() {
		if recover() == nil {
			t.Fatal("Must did not panic on Options with no Addr")
		}
	}()

	httpcache.Must(httpcache.Options{})
}
