// The central error map: domain sentinels classified in one place.
//
// The service layer returns plain sentinel errors that know nothing about
// HTTP. A single errors.NewOptions() value maps each sentinel to a canonical
// error code and app.Use(opts.Handler()) attaches that map to every request,
// so handlers only forward errors with ctx.OK or errors.HandleError. Matching
// goes through errors.Is, so wrapping a sentinel with fmt.Errorf("...: %w")
// anywhere in the call chain keeps the classification. An error the map does
// not know about renders as a generic 500 and is logged server-side.
//
// Learn: map domain sentinel errors to HTTP status codes once and forward them
// from handlers with ctx.OK, ctx.NoContent or errors.HandleError.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/books/1                  # 200
//	curl -i http://localhost:8080/books/42                 # 404 NOT_FOUND
//	curl -i -X POST http://localhost:8080/books/2/publish  # 403 PERMISSION_DENIED
//	curl -i -X POST http://localhost:8080/books/3/publish  # 204, no body
//	curl -i -X POST http://localhost:8080/books/1/publish  # 409 ALREADY_EXISTS
//	curl -i http://localhost:8080/books/1/stats            # 500 INTERNAL, generic body
//
// Next: ../handle-errorf
package main

import (
	"fmt"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	lib := library{
		1: {ID: 1, Title: "Iris in Action", Published: true},
		2: {ID: 2, Title: "Go in Depth", locked: true},
		3: {ID: 3, Title: "Iris Recipes"}, // publishable, for the 204.
	}

	// One map for the whole application. NewOptions starts with defaults
	// (sql.ErrNoRows, the kataras/jwt errors and more); every MapErrors call
	// extends the map, nothing is replaced.
	opts := errors.NewOptions().
		MapErrors(errors.NotFound, ErrBookNotFound).
		MapErrors(errors.PermissionDenied, ErrShelfLocked).
		MapErrors(errors.AlreadyExists, ErrPublished)

	app := iris.New()
	// Use covers the matched routes. UseRouter would also cover unmatched
	// paths (404) and method mismatches (405).
	app.Use(opts.Handler())

	app.Get("/books/{id:int}", func(ctx iris.Context) {
		book, err := lib.book(ctx.Params().GetInt("id"))
		ctx.OK(book, err) // 200 with the book, or the mapped error.
	})

	app.Post("/books/{id:int}/publish", func(ctx iris.Context) {
		err := lib.publish(ctx.Params().GetInt("id"))
		// ctx.NoContent carries the same map: 403, 404 or 409 for a mapped
		// error, and 204 when the publish went through.
		ctx.NoContent(true, err)
	})

	// An error the map does not know: the client gets the generic 500 body
	// ("server error") while the real text goes to the server log only.
	app.Get("/books/{id:int}/stats", func(ctx iris.Context) {
		err := errors.New("stats backend: dial tcp 10.0.0.7:9000: connect: connection refused")
		errors.HandleError(ctx, err)
	})

	app.Listen(":8080")
}

// Book is the stored entity and the JSON response shape.
type Book struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Published bool   `json:"published"`

	locked bool
}

// library stands in for a repository or service: it returns (value, error)
// pairs and never touches the Iris context.
type library map[int]*Book

func (l library) book(id int) (*Book, error) {
	book, ok := l[id]
	if !ok {
		// Wrapped with %w, so errors.Is still finds the sentinel.
		return nil, fmt.Errorf("id %d: %w", id, ErrBookNotFound)
	}

	return book, nil
}

func (l library) publish(id int) error {
	book, err := l.book(id)
	if err != nil {
		return err
	}

	if book.locked {
		return fmt.Errorf("book %q: %w", book.Title, ErrShelfLocked)
	}

	if book.Published {
		return fmt.Errorf("book %q: %w", book.Title, ErrPublished)
	}

	book.Published = true

	return nil
}

// The domain's sentinel errors: plain standard-library values (rest/errors
// re-exports errors.New) with no knowledge of HTTP.
var (
	ErrBookNotFound = errors.New("book not found")
	ErrPublished    = errors.New("book already published")
	ErrShelfLocked  = errors.New("shelf locked")
)
