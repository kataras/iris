package ipaccess_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/ipaccess"
)

func newApp(t *testing.T) *iris.Application {
	app := iris.New(iris.Defaults(func(opts *iris.Options) {
		// Enable the X-Forwarded-For header for Context.Addr(), which the middleware uses to get the client's IP address.
		opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
	}))

	allowedPatterns := []string{
		"127.0.0.1",
		"192.168.1.0/24",
		`^10\.0\.\d{1,3}\.\d{1,3}$`,
	}
	blockedPatterns := []string{
		"10.0.0.1",
		`^172\.16\.\d{1,3}\.\d{1,3}$`,
	}

	ipAccessControlMiddleware, err := ipaccess.New(allowedPatterns, blockedPatterns)
	if err != nil {
		t.Fatalf("failed to create IP access control middleware: %v", err)
	}
	app.Use(ipAccessControlMiddleware)

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	return app
}

func TestNew(t *testing.T) {
	app := newApp(t)
	e := httptest.New(t, app)

	// Test allowed IPs.
	e.GET("/").WithHeader("X-Forwarded-For", "127.0.0.1").Expect().Status(httptest.StatusOK).
		Body().IsEqual("Hello, World!")
	e.GET("/").WithHeader("X-Forwarded-For", "192.168.1.1").Expect().Status(httptest.StatusOK).
		Body().IsEqual("Hello, World!")
	e.GET("/").WithHeader("X-Forwarded-For", "10.0.1.1").Expect().Status(httptest.StatusOK).
		Body().IsEqual("Hello, World!")

	// Test blocked IPs.
	e.GET("/").WithHeader("X-Forwarded-For", "10.0.0.1").Expect().Status(httptest.StatusForbidden)
	e.GET("/").WithHeader("X-Forwarded-For", "172.16.0.1").Expect().Status(httptest.StatusForbidden)

	// Test IPs not in allowed patterns.
	e.GET("/").WithHeader("X-Forwarded-For", "192.168.2.1").Expect().Status(httptest.StatusForbidden)
}

// A glob-looking pattern is not an IP and not a CIDR, so it falls through to the
// regexp branch. It must be anchored: an unanchored match accepts any address
// whose text merely contains the pattern.
func TestGlobPatternIsAnchored(t *testing.T) {
	app := iris.New(iris.Defaults(func(opts *iris.Options) {
		opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
	}))

	mw, err := ipaccess.New([]string{`10\.0\.0\.\d{1,3}`}, nil)
	if err != nil {
		t.Fatal(err)
	}
	app.Use(mw)
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	e.GET("/").WithHeader("X-Forwarded-For", "10.0.0.7").Expect().Status(httptest.StatusOK)

	// The bypass: a public address that contains the allowed pattern.
	e.GET("/").WithHeader("X-Forwarded-For", "110.0.0.99").Expect().Status(httptest.StatusForbidden)
	e.GET("/").WithHeader("X-Forwarded-For", "210.0.0.5").Expect().Status(httptest.StatusForbidden)
}

// An empty allowlist means "no allowlist", not "deny everything": New(nil, blocked)
// is the natural way to express a blocklist-only policy and used to 403 every
// single request.
func TestEmptyAllowlistIsBlocklistOnly(t *testing.T) {
	app := iris.New(iris.Defaults(func(opts *iris.Options) {
		opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
	}))

	mw, err := ipaccess.New(nil, []string{"10.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	app.Use(mw)
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	e.GET("/").WithHeader("X-Forwarded-For", "8.8.8.8").Expect().Status(httptest.StatusOK)
	e.GET("/").WithHeader("X-Forwarded-For", "10.0.0.1").Expect().Status(httptest.StatusForbidden)
}

// Exact matching compares parsed addresses, so the two spellings of the IPv6
// loopback are the same address.
func TestExactIPv6FormsAreEquivalent(t *testing.T) {
	app := iris.New(iris.Defaults(func(opts *iris.Options) {
		opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
	}))

	mw, err := ipaccess.New([]string{"::1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	app.Use(mw)
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	e.GET("/").WithHeader("X-Forwarded-For", "::1").Expect().Status(httptest.StatusOK)
	e.GET("/").WithHeader("X-Forwarded-For", "0:0:0:0:0:0:0:1").Expect().Status(httptest.StatusOK)
	e.GET("/").WithHeader("X-Forwarded-For", "::2").Expect().Status(httptest.StatusForbidden)
}

// A prefix pattern is the documented way to cover a whole /24 in a blocklist.
// Anchoring both ends would silently stop it matching anything, so only the
// start is anchored.
func TestPrefixBlocklistPatternStillMatches(t *testing.T) {
	app := iris.New(iris.Defaults(func(opts *iris.Options) {
		opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
	}))

	mw, err := ipaccess.New(nil, []string{`^198\.51\.100\.`})
	if err != nil {
		t.Fatal(err)
	}
	app.Use(mw)
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)

	// Every address in the /24 is blocked.
	e.GET("/").WithHeader("X-Forwarded-For", "198.51.100.5").Expect().Status(httptest.StatusForbidden)
	e.GET("/").WithHeader("X-Forwarded-For", "198.51.100.250").Expect().Status(httptest.StatusForbidden)

	// Neighbouring ranges are not, and neither is an address that merely
	// contains the pattern later in the string.
	e.GET("/").WithHeader("X-Forwarded-For", "198.51.101.5").Expect().Status(httptest.StatusOK)
	e.GET("/").WithHeader("X-Forwarded-For", "8.8.8.8").Expect().Status(httptest.StatusOK)
}

// Must matches rewrite.Must and httpcache.Must: it takes what New takes and
// panics on a malformed pattern, so a middleware registration is one expression
// in a main function.
func TestMust(t *testing.T) {
	app := iris.New(iris.Defaults(func(opts *iris.Options) {
		opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
	}))
	app.Use(ipaccess.Must([]string{"127.0.0.1"}, nil))
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)
	e.GET("/").WithHeader("X-Forwarded-For", "127.0.0.1").Expect().Status(httptest.StatusOK)
	e.GET("/").WithHeader("X-Forwarded-For", "10.0.0.9").Expect().Status(httptest.StatusForbidden)
}

func TestMustPanicsOnAMalformedPattern(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Must returned instead of panicking on a pattern that New rejects")
		}
	}()

	ipaccess.Must([]string{"["}, nil)
}
