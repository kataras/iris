package sessions_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/sessions"
)

// TestNewOptionsSecureDefaults locks the v14 secure-by-default cookie settings.
// Apps serving over plain HTTP (e.g. local development) must opt out by setting
// Secure to false; see MIGRATION.md.
func TestNewOptionsSecureDefaults(t *testing.T) {
	opts := sessions.NewOptions()

	if !opts.Secure {
		t.Error("NewOptions().Secure = false, want true (secure-by-default)")
	}
	if opts.SameSite != http.SameSiteLaxMode {
		t.Errorf("NewOptions().SameSite = %v, want http.SameSiteLaxMode", opts.SameSite)
	}
}

// NewOptions returns a value and the database constructors take one by value:
// changing the caller's copy after the database was built must not reach it,
// and the session cookie must keep the settings that were passed in.
func TestOptionsAreCopiedByValue(t *testing.T) {
	opts := sessions.NewOptions([]byte("0123456789abcdef0123456789abcdef"))
	opts.MaxAge = time.Hour
	opts.Path = "/app"
	opts.Secure = false // so the test client keeps the cookie over plain HTTP.

	db := sessions.NewCookieDatabase[map[string]string](opts)

	// Mutating the local copy afterwards must not affect the database.
	opts.MaxAge = time.Nanosecond
	opts.Path = "/mutated"

	app := iris.New()
	app.Use(sessions.Handler(db, "mysession"))
	app.Get("/", func(ctx iris.Context) {
		sess := sessions.Get[map[string]string](ctx)
		if sess.Data == nil {
			sess.Data = make(map[string]string)
		}
		sess.Data["k"] = "v"
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)

	cookie := e.GET("/").Expect().Status(httptest.StatusOK).Cookie("mysession")
	cookie.Path().IsEqual("/app")
	cookie.MaxAge().IsEqual(time.Hour)
}
