package modrevision_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/modrevision"
)

func TestNew(t *testing.T) {
	app := iris.New()
	app.Get("/health", modrevision.New(modrevision.Options{
		ServerName:   "Iris Server",
		Env:          "development",
		Developer:    "kataras",
		UnixTime:     false,
		TimeLocation: time.FixedZone("Greece/Athens", 7200),
	}))

	e := httptest.New(t, app, httptest.Debug(true))
	e.GET("/health").Expect().Status(iris.StatusOK).
		Body().Contains("Iris Server (development)")
}
