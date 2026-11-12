package basicauth_test

import (
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/basicauth"
)

// Every error string is logged at Debug level by the middleware and can be
// rendered into the response body by a custom ErrorHandler, so none of them may
// contain the attempted password. A failed login is very often a valid user's
// typo of their real password.
func TestBasicAuthErrorsDoNotLeakPassword(t *testing.T) {
	const secret = "hunter2"

	errs := []error{
		basicauth.ErrCredentialsInvalid{Username: "makis", Password: secret, CurrentTries: 1},
		basicauth.ErrCredentialsForbidden{Username: "makis", Password: secret, Tries: 3},
		basicauth.ErrCredentialsExpired{Username: "makis", Password: secret},
	}

	for _, err := range errs {
		msg := err.Error()
		if strings.Contains(msg, secret) {
			t.Errorf("%T.Error() leaks the password: %q", err, msg)
		}
		if !strings.Contains(msg, "makis") {
			t.Errorf("%T.Error() should still name the user, got %q", err, msg)
		}
	}

	// The raw header is base64 of username:password, so it must not be printed either.
	const header = "Basic bWFraXM6aHVudGVyMg=="
	if msg := (basicauth.ErrCredentialsMissing{Header: header}).Error(); strings.Contains(msg, "bWFraXM6aHVudGVyMg") {
		t.Errorf("ErrCredentialsMissing.Error() leaks the header: %q", msg)
	}
}

// The MaxTries lockout used to live only in a client cookie, so a brute-forcer
// simply did not send it. The counter is now server-side; the cookie is still
// written for compatibility but is no longer authoritative.
func TestBasicAuthMaxTriesIsServerSide(t *testing.T) {
	app := iris.New()
	auth := basicauth.New(basicauth.Options[*iris.SimpleUser]{
		Allow:    basicauth.AllowUsersMap(map[string]string{"makis": "correct"}),
		MaxTries: 3,
	})
	app.Get("/protected", auth, func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	// A client that never returns the cookie still gets locked out.
	for range 2 {
		e.GET("/protected").WithBasicAuth("makis", "wrong").
			Expect().Status(httptest.StatusUnauthorized)
	}

	// The third failure crosses MaxTries: forbidden, not another 401 challenge.
	e.GET("/protected").WithBasicAuth("makis", "wrong").
		Expect().Status(httptest.StatusForbidden)

	// Further guesses stay forbidden, so the attacker never gets to try the
	// remaining candidates. (A correct password still clears the counter - the
	// feature throttles guessing, it does not lock the account.)
	e.GET("/protected").WithBasicAuth("makis", "also-wrong").
		Expect().Status(httptest.StatusForbidden)
	e.GET("/protected").WithBasicAuth("makis", "another-guess").
		Expect().Status(httptest.StatusForbidden)
}

// A correct login must not be affected by another user's failures.
func TestBasicAuthMaxTriesPerUser(t *testing.T) {
	app := iris.New()
	auth := basicauth.New(basicauth.Options[*iris.SimpleUser]{
		Allow:    basicauth.AllowUsersMap(map[string]string{"makis": "correct", "other": "pass"}),
		MaxTries: 2,
	})
	app.Get("/protected", auth, func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	e.GET("/protected").WithBasicAuth("makis", "wrong").
		Expect().Status(httptest.StatusUnauthorized)

	e.GET("/protected").WithBasicAuth("other", "pass").
		Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
}

// The tries cookie rides along with credentials, so it is HttpOnly and SameSite=Lax.
func TestBasicAuthMaxTriesCookieFlags(t *testing.T) {
	app := iris.New()
	auth := basicauth.New(basicauth.Options[*iris.SimpleUser]{
		Allow:    basicauth.AllowUsersMap(map[string]string{"makis": "correct"}),
		MaxTries: 5,
	})
	app.Get("/protected", auth, func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)
	cookie := e.GET("/protected").WithBasicAuth("makis", "wrong").
		Expect().Status(httptest.StatusUnauthorized).Cookie(basicauth.DefaultMaxTriesCookie).Raw()
	if !cookie.HttpOnly {
		t.Fatal("expected the tries cookie to be HttpOnly")
	}
	if cookie.SameSite != 2 { // http.SameSiteLaxMode
		t.Fatalf("expected the tries cookie to be SameSite=Lax but got %v", cookie.SameSite)
	}
}

// HTTPSOnly is about transport confidentiality. It used to also require HTTP/2,
// which meant every legitimate TLS request over HTTP/1.1 - the majority behind
// most load balancers, and curl by default - was answered with 505.
func TestBasicAuthHTTPSOnlyRejectsPlaintext(t *testing.T) {
	app := iris.New()
	auth := basicauth.New(basicauth.Options[*iris.SimpleUser]{
		Allow:     basicauth.AllowUsersMap(map[string]string{"makis": "correct"}),
		HTTPSOnly: true,
	})
	app.Get("/protected", auth, func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	// httptest serves over plain HTTP, so the guard fires - which is correct.
	e.GET("/protected").WithBasicAuth("makis", "correct").
		Expect().Status(httptest.StatusHTTPVersionNotSupported)
}

// An unknown username must cost the same as a known one, otherwise the endpoint
// is a username oracle. Timing is not asserted here (it is inherently flaky);
// what is asserted is that both paths reach the password comparison.
func TestBasicAuthUnknownUserComparesPassword(t *testing.T) {
	compares := 0
	countingCompare := func(opts *basicauth.UserAuthOptions[*iris.SimpleUser]) {
		opts.ComparePassword = func(stored, userPassword string) bool {
			compares++
			return stored == userPassword
		}
	}

	allows := map[string]basicauth.AuthFunc[*iris.SimpleUser]{
		"AllowUsersMap": basicauth.AllowUsersMap(map[string]string{"makis": "correct"}, countingCompare),
		"AllowUsers":    basicauth.AllowUsers([]*iris.SimpleUser{{Username: "makis", Password: "correct"}}, countingCompare),
	}

	for name, allow := range allows {
		t.Run(name, func(t *testing.T) {
			compares = 0
			if _, ok := allow(nil, "makis", "wrong"); ok {
				t.Fatal("expected the known user with a wrong password to fail")
			}
			knownCompares := compares

			compares = 0
			if _, ok := allow(nil, "does-not-exist", "wrong"); ok {
				t.Fatal("expected the unknown user to fail")
			}

			if compares != knownCompares {
				t.Fatalf("unknown username ran %d password comparisons, a known one ran %d: "+
					"the difference is measurable and enumerates valid usernames", compares, knownCompares)
			}
		})
	}
}
