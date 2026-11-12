package context_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

type shelfInput struct {
	Name string `json:"name"`
}

func (in shelfInput) Validate() error {
	if in.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

type pagesInput struct {
	Pages int `json:"pages"`
}

// Pointer-receiver Validate must be discovered too.
func (in *pagesInput) Validate() error {
	return validation.Join(
		validation.Number("pages", in.Pages).Positive(),
	)
}

type alwaysInvalid struct{}

func (alwaysInvalid) Validate() error {
	return fmt.Errorf("always invalid")
}

func newReadJSONApp() *iris.Application {
	app := iris.New()

	app.Post("/shelves", func(ctx iris.Context) {
		in, ok := ctx.BindJSON[shelfInput]()
		if !ok {
			return
		}
		ctx.OK(in, nil)
	})

	app.Post("/books", func(ctx iris.Context) {
		in, ok := ctx.BindJSON[pagesInput]()
		if !ok {
			return
		}
		ctx.OK(in, nil)
	})

	app.Post("/optional", func(ctx iris.Context) {
		_, ok := ctx.BindJSONOptional[alwaysInvalid]()
		if !ok {
			return
		}
		ctx.WriteString("optional-ok")
	})

	app.Post("/list", func(ctx iris.Context) {
		items, ok := ctx.BindJSONList[shelfInput]()
		if !ok {
			return
		}
		ctx.OK(len(items), nil)
	})

	app.Post("/one-or-many", func(ctx iris.Context) {
		one, many, ok := ctx.BindJSONOneOrMany[shelfInput]()
		if !ok {
			return
		}
		if one != nil {
			ctx.WriteString("one:" + one.Name)
			return
		}
		ctx.Writef("many:%d", len(many))
	})

	return app
}

// ReadJSON must run the request type's Validate method after decoding and
// reject the request as a client error carrying the validation message.
func TestReadJSONRunsValidate(t *testing.T) {
	e := httptest.New(t, newReadJSONApp())

	e.POST("/shelves").WithJSON(shelfInput{Name: ""}).Expect().
		Status(httptest.StatusBadRequest).JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.InvalidArgument,
			Status:        400,
		},
		Message: "name is required",
	})
}

func TestReadJSONValidatePasses(t *testing.T) {
	e := httptest.New(t, newReadJSONApp())

	e.POST("/shelves").WithJSON(shelfInput{Name: "sci-fi"}).Expect().
		Status(httptest.StatusOK).JSON().Object().HasValue("name", "sci-fi")
}

// Field-level validation errors keep their structured formatting.
func TestReadJSONValidatePointerReceiverFieldErrors(t *testing.T) {
	e := httptest.New(t, newReadJSONApp())

	e.POST("/books").WithJSON(pagesInput{Pages: 0}).Expect().
		Status(httptest.StatusBadRequest)

	e.POST("/books").WithJSON(pagesInput{Pages: 42}).Expect().
		Status(httptest.StatusOK).JSON().Object().HasValue("pages", 42)
}

func TestReadJSONMalformedBody(t *testing.T) {
	e := httptest.New(t, newReadJSONApp())

	e.POST("/shelves").WithBytes([]byte("{")).
		WithHeader("Content-Type", "application/json").Expect().
		Status(httptest.StatusBadRequest)
}

// ReadJSONOptional: an empty body is the zero value, reported as success,
// and validation is skipped (the type here always fails validation).
func TestReadJSONOptionalEmptyBodySkipsValidation(t *testing.T) {
	e := httptest.New(t, newReadJSONApp())

	e.POST("/optional").Expect().
		Status(httptest.StatusOK).Body().IsEqual("optional-ok")
}

// ReadJSONOptional: a present body decodes and validates normally.
func TestReadJSONOptionalPresentBodyValidates(t *testing.T) {
	e := httptest.New(t, newReadJSONApp())

	e.POST("/optional").WithJSON(map[string]string{}).Expect().
		Status(httptest.StatusBadRequest)
}

// reservedNameValidator is an app-level ContextOptions.RequestValidator that
// rejects one specific value, so both outcomes can be asserted on one type.
type reservedNameValidator struct{}

func (reservedNameValidator) Struct(v any) error {
	if in, ok := v.(*shelfInput); ok && in.Name == "reserved" {
		return fmt.Errorf("validator: name is reserved")
	}
	return nil
}

// Failures of the app-level ContextOptions.RequestValidator must reject a
// BindJSON request as a client error, not be silently dropped: this is the
// ReadJSON branch (the validator runs inside ReadJSON, before the payload's
// own Validate), distinct from the validateDecoded path BindJSONOneOrMany
// takes.
func TestBindJSONRequestValidator(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Options().RequestValidator = reservedNameValidator{}
		ctx.Next()
	})
	app.Post("/shelves", func(ctx iris.Context) {
		in, ok := ctx.BindJSON[shelfInput]()
		if !ok {
			return
		}
		ctx.OK(in, nil)
	})

	e := httptest.New(t, app)

	// Passes its own Validate (non-empty) but fails the RequestValidator.
	e.POST("/shelves").WithJSON(shelfInput{Name: "reserved"}).Expect().
		Status(httptest.StatusBadRequest).JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.InvalidArgument,
			Status:        400,
		},
		Message: "validator: name is reserved",
	})

	e.POST("/shelves").WithJSON(shelfInput{Name: "allowed"}).Expect().
		Status(httptest.StatusOK).JSON().Object().HasValue("name", "allowed")
}

// BindJSONList validates every element.
func TestBindJSONListValidatesElements(t *testing.T) {
	e := httptest.New(t, newReadJSONApp())

	e.POST("/list").WithJSON([]shelfInput{{Name: "a"}, {Name: ""}}).Expect().
		Status(httptest.StatusBadRequest)

	e.POST("/list").WithJSON([]shelfInput{{Name: "a"}, {Name: "b"}}).Expect().
		Status(httptest.StatusOK)
}

// BindJSONOneOrMany reports an object through the first value and an array
// through the second.
func TestBindJSONOneOrMany(t *testing.T) {
	e := httptest.New(t, newReadJSONApp())

	e.POST("/one-or-many").WithJSON(shelfInput{Name: "a"}).Expect().
		Status(httptest.StatusOK).Body().IsEqual("one:a")

	e.POST("/one-or-many").WithJSON([]shelfInput{{Name: "a"}, {Name: "b"}}).Expect().
		Status(httptest.StatusOK).Body().IsEqual("many:2")
}
