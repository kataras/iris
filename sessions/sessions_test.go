package sessions_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/httptest"
	"github.com/kataras/iris/sessions"
)

func TestSessions(t *testing.T) {
	app := iris.New()

	sess := sessions.New(sessions.Config{Cookie: "mycustomsessionid"})
	testSessions(t, sess, app)
}

const (
	testEnableSubdomain = false
)

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

	if testEnableSubdomain {
		app.Party("subdomain.").Get("/get", func(ctx context.Context) {
			writeValues(ctx)
		})
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
	if testEnableSubdomain {
		es := httptest.New(t, app, httptest.URL("http://subdomain.example.com"))
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

	// test start on the same request but more than one times

	e.GET("/multi_start_set_get").Expect().Status(iris.StatusOK).Body().Equal("value")
}

func TestFlashMessages(t *testing.T) {
	app := iris.New()

	sess := sessions.New(sessions.Config{Cookie: "mycustomsessionid"})

	valueSingleKey := "Name"
	valueSingleValue := "iris-sessions"

	values := map[string]interface{}{
		valueSingleKey: valueSingleValue,
		"Days":         "1",
		"Secret":       "dsads£2132215£%%Ssdsa",
	}

	writeValues := func(ctx context.Context, values map[string]interface{}) error {
		_, err := ctx.JSON(values)
		return err
	}

	app.Post("/set", func(ctx context.Context) {
		vals := make(map[string]interface{}, 0)
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		s := sess.Start(ctx)
		for k, v := range vals {
			s.SetFlash(k, v)
		}

		ctx.StatusCode(iris.StatusOK)
	})

	writeFlashValues := func(ctx context.Context) {
		s := sess.Start(ctx)

		flashes := s.GetFlashes()
		if err := writeValues(ctx, flashes); err != nil {
			t.Fatalf("While serialize the flash values: %s", err.Error())
		}
	}

	app.Get("/get_single", func(ctx context.Context) {
		s := sess.Start(ctx)
		flashMsgString := s.GetFlashString(valueSingleKey)
		ctx.WriteString(flashMsgString)
	})

	app.Get("/get", func(ctx context.Context) {
		writeFlashValues(ctx)
	})

	app.Get("/clear", func(ctx context.Context) {
		s := sess.Start(ctx)
		s.ClearFlashes()
		writeFlashValues(ctx)
	})

	app.Get("/destroy", func(ctx context.Context) {
		sess.Destroy(ctx)
		writeFlashValues(ctx)
		ctx.StatusCode(iris.StatusOK)
		// the cookie and all values should be empty
	})

	// request cookie should be empty
	app.Get("/after_destroy", func(ctx context.Context) {
		ctx.StatusCode(iris.StatusOK)
	})

	e := httptest.New(t, app, httptest.URL("http://example.com"))

	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	// get all
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	// get the same flash on other request should return nothing because the flash message is removed after fetch once
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Empty()
	// test destroy which also clears first
	d := e.GET("/destroy").Expect().Status(iris.StatusOK)
	d.JSON().Object().Empty()
	e.GET("/after_destroy").Expect().Status(iris.StatusOK).Cookies().Empty()
	// set and clear again
	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/clear").Expect().Status(iris.StatusOK).JSON().Object().Empty()

	// set again in order to take the single one ( we don't test Cookies.NotEmpty because httpexpect default conf reads that from the request-only)
	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK)
	e.GET("/get_single").Expect().Status(iris.StatusOK).Body().Equal(valueSingleValue)
}
