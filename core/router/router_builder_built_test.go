package router_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
)

// The mux is compiled once from a snapshot of the route registry, so a route
// registered afterwards is never served. That used to happen in silence.
func TestRouteRegisteredAfterBuildWarns(t *testing.T) {
	app := iris.New()
	app.Get("/before", func(ctx iris.Context) {})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	app.Logger().SetLevel("warn")
	app.Logger().SetOutput(&buf)

	app.Get("/after", func(ctx iris.Context) {})

	got := buf.String()
	if !strings.Contains(got, "/after") {
		t.Fatalf("expected a warning naming the late route, got: %q", got)
	}
	// Only ".go:", not this file's name: getCaller walks past every frame under
	// github.com/kataras/iris/v14 to find the end-developer's, and an in-repo
	// test is such a frame, so it resolves to testing.go here. Outside the
	// module it names the caller's own file.
	if !strings.Contains(got, ".go:") {
		t.Fatalf("expected the warning to name the caller's file and line, got: %q", got)
	}
	if strings.Contains(got, "/before") {
		t.Fatalf("the route registered before the build must not be reported, got: %q", got)
	}
}
