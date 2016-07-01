//Package test -v ./... builds all tests
package test

import (
	"testing"

	"github.com/kataras/iris"
)

func TestSessions(t *testing.T) {
	sessionId := "mycustomsessionid"

	values := map[string]interface{}{
		"Name":   "iris",
		"Months": "4",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	api := iris.New()

	api.Config.Sessions.Cookie = sessionId
	writeValues := func(ctx *iris.Context) {
		sessValues := ctx.Session().GetAll()
		ctx.JSON(iris.StatusOK, sessValues)
	}

	if enable_subdomain_tests {
		api.Party(subdomain+".").Get("/get", func(ctx *iris.Context) {
			writeValues(ctx)
		})
	}

	api.Post("set", func(ctx *iris.Context) {
		vals := make(map[string]interface{}, 0)
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		for k, v := range vals {
			ctx.Session().Set(k, v)
		}
	})

	api.Get("/get", func(ctx *iris.Context) {
		writeValues(ctx)
	})

	api.Get("/clear", func(ctx *iris.Context) {
		ctx.Session().Clear()
		writeValues(ctx)
	})

	api.Get("/destroy", func(ctx *iris.Context) {
		ctx.SessionDestroy()
		writeValues(ctx)
		// the cookie and all values should be empty
	})

	h := tester(api, t)

	h.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	h.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	if enable_subdomain_tests {
		h.Request("GET", subdomainURL+"/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	}

	// test destory which also clears first
	d := h.GET("/destroy").Expect().Status(iris.StatusOK)
	d.JSON().Object().Empty()
	d.Cookies().ContainsOnly(sessionId)
	// set and clear again
	h.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	h.GET("/clear").Expect().Status(iris.StatusOK).JSON().Object().Empty()
}
