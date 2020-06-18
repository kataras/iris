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

func TestErrorAs(t *testing.T) {
	testErr := errors.New("as")
	err := &Error{Err: testErr}
	if expected, got := true, errors.As(err, &testErr); expected != got {
		t.Fatalf("[testErr as err] expected %v but got %v", expected, got)
	}
	if expected, got := true, errors.As(testErr, &err); expected != got {
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
		expectedEntries := 4
		_ = Walk(g, func(typ interface{}, err error) {
			g.IncludeChildren = false
			childAPIErrorsGroup.IncludeChildren = false
			childAPIErrorsGroup2.IncludeChildren = false
			childAPIErrorsGroup2Group1.IncludeChildren = false

			expectedEntries--
			var expected string

			switch typ {
			case apiErrorsType:
				expected = apiErrorsText
			case childAPIErrorsType:
				expected = childAPIErrorsText
			case childAPIErrors2Type:
				expected = childAPIErrors2Text
			case childAPIErrors2Type1:
				expected = childAPIErrors2Text1
			}

			if got := err.Error(); expected != got {
				t.Fatalf("[%v] expected '%s' but got '%s'", typ, expected, got)
			}
		})

		if expectedEntries != 0 {
			t.Fatalf("not valid number of errors [...%d]", expectedEntries)
		}

		g.IncludeChildren = true
		childAPIErrorsGroup.IncludeChildren = true
		childAPIErrorsGroup2.IncludeChildren = true
		childAPIErrorsGroup2Group1.IncludeChildren = true
	})
}
