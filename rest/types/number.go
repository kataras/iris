package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"math"
	"strconv"
)

// Number is a float64 which has a JSON null: NaN and the two infinities
// marshal as null and a JSON null unmarshals back to NaN. A plain float64
// cannot do that, because encoding/json refuses to encode a value which JSON
// has no literal for, and a computed average or ratio produces those values
// often enough that the whole document would fail to render.
//
// The zero value is the number zero, not null. Use NullNumber for the null
// one and IsNull to test for it.
//
// On the database side the same rule holds: a NULL column scans to NaN and a
// NaN value is written as NULL, so a float8 column and this type agree on what
// "no value" means.
type Number float64

var (
	_ json.MarshalerTo     = Number(0)
	_ json.UnmarshalerFrom = (*Number)(nil)
	// The v1 fallbacks are asserted too: they are the methods which rot
	// unnoticed when a signature drifts, because nothing in this module calls
	// them directly.
	_ json.Marshaler   = Number(0)
	_ json.Unmarshaler = (*Number)(nil)
	_ driver.Valuer    = Number(0)
	_ sql.Scanner      = (*Number)(nil)
)

// NullNumber returns the null Number, the one which marshals as null and is
// written to the database as NULL.
func NullNumber() Number {
	return Number(math.NaN())
}

// IsNull reports whether the number has no value: NaN, or one of the two
// infinities, all of which render as null.
func (n Number) IsNull() bool {
	f := float64(n)
	return math.IsNaN(f) || math.IsInf(f, 0)
}

// Float64 returns the number as a float64. A null Number returns NaN, so
// check IsNull first when the difference matters.
func (n Number) Float64() float64 {
	return float64(n)
}

// String returns the JSON text of the number, and "null" when it has no value.
// It is the encoder's own formatter, so String, MarshalJSON and MarshalJSONTo
// always agree: a large value renders as 1e+21 on all three, where a plain
// strconv 'f' would write it out in full on one of them only.
func (n Number) String() string {
	if n.IsNull() {
		return "null"
	}

	return string(jsontext.AppendFloat(nil, float64(n), 64))
}

// MarshalJSONTo writes the number straight into the encoder, which is the path
// ctx.JSON takes, with no intermediate byte slice.
func (n Number) MarshalJSONTo(enc *jsontext.Encoder) error {
	if n.IsNull() {
		return enc.WriteToken(jsontext.Null)
	}

	return enc.WriteToken(jsontext.Float(float64(n)))
}

// MarshalJSON renders the number for encoding/json (v1) interop: a downstream
// logger or a v1 encoder gets the same bytes MarshalJSONTo writes.
func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(n.String()), nil
}

// UnmarshalJSONFrom reads the number from the decoder. A JSON null becomes
// NaN, the value this type calls null.
func (n *Number) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == 'n' { // null.
		if _, err := dec.ReadToken(); err != nil {
			return err
		}

		*n = NullNumber()
		return nil
	}

	var f float64
	if err := json.UnmarshalDecode(dec, &f); err != nil {
		return err
	}

	*n = Number(f)
	return nil
}

// UnmarshalJSON reads the number for encoding/json (v1) interop.
// A JSON null becomes NaN.
//
// A quoted number is rejected here as it is by UnmarshalJSONFrom: the two
// paths accept exactly the same documents, so a payload cannot decode under
// one encoder and fail under the other.
func (n *Number) UnmarshalJSON(data []byte) error {
	if isNull(data) {
		*n = NullNumber()
		return nil
	}

	f, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("types: number: %w: %s", ErrInvalid, data)
	}

	*n = Number(f)
	return nil
}

// Value completes the driver.Valuer interface: a null Number is written as
// NULL, every other one as a float8.
func (n Number) Value() (driver.Value, error) {
	if n.IsNull() {
		return nil, nil
	}

	return float64(n), nil
}

// Scan completes the sql.Scanner interface. A NULL column becomes NaN, so a
// nullable float8 needs no sql.NullFloat64 wrapper. Text-mode drivers give
// the value as a string or as raw bytes; both are parsed.
//
// An empty text is an error and not a null: a number column which holds no
// value holds NULL, and empty text from a numeric column means the driver
// gave something unexpected. Weekday, whose column is text, takes the
// opposite view and leaves the value alone.
func (n *Number) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*n = NullNumber()
	case float64:
		*n = Number(v)
	case float32:
		*n = Number(v)
	case int64:
		*n = Number(v)
	case int:
		*n = Number(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("types: number: scan: %w: %q", ErrInvalid, v)
		}
		*n = Number(f)
	case []byte:
		return n.Scan(string(v))
	default:
		return fmt.Errorf("types: number: scan: unknown type of: %T", src)
	}

	return nil
}
