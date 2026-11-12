package context_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// RemoveCookie used to run the cookie options against a cookie whose Name was
// still empty. Every option that consults CookieIncluded bails out on an empty
// name, so CookieAllowSubdomains, CookieAllowReclaim and CookieEncoding were
// silent no-ops on removal.
//
// The visible consequence: a session cookie set with CookieAllowSubdomains
// carries a Domain, but its deletion was emitted WITHOUT one, so the browser
// kept the original cookie and the session survived "logout".
func TestRemoveCookieAppliesOptions(t *testing.T) {
	app := iris.New()

	app.Get("/set", func(ctx iris.Context) {
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "abc123", Path: "/"}, iris.CookieAllowSubdomains())
	})

	app.Get("/logout", func(ctx iris.Context) {
		ctx.RemoveCookie("session", iris.CookieAllowSubdomains())
	})

	e := httptest.New(t, app)

	setHeader := e.GET("/set").WithHost("example.com").Expect().Status(httptest.StatusOK).
		Raw().Header.Get("Set-Cookie")
	if !strings.Contains(setHeader, "Domain=example.com") {
		t.Fatalf("set: expected a Domain attribute, got %q", setHeader)
	}

	delHeader := e.GET("/logout").WithHost("example.com").Expect().Status(httptest.StatusOK).
		Raw().Header.Get("Set-Cookie")

	if !strings.Contains(delHeader, "Domain=example.com") {
		t.Errorf("logout: expected the deletion to carry the same Domain attribute, got %q", delHeader)
	}
	if !strings.Contains(delHeader, "Max-Age=0") && !strings.Contains(delHeader, "Max-Age=-1") {
		t.Errorf("logout: expected the deletion to expire the cookie, got %q", delHeader)
	}
	if !strings.Contains(delHeader, "session=") {
		t.Errorf("logout: expected the cookie name to be present, got %q", delHeader)
	}
}

// CookieAllowReclaim also strips the cookie from the CURRENT request, so a later
// handler in the same chain no longer sees the "removed" cookie.
func TestRemoveCookieReclaimsWithinTheSameRequest(t *testing.T) {
	app := iris.New()

	app.Get("/logout", func(ctx iris.Context) {
		ctx.RemoveCookie("session", iris.CookieAllowReclaim())

		if v := ctx.CookieValue("session"); v != "" {
			ctx.StopWithText(http.StatusInternalServerError,
				"the removed cookie is still readable within the same request: "+v)
			return
		}

		ctx.WriteString("reclaimed")
	})

	e := httptest.New(t, app)
	e.GET("/logout").WithCookie("session", "abc123").
		Expect().Status(httptest.StatusOK).Body().IsEqual("reclaimed")
}
