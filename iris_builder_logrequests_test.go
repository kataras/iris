package iris_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/accesslog"
)

// LogRequestsWith accepts a pre-configured access log: requests are recorded
// through it and its closer is registered on the application.
func TestBuilderLogRequestsWith(t *testing.T) {
	var buf bytes.Buffer
	ac := accesslog.New(&buf)

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequestsWith(ac).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Get("/ping", func(ctx iris.Context) {
				ctx.WriteString("pong")
			})
		}).
		Build()

	e := httptest.New(t, app)
	e.GET("/ping").Expect().Status(httptest.StatusOK).Body().IsEqual("pong")

	if got := buf.String(); !strings.Contains(got, "/ping") {
		t.Fatalf("expected the custom access log to record the request, got: %q", got)
	}
}
