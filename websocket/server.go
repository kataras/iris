package websocket

import (
	"sync"

	"github.com/iris-contrib/websocket"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/context"
)

type (
	// ConnectionFunc is the callback which fires when a client/connection is connected to the server.
	// Receives one parameter which is the Connection
	ConnectionFunc func(Connection)
	// Rooms is just a map with key a string and  value slice of string
	Rooms map[string][]string
	// Server is the websocket server
	Server interface {
		// Upgrade upgrades the client in order websocket works
		Upgrade(context.IContext) error
		// OnConnection registers a callback which fires when a connection/client is connected to the server
		OnConnection(ConnectionFunc)
	}

	// roomPayload is used as payload from the connection to the server
	roomPayload struct {
		roomName     string
		connectionID string
	}

	// payloads, connection -> server
	messagePayload struct {
		from string
		to   string
		data []byte
	}

	//

	server struct {
		config                *config.Websocket
		upgrader              websocket.Upgrader
		put                   chan *connection
		free                  chan *connection
		connections           map[string]*connection
		join                  chan roomPayload
		leave                 chan roomPayload
		rooms                 Rooms      // by default a connection is joined to a room which has the connection id as its name
		mu                    sync.Mutex // for rooms
		messages              chan messagePayload
		onConnectionListeners []ConnectionFunc
		//connectionPool        *sync.Pool // sadly I can't make this because the websocket connection is live until is closed.
	}
)

var _ Server = &server{}

// server implementation

func newServer(c config.Websocket) *server {
	s := &server{
		config:                &c,
		put:                   make(chan *connection),
		free:                  make(chan *connection),
		connections:           make(map[string]*connection),
		join:                  make(chan roomPayload, 1), // buffered because join can be called immediately on connection connected
		leave:                 make(chan roomPayload),
		rooms:                 make(Rooms),
		messages:              make(chan messagePayload, 1), // buffered because messages can be sent/received immediately on connection connected
		onConnectionListeners: make([]ConnectionFunc, 0),
	}

	s.upgrader = websocket.New(s.handleConnection)
	go s.serve() // start the server automatically

	return s
}

func (s *server) Upgrade(ctx context.IContext) error {
	return s.upgrader.Upgrade(ctx)
}

func (s *server) handleConnection(websocketConn *websocket.Conn) {
	c := newConnection(websocketConn, s)
	s.put <- c
	go c.writer()
	c.reader()
}

func (s *server) OnConnection(cb ConnectionFunc) {
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

func (s *server) serve() {
	for {
		select {
		case c := <-s.put: // connection connected
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
						s.leaveRoom(c.id, msg.to)
					}
				}

			} else { // it suppose to send the message to all opened connections or to all except the sender
				for connID, c := range s.connections {
					if msg.to != All { // if it's not suppose to send to all connections (including itself)
						if msg.to == NotMe && msg.from == connID { // if broadcast to other connections except this
							continue //here we do the opossite of previous block, just skip this connection when it's suppose to send the message to all connections except the sender
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

//
