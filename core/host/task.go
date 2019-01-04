package host

// the 24hour name was "Supervisor" but it's not cover its usage
// 100%, best name is Task or Thead, I'll chouse Task.
// and re-name the host to "Supervisor" because that is the really
// supervisor.
import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/kataras/iris/core/netutil"
)

// WriteStartupLogOnServe is a task which accepts a logger(io.Writer)
// and logs the listening address
// by a generated message based on the host supervisor's server and writes it to the "w".
// This function should be registered on Serve.
func WriteStartupLogOnServe(w io.Writer) func(TaskHost) {
	return func(h TaskHost) {
		guessScheme := netutil.ResolveScheme(h.Supervisor.manuallyTLS)
		listeningURI := netutil.ResolveURL(guessScheme, h.Supervisor.Server.Addr)
		interruptkey := "CTRL"
		if runtime.GOOS == "darwin" {
			interruptkey = "CMD"
		}
		w.Write([]byte(fmt.Sprintf("Now listening on: %s\nApplication started. Press %s+C to shut down.\n",
			listeningURI, interruptkey)))
	}
}

// ShutdownOnInterrupt terminates the supervisor and its underline server when CMD+C/CTRL+C pressed.
// This function should be registered on Interrupt.
func ShutdownOnInterrupt(su *Supervisor, shutdownTimeout time.Duration) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.TODO(), shutdownTimeout)
		defer cancel()
		su.Shutdown(ctx)
		su.RestoreFlow()
	}
}

// TaskHost contains all the necessary information
// about the host supervisor, its server
// and the exports the whole flow controller of it.
type TaskHost struct {
	Supervisor *Supervisor
}

// Serve can (re)run the server with the latest known configuration.
func (h TaskHost) Serve() error {
	// the underline server's serve, using the "latest known" listener from the supervisor.
	l, err := h.Supervisor.newListener()
	if err != nil {
		return err
	}

	// if http.serverclosed ignroe the error, it will have this error
	// from the previous close
	if err := h.Supervisor.Server.Serve(l); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// HostURL returns the listening full url (scheme+host)
// based on the supervisor's server's address.
func (h TaskHost) HostURL() string {
	return netutil.ResolveURLFromServer(h.Supervisor.Server)
}

// Hostname returns the underline server's hostname.
func (h TaskHost) Hostname() string {
	return netutil.ResolveHostname(h.Supervisor.Server.Addr)
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first closing all open
// listeners, then closing all idle connections, and then waiting
// indefinitely for connections to return to idle and then shut down.
// If the provided context expires before the shutdown is complete,
// then the context's error is returned.
//
// Shutdown does not attempt to close nor wait for hijacked
// connections such as WebSockets. The caller of Shutdown should
// separately notify such long-lived connections of shutdown and wait
// for them to close, if desired.
//
// This Shutdown calls the underline's Server's Shutdown, in order to be able to re-start the server
// from a task.
func (h TaskHost) Shutdown(ctx context.Context) error {
	// the underline server's Shutdown (otherwise we will cancel all tasks and do cycles)
	return h.Supervisor.Server.Shutdown(ctx)
}

func createTaskHost(su *Supervisor) TaskHost {
	return TaskHost{
		Supervisor: su,
	}
}
