// Package pgerrors restores the PostgreSQL-specific error mapping of the
// rest/errors package. Import it for its side effects when the application
// uses the github.com/kataras/pg (pgx) database driver:
//
//	import _ "github.com/kataras/iris/v14/rest/errors/pgerrors"
//
// Without this import the rest/errors package maps only database/sql's
// ErrNoRows, keeping the PostgreSQL driver out of the application's
// dependency graph.
package pgerrors

import (
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kataras/pg"
)

// foreignKeyViolation is the SQLSTATE class 23 code PostgreSQL reports when a
// row references a key which is not there, or when a referenced row is
// deleted while a child row still points at it.
const foreignKeyViolation = "23503"

func init() {
	errors.RegisterErrorResolver(Resolve)
}

// Resolve maps PostgreSQL driver errors to canonical REST errors:
// pg.ErrNoRows to NotFound, duplicate-key violations to AlreadyExists,
// foreign-key violations (SQLSTATE 23503) to InvalidArgument with the
// constraint name in the detail and SQLSTATE-tagged input errors to
// InvalidArgument with the SQLSTATE noise trimmed.
// It reports whether it handled the error.
func Resolve(ctx *context.Context, err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, pg.ErrNoRows) {
		errors.NotFound.Message(ctx, "resource not found")
		return true
	}

	if _, ok := pg.IsErrDuplicate(err); ok {
		// Log the raw database error server-side for diagnostics, but never
		// forward it to the client: pgx duplicate-key errors leak schema
		// details (table, column and constraint names, and the offending value).
		errors.LogError(ctx, err)
		errors.AlreadyExists.Detail(ctx, "already exists",
			"a resource with the given value already exists")
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		// A foreign key violation is bad input, not a server fault: the client
		// referenced something which does not exist. Only the constraint name
		// is forwarded, so the client can tell which reference was wrong; the
		// driver's own message and detail name the table, the column and the
		// offending value.
		errors.LogError(ctx, err)

		if pgErr.ConstraintName == "" {
			errors.InvalidArgument.Detail(ctx, "invalid reference",
				"a referenced resource does not exist")
			return true
		}

		errors.InvalidArgument.Detailf(ctx, "invalid reference",
			"the %q constraint is not satisfied: a referenced resource does not exist",
			pgErr.ConstraintName)
		return true
	}

	errText := err.Error()
	if strings.Contains(errText, "invalid input syntax") ||
		strings.Contains(errText, "operator does not exist") {
		errors.InvalidArgument.Message(ctx, trimSQLState(errText))
		return true
	}

	return false
}

// trimSQLState turns
// `ERROR: invalid input syntax for type uuid: "x" (SQLSTATE 22P02)`
// into `invalid input syntax for type uuid: "x"`.
func trimSQLState(errText string) string {
	errText = strings.TrimPrefix(errText, "ERROR: ")
	if before, _, found := strings.Cut(errText, " (SQLSTATE"); found {
		return before
	}

	return errText
}
