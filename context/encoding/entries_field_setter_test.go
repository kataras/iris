package encoding

import (
	"errors"
	"math"
	"testing"
)

// The macro evaluator decides the runtime type of a path parameter value: an
// untyped "{id}" yields a string, "{id:int}" an int, "{id:int64}" an int64.
// The destination field type is chosen independently by the end-developer.
//
// The setter must therefore size every write by the FIELD's type. Sizing it by
// the value's type lets a 16-byte string header land on an 8-byte int64 field,
// overwriting the next field's contents with a client-controlled value.

type mixedTypesRequest struct {
	ID    int64  `param:"id"`
	Role  string `param:"role"`
	Small int8   `param:"small"`
	Flag  bool   `param:"flag"`
}

func TestPathFieldSetterRejectsMismatchedValueType(t *testing.T) {
	dec, err := NewPathParamsDecoder[mixedTypesRequest](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// An untyped "{id}" produces a string. Writing it as a string header over
	// the int64 field would clobber Role's data pointer.
	params := PathParams{
		{Key: "id", Value: "not-a-number"},
		{Key: "role", Value: "admin"},
	}

	var got mixedTypesRequest
	err = dec.DecodeEntries(params, &got)
	if err == nil {
		t.Fatal("expected a type mismatch error for a string value written to an int64 field")
	}
	if !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("expected ErrTypeMismatch, got %v", err)
	}
}

func TestPathFieldSetterNumericConversion(t *testing.T) {
	dec, err := NewPathParamsDecoder[mixedTypesRequest](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// "{id:int}" yields an int; the field is an int64. Widening is supported.
	params := PathParams{
		{Key: "id", Value: 42},
		{Key: "role", Value: "admin"},
		{Key: "small", Value: 7},
		{Key: "flag", Value: true},
	}

	var got mixedTypesRequest
	if err := dec.DecodeEntries(params, &got); err != nil {
		t.Fatal(err)
	}

	if got.ID != 42 {
		t.Errorf("ID = %d, expected 42", got.ID)
	}
	if got.Role != "admin" {
		t.Errorf("Role = %q, expected %q", got.Role, "admin")
	}
	if got.Small != 7 {
		t.Errorf("Small = %d, expected 7", got.Small)
	}
	if !got.Flag {
		t.Error("Flag = false, expected true")
	}
}

func TestPathFieldSetterRejectsOverflow(t *testing.T) {
	dec, err := NewPathParamsDecoder[mixedTypesRequest](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// An int8 field cannot hold 5000: narrowing must be reported, not truncated
	// into whichever byte fits.
	params := PathParams{{Key: "small", Value: 5000}}

	var got mixedTypesRequest
	err = dec.DecodeEntries(params, &got)
	if err == nil {
		t.Fatal("expected an out-of-range error for 5000 written to an int8 field")
	}
	if !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("expected ErrTypeMismatch, got %v", err)
	}
	if got.Small != 0 {
		t.Errorf("Small = %d, expected the field to be left untouched", got.Small)
	}
}

type unsignedRequest struct {
	Count uint32 `param:"count"`
	Name  string `param:"name"`
}

func TestPathFieldSetterRejectsNegativeForUnsigned(t *testing.T) {
	dec, err := NewPathParamsDecoder[unsignedRequest](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var got unsignedRequest
	err = dec.DecodeEntries(PathParams{{Key: "count", Value: -1}}, &got)
	if err == nil {
		t.Fatal("expected an out-of-range error for -1 written to a uint32 field")
	}
	if got.Count != 0 {
		t.Errorf("Count = %d, expected the field to be left untouched", got.Count)
	}

	// The boundary value still fits.
	got = unsignedRequest{}
	if err := dec.DecodeEntries(PathParams{{Key: "count", Value: uint64(math.MaxUint32)}}, &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != math.MaxUint32 {
		t.Errorf("Count = %d, expected %d", got.Count, uint32(math.MaxUint32))
	}
}

// A ":path" wildcard parameter arrives as a single string and is split into a
// []string field. That is the one deliberate string -> non-string conversion.
type pathSliceRequest struct {
	Parts []string `param:"parts"`
}

func TestPathFieldSetterSplitsPathIntoSlice(t *testing.T) {
	dec, err := NewPathParamsDecoder[pathSliceRequest](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var got pathSliceRequest
	if err := dec.DecodeEntries(PathParams{{Key: "parts", Value: "a/b/c"}}, &got); err != nil {
		t.Fatal(err)
	}

	if len(got.Parts) != 3 || got.Parts[0] != "a" || got.Parts[2] != "c" {
		t.Fatalf("Parts = %v, expected [a b c]", got.Parts)
	}
}
