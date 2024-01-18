package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

type resource string

// content types that are used in the ./assets,
// we could use the detectContentType that iris do but it's better
// to do it manually so we can test if that returns the correct result on embedding files.
func (r resource) contentType() string {
	switch filepath.Ext(r.String()) {
	case ".js":
		return "text/javascript"
	case ".css":
		return "text/css"
	case ".ico":
		return "image/x-icon"
	case ".html":
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
	filename := r.strip("/static")

	fullpath := filepath.Join(dir, filename)

	b, err := os.ReadFile(fullpath)
	if err != nil {
		panic(fullpath + " failed with error: " + err.Error())
	}

	result := string(b)
	if runtime.GOOS != "windows" {
		result = strings.ReplaceAll(result, "\n", "\r\n")
		result = strings.ReplaceAll(result, "\r\r", "")
	}

	return result
}

var urls = []resource{
	"/static/css/main.css",
	"/static/js/main.js",
	"/static/favicon.ico",
}

// if bindata's values matches with the assets/... contents
// and secondly if the HandleDir had successfully registered
// the routes and gave the correct response.
func TestEmbeddingFilesIntoApp(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	route := app.GetRouteReadOnly("GET/static/{file:path}")
	if route == nil {
		t.Fatalf("expected a route to serve embedded files")
	}

	if runtime.GOOS != "windows" {
		// remove the embedded static favicon for !windows,
		// it should be built for unix-specific in order to be work
		urls = urls[0 : len(urls)-1]
	}

	for _, u := range urls {
		url := u.String()
		contents := u.loadFromBase("./assets")

		e.GET(url).Expect().
			Status(httptest.StatusOK).
			ContentType(u.contentType()).
			Body().IsEqual(contents)
	}
}
