// errors.Router: handlers and middleware that return their errors.
//
// errors.NewRouter wraps any Iris router so every handler has the signature
// func(iris.Context) error. A returned error goes straight through
// HandleError: the same pipeline and the same central map as
// errors.HandleError, which removes the repetitive
// "if err != nil { errors.HandleError(ctx, err); return }" block.
// Use registers middleware on matched routes; UseRouter registers pre-routing
// middleware that also runs for unmatched paths under the router's prefix.
// A nil return continues the chain, a non-nil error stops the request.
//
// An ordinary iris.Handler still belongs on the wrapped router underneath,
// reachable as r.Router: the guard below is middleware/apikey, which renders
// its own 401 and has no error to return.
//
// Learn: write handlers and middleware that return error and let
// errors.NewRouter render them through the central map.
// Before: ../validation-errors
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/api/notes/1  -H "X-API-Key: secret"   # 200
//	curl -i http://localhost:8080/api/notes/42 -H "X-API-Key: secret"   # 404 NOT_FOUND
//	curl -i http://localhost:8080/api/notes/1                           # 401 UNAUTHENTICATED
//	curl -i http://localhost:8080/api/unknown  -H "X-API-Key: secret"   # 404, no route
//	curl -i -X POST http://localhost:8080/api/notes -H "X-API-Key: secret" \
//	  -d '{"text":"read the errors chapter"}'                           # 201
//	curl -i -X POST http://localhost:8080/api/notes -H "X-API-Key: secret" \
//	  -d 'nope'                                                         # 400 INVALID_ARGUMENT
//	curl -i -X POST http://localhost:8080/api/notes -H "X-API-Key: secret" \
//	  -d '{"text":""}'                                                  # 400 INVALID_ARGUMENT
//
// Next: ../../09-authentication/basicauth
package main

import (
	"fmt"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/apikey"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	notes := map[int]Note{1: {ID: 1, Text: "buy milk"}}

	app := iris.New()

	// Wrap the /api router. Everything registered on r takes error-returning
	// handlers; app itself keeps its standard signatures.
	r := errors.NewRouter(app.Router("/api"))

	// The central map has to be on the context before anything is
	// classified, so it is the first pre-routing middleware.
	// HandlerWithError is the errors.Func form of opts.Handler().
	opts := errors.NewOptions().MapErrors(errors.NotFound, ErrNoteNotFound)
	r.UseRouter(opts.HandlerWithError())

	// The key guard, pre-routing as well, so a request to an unknown /api
	// path is rejected with 401 too. It is a plain iris.Handler, hence
	// r.Router: the wrapped router is embedded and keeps its own signatures.
	// The secret belongs in the environment; it is inline here so the curl
	// lines above run as they are.
	r.Router.UseRouter(apikey.New(apikey.Options{
		Header: "X-API-Key",
		Scheme: apikey.NoScheme,
		Keys:   map[string]string{"notes-client": "secret"},
	}))

	r.Get("/notes/{id:int}", func(ctx iris.Context) error {
		id := ctx.Params().GetInt("id")

		note, ok := notes[id]
		if !ok {
			// Returned, not written: the map turns the sentinel into 404.
			return fmt.Errorf("id %d: %w", id, ErrNoteNotFound)
		}

		return ctx.JSON(note)
	})

	r.Post("/notes", func(ctx iris.Context) error {
		note, ok := ctx.BindJSON[Note]()
		if !ok {
			// ctx.BindJSON rendered the 400 through this same pipeline, so
			// there is no error left to return.
			return nil
		}

		if note.Text == "" {
			// A returned error nothing recognizes renders as 500. Wrapping
			// with errors.ErrRequest marks it as the client's fault, so this
			// renders as a 400 INVALID_ARGUMENT instead, with the wrapped
			// text as the message.
			return fmt.Errorf("%w: text is required", errors.ErrRequest)
		}

		note.ID = len(notes) + 1
		notes[note.ID] = note

		ctx.Created(note, nil)
		return nil
	})

	app.Listen(":8080")
}

// Note is the stored entity and the JSON shape.
type Note struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

// ErrNoteNotFound is the domain sentinel, mapped to 404 in main.
var ErrNoteNotFound = errors.New("note not found")
