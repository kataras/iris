package main

import (
	_ "github.com/kataras/iris/_examples/routing/subdomains/redirect/multi-instances/other"
	_ "github.com/kataras/iris/_examples/routing/subdomains/redirect/multi-instances/root"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/apps"
)

// In this example, you wanna use three different applications exposed as one.
// The first one is the "other" package, the second is the "root",
// the third is the switcher application which will expose the above.
// Unlike the previous example, on this one we will NOT redirect,
// the Hosts switcher will just pass the request to the matched Application to handle.
// This is NOT an alternative of your favorite load balancer.
// Read the comments carefully, if you need more information
// you can head over to the "apps" package's godocs and tests.
func main() {
	// The `apps.Hosts` switch provider:
	// The pattern. A regexp for matching the host part of incoming requests.
	// The target. An iris.Application instance (created by iris.New())
	//      OR
	// You can use the Application's name (app.SetName("myapp")).
	// Example:
	// package rootdomain
	// func init() {
	//  app := iris.New().SetName("root app")
	//  ...
	// }
	// On the main package add an empty import statement: ` _ import "rootdomain"`
	// And set the "root app" as the key to reference that application (of the same program).
	// Thats the target we wanna use now ^ (see ../hosts file).
	//      OR
	// An external host or a local running in the same machine but different port or host behind proxy.
	switcher := apps.Switch(apps.Hosts{
		{"^(www.)?mydomain.com$", "root app"},
		{"^otherdomain.com$", "other app"},
	})
	// The registration order matters, so we can register a fallback server (when host no match)
	// using "*". However, you have alternatives by using the Switch Iris Application value
	// (let's call it "switcher"):
	// 1. Handle the not founds, e.g. switcher.OnErrorCode(404, ...)
	// 2. Use the switcher.WrapRouter, e.g. to log the flow of a request of all hosts exposed.
	// 3. Just register routes to match, e.g. switcher.Get("/", ...)
	switcher.Get("/", fallback)
	// OR
	// Change the response code to 502
	// instead of 404 and write a message:
	// switcher.OnErrorCode(iris.StatusNotFound, fallback)

	// The switcher is a common Iris Application, so you have access to the Iris features.
	// And it should be listening to a host:port in order to match and serve its apps.
	//
	// http://mydomain.com (OK)
	// http://www.mydomain.com (OK)
	// http://mydomain.com/dsa (404)
	// http://no.mydomain.com (502 Bad Host)
	//
	// http://otherdomain.com (OK)
	// http://www.otherdomain.com (502 Bad Host)
	// http://otherdomain.com/dsa (404 JSON)
	// ...
	switcher.Listen(":80")
}

func fallback(ctx iris.Context) {
	ctx.StatusCode(iris.StatusBadGateway)
	ctx.Writef("Bad Host %s", ctx.Host())
}
