// Multi-application composition: two independent Iris applications served behind a single listener through apps.Switch.
//
// The switcher matches the request host against each case in registration
// order and forwards the request to the first match, which serves it with its
// own context pool and options. Unmatched hosts fall through to the switcher's
// own router.
//
// Learn: run two applications on one port and pick one by request host, with a shared not-found for everything else.
// Before: ../file-server
//
// Run:
//
//	go run main.go
//
// The Host header picks the application, so no DNS setup is needed.
//
// Try:
//
//	curl -H "Host: api.localhost:8080" http://localhost:8080/status   # 200 {"app":"api","ok":true}
//	curl -H "Host: localhost:8080" http://localhost:8080/             # 200, served by the "website" application
//	curl -i -H "Host: unknown.example" http://localhost:8080/         # 404 from the switcher's own router
//
// Next: ../api-graph
package main

import (
	"log"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/apps"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	website := newWebsite()
	api := newAPI()

	// The registry is process-global: any package can look an application up by
	// its Name, no import cycle needed. apps.GetAll() returns all of them and
	// apps.OnApplicationRegistered fires for each future registration.
	log.Printf("%d applications registered, apps.Get(%q) == api: %v",
		len(apps.GetAll()), "api", apps.Get("api") == api)

	// Patterns are literals or regular expressions matched against the request
	// host (host:port), in registration order. Target accepts an
	// *iris.Application, the name of a registered one, or a plain host string
	// (which produces a permanent redirect to that host instead).
	switcher := apps.Switch(apps.Hosts{
		{Pattern: `^api\.`, Target: api},
		{Pattern: `^localhost`, Target: website},
	})

	// The switcher is an ordinary application: routes registered on it run only
	// when none of the cases above matched.
	switcher.Get("/", func(ctx iris.Context) {
		// The error code carries the 404, so there is no status code to set:
		// an unmatched host answers with the same envelope as any other
		// not-found.
		errors.NotFound.Messagef(ctx, "no application is responsible for host %q", ctx.Host())
	})

	// apps.SetHost("example.com") would rewrite the request host before the
	// matched application sees it, e.g. to serve www. and the root domain from
	// one application without a redirect.
	switcher.Listen(":8080")
}

// newWebsite returns the public site. The Name option also registers the
// application under that name in the process-wide registry, so any package can
// retrieve it with apps.Get("website") without an import cycle.
func newWebsite() *iris.Application {
	app := iris.New(iris.Options{Name: "website"})

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("served by the %q application\n", ctx.Application().String())
	})

	return app
}

// newAPI returns the JSON API application.
func newAPI() *iris.Application {
	app := iris.New(iris.Options{Name: "api"})

	app.Get("/status", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"app": ctx.Application().String(), "ok": true})
	})

	return app
}
