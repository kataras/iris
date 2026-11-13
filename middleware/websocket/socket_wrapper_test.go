package websocket_test

import (
	"context"
	"errors"
	"net/http"
	nethttptest "net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/websocket"

	"github.com/kataras/neffos"
	"github.com/kataras/neffos/gorilla"
)

const wrapperNamespace = "default"

// serveThroughNew mounts the neffos server on an Iris application with
// websocket.New, starts a real listener and returns its ws:// endpoint.
func serveThroughNew(t *testing.T, server *neffos.Server) string {
	t.Helper()

	app := iris.New()
	app.Get("/ws", websocket.New(server, websocket.Options{}))
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	t.Cleanup(srv.Close)
	t.Cleanup(server.Close)

	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
}

// dialNamespace connects a neffos client to the endpoint and to the namespace.
func dialNamespace(t *testing.T, dialer neffos.Dialer, endpoint string) (*neffos.Client, *neffos.NSConn) {
	t.Helper()

	client, err := neffos.Dial(t.Context(), dialer, endpoint,
		neffos.Namespaces{wrapperNamespace: neffos.Events{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)

	nsConn, err := client.Connect(t.Context(), wrapperNamespace)
	if err != nil {
		t.Fatal(err)
	}

	return client, nsConn
}

// pingCountingDialer is the gorilla dialer with a ping handler that reports
// every ping frame of the server on pings before it answers with the pong.
// The handler is set before neffos starts reading from the socket.
func pingCountingDialer(pings chan<- struct{}) neffos.Dialer {
	return func(ctx context.Context, url string) (neffos.Socket, error) {
		socket, err := gorilla.DefaultDialer(ctx, url)
		if err != nil {
			return nil, err
		}

		conn := socket.(*gorilla.Socket).UnderlyingConn
		pong := conn.PingHandler()
		conn.SetPingHandler(func(appData string) error {
			select {
			case pings <- struct{}{}:
			default:
			}
			return pong(appData)
		})

		return socket, nil
	}
}

// Conn.Terminate sends a close frame with its code and reason when the socket
// is a neffos.SocketCloser. The wrapper that carries the Iris Context hid that
// method of the gorilla socket, so neffos dropped the network connection
// instead and the client saw 1006 (abnormal closure) with no reason.
func TestNewTerminateSendsTheCloseCode(t *testing.T) {
	server := neffos.New(gorilla.DefaultUpgrader, neffos.Namespaces{
		wrapperNamespace: neffos.Events{
			"ban": func(nsConn *neffos.NSConn, msg neffos.Message) error {
				go nsConn.Conn.Terminate(neffos.ClosePolicyViolation, "banned")
				return nil
			},
		},
	})

	client, nsConn := dialNamespace(t, gorilla.DefaultDialer, serveThroughNew(t, server))
	if !nsConn.Emit("ban", nil) {
		t.Fatal("emit failed")
	}

	select {
	case <-client.NotifyClose:
	case <-time.After(5 * time.Second):
		t.Fatal("the client connection was not closed")
	}

	err := client.Conn().Err()
	ce, ok := errors.AsType[neffos.CloseError](err)
	if !ok || ce.Code != neffos.ClosePolicyViolation || ce.Reason != "banned" {
		t.Fatalf("client close error = %v, want [%d] %q: the close frame of Terminate did not reach the client",
			err, neffos.ClosePolicyViolation, "banned")
	}
}

// WithTimeout.MaxMessageSize is set on the socket through
// neffos.SocketReadLimiter. Behind the wrapper it was never set, so a 4096
// byte message passed a limit of 64 bytes and reached the event.
func TestNewAppliesMaxMessageSize(t *testing.T) {
	received := make(chan int, 1)

	server := neffos.New(gorilla.DefaultUpgrader, neffos.WithTimeout{
		MaxMessageSize: 64,
		Namespaces: neffos.Namespaces{
			wrapperNamespace: neffos.Events{
				"chat": func(nsConn *neffos.NSConn, msg neffos.Message) error {
					received <- len(msg.Body)
					return nil
				},
			},
		},
	})

	serverErrs := make(chan error, 1)
	server.OnDisconnect = func(c *neffos.Conn) { serverErrs <- c.Err() }

	client, nsConn := dialNamespace(t, gorilla.DefaultDialer, serveThroughNew(t, server))
	nsConn.Emit("chat", []byte(strings.Repeat("x", 4096)))

	select {
	case n := <-received:
		t.Fatalf("the server handled a %d byte message with MaxMessageSize 64", n)
	case <-client.NotifyClose:
	case <-time.After(5 * time.Second):
		t.Fatal("the message was neither handled nor refused")
	}

	if err := client.Conn().Err(); neffos.CloseStatus(err) != neffos.CloseMessageTooBig {
		t.Fatalf("client close error = %v, want code %d", err, neffos.CloseMessageTooBig)
	}

	select {
	case err := <-serverErrs:
		if !errors.Is(err, neffos.ErrMessageTooBig) {
			t.Fatalf("server close error = %v, want ErrMessageTooBig", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the server connection was not closed")
	}
}

// WithTimeout.PingInterval starts the heartbeat only for a socket that is a
// neffos.SocketPinger. Behind the wrapper no ping was ever sent.
func TestNewRunsTheHeartbeat(t *testing.T) {
	server := neffos.New(gorilla.DefaultUpgrader, neffos.WithTimeout{
		PingInterval: 50 * time.Millisecond,
		Namespaces:   neffos.Namespaces{wrapperNamespace: neffos.Events{}},
	})

	pings := make(chan struct{}, 1)
	dialNamespace(t, pingCountingDialer(pings), serveThroughNew(t, server))

	select {
	case <-pings:
	case <-time.After(3 * time.Second):
		t.Fatal("the server sent no ping with PingInterval 50ms")
	}
}

// The heartbeat closes a connection whose ping gets no pong, so the error of
// Ping has to come back through the wrapper as well. Behind the old wrapper
// the server kept the connection of a peer that had stopped answering.
func TestNewHeartbeatClosesAPeerThatSendsNoPong(t *testing.T) {
	server := neffos.New(gorilla.DefaultUpgrader, neffos.WithTimeout{
		PingInterval: 50 * time.Millisecond,
		Namespaces:   neffos.Namespaces{wrapperNamespace: neffos.Events{}},
	})

	serverErrs := make(chan error, 1)
	server.OnDisconnect = func(c *neffos.Conn) { serverErrs <- c.Err() }

	// The gorilla dialer with a ping handler that sends no pong back.
	silentDialer := func(ctx context.Context, url string) (neffos.Socket, error) {
		socket, err := gorilla.DefaultDialer(ctx, url)
		if err != nil {
			return nil, err
		}

		socket.(*gorilla.Socket).UnderlyingConn.SetPingHandler(func(string) error { return nil })
		return socket, nil
	}

	dialNamespace(t, silentDialer, serveThroughNew(t, server))

	select {
	case err := <-serverErrs:
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("server close error = %v, want the pong timeout of the heartbeat", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the server kept a connection that answers no ping with PingInterval 50ms")
	}
}

// bareSocket is a neffos.Socket with none of the optional extensions: the
// embedded interface exposes the methods of neffos.Socket and nothing else.
type bareSocket struct {
	neffos.Socket
}

func bareUpgrader(w http.ResponseWriter, r *http.Request) (neffos.Socket, error) {
	socket, err := gorilla.DefaultUpgrader(w, r)
	if err != nil {
		return nil, err
	}

	return bareSocket{socket}, nil
}

// A socket without the optional extensions keeps the behaviour neffos gives
// it when it is not wrapped: MaxMessageSize and PingInterval do nothing and
// Terminate closes the network connection, with no close frame.
func TestNewKeepsTheFallbacksOfABareSocket(t *testing.T) {
	received := make(chan int, 1)

	server := neffos.New(bareUpgrader, neffos.WithTimeout{
		PingInterval:   20 * time.Millisecond,
		MaxMessageSize: 64,
		Namespaces: neffos.Namespaces{
			wrapperNamespace: neffos.Events{
				"chat": func(nsConn *neffos.NSConn, msg neffos.Message) error {
					received <- len(msg.Body)
					return nil
				},
				"ban": func(nsConn *neffos.NSConn, msg neffos.Message) error {
					go nsConn.Conn.Terminate(neffos.ClosePolicyViolation, "banned")
					return nil
				},
			},
		},
	})

	pings := make(chan struct{}, 1)
	client, nsConn := dialNamespace(t, pingCountingDialer(pings), serveThroughNew(t, server))

	nsConn.Emit("chat", []byte(strings.Repeat("x", 4096)))
	select {
	case n := <-received:
		if n != 4096 {
			t.Fatalf("the server handled %d bytes, want 4096", n)
		}
	case <-client.NotifyClose:
		t.Fatalf("the connection was closed (%v): a socket that cannot limit its reads must take the message", client.Conn().Err())
	case <-time.After(5 * time.Second):
		t.Fatal("the message was neither handled nor refused")
	}

	// Ten ping intervals: a heartbeat would have pinged, and a failed ping
	// would have closed the connection.
	select {
	case <-pings:
		t.Fatal("the server pinged through a socket that cannot ping")
	case <-client.NotifyClose:
		t.Fatalf("the connection was closed while idle: %v", client.Conn().Err())
	case <-time.After(200 * time.Millisecond):
	}

	if !nsConn.Emit("ban", nil) {
		t.Fatal("emit failed")
	}

	select {
	case <-client.NotifyClose:
	case <-time.After(5 * time.Second):
		t.Fatal("Terminate did not close the connection of a socket that cannot send a close frame")
	}

	if err := client.Conn().Err(); neffos.CloseStatus(err) == neffos.ClosePolicyViolation {
		t.Fatalf("client close error = %v: a close frame was sent through a socket that cannot send one", err)
	}
}
