package signalhandler

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kataras/iris/v14/core/server/closer"
)

var (
	// A global variable to hold registered functions.
	// Each function now returns an error.
	closers closer.Closers
	// Once to ensure that the listener is set up only once.
	once sync.Once
	// Mutex to protect concurrent access to closers.
	mu sync.Mutex
)

// Register takes a function and appends it to the list of functions to be called on an interrupt signal.
// It also sets up the listener on the first call.
func Register(closer any) {
	mu.Lock()
	defer mu.Unlock()

	once.Do(setupListener)
	closers.AddCloser(closer)
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
		mu.Lock() // Lock the mutex before accessing closers.
		err := closers.CloseWithContext(context.Background())
		mu.Unlock() // Unlock the mutex after accessing closers.

		if err != nil {
			log.Fatalf("Received an error during shutdown: %v", err)
		}
		os.Exit(0)
	}()
}
