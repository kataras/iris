package graceful

import (
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/logger"
	"github.com/kataras/iris/server"
	"golang.org/x/net/netutil"
)

// Server wraps an iris.Server with graceful connection handling.
// It may be used directly in the same way as iris.Server, or may
// be constructed with the global functions in this package.
type Server struct {
	*server.Server
	station *iris.Iris
	// Timeout is the duration to allow outstanding requests to survive
	// before forcefully terminating them.
	Timeout time.Duration

	// Limit the number of outstanding requests
	ListenLimit int

	// BeforeShutdown is an optional callback function that is called
	// before the listener is closed.
	BeforeShutdown func()

	// ShutdownInitiated is an optional callback function that is called
	// when shutdown is initiated. It can be used to notify the client
	// side of long lived connections (e.g. websockets) to reconnect.
	ShutdownInitiated func()

	// NoSignalHandling prevents graceful from automatically shutting down
	// on SIGINT and SIGTERM. If set to true, you must shut down the server
	// manually with Stop().
	NoSignalHandling bool

	// Logger used to notify of errors on startup and on stop.
	Logger *logger.Logger

	// Interrupted is true if the server is handling a SIGINT or SIGTERM
	// signal and is thus shutting down.
	Interrupted bool

	// interrupt signals the listener to stop serving connections,
	// and the server to shut down.
	interrupt chan os.Signal

	// stopLock is used to protect against concurrent calls to Stop
	stopLock sync.Mutex

	// stopChan is the channel on which callers may block while waiting for
	// the server to stop.
	stopChan chan struct{}

	// chanLock is used to protect access to the various channel constructors.
	chanLock sync.RWMutex

	// connections holds all connections managed by graceful
	connections map[net.Conn]struct{}
}

// Run serves the http.Handler with graceful shutdown enabled.
//
// timeout is the duration to wait until killing active requests and stopping the server.
// If timeout is 0, the server never times out. It waits for all active requests to finish.
// we don't pass an iris.RequestHandler , because we need iris.station.server to be setted in order the station.Close() to work
func Run(addr string, timeout time.Duration, n *iris.Iris) {
	srv := &Server{
		Timeout: timeout,
		Logger:  DefaultLogger(),
	}
	srv.station = n
	srv.Server = srv.station.PreListen(config.Server{ListeningAddr: addr})

	if err := srv.listenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			srv.Logger.Fatal(err)
		}
	}

}

// RunWithErr is an alternative version of Run function which can return error.
//
// Unlike Run this version will not exit the program if an error is encountered but will
// return it instead.
func RunWithErr(addr string, timeout time.Duration, n *iris.Iris) error {
	srv := &Server{
		Timeout: timeout,
		Logger:  DefaultLogger(),
	}
	srv.station = n
	srv.Server = srv.station.PreListen(config.Server{ListeningAddr: addr})
	return srv.listenAndServe()
}

// ListenAndServe is equivalent to iris.Listen with graceful shutdown enabled.
func (srv *Server) listenAndServe() error {
	// Create the listener so we can control their lifetime

	addr := srv.Config.ListeningAddr
	if addr == "" {
		addr = ":http"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.serve(l)
}

// Serve is equivalent to iris.Server.Serve with graceful shutdown enabled.
func (srv *Server) serve(listener net.Listener) error {

	if srv.ListenLimit != 0 {
		listener = netutil.LimitListener(listener, srv.ListenLimit)
	}

	// Track connection state
	add := make(chan net.Conn)
	remove := make(chan net.Conn)

	// Manage open connections
	shutdown := make(chan chan struct{})
	kill := make(chan struct{})
	go srv.manageConnections(add, remove, shutdown, kill)

	interrupt := srv.interruptChan()
	// Set up the interrupt handler
	if !srv.NoSignalHandling {
		signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	}
	quitting := make(chan struct{})
	go srv.handleInterrupt(interrupt, quitting, listener)

	// Serve with graceful listener.
	// Execution blocks here until listener.Close() is called, above.
	srv.station.PostListen()
	err := srv.Server.Serve(listener)
	if err != nil {
		// If the underlying listening is closed, Serve returns an error
		// complaining about listening on a closed socket. This is expected, so
		// let's ignore the error if we are the ones who explicitly closed the
		// socket.
		select {
		case <-quitting:
			err = nil
		default:
		}
	}

	srv.shutdown(shutdown, kill)

	return err
}

// Stop instructs the type to halt operations and close
// the stop channel when it is finished.
//
// timeout is grace period for which to wait before shutting
// down the server. The timeout value passed here will override the
// timeout given when constructing the server, as this is an explicit
// command to stop the server.
func (srv *Server) Stop(timeout time.Duration) {
	srv.stopLock.Lock()
	defer srv.stopLock.Unlock()

	srv.Timeout = timeout
	interrupt := srv.interruptChan()
	interrupt <- syscall.SIGINT
}

// StopChan gets the stop channel which will block until
// stopping has completed, at which point it is closed.
// Callers should never close the stop channel.
func (srv *Server) StopChan() <-chan struct{} {
	srv.chanLock.Lock()
	defer srv.chanLock.Unlock()

	if srv.stopChan == nil {
		srv.stopChan = make(chan struct{})
	}
	return srv.stopChan
}

// DefaultLogger returns the logger used by Run, RunWithErr, ListenAndServe, ListenAndServeTLS and Serve.
// The logger outputs to STDERR by default.
func DefaultLogger() *logger.Logger {
	return logger.New()
}

func (srv *Server) manageConnections(add, remove chan net.Conn, shutdown chan chan struct{}, kill chan struct{}) {
	var done chan struct{}
	srv.connections = map[net.Conn]struct{}{}
	for {
		select {
		case conn := <-add:
			srv.connections[conn] = struct{}{}
		case conn := <-remove:
			delete(srv.connections, conn)
			if done != nil && len(srv.connections) == 0 {
				done <- struct{}{}
				return
			}
		case done = <-shutdown:
			if len(srv.connections) == 0 {
				done <- struct{}{}
				return
			}
		case <-kill:
			for k := range srv.connections {
				if err := k.Close(); err != nil {
					srv.log("[IRIS GRACEFUL ERROR] %s", err.Error())
				}
			}
			return
		}
	}
}

func (srv *Server) interruptChan() chan os.Signal {
	srv.chanLock.Lock()
	defer srv.chanLock.Unlock()

	if srv.interrupt == nil {
		srv.interrupt = make(chan os.Signal, 1)
	}

	return srv.interrupt
}

func (srv *Server) handleInterrupt(interrupt chan os.Signal, quitting chan struct{}, listener net.Listener) {
	for _ = range interrupt {
		if srv.Interrupted {
			srv.log("already shutting down")
			continue
		}
		srv.log("shutdown initiated")
		srv.Interrupted = true
		if srv.BeforeShutdown != nil {
			srv.BeforeShutdown()
		}

		close(quitting)
		srv.Server.DisableKeepalive = true
		if err := listener.Close(); err != nil {
			srv.log("[IRIS GRACEFUL ERROR] %s", err.Error())
		}

		if srv.ShutdownInitiated != nil {
			srv.ShutdownInitiated()
		}
	}
}

func (srv *Server) log(fmt string, v ...interface{}) {
	if srv.Logger != nil {
		srv.Logger.Printf(fmt, v...)
	}
}

func (srv *Server) shutdown(shutdown chan chan struct{}, kill chan struct{}) {
	// Request done notification
	done := make(chan struct{})
	shutdown <- done

	if srv.Timeout > 0 {
		select {
		case <-done:
		case <-time.After(srv.Timeout):
			close(kill)
		}
	} else {
		<-done
	}
	// Close the stopChan to wake up any blocked goroutines.
	srv.chanLock.Lock()
	if srv.stopChan != nil {
		close(srv.stopChan)
	}
	// notify the iris plugins
	srv.station.Close()
	srv.chanLock.Unlock()
}
