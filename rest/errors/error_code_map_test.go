package errors

import (
	"fmt"
	"testing"
)

type myErr struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

func (e myErr) Error() string {
	return "my custom error"
}

func TestErrorCodeMap(t *testing.T) {
	var ErrDomainLevelError = fmt.Errorf("a domain error")

	m := make(ErrorCodeMap)
	m.MapErrors(Unauthenticated, ErrDomainLevelError)
	m.MapErrorTypesTransformed(InvalidArgument, myErr{})

	codeName, err := m.Filter(nil, ErrDomainLevelError)
	if codeName != Unauthenticated {
		t.Fatalf("expected: %s but got: %s\n", Unauthenticated, codeName)
	}

	if err != ErrDomainLevelError {
		t.Fatalf("expected: %s but got: %s\n", ErrDomainLevelError, err)
	}

	checkMyErr := func(codeName ErrorCodeName, err error) {
		if codeName != InvalidArgument {
			t.Fatalf("expected: %s but got: %s\n", Unauthenticated, codeName)
		}

		if asMyErr, ok := err.(myErr); ok {
			if asMyErr.Field1 != "field1" || asMyErr.Field2 != 42 {
				t.Fatalf("expected: %s but got: %s\n", myErr{"field1", 42}, asMyErr)
			}
		} else {
			t.Fatalf("expected: %s but got: %s\n", myErr{"field1", 42}, err)
		}
	}
	codeName, err = m.Filter(nil, myErr{"field1", 42})
	checkMyErr(codeName, err)

	// test wrapped custom error, because it's mapped as a type it should return only myErr.
	codeName, err = m.Filter(nil, fmt.Errorf("title error, and detail: %w", myErr{"field1", 42}))
	checkMyErr(codeName, err)

	// test simple custom error.
	codeName, err = m.Filter(nil, fmt.Errorf("a random error"))
	if string(codeName) != "" {
		t.Fatalf("expected: empty but got: %s\n", codeName)
	}

	if err.Error() != "a random error" {
		t.Fatalf("expected: %s but got: %s\n", "a random error", err.Error())
	}
}
