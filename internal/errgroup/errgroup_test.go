package errgroup

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrorError(t *testing.T) {
	testErr := errors.New("error")
	err := &Error{Err: testErr}
	if expected, got := testErr.Error(), err.Error(); expected != got {
		t.Fatalf("expected %s but got %s", expected, got)
	}
}

func TestErrorUnwrap(t *testing.T) {
	wrapped := errors.New("unwrap")

	err := &Error{Err: fmt.Errorf("this wraps:%w", wrapped)}
	if expected, got := wrapped, errors.Unwrap(err); expected != got {
		t.Fatalf("expected %#+v but got %#+v", expected, got)
	}

}
func TestErrorIs(t *testing.T) {
	testErr := errors.New("is")
	err := &Error{Err: fmt.Errorf("this is: %w", testErr)}
	if expected, got := true, errors.Is(err, testErr); expected != got {
		t.Fatalf("expected %v but got %v", expected, got)
	}
}

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

func TestErrorAs(t *testing.T) {
	testErr := &errorString{"as"}
	err := &Error{Err: testErr}
	if expected, got := true, errors.As(err, &testErr); expected != got {
		t.Fatalf("[testErr as err] expected %v but got %v", expected, got)
	}
	if expected, got := false, errors.As(testErr, &err); expected != got /* errorString does not implemeny As, so the std/default functionality will be applied */ {
		t.Fatalf("[err as testErr] expected %v but got %v", expected, got)
	}
}

func TestGroupError(t *testing.T) {
	g := New(0)
	tests := []string{"error 1", "error 2", "error 3"}
	for _, tt := range tests {
		g.Add(errors.New(tt))
	}

	if expected, got := strings.Join(tests, "\n"), g.Error(); expected != got {
		t.Fatalf("expected '%s' but got '%s'", expected, got)
	}
}

func TestGroup(t *testing.T) {
	const (
		apiErrorsType = iota + 1
		childAPIErrorsType
		childAPIErrors2Type  = "string type 1"
		childAPIErrors2Type1 = "string type 2"

		apiErrorsText        = "apiErrors error 1"
		childAPIErrorsText   = "apiErrors:child error 1"
		childAPIErrors2Text  = "apiErrors:child2 error 1"
		childAPIErrors2Text1 = "apiErrors:child2_1 error 1"
	)

	g := New(nil)
	apiErrorsGroup := g.Group(apiErrorsType)
	apiErrorsGroup.Errf(apiErrorsText)

	childAPIErrorsGroup := apiErrorsGroup.Group(childAPIErrorsType)
	childAPIErrorsGroup.Addf(childAPIErrorsText)
	childAPIErrorsGroup2 := apiErrorsGroup.Group(childAPIErrors2Type)
	childAPIErrorsGroup2.Addf(childAPIErrors2Text)
	childAPIErrorsGroup2Group1 := childAPIErrorsGroup2.Group(childAPIErrors2Type1)
	childAPIErrorsGroup2Group1.Addf(childAPIErrors2Text1)

	if apiErrorsGroup.Type != apiErrorsType {
		t.Fatal("invalid type")
	}

	if childAPIErrorsGroup.Type != childAPIErrorsType {
		t.Fatal("invalid type")
	}

	if childAPIErrorsGroup2.Type != childAPIErrors2Type {
		t.Fatal("invalid type")
	}

	if childAPIErrorsGroup2Group1.Type != childAPIErrors2Type1 {
		t.Fatal("invalid type")
	}

	if expected, got := 2, len(apiErrorsGroup.children); expected != got {
		t.Fatalf("expected %d but got %d", expected, got)
	}

	if expected, got := 0, len(childAPIErrorsGroup.children); expected != got {
		t.Fatalf("expected %d but got %d", expected, got)
	}

	if expected, got := 1, len(childAPIErrorsGroup2.children); expected != got {
		t.Fatalf("expected %d but got %d", expected, got)
	}

	if expected, got := 0, len(childAPIErrorsGroup2Group1.children); expected != got {
		t.Fatalf("expected %d but got %d", expected, got)
	}

	if expected, got := 1, apiErrorsGroup.index; expected != got {
		t.Fatalf("expected index %d but got %d", expected, got)
	}

	if expected, got := 2, childAPIErrorsGroup.index; expected != got {
		t.Fatalf("expected index %d but got %d", expected, got)
	}

	if expected, got := 3, childAPIErrorsGroup2.index; expected != got {
		t.Fatalf("expected index %d but got %d", expected, got)
	}

	if expected, got := 4, childAPIErrorsGroup2Group1.index; expected != got {
		t.Fatalf("expected index %d but got %d", expected, got)
	}

	t.Run("Error", func(t *testing.T) {
		if expected, got :=
			strings.Join([]string{apiErrorsText, childAPIErrorsText, childAPIErrors2Text, childAPIErrors2Text1}, delim), g.Error(); expected != got {
			t.Fatalf("expected '%s' but got '%s'", expected, got)
		}
	})

	t.Run("Walk", func(t *testing.T) {
		// Errf stores an *Error that carries the group's type. Addf stores the
		// error as it is, so Walk reports it with a nil type and its own text.
		type visit struct {
			typ  any
			text string
		}
		expected := []visit{
			{apiErrorsType, apiErrorsText},
			{nil, childAPIErrorsText},
			{nil, childAPIErrors2Text},
			{nil, childAPIErrors2Text1},
		}

		var got []visit
		_ = Walk(g, func(typ any, err error) {
			got = append(got, visit{typ, err.Error()})
		})

		if len(got) != len(expected) {
			t.Fatalf("expected %d visits but got %d: %v", len(expected), len(got), got)
		}

		for i := range expected {
			if got[i] != expected[i] {
				t.Fatalf("visit %d: expected %v but got %v", i, expected[i], got[i])
			}
		}
	})
}

// Group.Add and Group.Addf store the error as given, not as an *Error, so Walk
// must hand the visitor that entry and not the whole group.
func TestWalkPassesPlainEntries(t *testing.T) {
	plain := errors.New("plain")
	g := New("root")
	g.Add(plain)
	g.Err(errors.New("typed"))

	var got []error
	Walk(g, func(_ any, err error) {
		got = append(got, err)
	})

	if len(got) != 2 {
		t.Fatalf("expected 2 visits but got %d", len(got))
	}

	if got[0] != plain {
		t.Fatalf("expected the visitor to receive the plain entry %q but got %T %q", plain, got[0], got[0])
	}
}
