package main

import (
	"net"
	"os"
	"path/filepath"
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

func (r resource) loadFromBase(dir string) string {
	filename := r.String()

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

func TestFileServerSubdomainBasic(t *testing.T) {
	urls := []resource{
		"/css/main.css",
		"/js/jquery-2.1.1.js",
		"/favicon.ico",
		"/app2",
		"/app2/app2app3",
		"/",
	}

	app := newApp()
	e := httptest.New(t, app)

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	host = "http://" + subdomain + "." + host

	for _, u := range urls {
		url := u.String()
		contents := u.loadFromBase("./assets")

		e.GET(url).WithURL(host).Expect().
			Status(httptest.StatusOK).
			ContentType(u.contentType(), app.ConfigurationReadOnly().GetCharset()).
			Body().IsEqual(contents)
	}
}
