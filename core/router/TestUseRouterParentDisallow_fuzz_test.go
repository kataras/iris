package router_test

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"testing"
)

func FuzzTestUseRouterParentDisallow(f *testing.F) {
	f.Add("no_userouter_allowed", "always", "_2", "_3", "/index", "/", "/user")
	f.Fuzz(func(t *testing.T, data1 string, data2 string, data3 string, data4 string, data5 string,
		data6 string, data7 string) {
		app := iris.New()
		app.UseRouter(func(ctx iris.Context) {
			ctx.WriteString(data2)
			ctx.Next()
		})
		app.Get(data5, func(ctx iris.Context) {
			ctx.WriteString(data1)
		})

		app.SetPartyMatcher(func(ctx iris.Context, p iris.Party) bool {
			// modifies the PartyMatcher to not match any UseRouter,
			// tests should receive the handlers response alone.
			return false
		})

		app.PartyFunc(data6, func(p iris.Party) { // it's the same instance of app.
			p.UseRouter(func(ctx iris.Context) {
				ctx.WriteString(data3)
				ctx.Next()
			})
			p.Get(data6, func(ctx iris.Context) {
				ctx.WriteString(data1)
			})
		})

		app.PartyFunc(data7, func(p iris.Party) {
			p.UseRouter(func(ctx iris.Context) {
				ctx.WriteString(data4)
				ctx.Next()
			})

			p.Get(data6, func(ctx iris.Context) {
				ctx.WriteString(data1)
			})
		})

		e := httptest.New(t, app)
		e.GET(data5)
		e.GET(data6)
		e.GET(data7)

	})
}
