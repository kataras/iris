package signalhandler

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kataras/iris/v14/core/server/closer"
)

var (
	// A global variable to hold registered functions.
	// Each function now returns an error.
	closers closer.Closers
	// Once to ensure that the listener is set up only once.
	once sync.Once
	// Mutex to protect concurrent access to closers and to timeout.
	mu sync.Mutex
	// timeout bounds the context the closers receive. It is the largest one a
	// server gave through RegisterWithTimeout; zero sets no deadline.
	timeout time.Duration
	// unbounded is set when a server registered with no deadline, a zero or
	// negative timeout. It lifts the deadline the other servers asked for.
	unbounded bool

	// triggered is closed when a signal arrives and finished when the closers
	// of that signal have returned with no error. Wait blocks between the two.
	triggered = make(chan struct{})
	finished  = make(chan struct{})

	// exit ends the process once the closers are done. A test replaces it.
	exit = os.Exit
)

// Register takes a function and appends it to the list of functions to be called on an interrupt signal.
// It also sets up the listener on the first call.
//
// The function receives a context bounded by the
// ServerOptions.Timeout.ShutdownOnInterruptTimeout of the running server, see
// RegisterWithTimeout.
func Register(closer any) {
	mu.Lock()
	defer mu.Unlock()

	once.Do(setupListener)
	closers.AddCloser(closer)
}

// RegisterWithTimeout is Register for a server's own closer, which also names
// how long the closers of an interrupt may take: its
// ServerOptions.Timeout.ShutdownOnInterruptTimeout.
//
// Every closer of the signal receives a context carrying that as its deadline,
// so a callback registered with Register, iris.RegisterOnInterrupt(app.Shutdown)
// above all, is bounded by the same option that bounds the server's own drain.
// A zero or negative value sets no deadline, as that option documents, and so
// does a process whose servers never registered one.
//
// With more than one server in the process the one that asked for the most
// wins: the largest value of all of them, and no deadline at all once one of
// them registered a zero or negative value. A smaller one would cut the drain
// of the server that asked for more.
//
// One limit of that rule: a value is never taken back, so a server that has
// stopped still counts. One that asked for no deadline lifts it for the rest of
// the process, whatever the servers started after it ask for.
func RegisterWithTimeout(closer any, d time.Duration) {
	mu.Lock()
	if d <= 0 {
		unbounded = true
	} else if d > timeout {
		timeout = d
	}
	mu.Unlock()

	Register(closer)
}

// Wait blocks while the closers of an interrupt signal are running, so a caller
// that is about to let the process end gives them the chance to finish. It
// returns at once when no signal has been received.
//
// The process exits as soon as they are done, so a Wait that blocked returns at
// the moment the process is going away. When a closer returned an error it does
// not return at all: the process exits with status 1 while it is still blocked.
func Wait() {
	select {
	case <-triggered:
		<-finished
	default:
	}
}

// setupListener is responsible for listening to the interrupt signals and calling the registered functions.
func setupListener() {
	// Remember that os.Interrupt is a portable way of specifying an interrupt signal and
	// is equivalent to syscall.SIGINT on Unix-based systems.
	// Including both os.Interrupt and syscall.SIGINT is redundant on Unix, but it does not cause any issues.
	// It ensures that your program responds to Ctrl+C on all platforms,
	// including Windows, where os.Interrupt is the signal for an interrupt from the keyboard.
	//
	// Also syscall.SIGHUP can be used to reload the configuration of the server without stopping the process (we don't need it at the moment).

	// Create a channel to receive os.Signal values.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-c
		onSignal()
	}()
}

// onSignal runs the registered closers and ends the process. It is what the
// signal's goroutine does, as its own function so that a test can run it.
func onSignal() {
	close(triggered)

	ctx := context.Background()
	mu.Lock() // Lock the mutex before accessing closers.
	if timeout > 0 && !unbounded {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	err := closers.CloseWithContext(ctx)
	mu.Unlock() // Unlock the mutex after accessing closers.

	if err != nil {
		// Before the Wait calls are released: a main that Run handed back to
		// would end the process itself, with status 0, and could do so before
		// this line is printed.
		log.Printf("Received an error during shutdown: %v", err)
		exit(1)
		return
	}

	// Release the Wait calls: every closer has returned and the process is
	// about to end.
	close(finished)
	exit(0)
}
