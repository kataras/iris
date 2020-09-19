/*Package main is a proxy + accesslog example.
In this example we will make a small proxy which listens requests on "/proxy/+path".
With two accesslog instances, one for the main application and one for the /proxy/ requests.
Of cource, you could a single accesslog for the whole application, but for the sake of the example
let's log them separately.

We will make use of iris.StripPrefix and host.ProxyHandler.*/
package main

import (
	"net/url"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/host"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/kataras/iris/v12/middleware/recover"
)

func main() {
	app := iris.New()
	app.Get("/", index)

	ac := accesslog.File("access.log")
	defer ac.Close()
	ac.Async = true
	ac.RequestBody = true
	ac.ResponseBody = true
	ac.BytesReceived = false
	ac.BytesSent = false

	app.UseRouter(ac.Handler)
	app.UseRouter(recover.New())

	proxy := app.Party("/proxy")
	{
		acProxy := accesslog.File("proxy_access.log")
		defer acProxy.Close()
		acProxy.Async = true
		acProxy.RequestBody = true
		acProxy.ResponseBody = true
		acProxy.BytesReceived = false
		acProxy.BytesSent = false

		// Unlike Use, the UseRouter method replaces any duplications automatically.
		// (see UseOnce for the same behavior on Use).
		// Therefore, this statement removes the parent's accesslog and registers this new one.
		proxy.UseRouter(acProxy.Handler)
		proxy.UseRouter(recover.New())
		proxy.Use(func(ctx iris.Context) {
			ctx.CompressReader(true)
			ctx.Next()
		})

		/* Listen for specific proxy paths:
		// Listen on "/proxy" for "http://localhost:9090/read-write"
		proxy.Any("/", iris.StripPrefix("/proxy",
			newProxyHandler("http://localhost:9090/read-write")))
		*/

		// You can register an access log only for proxied requests, e.g. proxy_access.log:
		// proxy.UseRouter(ac2.Handler)

		// Listen for any proxy path.
		// Proxies the "/proxy/+$path" to "http://localhost:9090/$path".
		proxy.Any("/{p:path}", iris.StripPrefix("/proxy",
			newProxyHandler("http://localhost:9090")))
	}

	// $ go run target/main.go
	// open new terminal
	// $ go run main.go
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.WriteString("OK")
}

func newProxyHandler(proxyURL string) iris.Handler {
	target, err := url.Parse(proxyURL)
	if err != nil {
		panic(err)
	}
	reverseProxy := host.ProxyHandler(target)
	return iris.FromStd(reverseProxy)
}
