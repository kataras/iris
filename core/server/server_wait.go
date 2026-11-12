package server

import (
	"context"
	"fmt"
	"math"
	"net"
	"time"
)

// Wait blocks until the server is up and running.
// If the server is already up and running it returns immediately.
// If the server is not running, it will retry to connect to the server with an exponential backoff.
// The context can be used to cancel the wait operation.
// If the server errors before it is up and running, the error is returned.
//
// This is useful when you want to start the server on a separate go routine and
// wait somewhere else for it to be up and running before continuing.
// Useful also inside unit testing, so you don't have to use time.Sleep to
// wait for the server to be up and running before sending requests.
func (s *Server) Wait(ctx context.Context) error {
	// First check if there is an error already from the ListenAndServe.
	if err := s.getServerError(); err != nil {
		return err
	}

	// Set the base for exponential backoff.
	base := 2.0

	// Get the maximum number of retries by context or force to 7 retries.
	var maxRetries int
	// Get the deadline of the context.
	if deadline, ok := ctx.Deadline(); ok {
		now := time.Now()
		timeout := deadline.Sub(now)

		maxRetries = getMaxRetries(timeout, base)
	} else {
		maxRetries = 7 // 256 seconds max.
	}

	// Set the initial retry interval.
	retryInterval := time.Second

	return s.tryConnect(ctx, maxRetries, retryInterval, base)
}

func (s *Server) setServerError(err error) {
	s.serverErrorMu.Lock()
	s.serverError = err
	s.serverErrorMu.Unlock()
}

func (s *Server) getServerError() error {
	s.serverErrorMu.RLock()
	serverError := s.serverError
	s.serverErrorMu.RUnlock()
	return serverError
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
func (s *Server) tryConnect(ctx context.Context, maxRetries int, retryInterval time.Duration, base float64) error {
	// Try to connect to the server in a loop.
	for i := 0; i < maxRetries; i++ {
		// Check the context before each attempt.
		select {
		case <-ctx.Done():
			// Context is canceled, return the context error.
			return ctx.Err()
		default:
			address := s.options.Addr // Get this server's listening address.
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
			// After each failed attempt, check the server error again.
			if err := s.getServerError(); err != nil {
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
