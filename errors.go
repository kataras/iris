package iris

import (
	"fmt"
	"net/http"
	"reflect"
)

// ErrorHandler creates a handler which is responsible to send a particular error to the client
func ErrorHandler(message string, errCode int) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.Header().Set("X-Content-Type-Options", "nosniff")
		res.WriteHeader(errCode)
		fmt.Fprintln(res, message)
	})
}

// ErrorHandlers just a dictionary map[int] http.Handler
type ErrorHandlers map[int]http.Handler

// HTTPErrors is the struct which contains the handlers which will execute if http error occurs
// One struct per Server instance, the meaning of this is that the developer can change the default error message and replace them with his/her own completely custom handlers
type HTTPErrors struct {
	//developer can do Errors.On(500, a  http.Handler || func(res,req)
	errorHanders ErrorHandlers
}

// DefaultHTTPErrors creates and returns an instance of HTTPErrors with default handlers
func DefaultHTTPErrors() *HTTPErrors {
	httperrors := new(HTTPErrors)
	httperrors.errorHanders = make(ErrorHandlers, 0)
	httperrors.SetNotFound(ErrorHandler("404 not found", http.StatusNotFound))
	httperrors.SetMethodNotAllowed(ErrorHandler("405 method not allowed", http.StatusMethodNotAllowed))

	return httperrors
}

// On Registers a handler for a specific http error status ( overrides the NotFound and MethodNotAllowed)
// Possible parameter: http.Handler || func(http.ResponseWriter,req *http.Request)
func (he *HTTPErrors) On(httpStatus int, handler HTTPHandler) {
	if httpStatus == http.StatusOK {
		return
	}

	httpHandlerOfficialType := reflect.TypeOf((*http.Handler)(nil)).Elem()
	if !reflect.TypeOf(handler).Implements(httpHandlerOfficialType) {
		//it is not a http.Handler
		//it is func(res,req) we will convert it to a handler using http.HandlerFunc
		handler = http.HandlerFunc(handler.(func(res http.ResponseWriter, req *http.Request)))
	}
	he.errorHanders[httpStatus] = handler.(http.Handler)
}

// SetNotFound this func could named it OnNotFound also, registers a custom StatusNotFound error 404 handler
// Possible parameter: http.Handler || func(http.ResponseWriter,req *http.Request)
func (he *HTTPErrors) SetNotFound(handler HTTPHandler) {
	he.On(http.StatusNotFound, handler)
}

// SetMethodNotAllowed this func could named it OnMethodNotAllowed also, registers a custom StatusMethodNotAllowed error 405 handler
func (he *HTTPErrors) SetMethodNotAllowed(handler HTTPHandler) {
	he.On(http.StatusMethodNotAllowed, handler)
}

// Emit executes the handler of the given error http status code
func (he *HTTPErrors) Emit(errCode int, res http.ResponseWriter) {
	if handler := he.errorHanders[errCode]; handler != nil {
		handler.ServeHTTP(res, nil)
	}
}

// NotFound emits the registed NotFound (404) custom (or not) handler
func (he *HTTPErrors) NotFound(res http.ResponseWriter) {
	he.Emit(http.StatusNotFound, res)
}

// MethodNotAllowed emits the registed MethodNotAllowed(405) custom (or not) handler
func (he *HTTPErrors) MethodNotAllowed(res http.ResponseWriter) {
	he.Emit(http.StatusMethodNotAllowed, res)
}
