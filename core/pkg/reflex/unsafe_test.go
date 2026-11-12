package reflex

import (
	"testing"
	"unsafe"
)

type MyStruct struct {
	Field1    int
	Field2    float64
	field3    string // un-exported field
	FieldFunc func(int) int
}

func (m *MyStruct) SumFields() float64 {
	return float64(m.Field1) + m.Field2
}

func TestUnsafeCopy(t *testing.T) {
	src := MyStruct{
		Field1:    42,
		Field2:    3.14,
		field3:    "Hello",
		FieldFunc: func(x int) int { return x * 2 },
	}
	dst := MyStruct{}

	UnsafeCopy(unsafe.Pointer(&dst), unsafe.Pointer(&src), unsafe.Sizeof(src))

	if dst.Field1 != src.Field1 {
		t.Errorf("Field1: expected %d, got %d", src.Field1, dst.Field1)
	}
	if dst.Field2 != src.Field2 {
		t.Errorf("Field2: expected %f, got %f", src.Field2, dst.Field2)
	}
	if dst.field3 != src.field3 {
		t.Errorf("field3: expected %s, got %s", src.field3, dst.field3)
	}
	if dst.SumFields() != src.SumFields() {
		t.Errorf("SumFields: expected %f, got %f", src.SumFields(), dst.SumFields())
	}
	if dst.FieldFunc == nil || dst.FieldFunc(10) != src.FieldFunc(10) {
		t.Errorf("FieldFunc: expected %d, got %d", src.FieldFunc(10), dst.FieldFunc(10))
	}
}

func TestUnsafeCopyDifferentValues(t *testing.T) {
	src := MyStruct{
		Field1:    100,
		Field2:    6.28,
		field3:    "World",
		FieldFunc: func(x int) int { return x + 1 },
	}
	dst := MyStruct{}

	UnsafeCopy(unsafe.Pointer(&dst), unsafe.Pointer(&src), unsafe.Sizeof(src))

	if dst.Field1 != src.Field1 {
		t.Errorf("Field1: expected %d, got %d", src.Field1, dst.Field1)
	}
	if dst.Field2 != src.Field2 {
		t.Errorf("Field2: expected %f, got %f", src.Field2, dst.Field2)
	}
	if dst.field3 != src.field3 {
		t.Errorf("field3: expected %s, got %s", src.field3, dst.field3)
	}
	if dst.SumFields() != src.SumFields() {
		t.Errorf("SumFields: expected %f, got %f", src.SumFields(), dst.SumFields())
	}
	if dst.FieldFunc == nil || dst.FieldFunc(10) != src.FieldFunc(10) {
		t.Errorf("FieldFunc: expected %d, got %d", src.FieldFunc(10), dst.FieldFunc(10))
	}

	dst.FieldFunc = func(i int) int {
		return 5
	}
	if src.FieldFunc(10) == 5 {
		t.Errorf("FieldFunc: expected source to not be modified")
	}
}

func TestUnsafeCopyEmptyStruct(t *testing.T) {
	src := MyStruct{}
	dst := MyStruct{
		Field1:    1,
		Field2:    2.0,
		field3:    "Test",
		FieldFunc: func(x int) int { return x - 1 },
	}

	UnsafeCopy(unsafe.Pointer(&dst), unsafe.Pointer(&src), unsafe.Sizeof(src))

	if dst.Field1 != src.Field1 {
		t.Errorf("Field1: expected %d, got %d", src.Field1, dst.Field1)
	}
	if dst.Field2 != src.Field2 {
		t.Errorf("Field2: expected %f, got %f", src.Field2, dst.Field2)
	}
	if dst.field3 != src.field3 {
		t.Errorf("field3: expected %s, got %s", src.field3, dst.field3)
	}
	if dst.SumFields() != src.SumFields() {
		t.Errorf("SumFields: expected %f, got %f", src.SumFields(), dst.SumFields())
	}

	if dst.FieldFunc != nil {
		t.Errorf("FieldFunc: expected nil")
	}
}

func TestUnsafeCloneBasicFields(t *testing.T) {
	src := MyStruct{
		Field1:    42,
		Field2:    3.14,
		field3:    "Hello",
		FieldFunc: func(x int) int { return x * 2 },
	}
	dst := UnsafeClone(&src)

	if dst.Field1 != src.Field1 {
		t.Errorf("Field1: expected %d, got %d", src.Field1, dst.Field1)
	}
	if dst.Field2 != src.Field2 {
		t.Errorf("Field2: expected %f, got %f", src.Field2, dst.Field2)
	}
	if dst.field3 != src.field3 {
		t.Errorf("field3: expected %s, got %s", src.field3, dst.field3)
	}
	if dst.FieldFunc(10) != src.FieldFunc(10) {
		t.Errorf("FieldFunc: expected %d, got %d", src.FieldFunc(10), dst.FieldFunc(10))
	}
}

type MyStruct2 struct {
	Field1       int
	Field2       float64
	field3       string // un-exported field
	FieldFunc    func(int) int
	PointerField *int
}

func TestUnsafeClonePointerField(t *testing.T) {
	intValue := 100
	src := MyStruct2{
		Field1:       42,
		Field2:       3.14,
		field3:       "Hello",
		FieldFunc:    func(x int) int { return x * 2 },
		PointerField: &intValue,
	}
	dst := UnsafeClone(&src)

	if dst.PointerField == nil {
		t.Errorf("PointerField: expected non-nil, got nil")
	}
	if *dst.PointerField != *src.PointerField {
		t.Errorf("PointerField: expected %d, got %d", *src.PointerField, *dst.PointerField)
	}
}

func TestUnsafeCloneNilPointerField(t *testing.T) {
	src := MyStruct2{
		Field1:       42,
		Field2:       3.14,
		field3:       "Hello",
		FieldFunc:    func(x int) int { return x * 2 },
		PointerField: nil,
	}
	dst := UnsafeClone(&src)

	if dst.PointerField != nil {
		t.Errorf("PointerField: expected nil, got non-nil")
	}
}

func TestUnsafeCloneDifferentValues(t *testing.T) {
	intValue := 200
	src := MyStruct2{
		Field1:       100,
		Field2:       6.28,
		field3:       "World",
		FieldFunc:    func(x int) int { return x + 1 },
		PointerField: &intValue,
	}
	dst := UnsafeClone(&src)

	if dst.Field1 != src.Field1 {
		t.Errorf("Field1: expected %d, got %d", src.Field1, dst.Field1)
	}
	if dst.Field2 != src.Field2 {
		t.Errorf("Field2: expected %f, got %f", src.Field2, dst.Field2)
	}
	if dst.field3 != src.field3 {
		t.Errorf("field3: expected %s, got %s", src.field3, dst.field3)
	}
	if dst.FieldFunc(10) != src.FieldFunc(10) {
		t.Errorf("FieldFunc: expected %d, got %d", src.FieldFunc(10), dst.FieldFunc(10))
	}
	if dst.PointerField == nil {
		t.Errorf("PointerField: expected non-nil, got nil")
	}
	if *dst.PointerField != *src.PointerField {
		t.Errorf("PointerField: expected %d, got %d", *src.PointerField, *dst.PointerField)
	}
}

// more internals.

func TestUnsafeInt(t *testing.T) {
	var i int = 42
	ui := UnsafeInterfaceOf(i)
	if ui.UnsafeKind() != UnsafeIntKind {
		t.Errorf("Expected kind %v, got %v", UnsafeIntKind, ui.UnsafeKind())
	}
	if ui.ValueInt() != 42 {
		t.Errorf("Expected value %v, got %v", 42, ui.ValueInt())
	}
}

func TestUnsafeString(t *testing.T) {
	var s string = "hello"
	ui := UnsafeInterfaceOf(s)
	if ui.UnsafeKind() != UnsafeStringKind {
		t.Errorf("Expected kind %v, got %v", UnsafeStringKind, ui.UnsafeKind())
	}
	if ui.ValueString() != "hello" {
		t.Errorf("Expected value %v, got %v", "hello", ui.ValueString())
	}
}

func TestUnsafeComplex128(t *testing.T) {
	var c complex128 = complex(1, 1)
	ui := UnsafeInterfaceOf(c)
	if ui.UnsafeKind() != UnsafeComplex128Kind {
		t.Errorf("Expected kind %v, got %v", UnsafeComplex128Kind, ui.UnsafeKind())
	}
	if ui.ValueComplex128() != c {
		t.Errorf("Expected value %v, got %v", c, ui.ValueComplex128())
	}
}
