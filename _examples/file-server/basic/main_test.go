package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kataras/iris/httptest"
)

type resource string

func (r resource) contentType() string {
	switch filepath.Ext(r.String()) {
	case ".js":
		return "application/javascript"
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

func (r resource) loadFromBase(dir string) string {
	filename := r.String()

	filename = r.strip("/static")
	if filepath.Ext(filename) == "" {
		// root /.
		filename = filename + "/index.html"
	}

	fullpath := filepath.Join(dir, filename)

	b, err := ioutil.ReadFile(fullpath)
	if err != nil {
		panic(fullpath + " failed with error: " + err.Error())
	}

	result := string(b)

	return result
}

func TestFileServerBasic(t *testing.T) {
	urls := []resource{
		"/static/css/main.css",
		"/static/js/jquery-2.1.1.js",
		"/static/favicon.ico",
		"/static/app2",
		"/static/app2/app2app3",
		"/static",
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
		contents := u.loadFromBase("./assets")

		e.GET(url).Expect().
			Status(httptest.StatusOK).
			ContentType(u.contentType(), app.ConfigurationReadOnly().GetCharset()).
			Body().Equal(contents)
	}
}
