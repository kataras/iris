package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

type resource string

func (r resource) contentType() string {
	switch filepath.Ext(r.String()) {
	case ".js":
		return "text/javascript"
	case ".css":
		return "text/css"
	case ".ico":
		return "image/x-icon"
	case ".html", "":
		return "text/html"
	default:
		return "text/plain"
	}
}

func (r resource) String() string {
	return string(r)
}

func (r resource) strip(strip string) string {
	s := r.String()
	return strings.TrimPrefix(s, strip)
}

func (r resource) loadFromBase(dir string, strip string) string {
	filename := r.String()

	filename = r.strip(strip)
	if filepath.Ext(filename) == "" {
		// root /.
		filename = filename + "/index.html"
	}

	fullpath := filepath.Join(dir, filename)

	b, err := os.ReadFile(fullpath)
	if err != nil {
		panic(fullpath + " failed with error: " + err.Error())
	}

	result := string(b)

	return result
}

func TestFileServerBasic(t *testing.T) {
	urls := []resource{
		"/v1/static/css/main.css",
		"/v1/static/js/main.js",
		"/v1/static/favicon.ico",
		"/v1/static/app2",
		"/v1/static/app2/app2app3",
		"/v1/static",
	}

	app := newApp()
	// route := app.GetRouteReadOnly("GET/{file:path}")
	// if route == nil {
	// 	app.Logger().Fatalf("expected a route to serve files")
	// }

	// if expected, got := "./assets", route.StaticDir(); expected != got {
	// 	app.Logger().Fatalf("expected route's static directory to be: '%s' but got: '%s'", expected, got)
	// }

	// if !route.StaticDirContainsIndex() {
	// 	app.Logger().Fatalf("epxected ./assets to contain an %s file", "/index.html")
	// }

	e := httptest.New(t, app)
	for _, u := range urls {
		url := u.String()
		contents := u.loadFromBase("./assets", "/v1/static")

		e.GET(url).Expect().
			Status(httptest.StatusOK).
			ContentType(u.contentType(), app.ConfigurationReadOnly().GetCharset()).
			Body().IsEqual(contents)
	}
}

// Tests subdomain + request path and system directory with a name that contains a dot(.)
func TestHandleDirDot(t *testing.T) {
	urls := []resource{
		"/v1/assets.system/css/main.css",
	}
	app := newApp()
	app.Subdomain("test").Party("/v1").HandleDir("/assets.system", iris.Dir("./assets.system"))

	e := httptest.New(t, app, httptest.URL("http://test.example.com"))
	for _, u := range urls {
		url := u.String()
		contents := u.loadFromBase("./assets.system", "/v1/assets.system")

		e.GET(url).Expect().
			Status(httptest.StatusOK).
			ContentType(u.contentType(), app.ConfigurationReadOnly().GetCharset()).
			Body().IsEqual(contents)
	}
}
