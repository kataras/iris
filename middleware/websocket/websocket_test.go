package websocket_test

import (
	"net/http"
	nethttptest "net/http/httptest"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/websocket"

	"github.com/kataras/neffos"
	"github.com/kataras/neffos/gorilla"
)

// New returns a handler that upgrades the connection; a plain GET with no
// websocket handshake headers must be refused rather than reaching the route's
// next handler.
func TestNewRefusesANonWebsocketRequest(t *testing.T) {
	server := neffos.New(gorilla.DefaultUpgrader, neffos.Namespaces{
		"default": neffos.Events{},
	})
	defer server.Close()

	app := iris.New()
	app.Get("/ws", websocket.New(server, websocket.Options{}))
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/ws")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatalf("a non-websocket GET was upgraded; status = %d", resp.StatusCode)
	}
}

// The handshake succeeds and Options.IDGenerator names the connection, proving
// the second argument reaches the upgrader.
func TestNewUsesOptionsIDGenerator(t *testing.T) {
	const namespace = "default"

	connected := make(chan string, 1)

	server := neffos.New(gorilla.DefaultUpgrader, neffos.Namespaces{
		namespace: neffos.Events{
			neffos.OnNamespaceConnected: func(nsConn *neffos.NSConn, msg neffos.Message) error {
				connected <- nsConn.Conn.ID()
				return nil
			},
		},
	})
	defer server.Close()

	app := iris.New()
	app.Get("/ws", websocket.New(server, websocket.Options{
		IDGenerator: func(ctx iris.Context) string {
			return "fixed-id-" + ctx.Header("X-Client")
		},
	}))
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	defer srv.Close()

	client, err := neffos.Dial(t.Context(), gorilla.DefaultDialer,
		"ws"+strings.TrimPrefix(srv.URL, "http")+"/ws",
		neffos.Namespaces{namespace: neffos.Events{}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if _, err = client.Connect(t.Context(), namespace); err != nil {
		t.Fatal(err)
	}

	select {
	case id := <-connected:
		if !strings.HasPrefix(id, "fixed-id-") {
			t.Fatalf("connection id = %q, want the Options.IDGenerator value", id)
		}
	case <-t.Context().Done():
		t.Fatal("no connection was established")
	}
}

// A refused upgrade must leave the Context to the router. The handler used to
// force-release it, which put it back in the pool while the router still held
// it: the router's own Release put the very same Context a second time, so two
// later requests would be served from one object, and this request's release
// hooks were dropped on the way.
func TestNewRefusedUpgradeReleasesTheContextOnce(t *testing.T) {
	server := neffos.New(gorilla.DefaultUpgrader, neffos.Namespaces{
		"default": neffos.Events{},
	})
	defer server.Close()

	app := iris.New()

	var hookRuns int
	app.Get("/ws", func(ctx iris.Context) {
		ctx.OnRelease(func() { hookRuns++ })
		ctx.Next()
	}, websocket.New(server, websocket.Options{}))

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	rec := nethttptest.NewRecorder()
	app.ServeHTTP(rec, nethttptest.NewRequest(http.MethodGet, "/ws", nil))

	if rec.Code == http.StatusOK {
		t.Fatalf("a non-websocket GET was upgraded; status = %d", rec.Code)
	}

	if hookRuns != 1 {
		t.Fatalf("release hooks ran %d time(s), want 1: the refused upgrade released the Context behind the router's back", hookRuns)
	}

	pool := app.ContextPool()
	if first, second := pool.Acquire(), pool.Acquire(); first == second {
		t.Fatal("the pool handed the same *Context to two acquirers: the refused upgrade put it back twice, so two in-flight requests would share values, User and response writer")
	}
}

// The socket keeps its own copy of the upgrade request's Context. The router
// recycles the original as soon as the handler chain returns, and later
// requests reuse it, but an event must still read the values the middleware
// set before the upgrade.
func TestGetContextOutlivesTheRecycledRequestContext(t *testing.T) {
	const namespace = "default"

	users := make(chan string, 1)

	server := neffos.New(gorilla.DefaultUpgrader, neffos.Namespaces{
		namespace: neffos.Events{
			"whoami": func(nsConn *neffos.NSConn, msg neffos.Message) error {
				users <- websocket.GetContext(nsConn.Conn).Values().GetString("user")
				return nil
			},
		},
	})
	defer server.Close()

	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Values().Set("user", ctx.Request().URL.Query().Get("user"))
		ctx.Next()
	})
	app.Get("/ws", websocket.New(server, websocket.Options{}))
	app.Get("/", func(ctx iris.Context) {})
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	defer srv.Close()

	client, err := neffos.Dial(t.Context(), gorilla.DefaultDialer,
		"ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?user=alice",
		neffos.Namespaces{namespace: neffos.Events{}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	nsConn, err := client.Connect(t.Context(), namespace)
	if err != nil {
		t.Fatal(err)
	}

	// Serve other requests so a recycled Context would have been overwritten.
	for range 8 {
		resp, err := srv.Client().Get(srv.URL + "/?user=mallory")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	if !nsConn.Emit("whoami", nil) {
		t.Fatal("emit failed")
	}

	select {
	case user := <-users:
		if user != "alice" {
			t.Fatalf("user = %q, want %q: the socket read a Context that served another request", user, "alice")
		}
	case <-t.Context().Done():
		t.Fatal("the event never ran")
	}
}
