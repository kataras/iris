package types_test

import (
	"encoding"
	"testing"

	"github.com/kataras/iris/v14/rest/types"
)

var (
	_ encoding.TextUnmarshaler = (*types.SimpleDate)(nil)
	_ encoding.TextMarshaler   = types.SimpleDate{}
)

// UnmarshalText reads what the JSON decoding reads, without the quotes:
// a strict "2006-01-02".
// Issue #2486.
func TestSimpleDateUnmarshalText(t *testing.T) {
	tests := []struct {
		in       string
		expected string
		fail     bool
	}{
		{in: "2024-09-01", expected: "2024-09-01"},
		{in: "2024-02-29", expected: "2024-02-29"},
		{in: "2024-9-1", fail: true},
		{in: "2024/09/01", fail: true},
		{in: "01.09.2024", fail: true},
		{in: "2024-02-30", fail: true},
		{in: "2024-09-01T00:00:00Z", fail: true},
		{in: " 2024-09-01", fail: true},
		{in: `"2024-09-01"`, fail: true},
		{in: "null", fail: true},
	}

	const preset = "1999-12-31"

	for _, tt := range tests {
		date := testSimpleDate(t, preset)
		err := date.UnmarshalText([]byte(tt.in))

		if tt.fail {
			if err == nil {
				t.Errorf("%q: expected an error but got the date %q", tt.in, date)
			} else if date.String() != preset {
				t.Errorf("%q: a failed parse changed the date to %q", tt.in, date)
			}
			continue
		}

		if err != nil {
			t.Errorf("%q: %v", tt.in, err)
			continue
		}

		if date.String() != tt.expected {
			t.Errorf("%q: expected %q but got %q", tt.in, tt.expected, date)
		}
	}

	// An empty text is not an error and leaves the date as it was,
	// the way UnmarshalJSON treats "".
	for _, empty := range [][]byte{nil, {}} {
		date := testSimpleDate(t, preset)
		if err := date.UnmarshalText(empty); err != nil {
			t.Fatalf("empty text: %v", err)
		}

		if date.String() != preset {
			t.Fatalf("empty text: expected the date to stay %q but got %q", preset, date)
		}

		var zero types.SimpleDate
		if err := zero.UnmarshalText(empty); err != nil || !zero.IsZero() {
			t.Fatalf("empty text: expected a zero date and no error but got %q, %v", zero, err)
		}
	}
}

// MarshalText writes what UnmarshalText reads, and nothing for the zero date.
func TestSimpleDateMarshalText(t *testing.T) {
	date := testSimpleDate(t, "2024-09-01")

	text, err := date.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if string(text) != "2024-09-01" {
		t.Fatalf("expected %q but got %q", "2024-09-01", text)
	}

	var back types.SimpleDate
	if err = back.UnmarshalText(text); err != nil {
		t.Fatal(err)
	}

	if !back.Equal(date) {
		t.Fatalf("expected %q back but got %q", date, back)
	}

	text, err = types.SimpleDate{}.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if len(text) != 0 {
		t.Fatalf("expected no text for the zero date but got %q", text)
	}

	if err = back.UnmarshalText(text); err != nil {
		t.Fatalf("the text of the zero date does not read back: %v", err)
	}
}

// A map key is the one place where encoding/json asks for the text methods
// and never for MarshalJSON and UnmarshalJSON. Without them it could neither
// write nor read a map keyed by SimpleDate, while encoding/json/v2 always
// could, through the JSON methods. The two now agree.
func TestSimpleDateJSONMapKey(t *testing.T) {
	date := testSimpleDate(t, "2024-09-01")

	for _, codec := range testSimpleDateJSONCodecs {
		got, err := codec.marshal(map[types.SimpleDate]int{date: 1})
		if err != nil {
			t.Errorf("%s: marshal: %v", codec.name, err)
		} else if string(got) != `{"2024-09-01":1}` {
			t.Errorf("%s: marshal: expected %s but got %s", codec.name, `{"2024-09-01":1}`, got)
		}

		var counts map[types.SimpleDate]int
		if err = codec.unmarshal([]byte(`{"2024-09-01":1}`), &counts); err != nil {
			t.Errorf("%s: unmarshal: %v", codec.name, err)
		} else if len(counts) != 1 || counts[date] != 1 {
			t.Errorf("%s: unmarshal: expected one entry under %q but got %v", codec.name, date, counts)
		}

		if err = codec.unmarshal([]byte(`{"2024-9-1":1}`), &counts); err == nil {
			t.Errorf("%s: unmarshal: expected an error for a key that is not a date", codec.name)
		}
	}
}
