package main

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kataras/iris/httptest"
	"github.com/klauspost/compress/gzip"
)

type resource string

// content types that are used in the ./assets,
// we could use the detectContentType that iris do but it's better
// to do it manually so we can test if that returns the correct result on embedding files.
func (r resource) contentType() string {
	switch filepath.Ext(r.String()) {
	case ".js":
		return "application/javascript"
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
	filename := r.String()

	filename = r.strip("/static")

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
	"/static/css/bootstrap.min.css",
	"/static/js/jquery-2.1.1.js",
	"/static/favicon.ico",
}

// if bindata's values matches with the assets/... contents
// and secondly if the StaticEmbedded had successfully registered
// the routes and gave the correct response.
func TestEmbeddingGzipFilesIntoApp(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	if runtime.GOOS != "windows" {
		// remove the embedded static favicon for !windows,
		// it should be built for unix-specific in order to be work
		urls = urls[0 : len(urls)-1]
	}

	for i, u := range urls {
		url := u.String()
		rawContents := u.loadFromBase("./assets")

		response := e.GET(url).Expect()
		response.ContentType(u.contentType(), app.ConfigurationReadOnly().GetCharset())

		if expected, got := response.Raw().StatusCode, httptest.StatusOK; expected != got {
			t.Fatalf("[%d] of '%s': expected %d status code but got %d", i, url, expected, got)
		}

		func() {
			reader, err := gzip.NewReader(bytes.NewBuffer(response.Content))
			defer reader.Close()
			if err != nil {
				t.Fatalf("[%d] of '%s': %v", i, url, err)
			}
			buf := new(bytes.Buffer)
			reader.WriteTo(buf)
			if expected, got := rawContents, buf.String(); expected != got {
				// t.Fatalf("[%d] of '%s': expected body:\n%s but got:\n%s", i, url, expected, got)
				// let's reduce the output here...
				// they are big files, no need to check for length here.
				t.Fatalf("[%d] %s, expected body to look like: '%s...%s' but got '%s...%s'", i, url, expected[:40], expected[len(rawContents)-40:], got[:40], got[len(got)-40:])
			}
		}()
	}
}
