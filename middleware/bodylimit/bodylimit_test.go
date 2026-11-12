package bodylimit_test

import (
	"encoding/json"
	stdErrors "errors"
	"io"
	nethttptest "net/http/httptest"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/bodylimit"
	"github.com/kataras/iris/v14/rest/errors"
)

func TestBodyLimit(t *testing.T) {
	limit := int64(10) // Set the limit to 10 bytes

	handler := func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			ctx.StopWithPlainError(iris.StatusInternalServerError, err)
			return
		}
		ctx.Write(body)
	}

	app := iris.New()
	app.Use(bodylimit.New(limit))
	app.Post("/", handler)

	e := httptest.New(t, app)

	// Test with a body that is smaller than the limit.
	e.POST("/").WithText("123456789").Expect().Status(iris.StatusOK).Body().IsEqual("123456789")

	// Test with a body that is equal to the limit.
	e.POST("/").WithText("1234567890").Expect().Status(iris.StatusOK).Body().IsEqual("1234567890")

	// Test with a body that is bigger than the limit.
	e.POST("/").WithText("12345678910").Expect().Status(iris.StatusRequestEntityTooLarge)
}

// A body past the limit is one condition, found at one of two moments: before
// the handler runs when the request declares its length, or while the handler
// reads when it does not (a chunked upload). A client must not be able to
// tell which: the status, the code and the message are the same.
func TestBodyLimitAnswersTheSameOnBothPaths(t *testing.T) {
	const (
		limit   = 10
		tooLong = "12345678910"
	)

	var (
		limited []bool
		stored  []error
	)

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.Next()

		if err := ctx.GetErr(); err != nil {
			stored = append(stored, err)
		}
	})
	app.UseRouter(errors.Handler)
	app.Use(bodylimit.New(limit))
	app.Post("/", func(ctx iris.Context) {
		limited = append(limited, bodylimit.IsBodyLimited(ctx))

		body, err := ctx.Body()
		if err != nil {
			errors.HandleRequestError(ctx, err)
			return
		}

		ctx.Write(body)
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	defer srv.Close()

	type answer struct {
		Status        int
		CanonicalName string
		Message       string
		Detail        string
	}

	post := func(body io.Reader) answer {
		t.Helper()

		res, err := srv.Client().Post(srv.URL, "text/plain", body)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		var doc struct {
			ErrorCode struct {
				CanonicalName string `json:"canonical_name"`
			} `json:"http_error_code"`
			Message string `json:"message"`
			Detail  string `json:"detail"`
		}

		raw, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("expected the rest/errors envelope, got %q: %v", raw, err)
		}

		return answer{res.StatusCode, doc.ErrorCode.CanonicalName, doc.Message, doc.Detail}
	}

	expected := answer{
		Status:        iris.StatusRequestEntityTooLarge,
		CanonicalName: "TOO_LARGE",
		Message:       "request body too large",
	}

	// A *strings.Reader has a length net/http knows, so the request declares
	// it and the middleware refuses it before the handler runs.
	if got := post(strings.NewReader(tooLong)); got != expected {
		t.Errorf("declared length: expected %+v, got %+v", expected, got)
	}

	if len(limited) != 0 {
		t.Fatalf("declared length: the handler ran for a request the middleware refused")
	}

	if len(stored) != 1 || !stdErrors.Is(stored[0], context.ErrRequestEntityTooLarge) {
		t.Errorf("declared length: expected the stored error to be context.ErrRequestEntityTooLarge, got %v", stored)
	} else if !stdErrors.Is(stored[0], context.ErrReadRequestBody) {
		// What it was before, for code that already checks for it.
		t.Errorf("declared length: expected the stored error to still be context.ErrReadRequestBody, got %v", stored[0])
	}

	// The same bytes behind a reader of unknown length go out chunked, and the
	// limit is found while the handler reads.
	if got := post(io.NopCloser(strings.NewReader(tooLong))); got != expected {
		t.Errorf("no declared length: expected %+v, got %+v", expected, got)
	}

	if len(limited) != 1 || !limited[0] {
		t.Fatalf("no declared length: expected the handler to run once with a limited body, got %v", limited)
	}
}
