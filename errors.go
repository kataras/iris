package iris

import (
	"net/http"
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
	NotFound         http.Handler
	MethodNotAllowed http.Handler

	// other errors, developer can do Errors.On(500, new http.Handler here...)
	other map[int]http.Handler
}

// DefaultHTTPErrors creates and returns an instance of HTTPErrors with default handlers
func DefaultHTTPErrors() *HTTPErrors {
	httperrors := new(HTTPErrors)
	httperrors.NotFound = http.NotFoundHandler()
	httperrors.MethodNotAllowed = http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "405 method not allowed", http.StatusMethodNotAllowed)
	})
	httperrors.other = make(map[int]http.Handler,0)
	return httperrors

}

// On Registers a handler for a specific http error status ( overrides the NotFound and MethodNotAllowed)
func (he *HTTPErrors) On(httpStatus int, handler http.Handler) {
	he.other[httpStatus] = handler
	// if one of the pre-defined errors setted here then replace them too
	if httpStatus == http.StatusNotFound {
		he.NotFound = handler
	} else if httpStatus == http.StatusMethodNotAllowed {
		he.MethodNotAllowed = handler
	}
}
