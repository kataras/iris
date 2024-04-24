package websocket

import (
	"github.com/kataras/neffos"
	"github.com/kataras/neffos/gobwas"
	"github.com/kataras/neffos/gorilla"
	"github.com/kataras/neffos/stackexchange/redis"
)

type (
	// Dialer is the definition type of a dialer, gorilla or gobwas or custom.
	// It is the second parameter of the `Dial` function.
	Dialer = neffos.Dialer
	// GorillaDialerOptions is just an alias for the `gobwas/ws.Dialer` struct type.
	GorillaDialerOptions = gorilla.Options
	// GobwasDialerOptions is just an alias for the `gorilla/websocket.Dialer` struct type.
	GobwasDialerOptions = gobwas.Options
	// GobwasHeader is an alias to the adapter that allows the use of `http.Header` as
	// handshake headers.
	GobwasHeader = gobwas.Header

	// Conn describes the main websocket connection's functionality.
	// Its `Connection` will return a new `NSConn` instance.
	// Each connection can connect to one or more declared namespaces.
	// Each `NSConn` can join to multiple rooms.
	Conn = neffos.Conn
	// NSConn describes a connection connected to a specific namespace,
	// it emits with the `Message.Namespace` filled and it can join to multiple rooms.
	// A single `Conn` can be connected to one or more namespaces,
	// each connected namespace is described by this structure.
	NSConn = neffos.NSConn
	// Room describes a connected connection to a room,
	// emits messages with the `Message.Room` filled to the specific room
	// and `Message.Namespace` to the underline `NSConn`'s namespace.
	Room = neffos.Room
	// CloseError can be used to send and close a remote connection in the event callback's return statement.
	CloseError = neffos.CloseError

	// MessageHandlerFunc is the definition type of the events' callback.
	// Its error can be written to the other side on specific events,
	// i.e on `OnNamespaceConnect` it will abort a remote namespace connection.
	// See examples for more.
	MessageHandlerFunc = neffos.MessageHandlerFunc
	// ConnHandler is the interface which namespaces and events can be retrieved through.
	ConnHandler = neffos.ConnHandler
	// Events completes the `ConnHandler` interface.
	// It is a map which its key is the event name
	// and its value the event's callback.
	//
	// Events type completes the `ConnHandler` itself therefore,
	// can be used as standalone value on the `New` and `Dial` functions
	// to register events on empty namespace as well.
	//
	// See `Namespaces`, `New` and `Dial` too.
	Events = neffos.Events
	// Namespaces completes the `ConnHandler` interface.
	// Can be used to register one or more namespaces on the `New` and `Dial` functions.
	// The key is the namespace literal and the value is the `Events`,
	// a map with event names and their callbacks.
	//
	// See `WithTimeout`, `New` and `Dial` too.
	Namespaces = neffos.Namespaces
	// WithTimeout completes the `ConnHandler` interface.
	// Can be used to register namespaces and events or just events on an empty namespace
	// with Read and Write timeouts.
	//
	// See `New` and `Dial`.
	WithTimeout = neffos.WithTimeout
	// Struct completes the `ConnHandler` interface.
	// It uses a structure to register a specific namespace and its events.
	Struct = neffos.Struct
	// StructInjector can be used to customize the value creation that can is used on serving events.
	StructInjector = neffos.StructInjector
	// The Message is the structure which describes the incoming and outcoming data.
	// Emitter's "body" argument is the `Message.Body` field.
	// Emitter's return non-nil error is the `Message.Err` field.
	// If native message sent then the `Message.Body` is filled with the body and
	// when incoming native message then the `Message.Event` is the `OnNativeMessage`,
	// native messages are allowed only when an empty namespace("") and its `OnNativeMessage` callback are present.
	Message = neffos.Message
	// StackExchange is an optional interface
	// that can be used to change the way neffos
	// sends messages to its clients, i.e
	// communication between multiple neffos servers.
	//
	// See `NewRedisStackExchange` to create a new redis StackExchange.
	StackExchange = neffos.StackExchange
	// RedisStackExchange is a `neffos.StackExchange` for redis.
	RedisStackExchange = redis.StackExchange
	// RedisConfig is used on the `NewRedisStackExchange` package-level function.
	// Can be used to customize the redis client dialer.
	RedisConfig = redis.Config
)
