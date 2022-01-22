package client

import (
	"context"
	"net/http"
	"sync"
)

// RequestHandler can be set to each Client instance and it should be
// responsible to handle the begin and end states of each request.
// Its BeginRequest fires right before the client talks to the server
// and its EndRequest fires right after the client receives a response from the server.
// If one of them return a non-nil error then the execution of client will stop and return that error.
type RequestHandler interface {
	BeginRequest(context.Context, *http.Request) error
	EndRequest(context.Context, *http.Response, error) error
}

var (
	defaultRequestHandlers []RequestHandler
	mu                     sync.Mutex
)

// RegisterRequestHandler registers one or more request handlers
// to be ran before and after of each request on all newly created Iris HTTP Clients.
// Useful for Iris HTTP Client 3rd-party libraries
// e.g. on init register a custom request-response lifecycle logging.
func RegisterRequestHandler(reqHandlers ...RequestHandler) {
	mu.Lock()
	defaultRequestHandlers = append(defaultRequestHandlers, reqHandlers...)
	mu.Unlock()
}
