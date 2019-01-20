package main

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
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

	if filename == "/" {
		filename = "/index.html"
	}

	fullpath := filepath.Join(dir, filename)

	b, err := ioutil.ReadFile(fullpath)
	if err != nil {
		panic(fullpath + " failed with error: " + err.Error())
	}
	result := string(b)
	if runtime.GOOS != "windows" {
		// result = strings.Replace(result, "\n", "\r\n", -1)
	}
	return result
}

var urls = []resource{
	"/",
	"/index.html",
	"/app.js",
	"/css/main.css",
}

func TestSPAEmbedded(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	for _, u := range urls {
		url := u.String()
		contents := u.loadFromBase("./public")
		contents = strings.Replace(contents, "{{ .Page.Title }}", page.Title, 1)

		e.GET(url).Expect().
			Status(httptest.StatusOK).
			ContentType(u.contentType(), app.ConfigurationReadOnly().GetCharset()).
			Body().Equal(contents)
	}
}
