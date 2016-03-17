package iris

///TODO: I must re-write this file

import (
	"fmt"
	"net/http"
)

// ErrorHandler creates a handler which is responsible to send a particular error to the client
func ErrorHandler(message string, errCode int) Handler {
	return HandlerFunc(func(ctx *Context) {
		ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
		ctx.ResponseWriter.Header().Set("X-Content-Type-Options", "nosniff")
		ctx.ResponseWriter.WriteHeader(errCode)
		fmt.Fprintln(ctx.ResponseWriter, message)
	})
}

// ErrorHandlers just an array of struct{ code int, handler http.Handler}
type errorHandler struct {
	code    int
	handler Handler
}

// HTTPErrors is the struct which contains the handlers which will execute if http error occurs
// One struct per Server instance, the meaning of this is that the developer can change the default error message and replace them with his/her own completely custom handlers
type HTTPErrors struct {
	//developer can do Errors.On(500, iris.Handler)
	ErrorHanders []*errorHandler
}

// DefaultHTTPErrors creates and returns an instance of HTTPErrors with default handlers
func DefaultHTTPErrors() *HTTPErrors {
	httperrors := new(HTTPErrors)
	httperrors.ErrorHanders = make([]*errorHandler, 0)
	httperrors.SetNotFound(ErrorHandler("404 not found", http.StatusNotFound))
	httperrors.SetMethodNotAllowed(ErrorHandler("405 method not allowed", http.StatusMethodNotAllowed))

	return httperrors
}

func (he *HTTPErrors) getByCode(httpStatus int) *errorHandler {
	if he == nil {
		return nil
	}
	for _, h := range he.ErrorHanders {
		if h.code == httpStatus {
			return h
		}
	}
	return nil
}

// On Registers a handler for a specific http error status ( overrides the NotFound and MethodNotAllowed)
func (he *HTTPErrors) On(httpStatus int, handler Handler) {
	if httpStatus == http.StatusOK {
		return
	}

	/*	httpHandlerOfficialType := reflect.TypeOf((*http.Handler)(nil)).Elem()
		if !reflect.TypeOf(handler).Implements(httpHandlerOfficialType) {
			//it is not a http.Handler
			//it is func(res,req) we will convert it to a handler using http.HandlerFunc
			handler = ToHandlerFunc(handler.(func(res http.ResponseWriter, req *http.Request)))
		}
	*/
	if errH := he.getByCode(httpStatus); errH != nil {
		errH.handler = handler
	} else {
		he.ErrorHanders = append(he.ErrorHanders, &errorHandler{code: httpStatus, handler: handler})
	}

}

// SetNotFound this func could named it OnNotFound also, registers a custom StatusNotFound error 404 handler
// Possible parameter: iris.Handler or iris.HandlerFunc(func(ctx *Context){})
func (he *HTTPErrors) SetNotFound(handler Handler) {
	he.On(http.StatusNotFound, handler)
}

// SetMethodNotAllowed this func could named it OnMethodNotAllowed also, registers a custom StatusMethodNotAllowed error 405 handler
func (he *HTTPErrors) SetMethodNotAllowed(handler Handler) {
	he.On(http.StatusMethodNotAllowed, handler)
}

// Emit executes the handler of the given error http status code
func (he *HTTPErrors) Emit(errCode int, res http.ResponseWriter) {
	if errHandler := he.getByCode(errCode); errHandler != nil {
		errHandler.handler.Serve(&Context{ResponseWriter: res})

	}
}

// EmitWithContext executes the handler of the given error code to the context's response writer
func (he *HTTPErrors) EmitWithContext(errCode int, ctx *Context) {
	if errHandler := he.getByCode(errCode); errHandler != nil {
		errHandler.handler.Serve(ctx)

	}
}

// NotFound emits the registed NotFound (404) custom (or not) handler
func (he *HTTPErrors) NotFound(res http.ResponseWriter) {
	he.Emit(http.StatusNotFound, res)
}
