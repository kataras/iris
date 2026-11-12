package context_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"

	"uuid"
)

type bindItem struct {
	Name string `json:"name"`
}

// rejectingValidator is a ContextOptions.RequestValidator that rejects
// every payload, so a test can tell whether the validator ran at all.
type rejectingValidator struct{}

func (rejectingValidator) Struct(any) error { return errors.New("name is reserved") }

// BindJSONOneOrMany reads the raw body itself; a body-read failure is not a
// JSON decoder error, so the JSON error mapper does not recognize it and the
// caller used to continue with (nil, nil, true) and a 200.
func TestBindJSONOneOrManyOversizeBody(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Options().BodyMaxMemory = 16
		ctx.Next()
	})

	var ok bool
	app.Post("/items", func(ctx iris.Context) {
		_, _, ok = ctx.BindJSONOneOrMany[bindItem]()
		if !ok {
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	body := `[{"name":"` + strings.Repeat("x", 10*1024) + `"}]`
	e.POST("/items").WithHeader("Content-Type", "application/json").WithBytes([]byte(body)).
		Expect().Status(httptest.StatusRequestEntityTooLarge)

	if ok {
		t.Fatal("an oversize body must report ok == false")
	}
}

// The app-level RequestValidator runs on every value BindJSONOneOrMany
// decodes, for the single-object and the array shape alike.
func TestBindJSONOneOrManyRequestValidator(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Options().RequestValidator = rejectingValidator{}
		ctx.Next()
	})
	app.Post("/items", func(ctx iris.Context) {
		if _, _, ok := ctx.BindJSONOneOrMany[bindItem](); !ok {
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	for _, body := range []string{`{"name":"x"}`, `[{"name":"x"}]`} {
		e.POST("/items").WithHeader("Content-Type", "application/json").WithBytes([]byte(body)).
			Expect().Status(httptest.StatusBadRequest)
	}
}

// Body guards against a nil request body itself instead of leaving it to each
// reader: ReadMsgPack, ReadProtobuf and BindJSONOneOrMany call it directly.
func TestBodyNilRequestBody(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Request().Body = nil
		ctx.Next()
	})

	var gotErr error
	app.Post("/body", func(ctx iris.Context) {
		_, gotErr = ctx.Body()
		ctx.WriteString("done")
	})

	e := httptest.New(t, app)
	e.POST("/body").Expect().Status(httptest.StatusOK).Body().IsEqual("done")

	if !errors.Is(gotErr, context.ErrReadRequestBody) {
		t.Fatalf("expected ErrReadRequestBody, got: %v", gotErr)
	}
}

// The 400 message of a path parameter with an unexpected type names the
// parameter, never the server's Go type.
func TestBindParamUnexpectedTypeMessage(t *testing.T) {
	app := iris.New()
	app.Get("/users/{id:int}", func(ctx iris.Context) {
		if _, ok := ctx.BindParam[string]("id"); !ok {
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.GET("/users/42").Expect().Status(httptest.StatusBadRequest).
		JSON().Object().HasValue("message", "param id has an unexpected type")
}

// IsErrCanceled classifies by sentinel and type only; the old "closed pool"
// text match (a database pool sentinel from a dependency the framework no
// longer imports) is gone.
func TestIsErrCanceledNoTextMatch(t *testing.T) {
	if context.IsErrCanceled(errors.New("closed pool")) {
		t.Fatal(`IsErrCanceled must not match the "closed pool" error text`)
	}
}

// The "uuid" macro stores a uuid.UUID, so BindParam[uuid.UUID] is the typed
// read of a ":uuid" parameter. BindParam[string] on the same route is the
// documented type mismatch, a 400.
func TestBindParamUUID(t *testing.T) {
	const canonical = "a6ce7330-d584-4675-af9e-2c2d2c40de07"

	app := iris.New()
	app.Get("/typed/{id:uuid}", func(ctx iris.Context) {
		id, ok := ctx.BindParam[uuid.UUID]("id")
		if !ok {
			return
		}
		ctx.WriteString(id.String())
	})
	app.Get("/text/{id:uuid}", func(ctx iris.Context) {
		if _, ok := ctx.BindParam[string]("id"); !ok {
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.GET("/typed/A6CE7330-D584-4675-AF9E-2C2D2C40DE07").Expect().Status(httptest.StatusOK).Body().IsEqual(canonical)
	e.GET("/text/"+canonical).Expect().Status(httptest.StatusBadRequest).
		JSON().Object().HasValue("message", "param id has an unexpected type")
}
