package router_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

type auditAPI struct{}

func newAuditAPI() *auditAPI { return &auditAPI{} }

func (api *auditAPI) Routing(r iris.Router) {
	r.Get("/audit", func(ctx iris.Context) {
		ctx.WriteString("audit")
	})
}

// A bare func(Router) is one controller among many: the controllers that
// follow it in the same Router call must be registered too.
func TestRouterRegistersControllersAfterRouterFunc(t *testing.T) {
	app := iris.New()
	app.Router("/admin",
		func(r iris.Router) {
			r.Get("/dashboard", func(ctx iris.Context) {
				ctx.WriteString("dashboard")
			})
		},
		newAuditAPI(),
	)

	e := httptest.New(t, app)
	e.GET("/admin/dashboard").Expect().Status(httptest.StatusOK).Body().IsEqual("dashboard")
	e.GET("/admin/audit").Expect().Status(httptest.StatusOK).Body().IsEqual("audit")
}
