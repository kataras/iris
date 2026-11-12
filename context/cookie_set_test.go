package context_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/iris-contrib/httpexpect/v2"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

// setCookieLines sends GET path and returns every Set-Cookie line of the response.
func setCookieLines(t *testing.T, e *httpexpect.Expect, path string) []string {
	t.Helper()
	return e.GET(path).Expect().Status(httptest.StatusOK).Raw().Header.Values("Set-Cookie")
}

func linesWithName(lines []string, name string) []string {
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(line, name+"=") {
			out = append(out, line)
		}
	}
	return out
}

// RFC 6265 section 4.1.1: a server should not send two Set-Cookie lines with
// the same cookie name, path and domain in one response. SetCookie replaces,
// as SetHeader does.
func TestSetCookieReplacesSameNamePathDomain(t *testing.T) {
	app := iris.New()

	app.Get("/set-set", func(ctx iris.Context) {
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "first", Path: "/", Domain: "example.com"})
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "second", Path: "/", Domain: ".Example.com"})
	})

	app.Get("/add-add", func(ctx iris.Context) {
		ctx.AddCookie("session", "first")
		ctx.AddCookie("session", "second")
	})

	app.Get("/set-remove", func(ctx iris.Context) {
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "first", Path: "/", Domain: "example.com"})
		ctx.RemoveCookie("session", iris.CookieDomain("example.com"), iris.CookiePath("/"))
	})

	e := httptest.New(t, app)

	for _, path := range []string{"/set-set", "/add-add"} {
		got := linesWithName(setCookieLines(t, e, path), "session")
		if len(got) != 1 {
			t.Fatalf("%s: expected exactly one Set-Cookie line for session, got %d: %q", path, len(got), got)
		}
		if !strings.HasPrefix(got[0], "session=second") {
			t.Fatalf("%s: expected the second value to win, got %q", path, got[0])
		}
	}

	got := linesWithName(setCookieLines(t, e, "/set-remove"), "session")
	if len(got) != 1 {
		t.Fatalf("/set-remove: expected exactly one Set-Cookie line for session, got %d: %q", len(got), got)
	}
	if !strings.HasPrefix(got[0], "session=;") || !strings.Contains(got[0], "Max-Age=0") {
		t.Fatalf("/set-remove: expected the line to be the removal, got %q", got[0])
	}
}

func TestSetCookieKeepsOtherDomain(t *testing.T) {
	app := iris.New()

	app.Get("/domains", func(ctx iris.Context) {
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "a", Path: "/", Domain: "a.example.com"})
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "b", Path: "/", Domain: "b.example.com"})
	})

	app.Get("/paths", func(ctx iris.Context) {
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "a", Path: "/a"})
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "b", Path: "/b"})
	})

	app.Get("/empty-path", func(ctx iris.Context) {
		// An empty path and "/" are different keys to the browser.
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "a"})
		ctx.SetCookie(&http.Cookie{Name: "session", Value: "b", Path: "/"})
	})

	e := httptest.New(t, app)

	for _, path := range []string{"/domains", "/paths", "/empty-path"} {
		got := linesWithName(setCookieLines(t, e, path), "session")
		if len(got) != 2 {
			t.Fatalf("%s: expected two Set-Cookie lines for session, got %d: %q", path, len(got), got)
		}
		if !strings.HasPrefix(got[0], "session=a") || !strings.HasPrefix(got[1], "session=b") {
			t.Fatalf("%s: expected a then b, got %q", path, got)
		}
	}
}

func TestSetCookieLeavesOtherNamesInOrder(t *testing.T) {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.SetCookie(&http.Cookie{Name: "a", Value: "1", Path: "/"})
		ctx.SetCookie(&http.Cookie{Name: "b", Value: "1", Path: "/"})
		ctx.SetCookie(&http.Cookie{Name: "a", Value: "2", Path: "/"})
		ctx.SetCookie(&http.Cookie{Name: "c", Value: "1", Path: "/"})
	})

	e := httptest.New(t, app)

	got := setCookieLines(t, e, "/")
	want := []string{"a=2; Path=/", "b=1; Path=/", "c=1; Path=/"}
	if len(got) != len(want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: expected %q, got %q (all: %q)", i, want[i], got[i], got)
		}
	}
}
