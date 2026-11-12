// PostgreSQL error mapping: one blank import turns the driver's errors into 404, 409 and 400.
//
// rest/errors does not link the PostgreSQL driver any more, so the default
// map of errors.NewOptions() covers database/sql's ErrNoRows and nothing
// else. Applications on github.com/kataras/pg add one blank import and get
// the driver mapping back: pg.ErrNoRows to 404, duplicate-key violations to
// 409 (raw error logged, never sent) and SQLSTATE-tagged input errors to 400
// with the SQLSTATE noise trimmed. The import registers an
// errors.RegisterErrorResolver in its init; nothing else in the code changes.
//
// No database is opened here: the handlers hand the driver's errors to
// errors.HandleError directly, exactly as a repository would after a failed
// query. Resolvers run inside HandleError even when no Options middleware is
// registered, so the /pg routes below carry none.
//
// Learn: map a database driver's errors to HTTP statuses with one blank import and no code changes in your handlers.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/pg/no-rows    # 404 NOT_FOUND, "resource not found"
//	curl -i http://localhost:8080/pg/duplicate  # 409 ALREADY_EXISTS, generic detail
//	curl -i http://localhost:8080/pg/syntax     # 400 INVALID_ARGUMENT, SQLSTATE trimmed
//	curl -i http://localhost:8080/sql/no-rows   # 404, the default map, no pgerrors needed
//
// Next: ../pg-crud
package main

import (
	"database/sql"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"

	// Side-effect ("blank") import: nothing is referenced from the package,
	// its init runs and registers the PostgreSQL resolver.
	_ "github.com/kataras/iris/v14/rest/errors/pgerrors"

	"github.com/kataras/pg"
)

// The pgx driver reports these two conditions as plain text with a SQLSTATE
// suffix, and the resolver classifies them by content, which is why they
// can be reproduced verbatim here without a connection.
var (
	errDuplicateEmail = errors.New(`ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)`)
	errInvalidUUID    = errors.New(`ERROR: invalid input syntax for type uuid: "not-a-uuid" (SQLSTATE 22P02)`)
)

func main() {
	app := iris.New()

	pgAPI := app.Router("/pg")

	// 404 {"message": "resource not found"}.
	pgAPI.Get("/no-rows", func(ctx iris.Context) {
		errors.HandleError(ctx, pg.ErrNoRows)
	})

	// 409 with a generic detail; the constraint and column names stay in the
	// server log instead of travelling to the client.
	pgAPI.Get("/duplicate", func(ctx iris.Context) {
		errors.HandleError(ctx, errDuplicateEmail)
	})

	// 400 {"message": "invalid input syntax for type uuid: \"not-a-uuid\""}.
	pgAPI.Get("/syntax", func(ctx iris.Context) {
		errors.HandleError(ctx, errInvalidUUID)
	})

	// The default map lives in the Options value, so this route registers the
	// middleware; the resolvers above did not need it.
	sqlAPI := app.Router("/sql")
	sqlAPI.Use(errors.NewOptions().Handler())

	// 404 without any driver import: database/sql's ErrNoRows is mapped by
	// default. Note that pgx's ErrNoRows is a proxy for this one, so under an
	// Options middleware the default map claims pg.ErrNoRows first: still a
	// 404, only with the driver's own text as the message.
	sqlAPI.Get("/no-rows", func(ctx iris.Context) {
		errors.HandleError(ctx, sql.ErrNoRows)
	})

	app.Listen(":8080")
}
