package basicauth

import (
	stdContext "context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
)

// errorCapture records the last error handed to the ErrorHandler
// and then delegates to the DefaultErrorHandler.
type errorCapture struct {
	last error
}

func (c *errorCapture) handle(ctx *context.Context, err error) {
	c.last = err
	DefaultErrorHandler(ctx, err)
}

// newTestApp builds an application with the given middleware on /protected.
// The handler records the last authorization time it saw.
func newTestApp(t *testing.T, auth context.Handler, seen *[]time.Time) *iris.Application {
	t.Helper()

	app := iris.New()
	app.Get("/protected", auth, func(ctx *context.Context) {
		if seen != nil {
			at, _ := AuthorizedAt(ctx)
			*seen = append(*seen, at)
		}
		ctx.StatusCode(http.StatusNoContent)
	})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	return app
}

func login(app *iris.Application, username, password string) int {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.SetBasicAuth(username, password)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	return w.Code
}

func TestMaxAgeExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const maxAge = 10 * time.Minute

		capture := new(errorCapture)
		app := newTestApp(t, New(Options[*context.SimpleUser]{
			Allow:        AllowUsersMap(map[string]string{"kataras": "kataras_pass"}),
			MaxAge:       maxAge,
			ErrorHandler: capture.handle,
		}), nil)

		if code := login(app, "kataras", "kataras_pass"); code != http.StatusNoContent {
			t.Fatalf("first login: expected 204 but got %d", code)
		}

		time.Sleep(maxAge / 2)
		if code := login(app, "kataras", "kataras_pass"); code != http.StatusNoContent {
			t.Fatalf("login before expiration: expected 204 but got %d", code)
		}

		time.Sleep(maxAge/2 + time.Second)
		if code := login(app, "kataras", "kataras_pass"); code != http.StatusUnauthorized {
			t.Fatalf("login after expiration: expected 401 but got %d", code)
		}

		expired, ok := errors.AsType[ErrCredentialsExpired](capture.last)
		if !ok {
			t.Fatalf("expected ErrCredentialsExpired but got %T: %v", capture.last, capture.last)
		}
		if expired.Username != "kataras" {
			t.Fatalf("expected expired username to be kataras but got %q", expired.Username)
		}

		// The expired entry was removed, so the same credentials log in again as a first login.
		if code := login(app, "kataras", "kataras_pass"); code != http.StatusNoContent {
			t.Fatalf("re-login after expiration: expected 204 but got %d", code)
		}
	})
}

func TestGC(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			maxAge = 10 * time.Minute
			every  = time.Hour
		)

		ctx, cancel := stdContext.WithCancel(t.Context())
		defer cancel()

		capture := new(errorCapture)
		b := newBasicAuth(Options[*context.SimpleUser]{
			Allow:        AllowUsersMap(map[string]string{"kataras": "kataras_pass"}),
			MaxAge:       maxAge,
			ErrorHandler: capture.handle,
			GC:           GC{Context: ctx, Every: every},
		})
		app := newTestApp(t, b.handleRequest, nil)

		if code := login(app, "kataras", "kataras_pass"); code != http.StatusNoContent {
			t.Fatalf("first login: expected 204 but got %d", code)
		}

		b.mu.RLock()
		stored := len(b.credentials)
		b.mu.RUnlock()
		if stored != 1 {
			t.Fatalf("expected one stored credential but got %d", stored)
		}

		// Past the max age and past a GC tick: the expired entry has been collected,
		// so the next request is a fresh first login rather than an expired one (401).
		time.Sleep(every + time.Second)
		synctest.Wait()

		b.mu.RLock()
		stored = len(b.credentials)
		b.mu.RUnlock()
		if stored != 0 {
			t.Fatalf("expected the gc to remove the expired credential but %d remain", stored)
		}

		if code := login(app, "kataras", "kataras_pass"); code != http.StatusNoContent {
			t.Fatalf("login after gc: expected 204 but got %d (last error: %v)", code, capture.last)
		}
		if capture.last != nil {
			t.Fatalf("expected no error after gc but got: %v", capture.last)
		}
	})
}

// AuthorizedAt is the first login time behind the MaxAge entry, the same for
// every request within the period, and also reported by the SimpleUser that Default builds.
func TestAuthorizedAt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var seen []time.Time
		app := iris.New()
		app.Get("/protected", New(Options[*context.SimpleUser]{
			Allow:  AllowUsersMap(map[string]string{"kataras": "kataras_pass"}),
			MaxAge: 10 * time.Minute,
		}), func(ctx *context.Context) {
			at, ok := AuthorizedAt(ctx)
			if !ok {
				ctx.StopWithText(http.StatusInternalServerError, "AuthorizedAt reported false")
				return
			}

			userAt, err := ctx.User().GetAuthorizedAt()
			if err != nil || !userAt.Equal(at) {
				ctx.StopWithText(http.StatusInternalServerError, "ctx.User().GetAuthorizedAt() disagrees")
				return
			}

			seen = append(seen, at)
			ctx.StatusCode(http.StatusNoContent)
		})
		if err := app.Build(); err != nil {
			t.Fatal(err)
		}

		start := time.Now()
		if code := login(app, "kataras", "kataras_pass"); code != http.StatusNoContent {
			t.Fatalf("first login: expected 204 but got %d", code)
		}
		time.Sleep(time.Minute)
		if code := login(app, "kataras", "kataras_pass"); code != http.StatusNoContent {
			t.Fatalf("second login: expected 204 but got %d", code)
		}

		if len(seen) != 2 || !seen[0].Equal(start) || !seen[1].Equal(start) {
			t.Fatalf("expected both requests to report the first login time %v but got %v", start, seen)
		}
	})
}

func TestNewPanicsWithoutAllow(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected New to panic when Allow is nil")
		}
	}()
	New(Options[Map]{})
}

func TestGetUserWithoutMiddleware(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx *context.Context) {
		if _, ok := GetUser[Map](ctx); ok {
			ctx.StopWithText(http.StatusInternalServerError, "GetUser must report false without the middleware")
			return
		}
		if _, ok := AuthorizedAt(ctx); ok {
			ctx.StopWithText(http.StatusInternalServerError, "AuthorizedAt must report false without the middleware")
			return
		}
		ctx.StatusCode(http.StatusNoContent)
	})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 but got %d: %s", w.Code, w.Body.String())
	}
}
