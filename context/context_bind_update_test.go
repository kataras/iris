package context_test

import (
	"errors"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

type updateMeta struct {
	Notes string `json:"notes"`
}

type updatePayload struct {
	updateMeta
	Name   string `json:"name"`
	Email  string `json:"email,omitzero"`
	Age    int
	Secret string `json:"-"`
}

// updateEcho is what the BindUpdate test handler renders back, so a test can
// assert on the decoded value and on the exact columns slice at once.
type updateEcho struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	IsNil   bool     `json:"is_nil"`
}

func newBindUpdateApp() *iris.Application {
	app := iris.New()
	app.Patch("/users/{id}", func(ctx iris.Context) {
		value, columns, ok := ctx.BindUpdate[updatePayload]()
		if !ok {
			return
		}

		ctx.JSON(updateEcho{Name: value.Name, Columns: columns, IsNil: columns == nil})
	})
	return app
}

// BindUpdate decodes the body like BindJSON and reads the columns the client
// wants updated, validating each one against the payload's JSON members.
func TestBindUpdate(t *testing.T) {
	e := httptest.New(t, newBindUpdateApp())
	body := `{"name":"makis","email":"m@example.com","Age":30,"notes":"n"}`

	// No columns at all is a full update: nil, not an empty slice.
	obj := e.PATCH("/users/42").WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).JSON().Object()
	obj.HasValue("name", "makis")
	obj.HasValue("is_nil", true)

	// Repeated key form.
	e.PATCH("/users/42").WithQuery("columns", "name").WithQuery("columns", "email").
		WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).JSON().Object().
		HasValue("columns", []string{"name", "email"})

	// One comma-separated value, with spaces, empties and duplicates.
	e.PATCH("/users/42").WithQueryString("columns=name,%20email%20,,name,notes").
		WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).JSON().Object().
		HasValue("columns", []string{"name", "email", "notes"})

	// An untagged field is addressable by its Go field name.
	e.PATCH("/users/42").WithQuery("columns", "Age").
		WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).JSON().Object().
		HasValue("columns", []string{"Age"})
}

// An unknown column is a 400 naming the column: a typo used to update nothing.
func TestBindUpdateUnknownColumn(t *testing.T) {
	e := httptest.New(t, newBindUpdateApp())
	body := `{"name":"makis"}`

	for _, column := range []string{"nmae", "secret", "-", "updateMeta"} {
		resp := e.PATCH("/users/42").WithQuery("columns", column).
			WithText(body).WithHeader("Content-Type", "application/json").
			Expect().Status(httptest.StatusBadRequest)
		resp.Body().Contains(column)
		resp.Body().Contains("name") // the allowed member names.
	}
}

// A malformed body answers exactly like BindJSON does.
func TestBindUpdateBadBody(t *testing.T) {
	e := httptest.New(t, newBindUpdateApp())
	e.PATCH("/users/42").WithQuery("columns", "name").WithText(`{"name":`).
		WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusBadRequest)
}

// The app-level request validator rejects before any column is reported.
func TestBindUpdateValidator(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Options().RequestValidator = rejectingValidator{}
		ctx.Next()
	})
	app.Patch("/users/{id}", func(ctx iris.Context) {
		if _, _, ok := ctx.BindUpdate[updatePayload](); !ok {
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.PATCH("/users/42").WithQuery("columns", "name").WithText(`{"name":"makis"}`).
		WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusBadRequest).Body().Contains("name is reserved")
}

// A pointer request type is allowed: the member names come from the struct it
// points to.
func TestBindUpdatePointerType(t *testing.T) {
	app := iris.New()
	app.Patch("/users/{id}", func(ctx iris.Context) {
		value, columns, ok := ctx.BindUpdate[*updatePayload]()
		if !ok {
			return
		}

		ctx.JSON(updateEcho{Name: value.Name, Columns: columns, IsNil: columns == nil})
	})

	e := httptest.New(t, app)
	body := `{"name":"makis"}`

	e.PATCH("/users/42").WithQuery("columns", "name").
		WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).JSON().Object().
		HasValue("columns", []string{"name"})

	e.PATCH("/users/42").WithQuery("columns", "nmae").
		WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusBadRequest).Body().Contains("nmae")
}

// partialPayload validates itself the way a persisted row does: a full update
// needs every member, which is exactly what a partial update cannot provide.
type partialPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (p *partialPayload) Validate() error {
	if p.Name == "" || p.Email == "" {
		return errors.New("name and email are required")
	}

	return nil
}

// A partial update skips the payload's own Validate: the client names the
// columns it sends and every other member is zero by design, so a whole-value
// check would reject the request for members that are not even being written.
// A full update still validates exactly like BindJSON.
func TestBindUpdateSkipsValidateOnPartialUpdate(t *testing.T) {
	app := iris.New()
	app.Patch("/users/{id}", func(ctx iris.Context) {
		value, columns, ok := ctx.BindUpdate[partialPayload]()
		if !ok {
			return
		}

		ctx.JSON(updateEcho{Name: value.Name, Columns: columns, IsNil: columns == nil})
	})

	e := httptest.New(t, app)
	body := `{"name":"makis"}`

	e.PATCH("/users/42").WithQuery("columns", "name").
		WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusOK).JSON().Object().
		HasValue("columns", []string{"name"})

	e.PATCH("/users/42").
		WithText(body).WithHeader("Content-Type", "application/json").
		Expect().Status(httptest.StatusBadRequest).Body().Contains("name and email are required")
}
