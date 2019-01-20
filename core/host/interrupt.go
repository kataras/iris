package host

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// RegisterOnInterrupt registers a global function to call when CTRL+C/CMD+C pressed or a unix kill command received.
func RegisterOnInterrupt(cb func()) {
	Interrupt.Register(cb)
}

// Interrupt watches the os.Signals for interruption signals
// and fires the callbacks when those happens.
// A call of its `FireNow` manually will fire and reset the registered interrupt handlers.
var Interrupt = new(interruptListener)

type interruptListener struct {
	mu   sync.Mutex
	once sync.Once
	// onInterrupt contains a list of the functions that should be called when CTRL+C/CMD+C or
	// a unix kill command received.
	onInterrupt []func()
}

// Register registers a global function to call when CTRL+C/CMD+C pressed or a unix kill command received.
func (i *interruptListener) Register(cb func()) {
	if cb == nil {
		return
	}

	i.listenOnce()
	i.mu.Lock()
	i.onInterrupt = append(i.onInterrupt, cb)
	i.mu.Unlock()
}

// FireNow can be called more than one times from a Consumer in order to
// execute all interrupt handlers manually.
func (i *interruptListener) FireNow() {
	i.mu.Lock()
	for _, f := range i.onInterrupt {
		f()
	}
	i.onInterrupt = i.onInterrupt[0:0]
	i.mu.Unlock()
}

// listenOnce fires a goroutine which calls the interrupt handlers when CTRL+C/CMD+C and e.t.c.
// If `FireNow` called before then it does nothing when interrupt signal received,
// so it's safe to be used side by side with `FireNow`.
//
// Btw this `listenOnce` is called automatically on first register, it's useless for outsiders.
func (i *interruptListener) listenOnce() {
	i.once.Do(func() { go i.notifyAndFire() })
}

func (i *interruptListener) notifyAndFire() {
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
		i.FireNow()
	}
}
