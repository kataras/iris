package radix

import (
	"sync"
	"time"
)

// global pool of *time.Timer's
var timerPool sync.Pool

// get returns a timer that completes after the given duration.
func getTimer(d time.Duration) *time.Timer {
	if t, _ := timerPool.Get().(*time.Timer); t != nil {
		t.Reset(d)
		return t
	}

	return time.NewTimer(d)
}

// putTimer pools the given timer. putTimer stops the timer and handles any left over data in the channel.
func putTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}

	timerPool.Put(t)
}
