package main

import (
	"net/http"

	"github.com/kataras/iris/v12"

	"golang.org/x/net/http2"
)

/*
	Serve HTTP/2 without TLS and keep support for HTTP/1.1
*/

// $ go get golang.org/x/net/http2
// # Take a look at the golang.org/x/net/http2/h2c package as well,
// # you may want to use it.
// $ go run main.go
// $ brew install curl-openssl
// # Add curl-openssl to the front of your path.
// Test with the following commands:
// $ curl -v --http2 http://localhost:8080
// $ curl -v --http1.1 http://localhost:8080
func main() {
	// Initialize Iris Application.
	app := iris.New()

	// Build the API.
	app.Any("/", index)

	// Finally, listen and serve on port 8080 using h2c.
	app.Run(iris.Raw(func() error {
		host := app.NewHost(&http.Server{
			Addr: ":8080",
		})

		err := http2.ConfigureServer(host.Server, &http2.Server{
			MaxConcurrentStreams:         250,
			PermitProhibitedCipherSuites: true,
		})

		if err != nil {
			return err
		}

		return host.ListenAndServe()
	}))
}

func index(ctx iris.Context) {
	ctx.WriteString("Hello, World!\n")
}
