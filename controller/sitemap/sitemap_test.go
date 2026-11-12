package sitemap_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/sitemap"
	"github.com/kataras/iris/v14/httptest"
)

func TestSitemap(t *testing.T) {
	app := iris.New()
	app.Router("/", sitemap.New("http://localhost")) // order doesn't matter, because this specific router registered after route registrations.

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Index")
	})

	app.Get("/about", func(ctx iris.Context) {
		ctx.WriteString("About")
	})

	app.Get("/signup", func(ctx iris.Context) {
		ctx.WriteString("Signup Form")
	})
	app.Post("/signup", func(ctx iris.Context) {
		ctx.WriteString("Signup")
	})

	expectedSitemapContents := `<?xml version="1.0" encoding="utf-8" standalone="yes"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>http://localhost/</loc></url><url><loc>http://localhost/about</loc></url><url><loc>http://localhost/signup</loc></url></urlset>`

	e := httptest.New(t, app)
	e.GET("/sitemap.xml").Expect().Status(httptest.StatusOK).HasContentType("text/xml", "utf-8").
		Body().IsEqual(expectedSitemapContents)
}
