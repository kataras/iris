package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kataras/iris/httptest"
)

type resource string

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

	return result
}

var urls = []resource{
	"/",
	"/index.html",
	"/app.js",
	"/css/main.css",
}

func TestSPA(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.Debug(false))

	for _, u := range urls {
		url := u.String()
		contents := u.loadFromBase("./public")
		contents = strings.Replace(contents, "{{ .Page.Title }}", page.Title, 1)

		e.GET(url).Expect().
			Status(httptest.StatusOK).
			Body().Equal(contents)
	}
}
