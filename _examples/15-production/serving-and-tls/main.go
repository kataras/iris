// Serving: app.Listen for the plain case, app.Run(iris.ServerOptions{...}) when you need TLS, Let's Encrypt or server tuning.
//
// app.Listen(addr) is app.Run with a small set of defaults filled in
// (tcp network, keep-alive, socket sharding, the application's log writer).
// Everything else (certificates, timeouts, tunnels) is a field of
// iris.ServerOptions.
//
// The mode is chosen by an environment variable so this file compiles and
// runs plain HTTP with no certificates present:
//
//	IRIS_SERVE_MODE=plain        (default) HTTP on :8080 via app.Listen
//	IRIS_SERVE_MODE=options      HTTP on :8080 via app.Run, so the options
//	                             struct is visible
//	IRIS_SERVE_MODE=tls          HTTPS on :443 with TLS_CERT_FILE/TLS_KEY_FILE
//	IRIS_SERVE_MODE=letsencrypt  HTTPS on :443, certificates issued and
//	                             renewed automatically for TLS_HOSTS
//
// Learn: choose between app.Listen and app.Run(iris.ServerOptions{...}), and turn on TLS from PEM files or from Let's Encrypt.
//
// Run:
//
//	go run main.go
//	IRIS_SERVE_MODE=options go run main.go
//
// Try:
//
//	curl http://localhost:8080/   # 200 {"scheme":"http://","host":"localhost:8080"}
//
// Next: ../graceful-shutdown
package main

import (
	"os"
	"strings"
	"time"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"scheme": ctx.Scheme(), // "http://" or "https://"
			"host":   ctx.Host(),
		})
	})

	serve(app)
}

func serve(app *iris.Application) {
	switch os.Getenv("IRIS_SERVE_MODE") {
	case "options":
		// The explicit form of the default. Timeout, Network, KeepAlive,
		// SocketSharding and the log writer are all yours to set.
		app.Run(iris.ServerOptions{
			Addr:      ":8080",
			Network:   "tcp",
			KeepAlive: 3 * time.Minute,
		})

	case "tls":
		// Manual certificates: two PEM files on disk. Terminating TLS in the
		// application is the right choice when nothing else fronts it.
		app.Run(iris.ServerOptions{
			Addr:     ":443",
			CertFile: envOr("TLS_CERT_FILE", "cert.pem"),
			KeyFile:  envOr("TLS_KEY_FILE", "key.pem"),
		})

	case "letsencrypt":
		// Automatic certificates from Let's Encrypt. Hosts is the list the
		// certificate is valid for; the first entry is treated as the root
		// domain. The challenge is tls-alpn-01, answered by this TLS
		// listener: Let's Encrypt connects to port 443 of each host, so that
		// port must reach this server from the internet, and port 80 is not
		// needed. CertFile and KeyFile, when also given, are used as a local
		// fallback if issuance fails.
		hosts := strings.Split(envOr("TLS_HOSTS", "example.com"), ",")
		app.Run(iris.ServerOptions{
			Addr:        ":443",
			Letsencrypt: true,
			Hosts:       hosts,
		})

	default:
		// Plain HTTP with graceful shutdown on SIGINT/SIGTERM.
		app.Listen(":8080")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
