// Standard library interop: net/http handlers inside Iris, and the Iris application inside a plain http.Server.
//
// Iris converts in both directions.
//
// iris.FromStd turns anything net/http-shaped into an iris.Handler: an
// http.Handler value, a func(w, r) or a middleware-shaped
// func(w, r, next http.HandlerFunc). That is how packages like
// net/http/pprof, expvar or promhttp mount on an Iris route.
//
// In the other direction, *iris.Application implements http.Handler once
// Build has run, so it drops into any standard server, test harness or
// third-party listener. app.Listen and app.Run call Build for you; call it
// yourself only when handing the application to infrastructure you own,
// as below.
//
// The hand-written key check at the bottom is here only to have a piece of
// net/http middleware to adapt; a real API key belongs to middleware/apikey,
// which compares in constant time and renders the framework's 401 envelope.
//
// Learn: mount any net/http handler or middleware on an Iris route, and serve the Iris application from your own http.Server.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/                                   # 200 {"served_by":"net/http","framework":"iris"}
//	curl http://localhost:8080/healthz                            # 200 ok
//	curl http://localhost:8080/debug/vars                         # 200, expvar JSON
//	curl -i http://localhost:8080/guarded                         # 401 from the net/http middleware
//	curl -i -H "X-Api-Key: k3y" http://localhost:8080/guarded     # 200 {"secret":42}
//
// Next: ../typed-context
package main

import (
	"expvar"
	"log"
	"net/http"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"served_by": "net/http", "framework": "iris"})
	})

	// An http.Handler from the standard library, mounted on an Iris route.
	app.Get("/debug/vars", iris.FromStd(expvar.Handler()))

	// A plain func(w, r) works too.
	app.Get("/healthz", iris.FromStd(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("ok"))
	}))

	// Middleware-shaped net/http code: calling next continues the Iris chain,
	// skipping it stops the request here.
	app.Get("/guarded", iris.FromStd(requireAPIKey), func(ctx iris.Context) {
		ctx.JSON(iris.Map{"secret": 42})
	})

	// Build wires the router and the rest of the application's parts.
	if err := app.Build(); err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: app, // *iris.Application implements http.Handler.
	}

	log.Println("listening on http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}

// requireAPIKey is ordinary net/http middleware.
func requireAPIKey(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Header.Get("X-Api-Key") != "k3y" {
		http.Error(w, "missing or invalid X-Api-Key header", http.StatusUnauthorized)
		return
	}

	next(w, r)
}
