package iris_test

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"testing"
	"time"
)

var testMarkdownContents = `## Hello Markdown from Iris

This is an example of Markdown with Iris



Features
--------

All features of Sundown are supported, including:

*   **Compatibility**. The Markdown v1.0.3 test suite passes with
    the --tidy option.  Without --tidy, the differences are
    mostly in whitespace and entity escaping, where blackfriday is
    more consistent and cleaner.

*   **Common extensions**, including table support, fenced code
    blocks, autolinks, strikethroughs, non-strict emphasis, etc.

*   **Safety**. Blackfriday is paranoid when parsing, making it safe
    to feed untrusted user input without fear of bad things
    happening. The test suite stress tests this and there are no
    known inputs that make it crash.  If you find one, please let me
    know and send me the input that does it.

    NOTE: "safety" in this context means *runtime safety only*. In order to
    protect yourself against JavaScript injection in untrusted content, see
    [this example](https://github.com/russross/blackfriday#sanitize-untrusted-content).

*   **Fast processing**. It is fast enough to render on-demand in
    most web applications without having to cache the output.

*   **Thread safety**. You can run multiple parsers in different
    goroutines without ill effect. There is no dependence on global
    shared state.

*   **Minimal dependencies**. Blackfriday only depends on standard
    library packages in Go. The source code is pretty
    self-contained, so it is easy to add to any project, including
    Google App Engine projects.

*   **Standards compliant**. Output successfully validates using the
    W3C validation tool for HTML 4.01 and XHTML 1.0 Transitional.

	[this is a link](https://github.com/kataras/iris) `

// 10 seconds test
// EXAMPLE: https://github.com/iris-contrib/examples/tree/master/cache_body
func TestCacheCanRender(t *testing.T) {
	iris.ResetDefault()

	iris.Config.IsDevelopment = true
	defer iris.Close()
	var i = 1
	bodyHandler := func(ctx *iris.Context) {
		if i%2 == 0 { // only for testing
			ctx.SetStatusCode(iris.StatusNoContent)
			i++
			return
		}
		i++
		ctx.Markdown(iris.StatusOK, testMarkdownContents)
	}

	expiration := time.Duration(1 * time.Minute)

	iris.Get("/", iris.Cache(bodyHandler, expiration))

	e := httptest.New(iris.Default, t)

	expectedBody := iris.SerializeToString("text/markdown", testMarkdownContents)

	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(expectedBody)
	time.Sleep(5 * time.Second)                                          // let's sleep for a while in order to be saved in cache(running in goroutine)
	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(expectedBody) // the 1 minute didnt' passed so it should work

}

// CacheRemote IS not ready for production yet
// func TestCacheRemote(t *testing.T) {
// 	iris.ResetDefault()
// 	// setup the remote cache service listening on localhost:8888/cache
// 	remoteService := iris.New(iris.OptionDisableBanner(true))
// 	remoteService.Any("/cache", remoteService.ServeRemoteCache(5*time.Second)) // clear the gc every  5 seconds
// 	defer remoteService.Close()
// 	go remoteService.Listen("localhost:8888")
// 	<-remoteService.Available
//
// 	app := iris.New()
//
// 	n := 1
// 	bodyHandler := func(ctx *iris.Context) {
// 		n++
// 		ctx.Markdown(iris.StatusOK, testMarkdownContents)
// 	}
//
// 	app.Get("/", iris.RemoteCache("http://localhost:8888/cache", bodyHandler, 10*time.Second))
//
// 	e := httptest.New(app, t, httptest.Debug(false))
//
// 	expectedBody := app.SerializeToString("text/markdown", testMarkdownContents)
//
// 	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(expectedBody)
// 	time.Sleep(3 * time.Second) // let's wait a while because saving is going on a goroutine (in some ms, but travis is slow so 2 seconds wait)
// 	// we are in cache, so the 'n' should be 1
// 	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(expectedBody)
// 	if n > 1 {
// 		// n should be 1 because it doesn't changed after the first call
// 		t.Fatalf("Expected n = %d but got %d. Cache has problems!!", 1, n)
// 	}
//
// 	// let's wait 5 more seconds, the cache should be cleared now the n should be 2
// 	time.Sleep(5 * time.Second)
// 	e.GET("/").Expect().Status(iris.StatusNoContent).Body().Empty()
// 	if n != 2 {
// 		t.Fatalf("Expected n = %d but got %d. Cache has problems!!", 2, n)
// 	}
// }
