package rewrite_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/rewrite"

	"github.com/iris-contrib/httpexpect/v2"
)

// The documented rule `301 /seo/(.*) /$1` looks like it can only ever rewrite to
// a local path. But GET /seo//evil.com produces the target "//evil.com", which
// browsers resolve as https://evil.com - an open redirect.
func TestRewriteRelativeRuleDoesNotOpenRedirect(t *testing.T) {
	engine, err := rewrite.New(rewrite.Options{
		RedirectMatch: []string{"301 /seo/(.*) /$1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	app := iris.New()
	app.UseRouter(engine.Handler)
	app.Get("/{p:path}", func(ctx iris.Context) { ctx.WriteString("local") })
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("root") })

	e := httptest.New(t, app)

	// The ordinary case still redirects.
	e.GET("/seo/about").WithRedirectPolicy(httpexpect.DontFollowRedirects).
		Expect().Status(httptest.StatusMovedPermanently).
		Header("Location").IsEqual("/about")

	// The protocol-relative target must not become a Location.
	resp := e.GET("/seo//evil.com").WithRedirectPolicy(httpexpect.DontFollowRedirects).Expect()
	if loc := resp.Raw().Header.Get("Location"); loc == "//evil.com" {
		t.Fatalf("open redirect: Location = %q sends the client to evil.com", loc)
	}
}
