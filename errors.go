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
