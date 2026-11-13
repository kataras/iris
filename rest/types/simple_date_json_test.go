package types_test

import (
	jsonv1 "encoding/json"
	json "encoding/json/v2"
	"testing"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"
	"github.com/kataras/iris/v14/rest/types"
)

type (
	testSimpleDateField struct {
		D types.SimpleDate `json:"d"`
	}

	testSimpleDatePtrField struct {
		D *types.SimpleDate `json:"d"`
	}

	testSimpleDateOmitEmpty struct {
		D types.SimpleDate `json:"d,omitempty"`
	}

	testSimpleDateOmitZero struct {
		D types.SimpleDate `json:"d,omitzero"`
	}

	// testSimpleDateEmbedded gets every SimpleDate method by promotion.
	testSimpleDateEmbedded struct {
		types.SimpleDate
	}
)

// testSimpleDateJSONCodecs are the three ways a SimpleDate meets JSON:
// encoding/json, encoding/json/v2, and v2 with the options the Context
// readers and writers pass.
var testSimpleDateJSONCodecs = []struct {
	name      string
	v1        bool
	unmarshal func(data []byte, ptr any) error
	marshal   func(v any) ([]byte, error)
}{
	{
		name:      "encoding/json",
		v1:        true,
		unmarshal: jsonv1.Unmarshal,
		marshal:   jsonv1.Marshal,
	},
	{
		name:      "encoding/json/v2",
		unmarshal: func(data []byte, ptr any) error { return json.Unmarshal(data, ptr) },
		marshal:   func(v any) ([]byte, error) { return json.Marshal(v) },
	},
	{
		name: "encoding/json/v2 with the framework options",
		unmarshal: func(data []byte, ptr any) error {
			return json.Unmarshal(data, ptr, jsonutil.DefaultUnmarshalOptionsSlice...)
		},
		marshal: func(v any) ([]byte, error) {
			return json.Marshal(v, jsonutil.DefaultMarshalOptionsSlice...)
		},
	},
}

func testSimpleDate(t *testing.T, s string) types.SimpleDate {
	t.Helper()

	date, err := types.ParseSimpleDate(s)
	if err != nil {
		t.Fatal(err)
	}

	return date
}

// The JSON decoding of a SimpleDate belongs to its UnmarshalJSON alone:
// a strict "2006-01-02" string, where "" and null leave the value as it was.
// UnmarshalText (issue #2486) must not take over any of it, as it would on a
// type without an UnmarshalJSON of its own.
func TestSimpleDateJSONDecodingIgnoresUnmarshalText(t *testing.T) {
	const preset = "1999-12-31"

	tests := []struct {
		in string
		// The date after decoding into a field that held "preset",
		// and into a fresh value. Ignored when fail is set.
		overPreset string
		fresh      string
		fail       bool
	}{
		{in: `"2024-09-01"`, overPreset: "2024-09-01", fresh: "2024-09-01"},
		{in: `""`, overPreset: preset, fresh: ""},
		{in: `null`, overPreset: preset, fresh: ""},
		{in: `"2024-9-1"`, fail: true},
		{in: `"2024-02-30"`, fail: true},
		{in: `"2024-09-01T00:00:00Z"`, fail: true},
		{in: `" 2024-09-01 "`, fail: true},
		{in: `"null"`, fail: true},
		{in: `"garbage"`, fail: true},
		{in: `20240901`, fail: true},
		{in: `true`, fail: true},
		{in: `{}`, fail: true},
		{in: `[]`, fail: true},
	}

	for _, codec := range testSimpleDateJSONCodecs {
		for _, tt := range tests {
			check := func(target string, err error, got types.SimpleDate, expected string) {
				t.Helper()

				if tt.fail {
					if err == nil {
						t.Errorf("%s: %s: %s: expected an error but got the date %q", codec.name, target, tt.in, got)
					}
					return
				}

				if err != nil {
					t.Errorf("%s: %s: %s: %v", codec.name, target, tt.in, err)
					return
				}

				if got.String() != expected {
					t.Errorf("%s: %s: %s: expected %q but got %q", codec.name, target, tt.in, expected, got)
				}
			}

			object := []byte(`{"d":` + tt.in + `}`)

			field := testSimpleDateField{D: testSimpleDate(t, preset)}
			err := codec.unmarshal(object, &field)
			check("field over a preset", err, field.D, tt.overPreset)
			if tt.fail && field.D.String() != preset {
				t.Errorf("%s: %s: a failed decode changed the field to %q", codec.name, tt.in, field.D)
			}

			var bare types.SimpleDate
			err = codec.unmarshal([]byte(tt.in), &bare)
			check("bare value", err, bare, tt.fresh)

			var ptrField testSimpleDatePtrField
			err = codec.unmarshal(object, &ptrField)
			if tt.in == `null` {
				if err != nil || ptrField.D != nil {
					t.Errorf("%s: pointer field: null: expected a nil pointer but got %v, %v", codec.name, ptrField.D, err)
				}
			} else {
				var got types.SimpleDate
				if ptrField.D != nil {
					got = *ptrField.D
				}
				check("pointer field", err, got, tt.fresh)
			}

			var (
				list  []types.SimpleDate
				first types.SimpleDate
			)
			err = codec.unmarshal([]byte(`[`+tt.in+`]`), &list)
			if len(list) > 0 {
				first = list[0]
			}
			check("slice element", err, first, tt.fresh)

			var values map[string]types.SimpleDate
			err = codec.unmarshal([]byte(`{"k":`+tt.in+`}`), &values)
			check("map value", err, values["k"], tt.fresh)

			var embedded testSimpleDateEmbedded
			err = codec.unmarshal([]byte(tt.in), &embedded)
			check("embedded", err, embedded.SimpleDate, tt.fresh)
		}
	}
}

// The JSON encoding of a SimpleDate belongs to its MarshalJSON alone:
// a "2006-01-02" string, and "" for the zero date. MarshalText (issue #2486)
// must not take over any of it.
func TestSimpleDateJSONEncodingIgnoresMarshalText(t *testing.T) {
	var (
		date = testSimpleDate(t, "2024-09-01")
		zero types.SimpleDate
	)

	tests := []struct {
		name     string
		value    any
		expected string
		// What encoding/json writes when it differs from encoding/json/v2.
		expectedV1 string
	}{
		{name: "bare", value: date, expected: `"2024-09-01"`},
		{name: "bare zero", value: zero, expected: `""`},
		{name: "pointer", value: &date, expected: `"2024-09-01"`},
		{name: "pointer to zero", value: &zero, expected: `""`},
		{name: "field", value: testSimpleDateField{D: date}, expected: `{"d":"2024-09-01"}`},
		{name: "field zero", value: testSimpleDateField{}, expected: `{"d":""}`},
		{name: "pointer to struct", value: &testSimpleDateField{D: date}, expected: `{"d":"2024-09-01"}`},
		{name: "pointer field", value: testSimpleDatePtrField{D: &date}, expected: `{"d":"2024-09-01"}`},
		{name: "pointer field to zero", value: testSimpleDatePtrField{D: &zero}, expected: `{"d":""}`},
		{name: "nil pointer field", value: testSimpleDatePtrField{}, expected: `{"d":null}`},
		{name: "omitempty", value: testSimpleDateOmitEmpty{D: date}, expected: `{"d":"2024-09-01"}`},
		// encoding/json has no "empty" for a struct kind, v2 omits the "" it would write.
		{name: "omitempty zero", value: testSimpleDateOmitEmpty{}, expected: `{}`, expectedV1: `{"d":""}`},
		{name: "omitzero", value: testSimpleDateOmitZero{D: date}, expected: `{"d":"2024-09-01"}`},
		{name: "omitzero zero", value: testSimpleDateOmitZero{}, expected: `{}`},
		{name: "embedded", value: testSimpleDateEmbedded{date}, expected: `"2024-09-01"`},
		{name: "embedded zero", value: testSimpleDateEmbedded{}, expected: `""`},
		{name: "slice", value: []types.SimpleDate{date, zero}, expected: `["2024-09-01",""]`},
		{name: "SimpleDates", value: types.SimpleDates{date}, expected: `["2024-09-01"]`},
		{name: "array", value: [1]types.SimpleDate{date}, expected: `["2024-09-01"]`},
		{name: "map value", value: map[string]types.SimpleDate{"k": date}, expected: `{"k":"2024-09-01"}`},
		{name: "inside any", value: []any{date, &date, zero}, expected: `["2024-09-01","2024-09-01",""]`},
	}

	for _, codec := range testSimpleDateJSONCodecs {
		for _, tt := range tests {
			expected := tt.expected
			if codec.v1 && tt.expectedV1 != "" {
				expected = tt.expectedV1
			}

			got, err := codec.marshal(tt.value)
			if err != nil {
				t.Errorf("%s: %s: %v", codec.name, tt.name, err)
				continue
			}

			if string(got) != expected {
				t.Errorf("%s: %s: expected %s but got %s", codec.name, tt.name, expected, got)
			}
		}
	}
}
