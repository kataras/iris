package hero

import (
	"reflect"
	"testing"
)

type testInterface interface {
	Get() string
}

var testInterfaceTyp = reflect.TypeOf((*testInterface)(nil)).Elem()

type testImplPtr struct{}

func (*testImplPtr) Get() string { return "get_ptr" }

type testImpl struct{}

func (testImpl) Get() string { return "get" }

func TestEqualTypes(t *testing.T) {
	of := reflect.TypeOf

	var tests = map[reflect.Type]reflect.Type{
		of("string"):         of("input"),
		of(42):               of(10),
		testInterfaceTyp:     testInterfaceTyp,
		of(new(testImplPtr)): testInterfaceTyp,
		of(new(testImpl)):    testInterfaceTyp,
		of(testImpl{}):       testInterfaceTyp,
	}

	for binding, input := range tests {
		if !equalTypes(binding, input) {
			t.Fatalf("expected type of: %s to be equal to the binded one of: %s", input, binding)
		}
	}
}
