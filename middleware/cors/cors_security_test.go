package cors_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/cors"
)

func newCORSApp(c *cors.CORS) *iris.Application {
	app := iris.New()
	app.UseRouter(c.Handler())
	app.Get("/", func(ctx iris.Context) { ctx.Text("ok") })
	return app
}

// TestCORSReflectedOriginWithCredentialsIsTheDefault pins the DEFAULT behaviour
// of cors.New() as known and deliberate, not as correct.
//
// cors.New() reflects any Origin back and advertises credentials for it, which
// lets a page on any domain read a logged-in user's response. The default is
// kept for backwards compatibility; cors.Strict() is the safe entry point and is
// covered by the next test. If this test ever starts failing because the default
// changed, that is a deliberate breaking change - update it, do not "fix" the
// middleware back.
func TestCORSReflectedOriginWithCredentialsIsTheDefault(t *testing.T) {
	e := httptest.New(t, newCORSApp(cors.New()))

	resp := e.GET("/").WithHeader("Origin", "https://evil.com").Expect().Status(httptest.StatusOK)
	resp.Header("Access-Control-Allow-Origin").IsEqual("https://evil.com")
	resp.Header("Access-Control-Allow-Credentials").IsEqual("true")
}

// Strict() keeps everything New() does except that credentials are advertised
// only when the developer asks for them.
func TestCORSStrictSuppressesCredentials(t *testing.T) {
	e := httptest.New(t, newCORSApp(cors.Strict()))

	resp := e.GET("/").WithHeader("Origin", "https://evil.com").Expect().Status(httptest.StatusOK)
	resp.Header("Access-Control-Allow-Origin").IsEqual("https://evil.com")
	resp.Header("Access-Control-Allow-Credentials").IsEqual("false")

	// ...and it can still be opted into.
	e2 := httptest.New(t, newCORSApp(cors.Strict().AllowOrigin("https://app.example.com").AllowCredentials()))

	resp2 := e2.GET("/").WithHeader("Origin", "https://app.example.com").Expect().Status(httptest.StatusOK)
	resp2.Header("Access-Control-Allow-Origin").IsEqual("https://app.example.com")
	resp2.Header("Access-Control-Allow-Credentials").IsEqual("true")
}

// An allowlist used to turn every request WITHOUT an Origin header - plain
// browser navigations, health checks, curl - into a 403, because the empty
// origin was substituted with "*" and then matched against the allowlist, which
// never contains "*".
func TestCORSAllowlistDoesNotBlockRequestsWithoutOrigin(t *testing.T) {
	e := httptest.New(t, newCORSApp(cors.New().AllowOrigin("https://app.example.com")))

	// No Origin header at all: not a CORS request, must pass through.
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")

	// An allowed origin still passes.
	e.GET("/").WithHeader("Origin", "https://app.example.com").
		Expect().Status(httptest.StatusOK).Body().IsEqual("ok")

	// A disallowed origin is still rejected.
	e.GET("/").WithHeader("Origin", "https://evil.com").
		Expect().Status(httptest.StatusForbidden)
}

// StrictOriginExtractor is the opt-in that rejects requests without an Origin.
func TestCORSStrictOriginExtractorRejectsMissingOrigin(t *testing.T) {
	e := httptest.New(t, newCORSApp(cors.New().ExtractOriginFunc(cors.StrictOriginExtractor)))

	e.GET("/").Expect().Status(httptest.StatusForbidden)
	e.GET("/").WithHeader("Origin", "https://example.com").Expect().Status(httptest.StatusOK)
}

// A trailing "*" in a TLD position must match one label, not an arbitrary
// suffix, otherwise an attacker registers evil.com and serves
// www.example.evil.com.
func TestCORSWildcardTLDPatternDoesNotMatchArbitraryDomain(t *testing.T) {
	e := httptest.New(t, newCORSApp(cors.New().AllowOrigin("https://www.example.*")))

	e.GET("/").WithHeader("Origin", "https://www.example.com").
		Expect().Status(httptest.StatusOK)
	e.GET("/").WithHeader("Origin", "https://www.example.org").
		Expect().Status(httptest.StatusOK)

	e.GET("/").WithHeader("Origin", "https://www.example.evil.com").
		Expect().Status(httptest.StatusForbidden)
}

// Access-Control-Max-Age is delta-seconds: digits only. "8.64E+04" is discarded
// by browsers, which then preflight every single request.
func TestCORSMaxAgeIsDeltaSeconds(t *testing.T) {
	e := httptest.New(t, newCORSApp(cors.New().MaxAge(24*time.Hour)))

	e.OPTIONS("/").WithHeader("Origin", "https://example.com").
		Expect().Status(httptest.StatusNoContent).
		Header("Access-Control-Max-Age").IsEqual("86400")
}

// A literal "*" is not a wildcard for a credentialed preflight, so the requested
// headers are echoed back instead.
func TestCORSPreflightEchoesRequestedHeaders(t *testing.T) {
	e := httptest.New(t, newCORSApp(cors.New()))

	e.OPTIONS("/").
		WithHeader("Origin", "https://example.com").
		WithHeader("Access-Control-Request-Method", "POST").
		WithHeader("Access-Control-Request-Headers", "Authorization, Content-Type").
		Expect().Status(httptest.StatusNoContent).
		Header("Access-Control-Allow-Headers").IsEqual("Authorization, Content-Type")

	// Without the request header, the configured value is used as before.
	e.OPTIONS("/").WithHeader("Origin", "https://example.com").
		Expect().Status(httptest.StatusNoContent).
		Header("Access-Control-Allow-Headers").IsEqual("*")
}
