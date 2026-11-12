// API keys: authenticating a service, a cron job or a webhook sender.
//
// apikey.New(apikey.Options{...}) returns a plain iris.Handler, so it guards
// exactly the router or the route it is mounted on. The key arrives on
// "Authorization: Bearer <key>" by default, or on a header of your own when
// the caller cannot set that one. It is compared against every configured key
// in constant time, then handed to Lookup when nothing matched, and the
// principal that won is stored on the Context for apikey.Get and for
// ctx.User().
//
// The default is to deny: a middleware with no Keys and no Lookup rejects
// every request and logs a warning, rather than quietly letting everyone
// through when a secret is missing from the environment.
//
// The three groups below are the three shapes worth knowing:
//
//	/reports   Authorization: Bearer, secrets in the configuration
//	/webhooks  X-API-Key with apikey.NoScheme, a bare value on a custom header
//	/partners  Options.Hash with apikey.SHA256: the configuration holds only
//	           digests, and Lookup resolves the keys that live in a database
//
// Secrets are inline here so the curl lines run as they are. In a real service
// they come from the environment, and with Hash set what the deployment holds
// is the digest, never a usable credential.
//
// The 401 body is the rest/errors envelope every other rejection in the
// framework uses ("id" is the tracking id written to the log too):
//
//	{"id":"98db6518-4156-49a2-9e26-8fc4b0afdf18",
//	 "http_error_code":{"canonical_name":"UNAUTHENTICATED","status":401},
//	 "message":"apikey: the presented key matched nothing: apikey: credentials are invalid",
//	 "data":[{"message":"apikey: credentials are invalid"}]}
//
// A request with no key at all reports apikey.ErrMissing instead, so a custom
// ErrorHandler can tell the two apart with errors.Is and log them differently.
//
// Learn: guard a router with API keys from the configuration, a custom header
// or a database Lookup, storing only digests on the server.
// Before: ../basicauth
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -H "Authorization: Bearer reporting-key" http://localhost:8080/reports/daily   # 200
//	curl -i -X POST -H "X-API-Key: stripe-key" http://localhost:8080/webhooks/stripe       # 200
//	curl -i -H "Authorization: Bearer partner-key" http://localhost:8080/partners/quota    # 200, matched a digest
//	curl -i -H "Authorization: Bearer rotated-key" http://localhost:8080/partners/quota    # 200, resolved by Lookup
//	curl -i -H "Authorization: Bearer wrong" http://localhost:8080/reports/daily           # 401 UNAUTHENTICATED
//	curl -i http://localhost:8080/reports/daily                                            # 401 UNAUTHENTICATED, apikey.ErrMissing
//
// Next: ../jwt-middleware
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/apikey"
)

// partnerKeys stands in for the table a real service issues keys from: the
// digest of the key, mapped to the partner it belongs to. Nothing here can be
// presented to the API, which is the point of storing digests.
var partnerKeys = map[string]string{
	// apikey.SHA256("rotated-key")
	"5be3b05a2339aecdb0c543afefc6a563085f5861e6c109dce236363b3319566e": "acme",
}

func main() {
	app := iris.New()

	// 1. The default presentation: Authorization: Bearer <key>. Infrastructure
	// (proxies, CDNs, log scrubbers) redacts that header out of the box, which
	// a custom one does not get, so prefer it whenever the caller can set it.
	// One entry per consumer, so a leak can be traced and one key revoked
	// without breaking the others.
	reports := app.Router("/reports", apikey.New(apikey.Options{
		Keys: map[string]string{
			"reporting": "reporting-key",
			"ops":       "ops-key",
		},
	}))

	reports.Get("/daily", func(ctx iris.Context) {
		// Get names the principal whose key matched. The second result is
		// false only when the request did not go through the middleware at
		// all, so below a guard the name is always there.
		principal, _ := apikey.Get(ctx)

		ctx.JSON(iris.Map{"called_by": principal, "orders": 128})
	})

	// 2. A webhook sender that can only send a fixed custom header. NoScheme
	// is a sentinel rather than the empty string, so leaving Scheme out still
	// means the Bearer default.
	webhooks := app.Router("/webhooks", apikey.New(apikey.Options{
		Header: "X-API-Key",
		Scheme: apikey.NoScheme,
		Keys:   map[string]string{"stripe": "stripe-key"},
	}))

	webhooks.Post("/stripe", handleWebhook)

	// 3. Hashed keys. Keys hold apikey.SHA256(secret) instead of the secret,
	// and Lookup covers what the process did not start with. A shared secret
	// is a high-entropy random string, not a password, so a plain digest is
	// the right hash here: there is nothing to brute-force.
	partners := app.Router("/partners", apikey.New(apikey.Options{
		Hash: apikey.SHA256,
		Keys: map[string]string{
			// apikey.SHA256("partner-key")
			"globex": "346e50af211b5135824bb2bb58fe0f9e6df228adcf10c58a37fbc46b57baee74",
		},
		Lookup: lookupPartner,
	}))

	partners.Get("/quota", func(ctx iris.Context) {
		principal, _ := apikey.Get(ctx)

		// ctx.User() is filled by the middleware when nothing else did, so an
		// access log or a handler written against the auth SDK sees the
		// caller without knowing which middleware authenticated it.
		method, _ := ctx.User().GetAuthorization()

		ctx.JSON(iris.Map{"partner": principal, "authorization": method, "remaining": 4_000})
	})

	app.Listen(":8080")
}

func handleWebhook(ctx iris.Context) {
	principal, _ := apikey.Get(ctx)

	ctx.JSON(iris.Map{"sender": principal, "received": true})
}

// lookupPartner resolves a key none of the configured Keys matched. With Hash
// set it receives the digest and never the raw key, so a store of digests
// compares directly and must not hash again.
//
// A returned error is a 500, not a 401: a store that is down must not read as
// an invalid key and send a partner off to rotate a credential that works.
func lookupPartner(ctx iris.Context, key string) (string, bool, error) {
	principal, ok := partnerKeys[key]
	return principal, ok, nil
}
