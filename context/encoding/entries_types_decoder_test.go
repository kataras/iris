package encoding

import (
	"errors"
	"testing"
	"time"

	"uuid"
)

// A "{id:uuid}" macro stores a uuid.UUID, a custom macro may store a
// time.Duration, and an untyped "{id}" stores a string. The setter accepts the
// typed value as is and parses the string form, and a string field keeps
// receiving the text of a typed value.
type richTypesRequest struct {
	ID  uuid.UUID     `param:"id" url:"id"`
	Raw string        `param:"raw" url:"raw"`
	TTL time.Duration `param:"ttl" url:"ttl"`
}

func TestPathFieldSetterRichTypesFromTypedValues(t *testing.T) {
	dec, err := NewPathParamsDecoder[richTypesRequest](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	id := uuid.MustParse(testCanonicalUUID)
	params := PathParams{
		{Key: "id", Value: id},
		{Key: "raw", Value: id}, // a string field on a uuid route.
		{Key: "ttl", Value: 90 * time.Second},
	}

	var got richTypesRequest
	if err := dec.DecodeEntries(params, &got); err != nil {
		t.Fatal(err)
	}

	if got.ID != id {
		t.Errorf("ID = %s, expected %s", got.ID, id)
	}
	if got.Raw != testCanonicalUUID {
		t.Errorf("Raw = %q, expected the canonical text %q", got.Raw, testCanonicalUUID)
	}
	if got.TTL != 90*time.Second {
		t.Errorf("TTL = %s, expected 1m30s", got.TTL)
	}
}

func TestPathFieldSetterRichTypesFromStrings(t *testing.T) {
	dec, err := NewPathParamsDecoder[richTypesRequest](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	params := PathParams{
		{Key: "id", Value: "{" + testCanonicalUUID + "}"},
		{Key: "raw", Value: "plain"},
		{Key: "ttl", Value: "1m30s"},
	}

	var got richTypesRequest
	if err := dec.DecodeEntries(params, &got); err != nil {
		t.Fatal(err)
	}

	if got.ID != uuid.MustParse(testCanonicalUUID) {
		t.Errorf("ID = %s, expected %s", got.ID, testCanonicalUUID)
	}
	if got.Raw != "plain" {
		t.Errorf("Raw = %q, expected %q", got.Raw, "plain")
	}
	if got.TTL != 90*time.Second {
		t.Errorf("TTL = %s, expected 1m30s", got.TTL)
	}
}

func TestPathFieldSetterRichTypesRejectGarbage(t *testing.T) {
	dec, err := NewPathParamsDecoder[richTypesRequest](PathParamsDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"id", "ttl"} {
		var got richTypesRequest
		err := dec.DecodeEntries(PathParams{{Key: key, Value: "garbage"}}, &got)
		if err == nil {
			t.Fatalf("%s: expected an error for an unparsable string", key)
		}
		if !errors.Is(err, ErrTypeMismatch) {
			t.Fatalf("%s: expected ErrTypeMismatch, got %v", key, err)
		}
	}
}

// ctx.ReadQuery, ReadForm and ReadHeader decode through DecodeKeyValues,
// where every value arrives as text.
func TestKeyValuesDecoderRichTypes(t *testing.T) {
	dec, err := NewQueryDecoder[richTypesRequest](QueryDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var got richTypesRequest
	err = dec.DecodeKeyValues(map[string][]string{
		"id":  {"A6CE7330-D584-4675-AF9E-2C2D2C40DE07"},
		"raw": {"plain"},
		"ttl": {"250ms"},
	}, &got)
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != uuid.MustParse(testCanonicalUUID) {
		t.Errorf("ID = %s, expected %s", got.ID, testCanonicalUUID)
	}
	if got.Raw != "plain" {
		t.Errorf("Raw = %q, expected %q", got.Raw, "plain")
	}
	if got.TTL != 250*time.Millisecond {
		t.Errorf("TTL = %s, expected 250ms", got.TTL)
	}

	for _, key := range []string{"id", "ttl"} {
		var got richTypesRequest
		if err := dec.DecodeKeyValues(map[string][]string{key: {"garbage"}}, &got); err == nil {
			t.Fatalf("%s: expected an error for an unparsable value", key)
		}
	}
}
