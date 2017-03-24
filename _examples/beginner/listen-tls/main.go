package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

const host = "127.0.0.1:443"

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())

	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from the SECURE server")
	})

	app.Get("/mypath", func(ctx *iris.Context) {
		ctx.Writef("Hello from the SECURE server on path /mypath")
	})

	// start a secondary server (HTTP) on port 80, this is a non-blocking func
	// redirects all http to the main server which is tls/ssl on port :443

	iris.Proxy(":80", "https://"+host)
	// start the MAIN server (HTTPS) on port 443, this is a blocking func
	app.ListenTLS(host, "mycert.cert", "mykey.key")

	// now if you navigate to http://127.0.0.1/mypath it will
	// send you back to https://127.0.0.1:443/mypath (https://127.0.0.1/mypath)
	//
	// go to the listen-letsencrypt example to view how you can integrate your server
	// to get automatic certification and key from the letsencrypt.org 's servers.
}
