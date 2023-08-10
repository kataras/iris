package main

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

func TestSitemap(t *testing.T) {
	const expectedFullSitemapXML = `<?xml version="1.0" encoding="utf-8" standalone="yes"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>http://localhost:8080/home</loc><lastmod>2019-12-13T21:50:33+02:00</lastmod><changefreq>hourly</changefreq><priority>1</priority></url><url><loc>http://localhost:8080/articles</loc><changefreq>daily</changefreq></url><url><loc>http://localhost:8080/path1</loc></url><url><loc>http://localhost:8080/path2</loc></url></urlset>`

	app := newApp()
	app.Configure(iris.WithSitemap(startURL))

	e := httptest.New(t, app)
	e.GET("/sitemap.xml").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedFullSitemapXML)
}
