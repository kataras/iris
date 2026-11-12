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
