package host

import (
	"context"
	"fmt"
	"math"
	"net"
	"sync"
	"time"
)

// Waiter is a helper for waiting for a server to be up and running.
type Waiter struct {
	defaultMaxRetries int
	addressFunc       func() string

	failure error // or runError for app.Run.
	mu      sync.RWMutex
}

// NewWaiter returns a new Waiter.
func NewWaiter(defaultMaxRetries int, addressFunc func() string) *Waiter {
	if defaultMaxRetries <= 0 {
		defaultMaxRetries = 7 // 256 seconds max.
	}

	return &Waiter{
		defaultMaxRetries: defaultMaxRetries,
		addressFunc:       addressFunc,
	}
}

// Wait blocks the main goroutine until the application is up and running.
func (w *Waiter) Wait(ctx context.Context) error {
	// First check if there is an error already from Done.
	if err := w.getFailure(); err != nil {
		return err
	}

	// Set the base for exponential backoff.
	base := 2.0

	// Get the maximum number of retries by context or force to default max retries (e.g. 7).
	var maxRetries int
	// Get the deadline of the context.
	if deadline, ok := ctx.Deadline(); ok {
		now := time.Now()
		timeout := deadline.Sub(now)

		maxRetries = getMaxRetries(timeout, base)
	} else {
		maxRetries = w.defaultMaxRetries
	}

	// Set the initial retry interval.
	retryInterval := time.Second

	return w.tryConnect(ctx, w.addressFunc, maxRetries, retryInterval, base)
}

// getMaxRetries calculates the maximum number of retries from the retry interval and the base.
func getMaxRetries(retryInterval time.Duration, base float64) int {
	// Convert the retry interval to seconds.
	seconds := retryInterval.Seconds()
	// Apply the inverse formula.
	retries := math.Log(seconds)/math.Log(base) - 1
	return int(math.Round(retries))
}

// tryConnect tries to connect to the server with the given context and retry parameters.
func (w *Waiter) tryConnect(ctx context.Context, addressFunc func() string, maxRetries int, retryInterval time.Duration, base float64) error {
	// Try to connect to the server in a loop.
	for i := 0; i < maxRetries; i++ {
		// Check the context before each attempt.
		select {
		case <-ctx.Done():
			// Context is canceled, return the context error.
			return ctx.Err()
		default:
			address := addressFunc() // Get this server's listening address.
			if address == "" {
				i-- // Note that this may be modified at another go routine of the serve method. So it may be empty at first chance. So retry fetching the VHost every 1 second.
				time.Sleep(time.Second)
				continue
			}

			// Context is not canceled, proceed with the attempt.
			conn, err := net.Dial("tcp", address)
			if err == nil {
				// Connection successful, close the connection and return nil.
				conn.Close()
				return nil // exit.
			} // ignore error.

			// Connection failed, wait for the retry interval and try again.
			time.Sleep(retryInterval)
			// After each failed attempt, check the server Run's error again.
			if err := w.getFailure(); err != nil {
				return err
			}

			// Increase the retry interval by the base raised to the power of the number of attempts.
			/*
				0	2 seconds
				1	4 seconds
				2	8 seconds
				3	~16 seconds
				4	~32 seconds
				5	~64 seconds
				6	~128 seconds
				7	~256 seconds
				8	~512 seconds
				...
			*/
			retryInterval = time.Duration(math.Pow(base, float64(i+1))) * time.Second
		}
	}
	// All attempts failed, return an error.
	return fmt.Errorf("failed to connect to the server after %d retries", maxRetries)
}

// Fail is called by the server's Run method when the server failed to start.
func (w *Waiter) Fail(err error) {
	w.mu.Lock()
	w.failure = err
	w.mu.Unlock()
}

func (w *Waiter) getFailure() error {
	w.mu.RLock()
	err := w.failure
	w.mu.RUnlock()
	return err
}
