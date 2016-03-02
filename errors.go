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

// ErrorHandlers just an array of struct{ code int, handler http.Handler}
type errorHandler struct {
	code    int
	handler http.Handler
}

// HTTPErrors is the struct which contains the handlers which will execute if http error occurs
// One struct per Server instance, the meaning of this is that the developer can change the default error message and replace them with his/her own completely custom handlers
type HTTPErrors struct {
	//developer can do Errors.On(500, a  http.Handler || func(res,req)
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

	if errH := he.getByCode(httpStatus); errH != nil {
		errH.handler = handler.(http.Handler)
	} else {
		he.ErrorHanders = append(he.ErrorHanders, &errorHandler{code: httpStatus, handler: handler.(http.Handler)})
	}

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
	if errHandler := he.getByCode(errCode); errHandler != nil {

		errHandler.handler.ServeHTTP(res, nil)

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
