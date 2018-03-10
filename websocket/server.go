package websocket

import (
	"sync"

	"github.com/kataras/iris/context"

	"github.com/gorilla/websocket"
)

type connectionKV struct {
	key   string // the connection ID
	value *connection
}

type connections []connectionKV

func (cs *connections) add(key string, value *connection) {
	args := *cs
	n := len(args)
	// check if already id/key exist, if yes replace the conn
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.key == key {
			kv.value = value
			return
		}
	}

	c := cap(args)
	// make the connections slice bigger and put the conn
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.key = key
		kv.value = value
		*cs = args
		return
	}
	// append to the connections slice and put the conn
	kv := connectionKV{}
	kv.key = key
	kv.value = value
	*cs = append(args, kv)
}

func (cs *connections) get(key string) *connection {
	args := *cs
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.key == key {
			return kv.value
		}
	}
	return nil
}

// returns the connection which removed and a bool value of found or not
// the connection is useful to fire the disconnect events, we use that form in order to
// make work things faster without the need of get-remove, just -remove should do the job.
func (cs *connections) remove(key string) (*connection, bool) {
	args := *cs
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.key == key {
			conn := kv.value
			// we found the index,
			// let's remove the item by appending to the temp and
			// after set the pointer of the slice to this temp args
			args = append(args[:i], args[i+1:]...)
			*cs = args
			return conn, true
		}
	}
	return nil, false
}

type (
	// ConnectionFunc is the callback which fires when a client/connection is connected to the Server.
	// Receives one parameter which is the Connection
	ConnectionFunc func(Connection)

	// websocketRoomPayload is used as payload from the connection to the Server
	websocketRoomPayload struct {
		roomName     string
		connectionID string
	}

	// payloads, connection -> Server
	websocketMessagePayload struct {
		from string
		to   string
		data []byte
	}

	// Server is the websocket Server's implementation.
	//
	// It listens for websocket clients (either from the javascript client-side or from any websocket implementation).
	// See `OnConnection` , to register a single event which will handle all incoming connections and
	// the `Handler` which builds the upgrader handler that you can register to a route based on an Endpoint.
	//
	// To serve the built'n javascript client-side library look the `websocket.ClientHandler`.
	Server struct {
		config                Config
		connections           connections
		rooms                 map[string][]string // by default a connection is joined to a room which has the connection id as its name
		mu                    sync.RWMutex        // for rooms
		onConnectionListeners []ConnectionFunc
		//connectionPool        sync.Pool // sadly we can't make this because the websocket connection is live until is closed.
		upgrader websocket.Upgrader
	}
)

// New returns a new websocket Server based on a configuration.
// See `OnConnection` , to register a single event which will handle all incoming connections and
// the `Handler` which builds the upgrader handler that you can register to a route based on an Endpoint.
//
// To serve the built'n javascript client-side library look the `websocket.ClientHandler`.
func New(cfg Config) *Server {
	cfg = cfg.Validate()
	return &Server{
		config: cfg,
		rooms:  make(map[string][]string, 0),
		onConnectionListeners: make([]ConnectionFunc, 0),
		upgrader: websocket.Upgrader{
			HandshakeTimeout:  cfg.HandshakeTimeout,
			ReadBufferSize:    cfg.ReadBufferSize,
			WriteBufferSize:   cfg.WriteBufferSize,
			Error:             cfg.Error,
			CheckOrigin:       cfg.CheckOrigin,
			Subprotocols:      cfg.Subprotocols,
			EnableCompression: cfg.EnableCompression,
		},
	}
}

// Handler builds the handler based on the configuration and returns it.
// It should be called once per Server, its result should be passed
// as a middleware to an iris route which will be responsible
// to register the websocket's endpoint.
//
// Endpoint is the path which the websocket Server will listen for clients/connections.
//
// To serve the built'n javascript client-side library look the `websocket.ClientHandler`.
func (s *Server) Handler() context.Handler {
	return func(ctx context.Context) {
		c := s.Upgrade(ctx)
		if c.Err() != nil {
			return
		}
		// NOTE TO ME: fire these first BEFORE startReader and startPinger
		// in order to set the events and any messages to send
		// the startPinger will send the OK to the client and only
		// then the client is able to send and receive from Server
		// when all things are ready and only then. DO NOT change this order.

		// fire the on connection event callbacks, if any
		for i := range s.onConnectionListeners {
			s.onConnectionListeners[i](c)
		}

		// start the ping and the messages reader
		c.Wait()
	}
}

// Upgrade upgrades the HTTP Server connection to the WebSocket protocol.
//
// The responseHeader is included in the response to the client's upgrade
// request. Use the responseHeader to specify cookies (Set-Cookie) and the
// application negotiated subprotocol (Sec--Protocol).
//
// If the upgrade fails, then Upgrade replies to the client with an HTTP error
// response and the return `Connection.Err()` is filled with that error.
//
// For a more high-level function use the `Handler()` and `OnConnecton` events.
// This one does not starts the connection's writer and reader, so after your `On/OnMessage` events registration
// the caller has to call the `Connection#Wait` function, otherwise the connection will be not handled.
func (s *Server) Upgrade(ctx context.Context) Connection {
	conn, err := s.upgrader.Upgrade(ctx.ResponseWriter(), ctx.Request(), ctx.ResponseWriter().Header())
	if err != nil {
		ctx.Application().Logger().Warnf("websocket error: %v\n", err)
		ctx.StatusCode(503) // Status Service Unavailable
		return &connection{err: err}
	}

	return s.handleConnection(ctx, conn)
}

// wrapConnection wraps an underline connection to an iris websocket connection.
// It does NOT starts its writer, reader and event mux, the caller is responsible for that.
func (s *Server) handleConnection(ctx context.Context, websocketConn UnderlineConnection) *connection {
	// use the config's id generator (or the default) to create a websocket client/connection id
	cid := s.config.IDGenerator(ctx)
	// create the new connection
	c := newConnection(ctx, s, websocketConn, cid)
	// add the connection to the Server's list
	s.connections.add(cid, c)

	// join to itself
	s.Join(c.ID(), c.ID())

	return c
}

/* Notes:
   We use the id as the signature of the connection because with the custom IDGenerator
	 the developer can share this ID with a database field, so we want to give the oportunnity to handle
	 his/her websocket connections without even use the connection itself.

	 Another question may be:
	 Q: Why you use Server as the main actioner for all of the connection actions?
	 	  For example the Server.Disconnect(connID) manages the connection internal fields, is this code-style correct?
	 A: It's the correct code-style for these type of applications and libraries, Server manages all, the connnection's functions
	 should just do some internal checks (if needed) and push the action to its parent, which is the Server, the Server is able to
	 remove a connection, the rooms of its connected and all these things, so in order to not split the logic, we have the main logic
	 here, in the Server, and let the connection with some exported functions whose exists for the per-connection action user's code-style.

	 Ok my english are s** I can feel it, but these comments are mostly for me.
*/

/*
   connection actions, same as the connection's method,
    but these methods accept the connection ID,
    which is useful when the developer maps
    this id with a database field (using config.IDGenerator).
*/

// OnConnection is the main event you, as developer, will work with each of the websocket connections.
func (s *Server) OnConnection(cb ConnectionFunc) {
	s.onConnectionListeners = append(s.onConnectionListeners, cb)
}

// IsConnected returns true if the connection with that ID is connected to the Server
// useful when you have defined a custom connection id generator (based on a database)
// and you want to check if that connection is already connected (on multiple tabs)
func (s *Server) IsConnected(connID string) bool {
	c := s.connections.get(connID)
	return c != nil
}

// Join joins a websocket client to a room,
// first parameter is the room name and the second the connection.ID()
//
// You can use connection.Join("room name") instead.
func (s *Server) Join(roomName string, connID string) {
	s.mu.Lock()
	s.join(roomName, connID)
	s.mu.Unlock()
}

// join used internally, no locks used.
func (s *Server) join(roomName string, connID string) {
	if s.rooms[roomName] == nil {
		s.rooms[roomName] = make([]string, 0)
	}
	s.rooms[roomName] = append(s.rooms[roomName], connID)
}

// IsJoined reports if a specific room has a specific connection into its values.
// First parameter is the room name, second is the connection's id.
//
// It returns true when the "connID" is joined to the "roomName".
func (s *Server) IsJoined(roomName string, connID string) bool {
	s.mu.RLock()
	room := s.rooms[roomName]
	s.mu.RUnlock()

	if room == nil {
		return false
	}

	for _, connid := range room {
		if connID == connid {
			return true
		}
	}

	return false
}

// LeaveAll kicks out a connection from ALL of its joined rooms
func (s *Server) LeaveAll(connID string) {
	s.mu.Lock()
	for name := range s.rooms {
		s.leave(name, connID)
	}
	s.mu.Unlock()
}

// Leave leaves a websocket client from a room,
// first parameter is the room name and the second the connection.ID()
//
// You can use connection.Leave("room name") instead.
// Returns true if the connection has actually left from the particular room.
func (s *Server) Leave(roomName string, connID string) bool {
	s.mu.Lock()
	left := s.leave(roomName, connID)
	s.mu.Unlock()
	return left
}

// leave used internally, no locks used.
func (s *Server) leave(roomName string, connID string) (left bool) {
	///THINK: we could add locks to its room but we still use the lock for the whole rooms or we can just do what we do with connections
	// I will think about it on the next revision, so far we use the locks only for rooms so we are ok...
	if s.rooms[roomName] != nil {
		for i := range s.rooms[roomName] {
			if s.rooms[roomName][i] == connID {
				s.rooms[roomName] = append(s.rooms[roomName][:i], s.rooms[roomName][i+1:]...)
				left = true
				break
			}
		}
		if len(s.rooms[roomName]) == 0 { // if room is empty then delete it
			delete(s.rooms, roomName)
		}
	}

	if left {
		// fire the on room leave connection's listeners
		s.connections.get(connID).fireOnLeave(roomName)
	}
	return
}

// GetTotalConnections returns the number of total connections
func (s *Server) GetTotalConnections() int {
	s.mu.RLock()
	l := len(s.connections)
	s.mu.RUnlock()
	return l
}

// GetConnections returns all connections
func (s *Server) GetConnections() []Connection {
	s.mu.RLock()
	conns := make([]Connection, len(s.connections), len(s.connections))
	for i, c := range s.connections {
		conns[i] = c.value
	}
	s.mu.RUnlock()
	return conns
}

// GetConnection returns single connection
func (s *Server) GetConnection(key string) Connection {
	return s.connections.get(key)
}

// GetConnectionsByRoom returns a list of Connection
// which are joined to this room.
func (s *Server) GetConnectionsByRoom(roomName string) []Connection {
	s.mu.Lock()
	var conns []Connection
	if connIDs, found := s.rooms[roomName]; found {
		for _, connID := range connIDs {
			conns = append(conns, s.connections.get(connID))
		}

	}
	s.mu.Unlock()
	return conns
}

// emitMessage is the main 'router' of the messages coming from the connection
// this is the main function which writes the RAW websocket messages to the client.
// It sends them(messages) to the correct room (self, broadcast or to specific client)
//
// You don't have to use this generic method, exists only for extreme
// apps which you have an external goroutine with a list of custom connection list.
//
// You SHOULD use connection.EmitMessage/Emit/To().Emit/EmitMessage instead.
// let's keep it unexported for the best.
func (s *Server) emitMessage(from, to string, data []byte) {
	if to != All && to != Broadcast && s.rooms[to] != nil {
		// it suppose to send the message to a specific room/or a user inside its own room
		for _, connectionIDInsideRoom := range s.rooms[to] {
			if c := s.connections.get(connectionIDInsideRoom); c != nil {
				c.writeDefault(data) //send the message to the client(s)
			} else {
				// the connection is not connected but it's inside the room, we remove it on disconnect but for ANY CASE:
				cid := connectionIDInsideRoom
				if c != nil {
					cid = c.id
				}
				s.Leave(cid, to)
			}
		}
	} else {
		// it suppose to send the message to all opened connections or to all except the sender
		for _, cKV := range s.connections {
			connID := cKV.key
			if to != All && to != connID { // if it's not suppose to send to all connections (including itself)
				if to == Broadcast && from == connID { // if broadcast to other connections except this
					continue //here we do the opossite of previous block,
					// just skip this connection when it's suppose to send the message to all connections except the sender
				}

			}
			// send to the client(s) when the top validators passed
			cKV.value.writeDefault(data)
		}
	}
}

// Disconnect force-disconnects a websocket connection based on its connection.ID()
// What it does?
// 1. remove the connection from the list
// 2. leave from all joined rooms
// 3. fire the disconnect callbacks, if any
// 4. close the underline connection and return its error, if any.
//
// You can use the connection.Disconnect() instead.
func (s *Server) Disconnect(connID string) (err error) {
	// leave from all joined rooms before remove the actual connection from the list.
	// note: we cannot use that to send data if the client is actually closed.
	s.LeaveAll(connID)

	// remove the connection from the list
	if c, ok := s.connections.remove(connID); ok {
		if !c.disconnected {
			c.disconnected = true

			// fire the disconnect callbacks, if any
			c.fireDisconnect()
			// close the underline connection and return its error, if any.
			err = c.underline.Close()
		}
	}

	return
}
