package main

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/httptest"
	"github.com/kataras/iris/sessions"

	"github.com/gorilla/securecookie"
)

func TestSessionsEncodeDecode(t *testing.T) {
	// test the sessions encode decode via gorilla.securecookie
	app := iris.New()
	// IMPORTANT
	cookieName := "mycustomsessionid"
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey := []byte("the-big-and-secret-fash-key-here")
	blockKey := []byte("lot-secret-of-characters-big-too")
	secureCookie := securecookie.New(hashKey, blockKey)
	sess := sessions.New(sessions.Config{
		Cookie: cookieName,
		Encode: secureCookie.Encode,
		Decode: secureCookie.Decode,
	})

	testSessions(t, sess, app)
}

func testSessions(t *testing.T, sess *sessions.Sessions, app *iris.Application) {
	values := map[string]interface{}{
		"Name":   "iris",
		"Months": "4",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	writeValues := func(ctx context.Context) {
		s := sess.Start(ctx)
		sessValues := s.GetAll()

		ctx.JSON(sessValues)
	}

	app.Post("/set", func(ctx context.Context) {
		s := sess.Start(ctx)
		vals := make(map[string]interface{}, 0)
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		for k, v := range vals {
			s.Set(k, v)
		}
	})

	app.Get("/get", func(ctx context.Context) {
		writeValues(ctx)
	})

	app.Get("/clear", func(ctx context.Context) {
		sess.Start(ctx).Clear()
		writeValues(ctx)
	})

	app.Get("/destroy", func(ctx context.Context) {
		sess.Destroy(ctx)
		writeValues(ctx)
		// the cookie and all values should be empty
	})

	// request cookie should be empty
	app.Get("/after_destroy", func(ctx context.Context) {
	})

	app.Get("/multi_start_set_get", func(ctx context.Context) {
		s := sess.Start(ctx)
		s.Set("key", "value")
		ctx.Next()
	}, func(ctx context.Context) {
		s := sess.Start(ctx)
		ctx.Writef(s.GetString("key"))
	})

	e := httptest.New(t, app, httptest.URL("http://example.com"))

	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)

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

	// test start on the same request but more than one times

	e.GET("/multi_start_set_get").Expect().Status(iris.StatusOK).Body().Equal("value")
}
