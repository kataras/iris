package iris_test

import (
	"testing"
	// developers can use any library to add a custom cookie encoder/decoder.
	// At this test code we use the gorilla's securecookie library:
	"github.com/gorilla/securecookie"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/sessions"
	"gopkg.in/kataras/iris.v6/httptest"
)

func TestSessions(t *testing.T) {
	app := iris.New()
	app.Adapt(httprouter.New())
	app.Adapt(sessions.New(sessions.Config{Cookie: "mycustomsessionid"}))
	testSessions(app, t)
}

func TestSessionsEncodeDecode(t *testing.T) {
	// test the sessions encode decode via gorilla.securecookie
	app := iris.New()
	app.Adapt(httprouter.New())
	// IMPORTANT
	cookieName := "mycustomsessionid"
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey := []byte("the-big-and-secret-fash-key-here")
	blockKey := []byte("lot-secret-of-characters-big-too")
	secureCookie := securecookie.New(hashKey, blockKey)

	app.Adapt(sessions.New(sessions.Config{
		Cookie: cookieName,
		Encode: secureCookie.Encode,
		Decode: secureCookie.Decode,
	}))
	testSessions(app, t)
}

func testSessions(app *iris.Framework, t *testing.T) {
	values := map[string]interface{}{
		"Name":   "iris",
		"Months": "4",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	writeValues := func(ctx *iris.Context) {
		sessValues := ctx.Session().GetAll()
		ctx.JSON(iris.StatusOK, sessValues)
	}

	if testEnableSubdomain {
		app.Party(testSubdomain+".").Get("/get", func(ctx *iris.Context) {
			writeValues(ctx)
		})
	}

	app.Post("set", func(ctx *iris.Context) {
		vals := make(map[string]interface{}, 0)
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		for k, v := range vals {
			ctx.Session().Set(k, v)
		}
	})

	app.Get("/get", func(ctx *iris.Context) {
		writeValues(ctx)
	})

	app.Get("/clear", func(ctx *iris.Context) {
		ctx.Session().Clear()
		writeValues(ctx)
	})

	app.Get("/destroy", func(ctx *iris.Context) {
		ctx.SessionDestroy()
		writeValues(ctx)
		// the cookie and all values should be empty
	})

	// request cookie should be empty
	app.Get("/after_destroy", func(ctx *iris.Context) {
	})
	app.Config.VHost = "mydomain.com"
	e := httptest.New(app, t)

	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	if testEnableSubdomain {
		es := subdomainTester(e, app)
		es.Request("GET", "/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	}

	// test destroy which also clears first
	d := e.GET("/destroy").Expect().Status(iris.StatusOK)
	d.JSON().Object().Empty()
	// 	This removed: d.Cookies().Empty(). Reason:
	// httpexpect counts the cookies setted or deleted at the response time, but cookie is not removed, to be really removed needs to SetExpire(now-1second) so,
	// test if the cookies removed on the next request, like the browser's behavior.
	e.GET("/after_destroy").Expect().Status(iris.StatusOK).Cookies().Empty()
	// set and clear again
	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/clear").Expect().Status(iris.StatusOK).JSON().Object().Empty()
}

func TestFlashMessages(t *testing.T) {
	app := iris.New()
	app.Adapt(httprouter.New())
	app.Adapt(sessions.New(sessions.Config{Cookie: "mycustomsessionid"}))

	valueSingleKey := "Name"
	valueSingleValue := "iris-sessions"

	values := map[string]interface{}{
		valueSingleKey: valueSingleValue,
		"Days":         "1",
		"Secret":       "dsads£2132215£%%Ssdsa",
	}

	writeValues := func(ctx *iris.Context, values map[string]interface{}) error {
		return ctx.JSON(iris.StatusOK, values)
	}

	app.Post("/set", func(ctx *iris.Context) {
		vals := make(map[string]interface{}, 0)
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		sess := ctx.Session()
		for k, v := range vals {
			sess.SetFlash(k, v)
		}

		ctx.SetStatusCode(iris.StatusOK)
	})

	writeFlashValues := func(ctx *iris.Context) {
		sess := ctx.Session()
		flashes := sess.GetFlashes()
		if err := writeValues(ctx, flashes); err != nil {
			t.Fatalf("While serialize the flash values: %s", err.Error())
		}
	}

	app.Get("/get_single", func(ctx *iris.Context) {
		sess := ctx.Session()
		flashMsgString := sess.GetFlashString(valueSingleKey)
		ctx.WriteString(flashMsgString)
	})

	app.Get("/get", func(ctx *iris.Context) {
		writeFlashValues(ctx)
	})

	app.Get("/clear", func(ctx *iris.Context) {
		sess := ctx.Session()
		sess.ClearFlashes()
		writeFlashValues(ctx)
	})

	app.Get("/destroy", func(ctx *iris.Context) {
		ctx.SessionDestroy()
		writeFlashValues(ctx)
		ctx.SetStatusCode(iris.StatusOK)
		// the cookie and all values should be empty
	})

	// request cookie should be empty
	app.Get("/after_destroy", func(ctx *iris.Context) {
		ctx.SetStatusCode(iris.StatusOK)
	})

	e := httptest.New(app, t)

	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	// get all
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	// get the same flash on other request should return nothing because the flash message is removed after fetch once
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Empty()
	// test destory which also clears first
	d := e.GET("/destroy").Expect().Status(iris.StatusOK)
	d.JSON().Object().Empty()
	e.GET("/after_destroy").Expect().Status(iris.StatusOK).Cookies().Empty()
	// set and clear again
	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/clear").Expect().Status(iris.StatusOK).JSON().Object().Empty()

	// set again in order to take the single one ( we don't test Cookies.NotEmpty because httpexpect default conf reads that from the request-only)
	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK)
	//	e.GET("/get/").Expect().Status(http.StatusOK).JSON().Object().Equal(values)
	e.GET("/get_single").Expect().Status(iris.StatusOK).Body().Equal(valueSingleValue)
}
