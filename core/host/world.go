package host

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// package-level interrupt notifier and event firing.

type world struct {
	mu sync.Mutex
	// onInterrupt contains a list of the functions that should be called when CTRL+C/CMD+C or
	// a unix kill command received.
	onInterrupt []func()
}

var w = &world{}

// RegisterOnInterrupt registers a global function to call when CTRL+C/CMD+C pressed or a unix kill command received.
func RegisterOnInterrupt(cb func()) {
	w.mu.Lock()
	w.onInterrupt = append(w.onInterrupt, cb)
	w.mu.Unlock()
}

func notifyInterrupt() {
	w.mu.Lock()
	for _, f := range w.onInterrupt {
		go f()
	}
	w.mu.Unlock()
}

func tryStartInterruptNotifier() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.onInterrupt) > 0 {
		// this can't be moved to the task interrupt's `Run` function
		// because it will not catch more than one ctrl/cmd+c, so
		// we do it here. These tasks are canceled already too.
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch,
				// kill -SIGINT XXXX or Ctrl+c
				os.Interrupt,
				syscall.SIGINT, // register that too, it should be ok
				// os.Kill  is equivalent with the syscall.SIGKILL
				os.Kill,
				syscall.SIGKILL, // register that too, it should be ok
				// kill -SIGTERM XXXX
				syscall.SIGTERM,
			)
			select {
			case <-ch:
				notifyInterrupt()
			}
		}()
	}
}
