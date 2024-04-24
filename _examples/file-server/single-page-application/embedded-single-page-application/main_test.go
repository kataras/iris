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

func (r resource) contentType() string {
	switch filepath.Ext(r.String()) {
	case ".js":
		return "text/javascript"
	case ".css":
		return "text/css"
	default:
		return "text/html"
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

	if strings.HasSuffix(filename, "/") {
		filename = filename + "index.html"
	}

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
	"/",
	"/app.js",
	"/css/main.css",
	"/app2/",
	"/app2/index.html",
}

func TestSPAEmbedded(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	for _, u := range urls {
		url := u.String()
		base := "./data/public"
		if u == "/" || u == "/index.html" {
			base = "./data/views"
		}
		contents := u.loadFromBase(base)
		contents = strings.Replace(contents, "{{ .Page.Title }}", page.Title, 1)

		e.GET(url).Expect().
			Status(httptest.StatusOK).
			ContentType(u.contentType(), app.ConfigurationReadOnly().GetCharset()).
			Body().IsEqual(contents)
	}

	e.GET("/index.html").Expect().Status(httptest.StatusNotFound) // only root is served.
}
