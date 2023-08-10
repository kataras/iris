package sessions_test

import (
	"sync"
	"testing"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/sessions"
)

func TestSessions(t *testing.T) {
	app := iris.New()

	sess := sessions.New(sessions.Config{Cookie: "mycustomsessionid"})
	app.Use(sess.Handler())

	testSessions(t, app)
}

const (
	testEnableSubdomain = true
)

func testSessions(t *testing.T, app *iris.Application) {
	values := map[string]interface{}{
		"Name":   "iris",
		"Months": "4",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	writeValues := func(ctx *context.Context) {
		s := sessions.Get(ctx)
		sessValues := s.GetAll()

		err := ctx.JSON(sessValues)
		if err != nil {
			t.Fatal(err)
		}
	}

	if testEnableSubdomain {
		app.Party("subdomain.").Get("/get", writeValues)
	}

	app.Post("/set", func(ctx *context.Context) {
		s := sessions.Get(ctx)
		vals := make(map[string]interface{})
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot read JSON. Trace %s", err.Error())
		}
		for k, v := range vals {
			s.Set(k, v)
		}
	})

	app.Get("/get", func(ctx *context.Context) {
		writeValues(ctx)
	})

	app.Get("/clear", func(ctx *context.Context) {
		sessions.Get(ctx).Clear()
		writeValues(ctx)
	})

	app.Get("/destroy", func(ctx *context.Context) {
		session := sessions.Get(ctx)
		if session.IsNew() {
			t.Fatal("expected session not to be nil on destroy")
		}

		session.Man.Destroy(ctx)

		if sessions.Get(ctx) != nil {
			t.Fatal("expected session inside Context to be nil after Manager's Destroy call")
		}

		ctx.JSON(struct{}{})
		// the cookie and all values should be empty
	})

	// cookie should be new.
	app.Get("/after_destroy_renew", func(ctx *context.Context) {
		isNew := sessions.Get(ctx).IsNew()
		ctx.Writef("%v", isNew)
	})

	app.Get("/multi_start_set_get", func(ctx *context.Context) {
		s := sessions.Get(ctx)
		s.Set("key", "value")
		ctx.Next()
	}, func(ctx *context.Context) {
		s := sessions.Get(ctx)
		_, err := ctx.Writef(s.GetString("key"))
		if err != nil {
			t.Fatal(err)
		}
	})

	e := httptest.New(t, app, httptest.URL("http://example.com"))

	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK).Cookies().NotEmpty()
	e.GET("/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	if testEnableSubdomain {
		es := e.Builder(func(req *httptest.Request) {
			req.WithURL("http://subdomain.example.com")
		})
		es.Request("GET", "/get").Expect().Status(iris.StatusOK).JSON().Object().Equal(values)
	}
	// test destroy which also clears first
	d := e.GET("/destroy").Expect().Status(iris.StatusOK)
	d.JSON().Object().Empty()

	d = e.GET("/after_destroy_renew").Expect().Status(iris.StatusOK)
	d.Body().IsEqual("true")
	d.Cookies().NotEmpty()

	// set and clear again
	e.POST("/set").WithJSON(values).Expect().Status(iris.StatusOK)
	e.GET("/clear").Expect().Status(iris.StatusOK).JSON().Object().Empty()

	// test start on the same request but more than one times

	e.GET("/multi_start_set_get").Expect().Status(iris.StatusOK).Body().IsEqual("value")
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

	writeValues := func(ctx *context.Context, values map[string]interface{}) error {
		return ctx.JSON(values)
	}

	app.Post("/set", func(ctx *context.Context) {
		vals := make(map[string]interface{})
		if err := ctx.ReadJSON(&vals); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		s := sess.Start(ctx)
		for k, v := range vals {
			s.SetFlash(k, v)
		}

		ctx.StatusCode(iris.StatusOK)
	})

	writeFlashValues := func(ctx *context.Context) {
		s := sess.Start(ctx)

		flashes := s.GetFlashes()
		if err := writeValues(ctx, flashes); err != nil {
			t.Fatalf("While serialize the flash values: %s", err.Error())
		}
	}

	app.Get("/get_single", func(ctx *context.Context) {
		s := sess.Start(ctx)
		flashMsgString := s.GetFlashString(valueSingleKey)
		ctx.WriteString(flashMsgString)
	})

	app.Get("/get", func(ctx *context.Context) {
		writeFlashValues(ctx)
	})

	app.Get("/clear", func(ctx *context.Context) {
		s := sess.Start(ctx)
		s.ClearFlashes()
		writeFlashValues(ctx)
	})

	app.Get("/destroy", func(ctx *context.Context) {
		sess.Destroy(ctx)
		writeFlashValues(ctx)
		ctx.StatusCode(iris.StatusOK)
		// the cookie and all values should be empty
	})

	// request cookie should be empty
	app.Get("/after_destroy", func(ctx *context.Context) {
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
	e.GET("/get_single").Expect().Status(iris.StatusOK).Body().IsEqual(valueSingleValue)
}

func TestSessionsUpdateExpiration(t *testing.T) {
	app := iris.New()

	cookieName := "mycustomsessionid"

	sess := sessions.New(sessions.Config{
		Cookie:       cookieName,
		Expires:      30 * time.Minute,
		AllowReclaim: true,
	})

	app.Use(sess.Handler())

	type response struct {
		SessionID string `json:"sessionID"`
		Logged    bool   `json:"logged"`
	}

	var writeResponse = func(ctx *context.Context) {
		session := sessions.Get(ctx)
		ctx.JSON(response{
			SessionID: session.ID(),
			Logged:    session.GetBooleanDefault("logged", false),
		})
	}

	app.Get("/get", func(ctx *context.Context) {
		writeResponse(ctx)
	})

	app.Get("/set", func(ctx iris.Context) {
		sessions.Get(ctx).Set("logged", true)
		writeResponse(ctx)
	})

	app.Post("/remember_me", func(ctx iris.Context) {
		// re-sends the cookie with the new Expires and MaxAge fields,
		// test checks that on same session id too.
		sessions.Get(ctx).Man.UpdateExpiration(ctx, 24*time.Hour)
		writeResponse(ctx)
	})

	app.Get("/destroy", func(ctx iris.Context) {
		sessions.Get(ctx).Man.Destroy(ctx) // this will delete the cookie too.
	})

	e := httptest.New(t, app, httptest.URL("http://example.com"))

	tt := e.GET("/set").Expect().Status(httptest.StatusOK)
	tt.Cookie(cookieName).MaxAge().InRange(29*time.Minute, 30*time.Minute)
	sessionID := tt.JSON().Object().Raw()["sessionID"].(string)

	expectedResponse := response{SessionID: sessionID, Logged: true}
	e.GET("/get").Expect().Status(httptest.StatusOK).
		JSON().IsEqual(expectedResponse)

	tt = e.POST("/remember_me").Expect().Status(httptest.StatusOK)
	tt.Cookie(cookieName).MaxAge().InRange(23*time.Hour, 24*time.Hour)
	tt.JSON().IsEqual(expectedResponse)

	// Test call `UpdateExpiration` when cookie is firstly created.
	e.GET("/destroy").Expect().Status(httptest.StatusOK)
	e.POST("/remember_me").Expect().Status(httptest.StatusOK).
		Cookie(cookieName).MaxAge().InRange(23*time.Hour, 24*time.Hour)
}

// go test -v -count=100 -run=TestSessionsUpdateExpirationConcurrently$
// #1488
func TestSessionsUpdateExpirationConcurrently(t *testing.T) {
	cookieName := "mycustomsessionid"
	sess := sessions.New(sessions.Config{
		Cookie:       cookieName,
		Expires:      30 * time.Minute,
		AllowReclaim: true,
	})

	app := iris.New()
	app.Use(sess.Handler())
	app.Use(func(ctx iris.Context) {
		// session will expire after 30 minute at the last visit
		sess.UpdateExpiration(ctx, 30*time.Minute)
		ctx.Next()
	})

	app.Get("/get", func(ctx iris.Context) {
		ctx.WriteString(sessions.Get(ctx).ID())
	})

	e := httptest.New(t, app, httptest.URL("http://example.com"))

	id := e.GET("/get").Expect().Status(httptest.StatusOK).Body().Raw()

	i := 0
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i < 1000 {
		go func() {
			tt := e.GET("/get").Expect().Status(httptest.StatusOK)
			tt.Body().IsEqual(id)
			tt.Cookie(cookieName).MaxAge().InRange(29*time.Minute, 30*time.Minute)
			wg.Done()
		}()
		i++
	}
	wg.Wait()
	tt := e.GET("/get").Expect()
	tt.Status(httptest.StatusOK).Body().IsEqual(id)
	tt.Cookie(cookieName).MaxAge().InRange(29*time.Minute, 30*time.Minute)
}
