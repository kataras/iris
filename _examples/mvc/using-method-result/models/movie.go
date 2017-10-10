// file: models/movie.go

package models

import "github.com/kataras/iris/context"

// Movie is our sample data structure.
type Movie struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Year   int    `json:"year"`
	Genre  string `json:"genre"`
	Poster string `json:"poster"`
}

// Dispatch completes the `kataras/iris/mvc#Result` interface.
// Sends a `Movie` as a controlled http response.
// If its ID is zero or less then it returns a 404 not found error
// else it returns its json representation,
// (just like the controller's functions do for custom types by default).
//
// Don't overdo it, the application's logic should not be here.
// It's just one more step of validation before the response,
// simple checks can be added here.
//
// It's just a showcase,
// imagine what possible this opens when you designing a bigger application.
//
// This is called where the return value from a controller's method functions
// is type of `Movie`.
// For example the `controllers/movie_controller.go#GetBy`.
func (m Movie) Dispatch(ctx context.Context) {
	if m.ID <= 0 {
		ctx.NotFound()
		return
	}
	ctx.JSON(m, context.JSON{Indent: " "})
}

// For those who wonder `iris.Context`(go 1.9 type alias feature) and
// `context.Context` is the same exact thing.
