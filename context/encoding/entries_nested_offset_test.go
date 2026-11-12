package encoding

import (
	"testing"

	"github.com/kataras/iris/v14/core/pkg/reflex"
)

// The decoders reach every field by unsafe.Add(targetPtr, field.Offset).
// reflect reports Offset relative to the struct that DECLARES the field, but
// reflex.LookupFields flattens nested and embedded structs into a single list,
// so an inner field's offset has to be rebased onto the outermost struct before
// it can be used for pointer arithmetic.
//
// Without that rebasing, an embedded struct that is NOT the first field writes
// attacker-controlled bytes over whichever outer field happens to sit at the
// inner offset - here, over the string header of Token.

// NestedPage must be exported: an embedded field whose type is unexported has a
// non-empty PkgPath, and every decoder in this package sets SkipUnexported.
type NestedPage struct {
	Limit  int `param:"limit" url:"limit"`
	Offset int `param:"offset" url:"offset"`
}

type embeddedAfterStrings struct {
	Token string `param:"token" url:"token"`
	Name  string `param:"name" url:"name"`
	NestedPage
}

func TestLookupFieldsAbsoluteOffsets(t *testing.T) {
	typ := reflex.TypeOf(embeddedAfterStrings{})

	fields, err := reflex.LookupFieldsWithOptions(&reflex.LookupFieldsOptions{
		SkipUnexported: true,
		FieldTag:       "param",
		FlatOffsets:    true,
	}, typ)
	if err != nil {
		t.Fatal(err)
	}

	byName := make(map[string]uintptr, len(fields))
	for _, f := range fields {
		byName[f.Name] = f.Offset
	}

	// Token and Name are two string headers, so the embedded nestedPage starts
	// after both of them. Limit must therefore report a non-zero offset; a zero
	// offset means it kept the offset it had inside nestedPage and would be
	// written straight over Token.
	tokenOffset, ok := byName["Token"]
	if !ok {
		t.Fatal("expected a Token field")
	}
	limitOffset, ok := byName["Limit"]
	if !ok {
		t.Fatal("expected the embedded Limit field to be flattened into the list")
	}

	if limitOffset == tokenOffset {
		t.Fatalf("embedded Limit reports offset %d, the same as the outer Token field: "+
			"writing to it would corrupt Token's string header", limitOffset)
	}

	// Cross-check against what reflect itself reports for the same field path.
	for _, f := range fields {
		want := typ.FieldByIndex(f.Index).Offset
		var base uintptr
		if len(f.Index) > 1 {
			// Sum the offsets along the path, which is what an absolute offset means.
			base = 0
			cur := typ
			for _, idx := range f.Index {
				sf := cur.Field(idx)
				base += sf.Offset
				cur = sf.Type
			}
			want = base
		}

		if f.Offset != want {
			t.Errorf("field %s: offset = %d, expected %d (absolute, relative to the outermost struct)", f.Name, f.Offset, want)
		}
	}
}

func TestDecodeEntriesNestedStructDoesNotCorruptOuterFields(t *testing.T) {
	dec, err := NewPathParamsDecoder[embeddedAfterStrings](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	params := PathParams{
		{Key: "token", Value: "secret-token-value"},
		{Key: "name", Value: "makis"},
		{Key: "limit", Value: 25},
		{Key: "offset", Value: 50},
	}

	var got embeddedAfterStrings
	if err := dec.DecodeEntries(params, &got); err != nil {
		t.Fatal(err)
	}

	if got.Token != "secret-token-value" {
		t.Errorf("Token = %q, expected %q: the embedded struct's field was written over it", got.Token, "secret-token-value")
	}
	if got.Name != "makis" {
		t.Errorf("Name = %q, expected %q", got.Name, "makis")
	}
	if got.Limit != 25 {
		t.Errorf("Limit = %d, expected 25", got.Limit)
	}
	if got.Offset != 50 {
		t.Errorf("Offset = %d, expected 50", got.Offset)
	}
}

func TestDecodeQueryNestedStructDoesNotCorruptOuterFields(t *testing.T) {
	dec, err := NewQueryDecoder[embeddedAfterStrings](QueryDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	query := map[string][]string{
		"token":  {"secret-token-value"},
		"name":   {"makis"},
		"limit":  {"25"},
		"offset": {"50"},
	}

	var got embeddedAfterStrings
	if err := dec.DecodeKeyValues(query, &got); err != nil {
		t.Fatal(err)
	}

	if got.Token != "secret-token-value" {
		t.Errorf("Token = %q, expected %q: the embedded struct's field was written over it", got.Token, "secret-token-value")
	}
	if got.Limit != 25 {
		t.Errorf("Limit = %d, expected 25", got.Limit)
	}
	if got.Offset != 50 {
		t.Errorf("Offset = %d, expected 50", got.Offset)
	}
}

// A struct that reaches itself must not recurse until the stack overflows.
// A Go stack overflow is a fatal runtime throw: no recover, no test failure,
// the whole process dies.
type recursiveNode struct {
	Name string         `param:"name" url:"name"`
	Next *recursiveNode `param:"next" url:"next"`
}

type mutualA struct {
	Name string   `param:"name" url:"name"`
	B    *mutualB `param:"b" url:"b"`
}

type mutualB struct {
	Title string   `param:"title" url:"title"`
	A     *mutualA `param:"a" url:"a"`
}

func TestLookupFieldsRecursiveStructTerminates(t *testing.T) {
	for _, typ := range []any{recursiveNode{}, mutualA{}, mutualB{}} {
		fields, err := reflex.LookupFieldsWithOptions(&reflex.LookupFieldsOptions{
			SkipUnexported: true,
			FieldTag:       "param",
		}, reflex.TypeOf(typ))
		if err != nil {
			t.Fatalf("%T: %v", typ, err)
		}
		if len(fields) == 0 {
			t.Errorf("%T: expected at least one field", typ)
		}
	}
}

func TestNewDecoderRecursiveStructTerminates(t *testing.T) {
	if _, err := NewPathParamsDecoder[recursiveNode](PathParamsDecoderOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewQueryDecoder[mutualA](QueryDecoderOptions{}); err != nil {
		t.Fatal(err)
	}
}
