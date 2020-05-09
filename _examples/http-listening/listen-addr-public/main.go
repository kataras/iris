package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>Hello World!</h1>")

		// Will print the ngrok public domain
		// that your app is using to be served online.
		ctx.Writef("From: %s",
			ctx.Application().ConfigurationReadOnly().GetVHost())
	})

	app.Listen(":8080", iris.WithTunneling, iris.WithLogLevel("debug"))

	/* The full configuration can be set as:
	app.Listen(":8080", iris.WithConfiguration(
		iris.Configuration{
			Tunneling: iris.TunnelingConfiguration{
				AuthToken:    "my-ngrok-auth-client-token",
				Bin:          "/bin/path/for/ngrok",
				Region:       "eu",
				WebInterface: "127.0.0.1:4040",
				Tunnels: []iris.Tunnel{
					{
						Name: "MyApp",
						Addr: ":8080",
					},
				},
			},
		}))
	*/
}
