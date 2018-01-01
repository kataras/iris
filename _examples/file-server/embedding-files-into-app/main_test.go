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

	fullpath := filepath.Join(dir, filename)

	b, err := ioutil.ReadFile(fullpath)
	if err != nil {
		panic(fullpath + " failed with error: " + err.Error())
	}

	result := string(b)
	if runtime.GOOS != "windows" {
		result = strings.Replace(result, "\n", "\r\n", -1)
	}
	return result
}

var urls = []resource{
	"/static/css/bootstrap.min.css",
	"/static/js/jquery-2.1.1.js",
	"/static/favicon.ico",
}

// if bindata's values matches with the assets/... contents
// and secondly if the StaticEmbedded had successfully registered
// the routes and gave the correct response.
func TestEmbeddingFilesIntoApp(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

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
			Body().Equal(contents)
	}
}
