package errors_test

import (
	"database/sql"
	stdErrors "errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
	"github.com/kataras/iris/v14/rest/errors/validation"
)

// privateAnswer is what a client may read for a private error: the status the
// handler set, its canonical code and a message that says nothing about the
// error.
type privateAnswer struct {
	status        int
	canonicalName string
	message       string
}

// expectPrivateAnswer requests path and fails when any fragment of the private
// text is in the body, or when the status, the code or the message is not the
// expected one. The body is checked first, so a failure shows what leaked.
func expectPrivateAnswer(t *testing.T, e *httptest.Expect, path string, expected privateAnswer, fragments ...string) {
	t.Helper()

	res := e.GET(path).Expect()

	body := res.Body().Raw()
	for _, fragment := range fragments {
		if strings.Contains(body, fragment) {
			t.Errorf("%s: the private error text %q reached the client with status %d: %s",
				path, fragment, res.Raw().StatusCode, body)
			return
		}
	}

	res.Status(expected.status)

	doc := res.JSON().Object()
	doc.Value("http_error_code").Object().
		HasValue("canonical_name", expected.canonicalName).
		HasValue("status", expected.status)
	doc.HasValue("message", expected.message)
}

var (
	unauthorizedAnswer = privateAnswer{iris.StatusUnauthorized, "UNAUTHENTICATED", "Unauthorized"}
	badRequestAnswer   = privateAnswer{iris.StatusBadRequest, "INVALID_ARGUMENT", "Bad Request"}
	forbiddenAnswer    = privateAnswer{iris.StatusForbidden, "PERMISSION_DENIED", "Forbidden"}
	internalAnswer     = privateAnswer{iris.StatusInternalServerError, "INTERNAL", "server error"}
	unavailableAnswer  = privateAnswer{iris.StatusServiceUnavailable, "UNAVAILABLE", "server error"}
)

// A stored *errors.Error is the one error Handler sends as it is on a status
// of 500 or above, because somebody built that document for the client. Marked
// private, it is not one: the mark wins and the client reads the generic
// answer with the status the handler set.
func TestHandlerPrivateWireErrorIsNotSentAsItIs(t *testing.T) {
	app, logged, _ := storedErrorApp(t, errors.NewOptions())

	app.Get("/unavailable", func(ctx iris.Context) {
		ctx.SetErr(iris.PrivateError(&errors.Error{Message: "PRIV-WIRE-5XX", Detail: "PRIV-WIRE-DETAIL"}))
		ctx.StopWithStatus(iris.StatusServiceUnavailable)
	})
	app.Get("/forbidden", func(ctx iris.Context) {
		ctx.SetErr(iris.PrivateError(&errors.Error{Message: "PRIV-WIRE-4XX", Detail: "PRIV-WIRE-DETAIL"}))
		ctx.StopWithStatus(iris.StatusForbidden)
	})

	e := httptest.New(t, app)

	expectPrivateAnswer(t, e, "/unavailable", unavailableAnswer, "PRIV-WIRE")
	if len(*logged) != 1 || !strings.Contains((*logged)[0], "PRIV-WIRE-5XX") {
		t.Fatalf("expected the configured logger to receive the private error once, got %q", *logged)
	}

	expectPrivateAnswer(t, e, "/forbidden", forbiddenAnswer, "PRIV-WIRE")
}

// A private error is not handed to the mappings. One that wraps a mapped
// error used to render through the mapping, with the mapping's status and the
// whole private text as the message. The answer is the one every private
// error gets, with the status the handler set next to it.
func TestHandlerPrivateErrorIsNotRenderedThroughAMapping(t *testing.T) {
	app, _, _ := storedErrorApp(t, errors.NewOptions().MapErrors(errors.NotFound, sql.ErrNoRows))

	private := func() error {
		return iris.PrivateError(fmt.Errorf("PRIV-MAPPED tenant db-eu-7: %w", sql.ErrNoRows))
	}

	app.Get("/unauthorized", func(ctx iris.Context) {
		ctx.StopWithError(iris.StatusUnauthorized, private())
	})
	app.Get("/internal", func(ctx iris.Context) {
		ctx.StopWithError(iris.StatusInternalServerError, private())
	})

	e := httptest.New(t, app)

	for path, expected := range map[string]privateAnswer{
		"/unauthorized": unauthorizedAnswer,
		"/internal":     internalAnswer,
	} {
		expectPrivateAnswer(t, e, path, expected, "PRIV-MAPPED", "db-eu-7", "no rows")
	}
}

// The same error, not marked private, still renders through its mapping:
// the mapping's status and code, and the error's text as the message.
func TestHandlerPublicMappedErrorStillRendersThroughItsMapping(t *testing.T) {
	app, _, _ := storedErrorApp(t, errors.NewOptions().MapErrors(errors.NotFound, sql.ErrNoRows))

	app.Get("/plan", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusUnauthorized, fmt.Errorf("plan 42: %w", sql.ErrNoRows))
	})

	e := httptest.New(t, app)

	doc := e.GET("/plan").Expect().Status(iris.StatusNotFound).JSON().Object()
	doc.Value("http_error_code").Object().
		HasValue("canonical_name", "NOT_FOUND").
		HasValue("status", iris.StatusNotFound)
	doc.HasValue("message", "plan 42: sql: no rows in result set")
}

// A mapping can still decide both things for an error that is stored as it
// is: the status, by recognising it, and that its text is not for the client,
// by returning it through errors.Private. That is the way to keep a mapping's
// status for an error whose text must stay on the server.
func TestHandlerMappingThatReturnsAPrivateErrorKeepsItsStatus(t *testing.T) {
	// An error of the test's own: NewOptions already maps sql.ErrNoRows to
	// NotFound, and the first mapping of a code that recognises an error wins.
	errPlanGone := stdErrors.New("plan row is gone")

	opts := errors.NewOptions().MapErrorFunc(errors.NotFound, func(err error) error {
		if stdErrors.Is(err, errPlanGone) {
			return errors.Private(err)
		}

		return nil
	})

	app, logged, _ := storedErrorApp(t, opts)

	app.Get("/plan", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusInternalServerError, fmt.Errorf("PRIV-BY-MAPPING tenant db-eu-7: %w", errPlanGone))
	})

	e := httptest.New(t, app)

	expectPrivateAnswer(t, e, "/plan", privateAnswer{iris.StatusNotFound, "NOT_FOUND", "server error"},
		"PRIV-BY-MAPPING", "db-eu-7", "gone")

	if len(*logged) != 1 || !strings.Contains((*logged)[0], "PRIV-BY-MAPPING") {
		t.Fatalf("expected the configured logger to receive the error once, got %q", *logged)
	}
}

// An error tagged with an ErrorCodeName is a mapping too: the tag names the
// code to render with. A private error that wraps one is still private.
func TestHandlerPrivateErrorWrappingACodeNameIsNotRendered(t *testing.T) {
	app, _, _ := storedErrorApp(t, errors.NewOptions())

	app.Get("/forbidden", func(ctx iris.Context) {
		tagged := errors.InvalidArgument.Wrap(stdErrors.New("PRIV-CODENAME secret"))
		ctx.StopWithError(iris.StatusForbidden, iris.PrivateError(tagged))
	})

	e := httptest.New(t, app)

	expectPrivateAnswer(t, e, "/forbidden", forbiddenAnswer, "PRIV-CODENAME", "secret")
}

// ErrPrivate promises the generic message whatever it is given. A
// *errors.Error used to go out as it is, message and detail included.
func TestErrPrivateDoesNotSendAWireErrorAsItIs(t *testing.T) {
	app, _, collected := storedErrorApp(t, errors.NewOptions())

	app.Get("/internal", func(ctx iris.Context) {
		errors.Internal.ErrPrivate(ctx, &errors.Error{Message: "ERRPRIVATE-WIRE", Detail: "ERRPRIVATE-DETAIL"})
	})

	e := httptest.New(t, app)

	expectPrivateAnswer(t, e, "/internal", internalAnswer, "ERRPRIVATE")

	if len(*collected) != 1 || !strings.Contains((*collected)[0], "ERRPRIVATE-WIRE") {
		t.Fatalf("expected the collector to receive the private error once, got %q", *collected)
	}
}

// The exception to the 500-and-above rule is unchanged for an error that is
// not private: a stored *errors.Error was built for the client and is sent as
// it is, with the code of the status the handler set.
func TestHandlerPublicWireErrorIsSentAsItIs(t *testing.T) {
	app, _, _ := storedErrorApp(t, errors.NewOptions())

	app.Get("/maintenance", func(ctx iris.Context) {
		ctx.SetErr(&errors.Error{Message: "down for maintenance", Detail: "back at 14:00 UTC"})
		ctx.StopWithStatus(iris.StatusServiceUnavailable)
	})

	e := httptest.New(t, app)

	doc := e.GET("/maintenance").Expect().Status(iris.StatusServiceUnavailable).JSON().Object()
	doc.Value("http_error_code").Object().
		HasValue("canonical_name", "UNAVAILABLE").
		HasValue("status", iris.StatusServiceUnavailable)
	doc.HasValue("message", "down for maintenance")
	doc.HasValue("detail", "back at 14:00 UTC")
}

// The private mark does not have to be the outermost value. errors.Private
// keeps the error's own chain and adds the mark to it, and a private error
// can be wrapped again on its way up. Both are private for Handler.
func TestHandlerErrorMarkedPrivateInItsChainIsNotRendered(t *testing.T) {
	app, _, _ := storedErrorApp(t, errors.NewOptions().MapErrors(errors.NotFound, sql.ErrNoRows))

	app.Get("/marked", func(ctx iris.Context) {
		ctx.StopWithPlainError(iris.StatusBadRequest, errors.Private(stdErrors.New("PRIV-CHAIN secret")))
	})
	app.Get("/rewrapped", func(ctx iris.Context) {
		private := iris.PrivateError(stdErrors.New("PRIV-CHAIN secret"))
		ctx.StopWithPlainError(iris.StatusBadRequest, fmt.Errorf("load: %w", private))
	})
	app.Get("/marked-mapped", func(ctx iris.Context) {
		private := errors.Private(fmt.Errorf("PRIV-CHAIN tenant db-eu-7: %w", sql.ErrNoRows))
		ctx.StopWithPlainError(iris.StatusUnauthorized, private)
	})

	e := httptest.New(t, app)

	expectPrivateAnswer(t, e, "/marked", badRequestAnswer, "PRIV-CHAIN", "secret")
	expectPrivateAnswer(t, e, "/rewrapped", badRequestAnswer, "PRIV-CHAIN", "secret")
	expectPrivateAnswer(t, e, "/marked-mapped", unauthorizedAnswer, "PRIV-CHAIN", "db-eu-7", "no rows")
}

// A validation error names the fields and the values the client sent, and it
// renders as a list of them. Stored private, or given to ErrPrivate, none of
// that is for the client.
func TestPrivateValidationErrorIsNotRendered(t *testing.T) {
	app, _, _ := storedErrorApp(t, errors.NewOptions())

	invalid := func() error {
		return validation.Join(validation.Field("PRIV-FIELD", 17).Func(validation.InRange(18, 200)))
	}

	app.Get("/stored", func(ctx iris.Context) {
		ctx.StopWithError(iris.StatusBadRequest, iris.PrivateError(invalid()))
	})
	app.Get("/err-private", func(ctx iris.Context) {
		errors.Internal.ErrPrivate(ctx, invalid())
	})

	e := httptest.New(t, app)

	expectPrivateAnswer(t, e, "/stored", badRequestAnswer, "PRIV-FIELD", "validation")
	expectPrivateAnswer(t, e, "/err-private", internalAnswer, "PRIV-FIELD", "validation")
}
