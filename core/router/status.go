package router

import (
	"net/http" // just for status codes
	"sync"

	"github.com/kataras/iris/context"
)

func statusCodeSuccessful(statusCode int) bool {
	return !context.StatusCodeNotSuccessful(statusCode)
}

// ErrorCodeHandler is the entry
// of the list of all http error code handlers.
type ErrorCodeHandler struct {
	StatusCode int
	Handlers   context.Handlers
	mu         sync.Mutex
}

// Fire executes the specific an error http error status.
// it's being wrapped to make sure that the handler
// will render correctly.
func (ch *ErrorCodeHandler) Fire(ctx context.Context) {
	// if we can reset the body
	if w, ok := ctx.IsRecording(); ok {
		if statusCodeSuccessful(w.StatusCode()) { // if not an error status code
			w.WriteHeader(ch.StatusCode) // then set it manually here, otherwise it should be setted via ctx.StatusCode(...)
		}
		// reset if previous content and it's recorder, keep the status code.
		w.ClearHeaders()
		w.ResetBody()
	} else if w, ok := ctx.ResponseWriter().(*context.GzipResponseWriter); ok {
		// reset and disable the gzip in order to be an expected form of http error result
		w.ResetBody()
		w.Disable()
	} else {
		// if we can't reset the body and the body has been filled
		// which means that the status code already sent,
		// then do not fire this custom error code.
		if ctx.ResponseWriter().Written() > 0 { // != -1, rel: context/context.go#EndRequest
			return
		}
	}

	// ctx.StopExecution() // not uncomment this, is here to remember why to.
	// note for me: I don't stopping the execution of the other handlers
	// because may the user want to add a fallback error code
	// i.e
	// users := app.Party("/users")
	// users.Done(func(ctx context.Context){ if ctx.StatusCode() == 400 { /*  custom error code for /users */ }})

	// use .HandlerIndex
	// that sets the current handler index to zero
	// in order to:
	// ignore previous runs that may changed the handler index,
	// via ctx.Next or ctx.StopExecution, if any.
	//
	// use .Do
	// that overrides the existing handlers and sets and runs these error handlers.
	// in order to:
	// ignore the route's after-handlers, if any.
	ctx.HandlerIndex(0)
	ctx.Do(ch.Handlers)
}

func (ch *ErrorCodeHandler) updateHandlers(handlers context.Handlers) {
	ch.mu.Lock()
	ch.Handlers = handlers
	ch.mu.Unlock()
}

// ErrorCodeHandlers contains the http error code handlers.
// User of this struct can register, get
// a status code handler based on a status code or
// fire based on a receiver context.
type ErrorCodeHandlers struct {
	handlers []*ErrorCodeHandler
}

func defaultErrorCodeHandlers() *ErrorCodeHandlers {
	chs := new(ErrorCodeHandlers)
	// register some common error handlers.
	// Note that they can be registered on-fly but
	// we don't want to reduce the performance even
	// on the first failed request.
	for _, statusCode := range []int{
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusInternalServerError} {
		chs.Register(statusCode, statusText(statusCode))
	}

	return chs
}

func statusText(statusCode int) context.Handler {
	return func(ctx context.Context) {
		ctx.WriteString(http.StatusText(statusCode))
	}
}

// Get returns an http error handler based on the "statusCode".
// If not found it returns nil.
func (s *ErrorCodeHandlers) Get(statusCode int) *ErrorCodeHandler {
	for i, n := 0, len(s.handlers); i < n; i++ {
		if h := s.handlers[i]; h.StatusCode == statusCode {
			return h
		}
	}
	return nil
}

// Register registers an error http status code
// based on the "statusCode" < 200 || >= 400 (`context.StatusCodeNotSuccessful`).
// The handler is being wrapepd by a generic
// handler which will try to reset
// the body if recorder was enabled
// and/or disable the gzip if gzip response recorder
// was active.
func (s *ErrorCodeHandlers) Register(statusCode int, handlers ...context.Handler) *ErrorCodeHandler {
	if statusCodeSuccessful(statusCode) {
		return nil
	}

	h := s.Get(statusCode)
	if h == nil {
		// create new and add it
		ch := &ErrorCodeHandler{
			StatusCode: statusCode,
			Handlers:   handlers,
		}

		s.handlers = append(s.handlers, ch)

		return ch
	}

	// otherwise update the handlers
	h.updateHandlers(handlers)
	return h
}

// Fire executes an error http status code handler
// based on the context's status code.
//
// If a handler is not already registered,
// then it creates & registers a new trivial handler on the-fly.
func (s *ErrorCodeHandlers) Fire(ctx context.Context) {
	statusCode := ctx.GetStatusCode()
	if statusCodeSuccessful(statusCode) {
		return
	}
	ch := s.Get(statusCode)
	if ch == nil {
		ch = s.Register(statusCode, statusText(statusCode))
	}
	ch.Fire(ctx)
}
