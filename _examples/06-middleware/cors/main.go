// CORS: allowed origins, answered preflights and a JSON body for rejected
// origins.
//
// Cross-Origin Resource Sharing is the browser mechanism that decides whether
// a page served from one origin may call an API on another. The cors
// middleware stamps the Access-Control-* response headers and answers
// preflights; it is configured through a fluent chain that ends in .Handler().
//
// Register it with UseRouter, not Use. A preflight is an automatic OPTIONS
// request the browser sends before certain cross-origin calls, and it usually
// matches no registered route - a per-route registration would never see it,
// the request would 404 first. UseRouter runs before routing, so the preflight
// is answered with 204 plus Access-Control-Allow-Methods, -Allow-Headers and
// -Max-Age either way. Max-Age tells the browser how long it may reuse that
// answer; it defaults to 86400 seconds (24 hours).
//
// There is no AllowMethods method on the chain: Access-Control-Allow-Methods
// is always "*", and route registration decides which verbs actually exist.
//
// When testing by hand: a request WITHOUT an Origin header is treated as
// origin "*", which does not match a specific allow-list, so plain curl gets
// the 403. Always pass -H "Origin: ...", or chain SkipLocalRequests(true)
// while developing locally. CORS also only constrains browsers - curl and
// other non-browser clients ignore the response headers entirely.
//
// Learn: configure the cors chain, register it with UseRouter so preflights are
// answered, and test it with an Origin header.
// Before: ../error-middleware
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X OPTIONS -H "Origin: https://app.example.com" \
//	    -H "Access-Control-Request-Method: POST" \
//	    -H "Access-Control-Request-Headers: Content-Type" \
//	    http://localhost:8080/api/items                                            # 204 preflight, Allow-Methods / Allow-Headers / Max-Age
//	curl -i -H "Origin: https://app.example.com" http://localhost:8080/api/items   # 200, Access-Control-Allow-Origin
//	curl -i -H "Origin: https://evil.example" http://localhost:8080/api/items      # 403 PERMISSION_DENIED from HandleErrorFunc
//
// Next: ../rate-limit
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	crs := cors.New().
		// One comma-separated line. AllowOrigins takes a list instead and
		// supports wildcards: "https://*.example.com", "https://www.example.*".
		// For dynamic decisions use AllowOriginMatcherFunc(func(origin string)
		// bool), the context-aware AllowOriginFunc, or AllowOriginRegex.
		AllowOrigin("https://example.com,https://app.example.com").
		// Request headers the browser is allowed to send.
		AllowHeaders("Content-Type", "Authorization").
		// Response headers browser JavaScript is allowed to read.
		ExposeHeaders("X-Total-Count").
		ReferrerPolicy(cors.StrictOriginWhenCrossOrigin).
		// Rejections render as the standard rest/errors JSON body instead of
		// the default bare 403.
		HandleErrorFunc(func(ctx iris.Context, err error) {
			errors.PermissionDenied.Err(ctx, err)
		}).
		Handler()

	// Before routing, so preflights never reach the 404 chain.
	app.UseRouter(crs)

	// The same chain fits iris.NewBuilder through its Cors step, which takes
	// the CORS value instead of an origin line:
	//
	//	iris.NewBuilder().
	//	    Prefix("/api").
	//	    Cors(cors.New().
	//	        HandleErrorFunc(errors.FailedPrecondition.Err).
	//	        AllowOrigin("https://app.example.com").
	//	        AllowCredentials()).
	//	    Compression(true).
	//	    // ...
	//
	// AllowOrigin("https://app.example.com") on the builder is the shorthand
	// for the same call with the FailedPrecondition error handler and nothing
	// else set. Cors uses your value as given, so set HandleErrorFunc yourself
	// or rejections render as a bare 403.

	app.Get("/api/items", func(ctx iris.Context) {
		ctx.SetHeader("X-Total-Count", "2")
		ctx.JSON([]string{"pen", "notebook"})
	})

	app.Post("/api/items", func(ctx iris.Context) {
		ctx.Created(iris.Map{"created": true}, nil)
	})

	app.Listen(":8080")
}

// Other methods of the chain, for reference:
//
//	AllowOrigins("https://a.com", "https://*.b.com")  a list, with wildcards
//	AllowOriginRegex(`^https://.*\.example\.com$`)    regular expressions
//	AllowCredentials()                                see the note below
//	DisallowCredentials()                             no credentialed requests
//	ExtractOriginFunc(cors.StrictOriginExtractor)     reject a missing Origin
//	SkipLocalRequests(true)                           no checks for localhost
//
// On AllowCredentials: browsers reject Access-Control-Allow-Credentials when
// the allowed origin is the wildcard "*", so the middleware suppresses that
// header in the wildcard case unless AllowCredentials was called explicitly.
// A request carrying an Origin gets that origin echoed back rather than "*",
// which is every browser request, so credentials are advertised as usual
// there. The wildcard case is a request with no Origin header at all.
