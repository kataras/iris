// Package websocket bridges the neffos websocket server with Iris.
// Its entry point is New(*neffos.Server, Options), which returns a Handler that
// upgrades the connection; it must be the last handler in the route's chain.
package websocket

import (
	"net/http"
	"time"

	"github.com/kataras/iris/v14/context"

	"github.com/kataras/neffos"
)

// Name is the name of the handler, it's used to register the websocket handler.
const Name = "iris.websocket"

func init() {
	context.SetHandlerName("iris/middleware/websocket.*", Name)
}

// IDGenerator is an iris-specific IDGenerator for new connections.
type IDGenerator func(*context.Context) string

// DefaultIDGenerator returns a universal unique identifier for a new connection.
// It's the default `IDGenerator` if missing.
func DefaultIDGenerator(ctx *context.Context) string {
	return neffos.DefaultIDGenerator(ctx.ResponseWriter(), ctx.Request())
}

func wrapIDGenerator(idGen IDGenerator) func(ctx *context.Context) neffos.IDGenerator {
	return func(ctx *context.Context) neffos.IDGenerator {
		return func(w http.ResponseWriter, r *http.Request) string {
			return idGen(ctx)
		}
	}
}

// Options is the configuration for the websocket handler.
// It can be used to set a custom IDGenerator.
type Options struct {
	// IDGenerator defaults to the DefaultIDGenerator but it can be overridden.
	IDGenerator IDGenerator
}

// New returns an Iris handler to be served in a route of an Iris application.
// Accepts the neffos websocket server as its first input argument
// and the Options as its second one.
//
// This SHOULD be the last handler in the route's chain as it hijacks the connection and the context.
//
// Usage:
//
// const namespace = "default"
//
//	var serverEvents = neffos.Namespaces{
//		namespace: neffos.Events{
//			neffos.OnNamespaceConnected: func(nsConn *neffos.NSConn, msg neffos.Message) error {
//				// with `websocket.GetContext` you can retrieve the Iris' `Context`.
//				ctx := websocket.GetContext(nsConn.Conn)
//				log.Printf("[%s] connected to namespace [%s] with IP [%s]",
//					nsConn, msg.Namespace,
//					ctx.RemoteAddr())
//				return nil
//			},
//			neffos.OnNamespaceDisconnect: func(nsConn *neffos.NSConn, msg neffos.Message) error {
//				log.Printf("[%s] disconnected from namespace [%s]", nsConn, msg.Namespace)
//				return nil
//			},
//			"chat": func(nsConn *neffos.NSConn, msg neffos.Message) error {
//				// room.String() returns -> NSConn.String() returns -> Conn.String() returns -> Conn.ID()
//				log.Printf("[%s] sent: %s", nsConn, string(msg.Body))
//				// Write message back to the client message owner with:
//				// nsConn.Emit("chat", msg)
//				// Write message to all except this client with:
//				nsConn.Conn.Server().Broadcast(nsConn, msg)
//				return nil
//			},
//		},
//	}
//
//	func main() {
//		app := iris.New()
//		websocketServer := neffos.New(
//			gorilla.DefaultUpgrader, /* gobwas.DefaultUpgrader can be used too. */
//			serverEvents)
//	    app.Any("/ws", websocket.New(websocketServer, websocket.Options{}))
//	    // [...]
//	}
func New(s *neffos.Server, opts Options) context.Handler {
	if opts.IDGenerator == nil {
		opts.IDGenerator = DefaultIDGenerator
	}

	return func(ctx *context.Context) {
		if ctx.IsStopped() {
			return
		}

		Upgrade(ctx, s, opts.IDGenerator)
	}
}

// Upgrade upgrades the request and returns a new websocket Conn.
// Use `New` for higher-level implementation instead.
func Upgrade(ctx *context.Context, s *neffos.Server, idGen IDGenerator) *neffos.Conn {
	// The socket outlives the handler chain, so it gets its own Context: a
	// copy of this request's values, params and request that no pool knows
	// about. The router recycles ctx as it does for any other request, and
	// the socket's copy is collected once the connection is gone.
	//
	// Handing ctx itself to the socket meant disabling the pool release here
	// and putting it back from the disconnect event, on neffos' goroutine,
	// while the router could still be running the release hooks and reading
	// the release flag on its own. A client that closed right after the
	// handshake raced the router for the same Context and could put it in
	// the pool twice.
	socketCtx := ctx.Clone()

	/* Do NOT return the error as it is captured on the OnUpgradeError listener,
	the end-developer should not be able to write to this http client directly. */
	conn, _ := s.Upgrade(ctx.ResponseWriter(), ctx.Request(), func(socket neffos.Socket) neffos.Socket {
		return &socketWrapper{
			Socket: socket,
			ctx:    socketCtx,
		}
	}, wrapIDGenerator(idGen)(ctx))

	return conn
}

// socketWrapper carries the Iris Context of the upgrade request next to the
// socket of the upgrader.
//
// The embedded interface promotes the methods of neffos.Socket and nothing
// else, while neffos finds its optional socket extensions with a type
// assertion on the wrapper. The methods below hand each extension to the
// wrapped socket, so the close frame, the heartbeat and the read limit work
// as they do for a server that is not mounted through Iris.
type socketWrapper struct {
	neffos.Socket
	ctx *context.Context
}

var (
	_ neffos.SocketCloser      = (*socketWrapper)(nil)
	_ neffos.SocketPinger      = (*socketWrapper)(nil)
	_ neffos.SocketReadLimiter = (*socketWrapper)(nil)
)

// Close implements neffos.SocketCloser. A wrapped socket that cannot send a
// close frame gets its network connection closed, which is what neffos does
// with such a socket.
func (s *socketWrapper) Close(code int, reason string, timeout time.Duration) error {
	if closer, ok := s.Socket.(neffos.SocketCloser); ok {
		return closer.Close(code, reason, timeout)
	}

	return s.NetConn().Close()
}

// Ping implements neffos.SocketPinger. A wrapped socket that cannot ping
// reports no error, so the heartbeat never closes its connection: a
// PingInterval does nothing for such a socket.
func (s *socketWrapper) Ping(timeout time.Duration) error {
	if pinger, ok := s.Socket.(neffos.SocketPinger); ok {
		return pinger.Ping(timeout)
	}

	return nil
}

// SetReadLimit implements neffos.SocketReadLimiter. It does nothing for a
// wrapped socket that cannot limit its reads.
func (s *socketWrapper) SetReadLimit(n int64) {
	if limiter, ok := s.Socket.(neffos.SocketReadLimiter); ok {
		limiter.SetReadLimit(n)
	}
}

// GetContext returns the Iris Context from a websocket connection. It is a
// copy of the Context that served the upgrade request, taken before the
// handshake, with that request's values, params and headers; the original
// went back to the pool when the handler chain returned.
//
// Note that writing to the client connection through this Context is not allowed
// from a websocket event because the connection is hijacked.
// If used, you are limited for read-only access of the request e.g. read the request headers.
func GetContext(c *neffos.Conn) *context.Context {
	if sw, ok := c.Socket().(*socketWrapper); ok {
		return sw.ctx
	}
	return nil
}
