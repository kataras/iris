package gapi

import (
	"net/http"
)

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

func NotFoundRoute() http.Handler {
	return ErrorHandler("<h1> Sorry the route was not found! </h1>", 404)
}
