package rewrite_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rewrite"
)

// Must is New's panicking companion: on valid options it returns the Engine and
// the registered handler performs the redirect.
func TestMustRedirects(t *testing.T) {
	engine := rewrite.Must(rewrite.Options{
		RedirectMatch: []string{"301 /seo/(.*) /$1"},
	})

	app := iris.New()
	app.UseRouter(engine.Handler)
	app.Get("/about", func(ctx iris.Context) {
		ctx.WriteString("about")
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://example.com/seo/about", nil))

	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMovedPermanently)
	}

	if got, want := w.Header().Get("Location"), "/about"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

// A rule New would report as an error makes Must panic.
func TestMustPanicsOnBadRule(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Must did not panic on an unparsable redirect rule")
		}
	}()

	rewrite.Must(rewrite.Options{RedirectMatch: []string{"this is not a rule"}})
}
