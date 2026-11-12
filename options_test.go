package iris_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func TestApplicationNewOptions(t *testing.T) {
	app := iris.New(iris.Options{
		Context: iris.ContextOptions{
			RemoteAddrHeaders:  []string{"X-Forwarded-For"},
			FireEmptyFormError: true,
		},
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"options": ctx.Options(),
		})
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	if app.Routes().CountHandlers() != 1 { // after build.
		t.Fatalf("expected 1 handler as the context modifications are not stored in any handler, they are stored in the current Router, found: %d", app.Routes().CountHandlers())
	}

	e := httptest.New(t, app)

	// The custom fields are kept and every unset field inherits the defaults.
	obj := e.GET("/").Expect().Status(httptest.StatusOK).
		JSON().Object().Value("options").Object()
	obj.Value("remote_addr_headers").Array().ConsistsOf("X-Forwarded-For")
	obj.HasValue("fire_empty_form_error", true)
	obj.HasValue("charset", "utf-8")
	obj.HasValue("post_max_memory", 32<<20)
}
