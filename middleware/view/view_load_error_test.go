package view_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/view"

	"github.com/kataras/blocks"
)

// A template that fails to parse does not stop the application.
//
// The framework never loads templates on its own. Load hands the error to the
// caller, and the caller decides whether it is fatal. This is the answer to
// kataras/iris#2482, which asked v12 for a flag to keep the server running:
// log the error instead of panicking and every other route keeps serving.
func TestLoadErrorDoesNotStopTheApp(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.html")

	// The action is never closed, so the parse fails.
	if err := os.WriteFile(index, []byte(`<h1>{{ .Title</h1>`), 0o600); err != nil {
		t.Fatal(err)
	}

	templates := blocks.New(dir)
	if err := templates.Load(); err == nil {
		t.Fatal("expected Load to report the broken template")
	}

	app := iris.New()
	app.UseRouter(view.Handler(templates))
	app.Get("/ping", func(ctx iris.Context) {
		ctx.WriteString("pong")
	})
	app.Get("/", func(ctx iris.Context) {
		if err := view.Render(ctx, "index", "", iris.Map{"Title": "Page Title"}); err != nil {
			ctx.StopWithStatus(iris.StatusInternalServerError)
		}
	})

	e := httptest.New(t, app)

	// A route that renders nothing is not affected.
	e.GET("/ping").Expect().Status(httptest.StatusOK).Body().IsEqual("pong")
	// The route that renders gets the error back and answers for itself.
	e.GET("/").Expect().Status(httptest.StatusInternalServerError)

	// Fix the file and load again: the same running application renders it.
	if err := os.WriteFile(index, []byte(`<h1>{{ .Title }}</h1>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := templates.Load(); err != nil {
		t.Fatalf("load after the fix: %v", err)
	}

	e.GET("/").Expect().Status(httptest.StatusOK).Body().Contains("Page Title")
}
