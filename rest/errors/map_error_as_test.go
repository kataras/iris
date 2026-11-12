package errors_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// quotaError is a typed error with fields worth reporting to the client.
type quotaError struct {
	Limit int
}

func (e *quotaError) Error() string { return fmt.Sprintf("quota of %d exceeded", e.Limit) }

func TestMapErrorAsFilter(t *testing.T) {
	m := make(errors.ErrorCodeMap)
	m.MapErrorAs(errors.ResourceExhausted, func(e *quotaError) *errors.Error {
		if e.Limit == 0 {
			return nil // not handled, fall through to the next mapping.
		}

		return &errors.Error{
			Message: "quota exceeded",
			Detail:  fmt.Sprintf("the limit is %d", e.Limit),
		}
	})

	// A wrapped typed error is found through errors.As.
	codeName, mapped := m.Filter(nil, fmt.Errorf("create: %w", &quotaError{Limit: 10}))
	if codeName != errors.ResourceExhausted {
		t.Fatalf("expected: %s but got: %s", errors.ResourceExhausted, codeName)
	}

	restErr, ok := mapped.(*errors.Error)
	if !ok {
		t.Fatalf("expected a *errors.Error but got: %T", mapped)
	}

	if expected, got := "the limit is 10", restErr.Detail; expected != got {
		t.Fatalf("expected: %s but got: %s", expected, got)
	}

	// A nil render result falls through: no mapping.
	if codeName, _ := m.Filter(nil, &quotaError{Limit: 0}); codeName != "" {
		t.Fatalf("expected no mapping but got: %s", codeName)
	}

	// Another type is untouched.
	if codeName, _ := m.Filter(nil, fmt.Errorf("something else")); codeName != "" {
		t.Fatalf("expected no mapping but got: %s", codeName)
	}
}

func TestMapErrorAsOptionsHTTP(t *testing.T) {
	app := iris.New()

	// The method chains like every other mapping method, and E is inferred
	// from the render function.
	opts := errors.NewOptions().
		MapErrors(errors.NotFound, sql.ErrNoRows).
		MapErrorAs(errors.ResourceExhausted, func(e *quotaError) *errors.Error {
			return &errors.Error{Message: "quota exceeded", Detail: e.Error()}
		})
	app.Use(opts.Handler())

	app.Get("/create", func(ctx iris.Context) {
		errors.HandleError(ctx, fmt.Errorf("create: %w", &quotaError{Limit: 3}))
	})

	app.Get("/missing", func(ctx iris.Context) {
		errors.HandleError(ctx, fmt.Errorf("load: %w", sql.ErrNoRows))
	})

	e := httptest.New(t, app)
	// The mapping registered before MapErrorAs in the chain is still there.
	e.GET("/missing").Expect().Status(httptest.StatusNotFound)
	e.GET("/create").Expect().Status(httptest.StatusTooManyRequests).
		JSON().Object().ContainsSubset(errors.Error{
		ErrorCode: errors.ErrorCode{
			CanonicalName: errors.ResourceExhausted,
			Status:        httptest.StatusTooManyRequests,
		},
		Message: "quota exceeded",
		Detail:  "quota of 3 exceeded",
	})
}

func TestMapErrorContains(t *testing.T) {
	m := make(errors.ErrorCodeMap)
	m.MapErrorContains(errors.InvalidArgument, "violates check constraint", "invalid value")

	codeName, mapped := m.Filter(nil, fmt.Errorf(`ERROR: new row for relation "users" violates check constraint "age_positive"`))
	if codeName != errors.InvalidArgument {
		t.Fatalf("expected: %s but got: %s", errors.InvalidArgument, codeName)
	}

	if expected, got := "invalid value", mapped.Error(); expected != got {
		t.Fatalf("expected: %s but got: %s", expected, got)
	}

	// Case-sensitive: a different case does not match.
	if codeName, _ := m.Filter(nil, fmt.Errorf("VIOLATES CHECK CONSTRAINT")); codeName != "" {
		t.Fatalf("expected no mapping but got: %s", codeName)
	}
}

// A nil render classifies without transforming; E has to be named because
// there is nothing left to infer it from.
func TestMapErrorAsNilRender(t *testing.T) {
	original := fmt.Errorf("create: %w", &quotaError{Limit: 10})

	check := func(t *testing.T, m errors.ErrorCodeMap) {
		t.Helper()

		codeName, mapped := m.Filter(nil, original)
		if codeName != errors.ResourceExhausted {
			t.Fatalf("expected: %s but got: %s", errors.ResourceExhausted, codeName)
		}

		if mapped != original {
			t.Fatalf("expected the original error to pass through, got: %v", mapped)
		}

		// Another type is untouched.
		if codeName, _ := m.Filter(nil, fmt.Errorf("something else")); codeName != "" {
			t.Fatalf("expected no mapping but got: %s", codeName)
		}
	}

	t.Run("ErrorCodeMap", func(t *testing.T) {
		m := make(errors.ErrorCodeMap)
		m.MapErrorAs[*quotaError](errors.ResourceExhausted, nil)

		check(t, m)
	})

	t.Run("Options", func(t *testing.T) {
		opts := errors.NewOptions()
		opts.MapErrorAs[*quotaError](errors.ResourceExhausted, nil)

		check(t, opts.ErrorCodeMap())
	})
}

// The zero Options has no code map yet: MapErrorAs creates it, like the other
// mapping methods do.
func TestOptionsMapErrorAsZeroValue(t *testing.T) {
	var opts errors.Options
	opts.MapErrorAs[*quotaError](errors.ResourceExhausted, nil) // must not panic on the nil internal map.

	codeName, _ := opts.ErrorCodeMap().Filter(nil, &quotaError{Limit: 1})
	if codeName != errors.ResourceExhausted {
		t.Fatalf("expected: %s but got: %s", errors.ResourceExhausted, codeName)
	}
}

// An empty substring registers nothing: a missing constant must not turn into
// a mapping which answers every error.
func TestMapErrorContainsEmptySubstring(t *testing.T) {
	m := make(errors.ErrorCodeMap)
	m.MapErrorContains(errors.InvalidArgument, "", "invalid value")

	if codeName, _ := m.Filter(nil, fmt.Errorf("anything at all")); codeName != "" {
		t.Fatalf("expected no mapping but got: %s", codeName)
	}
}

// An empty message keeps the original error text.
func TestMapErrorContainsEmptyMessage(t *testing.T) {
	m := make(errors.ErrorCodeMap)
	m.MapErrorContains(errors.InvalidArgument, "violates", "")

	original := fmt.Errorf("row violates a constraint")

	codeName, mapped := m.Filter(nil, original)
	if codeName != errors.InvalidArgument {
		t.Fatalf("expected: %s but got: %s", errors.InvalidArgument, codeName)
	}

	if mapped != original {
		t.Fatalf("expected the original error to pass through, got: %v", mapped)
	}
}

func TestOptionsMapErrorContainsZeroValue(t *testing.T) {
	var opts errors.Options
	opts.MapErrorContains(errors.InvalidArgument, "boom", "invalid value") // must not panic on the nil internal map.

	codeName, _ := opts.ErrorCodeMap().Filter(nil, fmt.Errorf("boom"))
	if codeName != errors.InvalidArgument {
		t.Fatalf("expected: %s but got: %s", errors.InvalidArgument, codeName)
	}
}

// AsType forwards the standard errors.AsType, so a file that imports this
// package as "errors" needs no second import for it.
func TestAsType(t *testing.T) {
	wrapped := fmt.Errorf("create: %w", &quotaError{Limit: 7})

	quota, ok := errors.AsType[*quotaError](wrapped)
	if !ok {
		t.Fatal("expected the wrapped *quotaError to be found")
	}

	if quota.Limit != 7 {
		t.Fatalf("expected the found error to be the wrapped one, got limit %d", quota.Limit)
	}

	if found, ok := errors.AsType[*quotaError](fmt.Errorf("something else")); ok || found != nil {
		t.Fatalf("expected no match for another error, got %v", found)
	}

	if _, ok := errors.AsType[*quotaError](nil); ok {
		t.Fatal("expected no match for a nil error")
	}
}
