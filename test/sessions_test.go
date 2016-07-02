//Package test -v ./... builds all tests
package test

// Contains tests for sessions

import (
	"testing"

	"github.com/kataras/iris"
)

func TestSessions(t *testing.T) {

	values := map[string]interface{}{
		"Name":   "iris",
		"Months": "4",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	api := iris.New()
	api.Config.Sessions.Cookie = "mycustomsessionid"

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

	e := tester(api, t)

	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	if enable_subdomain_tests {
		e.Request("GET", subdomainURL+"/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	}

	// test destory which also clears first
	d := e.GET("/destroy").Expect().Status(iris.StatusOK)
	d.JSON().Object().Empty()
	d.Cookies().ContainsOnly(api.Config.Sessions.Cookie)
	// set and clear again
	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/clear").Expect().Status(iris.StatusOK).JSON().Object().Empty()
}

func FlashMessagesTest(t *testing.T) {
	api := iris.New()
	values := map[string]string{"name": "kataras", "package": "iris"}

	api.Put("/set", func(ctx *iris.Context) {
		for k, v := range values {
			ctx.SetFlash(k, v)
		}
	})

	api.Get("/get", func(ctx *iris.Context) {
		// one time one handler
		kv := make(map[string]string)
		for k := range values {
			kv[k], _ = ctx.GetFlash(k)
		}
		//second time on the same handler
		for k := range values {
			kv[k], _ = ctx.GetFlash(k)
		}

	}, func(ctx *iris.Context) {
		// third time on a next handler
		// test the if next handler has access to them(must) because flash are request lifetime now.
		kv := make(map[string]string)
		for k := range values {
			kv[k], _ = ctx.GetFlash(k)
		}
		// print them to the client for test the response also
		ctx.JSON(iris.StatusOK, kv)
	})

	e := tester(api, t)
	e.PUT("/set").Expect().Status(iris.StatusOK)
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	// secnd request lifetime ,the flash messages here should be not available
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Empty()

}
