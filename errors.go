package iris

import (
	"net/http"
	"reflect"
)

// ErrorHandler creates a handler which is responsible to send a particular error to the client
func ErrorHandler(message string, errCode ...int) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if errCode == nil {
			errCode = make([]int, 1)
			errCode[0] = 404
		}
		res.WriteHeader(errCode[0])
		res.Header().Add("Content Type", "text/html")
		res.Write([]byte(message))
	})
}

// NotFoundRoute a custom error handler for 404 not found error, it has not be used yet.
func NotFoundRoute() http.Handler {
	return ErrorHandler("<h1> Sorry the route was not found! </h1>", 404)
}

// HTTPErrors is the struct which contains the handlers which will execute if http error occurs
// One struct per Server instance, the meaning of this is that the developer can change the default error message and replace them with his/her own completely custom handlers
type HTTPErrors struct {
	//developer can do Errors.On(500, a  http.Handler || func(res,req)
	errorHander map[int]http.Handler
}

// DefaultHTTPErrors creates and returns an instance of HTTPErrors with default handlers
func DefaultHTTPErrors() *HTTPErrors {
	httperrors := new(HTTPErrors)
	httperrors.errorHander = make(map[int]http.Handler, 0)
	httperrors.NotFound(http.NotFoundHandler())
	httperrors.MethodNotAllowed(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "405 method not allowed", http.StatusMethodNotAllowed)
	}))

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
	he.errorHander[httpStatus] = handler.(http.Handler)
}

// NotFound Sets a StatusNotFound error 404 handler
// Possible parameter: http.Handler || func(http.ResponseWriter,req *http.Request)
func (he *HTTPErrors) NotFound(handler HTTPHandler) {
	he.On(http.StatusNotFound, handler)
}

// MethodNotAllowed Sets a StatusMethodNotAllowed error 405 handler
func (he *HTTPErrors) MethodNotAllowed(handler HTTPHandler) {
	he.On(http.StatusMethodNotAllowed, handler)
}
