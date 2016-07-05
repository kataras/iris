package iris

// Contains tests for sessions(sessions package) & flash messages(context)

import (
	"testing"
)

func TestSessions(t *testing.T) {

	values := map[string]interface{}{
		"Name":   "iris",
		"Months": "4",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	initDefault()
	HTTPServer.Config.ListeningAddr = "127.0.0.1:8080" // in order to test the sessions
	Config.Sessions.Cookie = "mycustomsessionid"

	writeValues := func(ctx *Context) {
		sessValues := ctx.Session().GetAll()
		ctx.JSON(StatusOK, sessValues)
	}

	if testEnableSubdomain {
		Party(testSubdomain+".").Get("/get", func(ctx *Context) {
			writeValues(ctx)
		})
	}

	Post("set", func(ctx *Context) {
		vals := make(map[string]interface{}, 0)
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		for k, v := range vals {
			ctx.Session().Set(k, v)
		}
	})

	Get("/get", func(ctx *Context) {
		writeValues(ctx)
	})

	Get("/clear", func(ctx *Context) {
		ctx.Session().Clear()
		writeValues(ctx)
	})

	Get("/destroy", func(ctx *Context) {
		ctx.SessionDestroy()
		writeValues(ctx)
		// the cookie and all values should be empty
	})

	e := Tester(t)

	e.POST("/set").WithJSON(values).Expect().Status(StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(StatusOK).JSON().Object().Equal(values)
	if testEnableSubdomain {
		es := subdomainTester(e)
		es.Request("GET", "/get").Expect().Status(StatusOK).JSON().Object().Equal(values)
	}

	// test destory which also clears first
	d := e.GET("/destroy").Expect().Status(StatusOK)
	d.JSON().Object().Empty()
	d.Cookies().ContainsOnly(Config.Sessions.Cookie)
	// set and clear again
	e.POST("/set").WithJSON(values).Expect().Status(StatusOK).Cookies().NotEmpty()
	e.GET("/clear").Expect().Status(StatusOK).JSON().Object().Empty()
}

func FlashMessagesTest(t *testing.T) {
	initDefault()
	values := map[string]string{"name": "kataras", "package": "iris"}

	Put("/set", func(ctx *Context) {
		for k, v := range values {
			ctx.SetFlash(k, v)
		}
	})

	//we don't get the flash so on the next request the flash messages should be available.
	Get("/get_no_getflash", func(ctx *Context) {})

	Get("/get", func(ctx *Context) {
		// one time one handler
		kv := make(map[string]string)
		for k := range values {
			kv[k], _ = ctx.GetFlash(k)
		}
		//second time on the same handler
		for k := range values {
			kv[k], _ = ctx.GetFlash(k)
		}

	}, func(ctx *Context) {
		// third time on a next handler
		// test the if next handler has access to them(must) because flash are request lifetime now.
		kv := make(map[string]string)
		for k := range values {
			kv[k], _ = ctx.GetFlash(k)
		}
		// print them to the client for test the response also
		ctx.JSON(StatusOK, kv)
	})

	e := Tester(t)
	e.PUT("/set").Expect().Status(StatusOK).Cookies().NotEmpty()
	// just a request which does not use the flash message, so flash messages should be available on the next request
	e.GET("/get_no_getflash").Expect().Status(StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(StatusOK).JSON().Object().Equal(values)
	// second request ,the flash messages here should be not available and cookie has been removed
	// (the true is that the cookie is removed from the first GetFlash, but is available though the whole request saved on context's values for faster get, keep that secret!)*
	g := e.GET("/get").Expect().Status(StatusOK)
	g.JSON().Object().Empty()
	g.Cookies().Empty()

}
