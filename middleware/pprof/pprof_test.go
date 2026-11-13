package pprof_test

import (
	"regexp"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/pprof"
)

// The index page must list the go1.27 goroutineleak profile, both as a
// table row (runtime/pprof.Profiles includes it) and with a non-empty
// entry in the descriptions list.
func TestIndexListsGoroutineLeakProfile(t *testing.T) {
	app := iris.New()
	app.Router("/debug/pprof", pprof.New())

	e := httptest.New(t, app)

	body := e.GET("/debug/pprof").Expect().
		Status(httptest.StatusOK).
		HasContentType("text/html", "utf-8").
		Body().Raw()

	// The profile row links to the actual profile endpoint.
	// html/template renders "=" as "&#61;" inside the unquoted href.
	if !regexp.MustCompile(`href=[^>]*goroutineleak\?debug(=|&#61;)1`).MatchString(body) {
		t.Fatalf("expected a goroutineleak profile row on the index page, got:\n%s", body)
	}

	// The description entry must not be blank.
	descRe := regexp.MustCompile(`goroutineleak:</div>\s*(.+)`)
	m := descRe.FindStringSubmatch(body)
	if m == nil || len(m[1]) == 0 || m[1] == "</li>" {
		t.Fatalf("expected a non-empty goroutineleak description, got:\n%s", body)
	}
}

// cmdline, profile, symbol and trace are not runtime/pprof profiles.
// net/http/pprof serves them with its Cmdline, Profile, Symbol and Trace
// functions, and its Handler(name) answers 404 "Unknown profile" for them.
// The index page links to them, so the middleware must serve them too.
// See https://github.com/kataras/iris/issues/1758.
func TestServesCmdlineProfileSymbolTrace(t *testing.T) {
	app := iris.New()
	app.Router("/debug/pprof", pprof.New())

	e := httptest.New(t, app)

	tests := []struct {
		action      string
		seconds     string // the shortest duration the endpoint accepts, if it takes one.
		contentType string
		disposition string
	}{
		{action: "cmdline", contentType: "text/plain"},
		{action: "symbol", contentType: "text/plain"},
		{action: "profile", seconds: "1", contentType: "application/octet-stream", disposition: `attachment; filename="profile"`},
		{action: "trace", seconds: "0.1", contentType: "application/octet-stream", disposition: `attachment; filename="trace"`},
	}

	for _, tt := range tests {
		req := e.GET("/debug/pprof/" + tt.action)
		if tt.seconds != "" {
			req.WithQuery("seconds", tt.seconds)
		}

		res := req.Expect().Status(httptest.StatusOK).HasContentType(tt.contentType)
		res.Header("Content-Disposition").IsEqual(tt.disposition)
		res.Body().NotEmpty()
	}

	// The runtime/pprof profiles and the unknown names keep their answers.
	e.GET("/debug/pprof/heap").Expect().Status(httptest.StatusOK)
	e.GET("/debug/pprof/nope").Expect().Status(httptest.StatusNotFound).
		Body().Contains("Unknown profile")
}
