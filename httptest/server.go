package httptest

import (
	"net/http/httptest"

	"github.com/kataras/iris/v12"
)

// NewServer is just a helper to create a new standard
// httptest.Server instance.
func NewServer(t IrisTesty, app *iris.Application) *httptest.Server {
	if err := app.Build(); err != nil {
		t.Fatal(err)
		return nil
	}

	return httptest.NewServer(app)
}
