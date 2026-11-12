package pgerrors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
	_ "github.com/kataras/iris/v14/rest/errors/pgerrors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kataras/pg"
)

func newPgErrorsApp() *iris.Application {
	app := iris.New()

	app.Get("/norows", func(ctx iris.Context) {
		errors.HandleError(ctx, fmt.Errorf("select customer: %w", pg.ErrNoRows))
	})

	app.Get("/duplicate", func(ctx iris.Context) {
		errors.HandleError(ctx, &pgconn.PgError{
			Severity:       "ERROR",
			Code:           "23505",
			Message:        `duplicate key value violates unique constraint "customers_email_key"`,
			ConstraintName: "customers_email_key",
		})
	})

	app.Get("/foreignkey", func(ctx iris.Context) {
		errors.HandleError(ctx, fmt.Errorf("insert order: %w", &pgconn.PgError{
			Severity:       "ERROR",
			Code:           "23503",
			Message:        `insert or update on table "orders" violates foreign key constraint "orders_customer_id_fkey"`,
			Detail:         `Key (customer_id)=(42) is not present in table "customers".`,
			TableName:      "orders",
			ConstraintName: "orders_customer_id_fkey",
		}))
	})

	app.Get("/foreignkey-unnamed", func(ctx iris.Context) {
		errors.HandleError(ctx, &pgconn.PgError{
			Severity: "ERROR",
			Code:     "23503",
			Message:  "violates foreign key constraint",
		})
	})

	app.Get("/syntax", func(ctx iris.Context) {
		errors.HandleError(ctx, fmt.Errorf(
			`ERROR: invalid input syntax for type uuid: "not-a-uuid" (SQLSTATE 22P02)`))
	})

	return app
}

// Importing pgerrors restores pg.ErrNoRows -> 404.
func TestPgErrNoRowsIsNotFound(t *testing.T) {
	e := httptest.New(t, newPgErrorsApp())

	e.GET("/norows").Expect().Status(httptest.StatusNotFound)
}

// Duplicate-key violations map to 409 with a generic client detail:
// the raw error (with schema names) must not leak.
func TestPgDuplicateIsConflictWithGenericDetail(t *testing.T) {
	e := httptest.New(t, newPgErrorsApp())

	body := e.GET("/duplicate").Expect().Status(httptest.StatusConflict).
		JSON().Object()
	body.HasValue("detail", "a resource with the given value already exists")
	body.NotContainsValue("customers_email_key")
}

// SQLSTATE-tagged syntax errors render as 400 with the SQLSTATE noise trimmed.
func TestPgSyntaxErrorTrimmed(t *testing.T) {
	e := httptest.New(t, newPgErrorsApp())

	e.GET("/syntax").Expect().Status(httptest.StatusBadRequest).
		JSON().Object().
		HasValue("message", `invalid input syntax for type uuid: "not-a-uuid"`)
}

// A foreign key violation is bad input, not a server fault: it maps to 400
// with the constraint name in the detail, so the client can tell which
// reference was wrong. The wrapped form must be found too.
func TestPgForeignKeyViolationIsInvalidArgument(t *testing.T) {
	e := httptest.New(t, newPgErrorsApp())

	body := e.GET("/foreignkey").Expect().Status(httptest.StatusBadRequest).
		JSON().Object()
	body.Value("detail").String().Contains("orders_customer_id_fkey")
	// The driver detail names the table and the offending value: it must not leak.
	body.NotContainsValue(`Key (customer_id)=(42) is not present in table "customers".`)
}

// Without a constraint name the detail stays generic instead of naming an
// empty constraint.
func TestPgForeignKeyViolationWithoutConstraintName(t *testing.T) {
	e := httptest.New(t, newPgErrorsApp())

	body := e.GET("/foreignkey-unnamed").Expect().Status(httptest.StatusBadRequest).
		JSON().Object()
	body.Value("detail").String().NotContains(`""`)
}
