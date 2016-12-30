package websocket

import (
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// --------------------------------Server implementation--------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

type (
	// ConnectionFunc is the callback which fires when a client/connection is connected to the server.
	// Receives one parameter which is the Connection
	ConnectionFunc func(Connection)
	// Rooms is just a map with key a string and  value slice of string
	Rooms map[string][]string

	// websocketRoomPayload is used as payload from the connection to the server
	websocketRoomPayload struct {
		roomName     string
		connectionID string
	}

	// payloads, connection -> server
	websocketMessagePayload struct {
		from string
		to   string
		data []byte
	}

	// Server is the websocket server, listens on the config's port, the critical part is the event OnConnection
	Server interface {
		// Set sets an option aka configuration field to the websocket server
		Set(...OptionSetter)
		// Handler returns the http.Handler which is setted to the 'Websocket Endpoint path', the client should target to this handler's developer's custom path
		// ex: http.Handle("/myendpoint", mywebsocket.Handler())
		// Handler calls the HandleConnection, so
		// Use Handler or HandleConnection manually, DO NOT USE both.
		// Note: you can always create your own upgrader which returns an UnderlineConnection and call only the HandleConnection manually (as Iris web framework does)
		Handler() http.Handler
		// HandleConnection creates & starts to listening to a new connection
		// DO NOT USE Handler() and HandleConnection at the sametime, see Handler for more
		//
		// Note: to see examples on how to manually use the HandleConnection, see one of my repositories:
		// look at https://github.com/iris-contrib/websocket, which is an edited version from gorilla/websocket to work with iris
		// and https://github.com/kataras/iris/blob/master/websocket.go
		// from fasthttp look at the https://github.com/fasthttp-contrib/websocket,  which is an edited version from gorilla/websocket to work with fasthttp
		HandleConnection(UnderlineConnection)
		// OnConnection this is the main event you, as developer, will work with each of the websocket connections
		OnConnection(cb ConnectionFunc)
		// Serve starts the websocket server, it's a non-blocking function (runs from a new goroutine)
		Serve()
	}

	server struct {
		once                  sync.Once
		config                Config
		put                   chan *connection
		free                  chan *connection
		connections           map[string]*connection
		join                  chan websocketRoomPayload
		leave                 chan websocketRoomPayload
		rooms                 Rooms      // by default a connection is joined to a room which has the connection id as its name
		mu                    sync.Mutex // for rooms
		messages              chan websocketMessagePayload
		onConnectionListeners []ConnectionFunc
		//connectionPool        *sync.Pool // sadly I can't make this because the websocket connection is live until is closed.
	}
)

var _ Server = &server{}

var defaultServer = newServer()

// server implementation

// New creates a websocket server and returns it
func New(setters ...OptionSetter) Server {
	return newServer(setters...)
}

// newServer creates a websocket server and returns it
func newServer(setters ...OptionSetter) *server {

	s := &server{
		put:                   make(chan *connection),
		free:                  make(chan *connection),
		connections:           make(map[string]*connection),
		join:                  make(chan websocketRoomPayload, 1), // buffered because join can be called immediately on connection connected
		leave:                 make(chan websocketRoomPayload),
		rooms:                 make(Rooms),
		messages:              make(chan websocketMessagePayload, 1), // buffered because messages can be sent/received immediately on connection connected
		onConnectionListeners: make([]ConnectionFunc, 0),
	}

	s.Set(setters...)

	// go s.serve() // start the ws server
	return s
}

// Set sets an option aka configuration field to the default websocket server
func Set(setters ...OptionSetter) {
	defaultServer.Set(setters...)
}

// Set sets an option aka configuration field to the websocket server
func (s *server) Set(setters ...OptionSetter) {
	for _, setter := range setters {
		setter.Set(&s.config)
	}

	s.config = s.config.Validate() // validate the fields on each call
}

// Handler returns the http.Handler which is setted to the 'Websocket Endpoint path', the client should target to this handler's developer's custom path
// ex: http.Handle("/myendpoint", mywebsocket.Handler())
// Handler calls the HandleConnection, so
// Use Handler or HandleConnection manually, DO NOT USE both.
// Note: you can always create your own upgrader which returns an UnderlineConnection and call only the HandleConnection manually (as Iris web framework does)
func Handler() http.Handler {
	return defaultServer.Handler()
}

func (s *server) Handler() http.Handler {
	// build the upgrader once
	c := s.config
	upgrader := websocket.Upgrader{ReadBufferSize: c.ReadBufferSize, WriteBufferSize: c.WriteBufferSize, Error: c.Error, CheckOrigin: c.CheckOrigin}
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
		//
		// The responseHeader is included in the response to the client's upgrade
		// request. Use the responseHeader to specify cookies (Set-Cookie) and the
		// application negotiated subprotocol (Sec--Protocol).
		//
		// If the upgrade fails, then Upgrade replies to the client with an HTTP error
		// response.
		conn, err := upgrader.Upgrade(res, req, res.Header())
		if err != nil {
			http.Error(res, "Websocket Error: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		s.handleConnection(conn)
	})
}

// HandleConnection creates & starts to listening to a new connection
func HandleConnection(websocketConn UnderlineConnection) {
	defaultServer.HandleConnection(websocketConn)
}

// HandleConnection creates & starts to listening to a new connection
func (s *server) HandleConnection(websocketConn UnderlineConnection) {
	s.handleConnection(websocketConn)
}

func (s *server) handleConnection(websocketConn UnderlineConnection) {
	c := newConnection(websocketConn, s)
	s.put <- c
	go c.writer()
	c.reader()
}

// OnConnection this is the main event you, as developer, will work with each of the websocket connections
func OnConnection(cb ConnectionFunc) {
	defaultServer.OnConnection(cb)
}

// OnConnection this is the main event you, as developer, will work with each of the websocket connections
func (s *server) OnConnection(cb ConnectionFunc) {
	// start the server here if was the first listener
	if len(s.onConnectionListeners) == 0 {
		s.Serve()
	}
	s.onConnectionListeners = append(s.onConnectionListeners, cb)
}

func (s *server) joinRoom(roomName string, connID string) {
	s.mu.Lock()
	if s.rooms[roomName] == nil {
		s.rooms[roomName] = make([]string, 0)
	}
	s.rooms[roomName] = append(s.rooms[roomName], connID)
	s.mu.Unlock()
}

func (s *server) leaveRoom(roomName string, connID string) {
	s.mu.Lock()
	if s.rooms[roomName] != nil {
		for i := range s.rooms[roomName] {
			if s.rooms[roomName][i] == connID {
				s.rooms[roomName][i] = s.rooms[roomName][len(s.rooms[roomName])-1]
				s.rooms[roomName] = s.rooms[roomName][:len(s.rooms[roomName])-1]
				break
			}
		}
		if len(s.rooms[roomName]) == 0 { // if room is empty then delete it
			delete(s.rooms, roomName)
		}
	}

	s.mu.Unlock()
}

// Serve starts the websocket server
func Serve() {
	defaultServer.Serve()
}

// Serve starts the websocket server
func (s *server) Serve() {
	s.once.Do(func() {
		go s.serve()
	})
}

func (s *server) serve() {
	for {
		select {
		case c := <-s.put: // connection established
			s.connections[c.id] = c
			// make and join a room with the connection's id
			s.rooms[c.id] = make([]string, 0)
			s.rooms[c.id] = []string{c.id}
			for i := range s.onConnectionListeners {
				s.onConnectionListeners[i](c)
			}
		case c := <-s.free: // connection closed
			if _, found := s.connections[c.id]; found {
				// leave from all rooms
				for roomName := range s.rooms {
					s.leaveRoom(roomName, c.id)
				}
				delete(s.connections, c.id)
				close(c.send)
				c.fireDisconnect()

			}
		case join := <-s.join:
			s.joinRoom(join.roomName, join.connectionID)
		case leave := <-s.leave:
			s.leaveRoom(leave.roomName, leave.connectionID)
		case msg := <-s.messages: // message received from the connection
			if msg.to != All && msg.to != NotMe && s.rooms[msg.to] != nil {
				// it suppose to send the message to a room
				for _, connectionIDInsideRoom := range s.rooms[msg.to] {
					if c, connected := s.connections[connectionIDInsideRoom]; connected {
						c.send <- msg.data //here we send it without need to continue below
					} else {
						// the connection is not connected but it's inside the room, we remove it on disconnect but for ANY CASE:
						cid := connectionIDInsideRoom
						if c != nil {
							cid = c.id
						}
						s.leaveRoom(cid, msg.to)
					}
				}

			} else { // it suppose to send the message to all opened connections or to all except the sender
				for connID, c := range s.connections {
					if msg.to != All { // if it's not suppose to send to all connections (including itself)
						if msg.to == NotMe && msg.from == connID { // if broadcast to other connections except this
							continue //here we do the opossite of previous block, just skip this connection when it's suppose to send the message to all connections except the sender
						} else if msg.to != connID { //it's not suppose to send to every one but to the one we'd like to
							continue
						}
					}
					select {
					case s.connections[connID].send <- msg.data: //send the message back to the connection in order to send it to the client
					default:
						close(c.send)
						delete(s.connections, connID)
						c.fireDisconnect()

					}

				}
			}

		}

	}
}
