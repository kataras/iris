// HandlerFunc: dependency injection for plain handlers and middleware.
//
// Controllers are not the only consumers of the container. app.HandlerFunc takes
// a function whose parameters are registered dependencies and which returns an
// iris.Handler. The outer function runs once, at startup, with the arguments
// injected; the handler it returns serves the requests. app.HandlerFuncs maps a
// whole list at once, and the builder's Middlewares/RouterMiddlewares steps
// accept the same shape mixed with plain handlers.
//
// It panics at startup on an unresolvable parameter, a nil result or a result
// that is not a handler - the graph is verified before the port is bound.
//
// Learn: inject registered dependencies into plain handlers and middleware with
// app.HandlerFunc and app.HandlerFuncs.
// Before: ../constructors
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/                        # 200, {"message":"hello"}
//	curl http://localhost:8080/stats                   # 200, hit counts per path
//	curl http://localhost:8080/healthz                 # 200, ok
//	curl -i http://localhost:8080/admin                # 403 PERMISSION_DENIED without the header
//	curl -H 'X-Admin: 1' http://localhost:8080/admin   # 200, {"area":"admin"}
//
// Next: ../request-scoped
package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	if err := app.RegisterDependency(&Config{Greeting: "hello"}, NewStats); err != nil {
		log.Fatal(err)
	}

	// As middleware: same shape, ctx.Next() continues the chain.
	app.Use(app.HandlerFunc(func(stats *Stats) iris.Handler {
		return func(ctx iris.Context) {
			stats.Hit(ctx.Path())
			ctx.Next()
		}
	}))

	// As a route handler.
	app.Get("/", app.HandlerFunc(func(cfg *Config) iris.Handler {
		return func(ctx iris.Context) {
			ctx.OK(iris.Map{"message": cfg.Greeting}, nil)
		}
	}))

	app.Get("/stats", app.HandlerFunc(func(stats *Stats) iris.Handler {
		return func(ctx iris.Context) {
			ctx.OK(stats.Snapshot(), nil)
		}
	}))

	// A net/http-style result is converted when possible, so third-party
	// handler factories fit without a wrapper.
	app.Get("/healthz", app.HandlerFunc(func(stats *Stats) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("ok"))
		}
	}))

	// HandlerFuncs maps several at once and returns iris.Handlers, which
	// Router accepts as the group's middleware.
	admin := app.Router("/admin", app.HandlerFuncs(
		func(stats *Stats) iris.Handler {
			return func(ctx iris.Context) {
				stats.Hit("admin")
				ctx.Next()
			}
		},
		func(cfg *Config) iris.Handler {
			return func(ctx iris.Context) {
				if ctx.Header("X-Admin") == "" {
					errors.PermissionDenied.Detail(ctx,
						"admin only", "send the X-Admin header")
					return // the errors senders stop the request.
				}
				ctx.Next()
			}
		},
	))
	admin.Get("/", func(ctx iris.Context) {
		ctx.OK(iris.Map{"area": "admin"}, nil)
	})

	app.Listen(":8080")
}

// Config is a plain value dependency.
type Config struct {
	Greeting string
}

// Stats is a singleton shared by every handler.
type Stats struct {
	mu   sync.Mutex
	hits map[string]int
}

func NewStats() *Stats {
	return &Stats{hits: make(map[string]int)}
}

func (s *Stats) Hit(path string) {
	s.mu.Lock()
	s.hits[path]++
	s.mu.Unlock()
}

func (s *Stats) Snapshot() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[string]int, len(s.hits))
	for k, v := range s.hits {
		out[k] = v
	}
	return out
}
