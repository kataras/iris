package examples

import (
	"net/http"

	"google.golang.org/appengine"
)

// GaeHandler creates http.Handler to run in the Google App Engine.
//
// Routes:
//  GET /ping   return "pong"
func GaeHandler() http.Handler {
	m := http.NewServeMux()

	m.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_ = appengine.NewContext(r)
		w.Write([]byte("pong"))
	})

	return m
}
