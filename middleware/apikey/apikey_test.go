package apikey

import (
	"bytes"
	"crypto/sha256"
	stdErrors "errors"
	"fmt"
	"net/http"
	nethttptest "net/http/httptest"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

const secret = "iris_secret-key"

func newApp(t *testing.T, opts Options) *iris.Application {
	t.Helper()

	app := iris.New()
	app.Logger().SetLevel("disable")

	app.Get("/guarded", New(opts), func(ctx iris.Context) {
		principal, ok := Get(ctx)
		if !ok {
			ctx.WriteString("anonymous")
			return
		}

		ctx.WriteString(principal)
	})

	return app
}

// serve sends one request carrying the given header. A headerName of ""
// sends no header at all.
func serve(t *testing.T, opts Options, headerName, headerValue string) int {
	t.Helper()

	e := httptest.New(t, newApp(t, opts))
	req := e.GET("/guarded")
	if headerName != "" {
		req = req.WithHeader(headerName, headerValue)
	}

	return req.Expect().Raw().StatusCode
}

func keyed() Options {
	return Options{Keys: map[string]string{"iris-backend": secret}}
}

func TestValidKeyIsAccepted(t *testing.T) {
	if got := serve(t, keyed(), "Authorization", "Bearer "+secret); got != http.StatusOK {
		t.Fatalf("status = %d with a valid bearer key, want 200", got)
	}
}

// RFC 7235 makes the auth-scheme case-insensitive.
func TestSchemeIsCaseInsensitive(t *testing.T) {
	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "BeArEr"} {
		t.Run(scheme, func(t *testing.T) {
			if got := serve(t, keyed(), "Authorization", scheme+" "+secret); got != http.StatusOK {
				t.Fatalf("status = %d for scheme %q, want 200", got, scheme)
			}
		})
	}
}

func TestExtraPaddingAfterSchemeIsTolerated(t *testing.T) {
	if got := serve(t, keyed(), "Authorization", "Bearer   "+secret); got != http.StatusOK {
		t.Fatalf("status = %d with padding after the scheme, want 200", got)
	}

	// net/textproto strips leading and trailing spaces and tabs from every
	// header value, so a padded key never reaches the middleware over a real
	// connection. stripScheme stays strict about a trailing space anyway,
	// see TestStripScheme. This one goes over a socket on purpose: an
	// in-process binder hands the value through unparsed.
	app := newApp(t, keyed())
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/guarded", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+secret+"  ")

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d with a trailing space, want 200", resp.StatusCode)
	}
}

// C1: the comparison must not depend on the length of the configured secret.
// subtle.ConstantTimeCompare returns 0 the moment two lengths differ, so
// comparing raw values would make a presented key whose length matches a
// configured one measurably slower than one that matches none. Both sides are
// digested first; this pins that every comparison sees two equal, fixed-width
// inputs, whatever the caller presents.
func TestComparisonIsLengthIndependent(t *testing.T) {
	entries := []keyEntry{
		{principal: "short", secret: digest("k")},
		{principal: "long", secret: digest(strings.Repeat("k", 4096))},
	}

	original := compare
	t.Cleanup(func() { compare = original })

	var seen [][2]int
	compare = func(x, y []byte) int {
		seen = append(seen, [2]int{len(x), len(y)})
		return original(x, y)
	}

	for _, presented := range []string{"", "k", "kk", strings.Repeat("k", 4096), strings.Repeat("k", 4097)} {
		seen = nil

		if _, ok := match(entries, presented); ok != (presented == "k" || presented == strings.Repeat("k", 4096)) {
			t.Fatalf("match(%d bytes) = %v", len(presented), ok)
		}

		if len(seen) != len(entries) {
			t.Fatalf("the compare ran %d times for %d keys: an early exit is a timing oracle", len(seen), len(entries))
		}

		for _, lengths := range seen {
			if lengths[0] != sha256.Size || lengths[1] != sha256.Size {
				t.Fatalf("compared %d against %d bytes; both must be a %d-byte digest, or the key length leaks",
					lengths[0], lengths[1], sha256.Size)
			}
		}
	}
}

// A wrong key whose length matches a configured one is rejected like any
// other, and takes the same path.
func TestWrongKeyOfMatchingLength(t *testing.T) {
	wrong := strings.Repeat("x", len(secret))
	if len(wrong) != len(secret) {
		t.Fatal("the test key is not the same length as the configured one")
	}

	if got := serve(t, keyed(), "Authorization", "Bearer "+wrong); got != http.StatusUnauthorized {
		t.Fatalf("status = %d for a wrong key of the same length, want 401", got)
	}
	if got := serve(t, keyed(), "Authorization", "Bearer "+wrong+"x"); got != http.StatusUnauthorized {
		t.Fatalf("status = %d for a wrong key of a different length, want 401", got)
	}
}

func TestRejection(t *testing.T) {
	cases := []struct {
		name        string
		headerName  string
		headerValue string
	}{
		{"no header", "", ""},
		{"empty value", "Authorization", ""},
		{"bare key without a scheme", "Authorization", secret},
		{"scheme with no key", "Authorization", "Bearer"},
		{"scheme with only spaces", "Authorization", "Bearer   "},
		{"wrong scheme", "Authorization", "Basic " + secret},
		{"scheme not separated", "Authorization", "Bearer" + secret},
		{"wrong key", "Authorization", "Bearer nope"},
		{"key as a prefix", "Authorization", "Bearer " + secret + "-extra"},
		{"another header is not accepted", "X-API-Key", secret},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := serve(t, keyed(), c.headerName, c.headerValue); got != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", got)
			}
		})
	}
}

// Two credentials on one request is not a valid presentation of either.
func TestDuplicateHeadersAreRejected(t *testing.T) {
	app := newApp(t, keyed())
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	req := nethttptest.NewRequest(http.MethodGet, "/guarded", nil)
	req.Header.Add("Authorization", "Bearer "+secret)
	req.Header.Add("Authorization", "Bearer smuggled")

	rec := nethttptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d with a smuggled second key, want 401", rec.Code)
	}
}

// A consumer that cannot send an Authorization header gets a custom header
// and no scheme.
func TestCustomHeaderWithoutScheme(t *testing.T) {
	opts := keyed()
	opts.Header = "X-API-Key"
	opts.Scheme = NoScheme

	if got := serve(t, opts, "X-API-Key", secret); got != http.StatusOK {
		t.Fatalf("status = %d with the custom header, want 200", got)
	}
	if got := serve(t, opts, "Authorization", "Bearer "+secret); got != http.StatusUnauthorized {
		t.Fatalf("status = %d on Authorization when a custom header is configured, want 401", got)
	}
	// With no scheme, a Bearer prefix is part of the key and must not match.
	if got := serve(t, opts, "X-API-Key", "Bearer "+secret); got != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", got)
	}
}

func TestCustomSchemeName(t *testing.T) {
	opts := keyed()
	opts.Scheme = "Token"

	if got := serve(t, opts, "Authorization", "Token "+secret); got != http.StatusOK {
		t.Fatalf("status = %d with the configured scheme, want 200", got)
	}
	if got := serve(t, opts, "Authorization", "Bearer "+secret); got != http.StatusUnauthorized {
		t.Fatalf("status = %d with the wrong scheme, want 401", got)
	}
}

func TestAnyConfiguredKeyIsAccepted(t *testing.T) {
	opts := Options{Keys: map[string]string{
		"iris-backend": "key-one",
		"ops":          "key-two",
	}}

	for _, key := range []string{"key-one", "key-two"} {
		if got := serve(t, opts, "Authorization", "Bearer "+key); got != http.StatusOK {
			t.Fatalf("status = %d for %q, want 200", got, key)
		}
	}
}

// Empty Keys is the dangerous configuration: default-deny unless the
// operator explicitly opted into anonymous access.
func TestEmptyKeys(t *testing.T) {
	t.Run("default denies everything", func(t *testing.T) {
		if got := serve(t, Options{}, "Authorization", "Bearer anything"); got != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", got)
		}
	})

	t.Run("AllowAnonymous opens the group", func(t *testing.T) {
		// Get reports false: nothing authenticated, so nothing is attributed.
		httptest.New(t, newApp(t, Options{AllowAnonymous: true})).GET("/guarded").
			Expect().Status(httptest.StatusOK).Body().IsEqual("anonymous")
	})

	t.Run("AllowAnonymous is ignored once a key exists", func(t *testing.T) {
		opts := keyed()
		opts.AllowAnonymous = true

		if got := serve(t, opts, "", ""); got != http.StatusUnauthorized {
			t.Fatalf("status = %d without a key, want 401", got)
		}
		if got := serve(t, opts, "Authorization", "Bearer "+secret); got != http.StatusOK {
			t.Fatalf("status = %d with the key, want 200", got)
		}
	})

	t.Run("AllowAnonymous is ignored once a Lookup exists", func(t *testing.T) {
		opts := Options{
			AllowAnonymous: true,
			Lookup: func(ctx iris.Context, key string) (string, bool, error) {
				return "", false, nil
			},
		}

		if got := serve(t, opts, "", ""); got != http.StatusUnauthorized {
			t.Fatalf("status = %d without a key, want 401", got)
		}
	})

	t.Run("blank key values are dropped", func(t *testing.T) {
		opts := Options{Keys: map[string]string{"broken": ""}}
		if got := serve(t, opts, "Authorization", "Bearer "); got != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", got)
		}
	})
}

// The principal is the map key, so a handler and an access log can say which
// consumer called.
func TestGetPrincipal(t *testing.T) {
	e := httptest.New(t, newApp(t, keyed()))
	e.GET("/guarded").WithHeader("Authorization", "Bearer "+secret).
		Expect().Status(httptest.StatusOK).Body().IsEqual("iris-backend")
}

// Downstream auth-style code reads ctx.User(); the middleware fills a
// SimpleUser so that it sees one.
func TestSimpleUserIsSet(t *testing.T) {
	app := iris.New()
	app.Logger().SetLevel("disable")

	app.Get("/user", New(keyed()), func(ctx iris.Context) {
		u := ctx.User()
		if u == nil {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		id, _ := u.GetID()
		username, _ := u.GetUsername()
		ctx.WriteString(id + "/" + username)
	})

	// An existing user is not replaced.
	app.Get("/existing", func(ctx iris.Context) {
		ctx.SetUser(&iris.SimpleUser{ID: "already", Username: "already"})
		ctx.Next()
	}, New(keyed()), func(ctx iris.Context) {
		id, _ := ctx.User().GetID()
		ctx.WriteString(id)
	})

	e := httptest.New(t, app)
	e.GET("/user").WithHeader("Authorization", "Bearer "+secret).
		Expect().Status(httptest.StatusOK).Body().IsEqual("iris-backend/iris-backend")
	e.GET("/existing").WithHeader("Authorization", "Bearer "+secret).
		Expect().Status(httptest.StatusOK).Body().IsEqual("already")
}

func TestQueryParameter(t *testing.T) {
	opts := keyed()
	opts.Query = "api_key"

	e := httptest.New(t, newApp(t, opts))

	e.GET("/guarded").WithQuery("api_key", secret).
		Expect().Status(httptest.StatusOK).Body().IsEqual("iris-backend")
	e.GET("/guarded").WithQuery("api_key", "nope").
		Expect().Status(httptest.StatusUnauthorized)

	// The header still works and wins when both are present.
	e.GET("/guarded").WithHeader("Authorization", "Bearer "+secret).
		WithQuery("api_key", "nope").Expect().Status(httptest.StatusOK)

	// The query parameter carries a bare key, with no scheme.
	e.GET("/guarded").WithQuery("api_key", "Bearer "+secret).
		Expect().Status(httptest.StatusUnauthorized)

	// It is off unless configured.
	httptest.New(t, newApp(t, keyed())).GET("/guarded").WithQuery("api_key", secret).
		Expect().Status(httptest.StatusUnauthorized)
}

// With Hash set, Keys hold the digest of the secret, so the configuration
// file (or the environment) never holds a usable credential.
func TestHash(t *testing.T) {
	opts := Options{
		Keys: map[string]string{"iris-backend": SHA256(secret)},
		Hash: SHA256,
	}

	if got := serve(t, opts, "Authorization", "Bearer "+secret); got != http.StatusOK {
		t.Fatalf("status = %d with the hashed key configured, want 200", got)
	}
	// The digest itself is not the credential.
	if got := serve(t, opts, "Authorization", "Bearer "+SHA256(secret)); got != http.StatusUnauthorized {
		t.Fatalf("status = %d presenting the digest, want 401", got)
	}
}

func TestLookup(t *testing.T) {
	t.Run("consulted after Keys", func(t *testing.T) {
		opts := keyed()
		opts.Lookup = func(ctx iris.Context, key string) (string, bool, error) {
			if key == "dynamic" {
				return "from-store", true, nil
			}

			return "", false, nil
		}

		e := httptest.New(t, newApp(t, opts))
		// A static key still wins without touching the store.
		e.GET("/guarded").WithHeader("Authorization", "Bearer "+secret).
			Expect().Status(httptest.StatusOK).Body().IsEqual("iris-backend")
		e.GET("/guarded").WithHeader("Authorization", "Bearer dynamic").
			Expect().Status(httptest.StatusOK).Body().IsEqual("from-store")
		e.GET("/guarded").WithHeader("Authorization", "Bearer nope").
			Expect().Status(httptest.StatusUnauthorized)
	})

	t.Run("an error is a 500, not a 401", func(t *testing.T) {
		opts := Options{
			Lookup: func(ctx iris.Context, key string) (string, bool, error) {
				return "", false, fmt.Errorf("the store is down")
			},
		}

		if got := serve(t, opts, "Authorization", "Bearer anything"); got != http.StatusInternalServerError {
			t.Fatalf("status = %d on a Lookup error, want 500", got)
		}
	})

	t.Run("Lookup alone is enough to guard", func(t *testing.T) {
		opts := Options{
			Lookup: func(ctx iris.Context, key string) (string, bool, error) {
				return "only-store", key == "dynamic", nil
			},
		}

		if got := serve(t, opts, "Authorization", "Bearer dynamic"); got != http.StatusOK {
			t.Fatalf("status = %d with no Keys and a matching Lookup, want 200", got)
		}
		if got := serve(t, opts, "", ""); got != http.StatusUnauthorized {
			t.Fatalf("status = %d with no credentials, want 401", got)
		}
	})
}

// The sentinels tell a custom handler which of the two failures happened.
func TestErrorHandlerSentinels(t *testing.T) {
	var got []error

	opts := keyed()
	opts.ErrorHandler = func(ctx iris.Context, err error) {
		got = append(got, err)
		errors.PermissionDenied.Err(ctx, err)
	}

	e := httptest.New(t, newApp(t, opts))
	e.GET("/guarded").Expect().Status(httptest.StatusForbidden)
	e.GET("/guarded").WithHeader("Authorization", "Bearer nope").
		Expect().Status(httptest.StatusForbidden)

	if len(got) != 2 {
		t.Fatalf("the error handler ran %d times, want 2", len(got))
	}
	if !stdErrors.Is(got[0], ErrMissing) {
		t.Fatalf("first error = %v, want ErrMissing", got[0])
	}
	if !stdErrors.Is(got[1], ErrInvalid) {
		t.Fatalf("second error = %v, want ErrInvalid", got[1])
	}
}

// New has no application to log through, so the default-deny warning is
// logged on the first request. It must be logged, and only once, or an
// operator either never learns the routes are locked down or drowns in it.
func TestUnconfiguredWarningIsLoggedOnce(t *testing.T) {
	for name, opts := range map[string]Options{
		"default deny":    {},
		"allow anonymous": {AllowAnonymous: true},
	} {
		t.Run(name, func(t *testing.T) {
			app := iris.New()

			var logged bytes.Buffer

			app.Get("/guarded", New(opts), func(ctx iris.Context) { ctx.WriteString("ok") })

			// httptest.New resets the level, so configure the logger through
			// it and capture the output afterwards.
			e := httptest.New(t, app, httptest.LogLevel("warn"))
			app.Logger().SetOutput(&logged)

			e.GET("/guarded").Expect()
			e.GET("/guarded").Expect()

			if got := strings.Count(logged.String(), "apikey:"); got != 1 {
				t.Fatalf("the warning was logged %d times over two requests, want 1: %s", got, logged.String())
			}
		})
	}
}

func TestStripScheme(t *testing.T) {
	cases := []struct {
		raw    string
		scheme string
		want   string
		ok     bool
	}{
		{"Bearer abc", "Bearer", "abc", true},
		{"bearer abc", "Bearer", "abc", true},
		{"Bearer   abc", "Bearer", "abc", true},
		{"Bearer abc ", "Bearer", "abc ", true}, // HTTP strips this before we see it
		{"Bearer", "Bearer", "", false},
		{"Bearer ", "Bearer", "", false},
		{"Bearerabc", "Bearer", "", false},
		{"Basic abc", "Bearer", "", false},
		{"abc", "Bearer", "", false},
		{"", "Bearer", "", false},
		{"abc", "", "abc", true},
		{"", "", "", false},
		{"Bearer abc", "", "Bearer abc", true},
	}

	for _, c := range cases {
		t.Run(c.raw+"/"+c.scheme, func(t *testing.T) {
			got, ok := stripScheme(c.raw, c.scheme)
			if got != c.want || ok != c.ok {
				t.Fatalf("stripScheme(%q, %q) = %q, %v; want %q, %v", c.raw, c.scheme, got, ok, c.want, c.ok)
			}
		})
	}
}
