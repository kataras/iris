package sessions_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/sessions"
)

func newApp() *iris.Application {
	type UserSession struct {
		ID       string
		Username string
	}

	app := iris.New()

	sessionOptions := sessions.NewOptions(
		[]byte("D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x"),
		[]byte("VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!"))
	sessionOptions.MaxAge = 1 * time.Hour
	// sessionOptions.Secure = false  // without https for testing purposes.
	sessionDB := sessions.NewCookieDatabase[UserSession](sessionOptions)

	app.Get("/set", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}

		session.Set(UserSession{
			ID:       "1",
			Username: "kataras",
		})
		if err := session.Commit(ctx); err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}

		ctx.WriteString("session set")
	})

	app.Get("/get", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}

		if session.Data.ID == "" {
			ctx.StopWithText(httptest.StatusInternalServerError, "session not found")
			return
		}

		ctx.JSON(session.Data)
	})

	app.Get("/delete", func(ctx iris.Context) {
		session, _ := sessionDB.Get(ctx, "my_session")

		session.Options.MaxAge = -1
		if err := session.Commit(ctx); err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}

		ctx.WriteString("session & cookie deleted")
	})

	app.Get("/set-flash", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}

		session.SetFlash("notice", "This is a flash message!")
		if err := session.Commit(ctx); err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}

		ctx.WriteString("Flash message set")
	})

	app.Get("/get-flash", func(ctx iris.Context) {
		session, err := sessionDB.Get(ctx, "my_session")
		if err != nil {
			ctx.StopWithText(httptest.StatusInternalServerError, err.Error())
			return
		}

		flash, ok := session.GetFlash[string]("notice")
		if ok {
			session.Commit(ctx)
			ctx.WriteString(flash)
		} else {
			ctx.WriteString("No 'notice' flash message")
		}
	})

	return app
}

func TestSessions(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app) // Let's try to handle cookies by ourselves instead of httptest.URL("https://example.com")) here.

	// Test setting a session
	setResp := e.GET("/set").Expect().Status(httptest.StatusOK)
	setResp.Body().IsEqual("session set")
	setResp.Cookies().NotEmpty()
	cookie := setResp.Cookie("my_session")
	cookieName := cookie.Name().Raw()
	cookieValue := cookie.Value().Raw()

	// Test getting the session
	getResp := e.GET("/get").WithCookie(cookieName, cookieValue).Expect().Status(httptest.StatusOK)
	getResp.JSON().Object().ContainsKey("ID").HasValue("ID", "1").ContainsKey("Username").HasValue("Username", "kataras")

	// Test deleting the session
	deleteResp := e.GET("/delete").WithCookie(cookieName, cookieValue).Expect().Status(httptest.StatusOK)
	deleteResp.Body().IsEqual("session & cookie deleted")
	deleteCookie := deleteResp.Cookie("my_session")
	deleteCookie.Expires().Lt(time.Now())

	// Test getting the session after deletion
	e.GET("/get").Expect().Status(httptest.StatusInternalServerError).Body().IsEqual("session not found")
}

func TestFlashMessages(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.URL("https://example.com"))

	// Test setting a flash message
	setFlashResp := e.GET("/set-flash").Expect().Status(httptest.StatusOK)
	setFlashResp.Cookies().NotEmpty()
	setFlashResp.Body().IsEqual("Flash message set")
	// cookie := setFlashResp.Cookie("my_session")
	// cookieName := cookie.Name().Raw()
	// cookieValue := cookie.Value().Raw()

	// Test getting the flash message
	e.GET("/get-flash").Expect().Status(httptest.StatusOK).Body().IsEqual("This is a flash message!")

	// Test getting the flash message again (should be gone)
	e.GET("/get-flash").Expect().Status(httptest.StatusOK).Body().IsEqual("No 'notice' flash message")
}

// TestSessionCookieSecurityAttributes verifies the v14 secure-by-default cookie
// attributes (Secure, SameSite=Lax) actually reach the Set-Cookie response header.
func TestSessionCookieSecurityAttributes(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	resp := e.GET("/set").Expect().Status(httptest.StatusOK)

	var sess *http.Cookie
	for _, c := range resp.Raw().Cookies() {
		if c.Name == "my_session" {
			sess = c
		}
	}
	if sess == nil {
		t.Fatal("my_session cookie was not set")
	}
	if !sess.Secure {
		t.Error("session cookie should carry the Secure attribute by default (v14)")
	}
	if sess.SameSite != http.SameSiteLaxMode {
		t.Errorf("session cookie SameSite = %v, want http.SameSiteLaxMode", sess.SameSite)
	}
}
