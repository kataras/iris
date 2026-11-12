package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"strings"
	"time"
)

// Weekday is a time.Weekday which travels as a lowercase name: "monday". A
// client reading a schedule gets a word it can print, not a number whose base
// it has to guess, and the JSON stays readable in a log.
//
// The numbering is time.Weekday's, Sunday being zero, so a Weekday converts
// to and from the standard type for free:
//
//	d := types.WeekdayOf(t.Weekday())
//	t.Weekday() == d.Std()
//
// Because Sunday is the zero value, the zero Weekday marshals as "sunday".
type Weekday time.Weekday

var (
	_ json.MarshalerTo     = Weekday(0)
	_ json.UnmarshalerFrom = (*Weekday)(nil)
	// The v1 fallbacks are asserted too: they are the methods which rot
	// unnoticed when a signature drifts, because nothing in this module calls
	// them directly.
	_ json.Marshaler   = Weekday(0)
	_ json.Unmarshaler = (*Weekday)(nil)
	_ driver.Valuer    = Weekday(0)
	_ sql.Scanner      = (*Weekday)(nil)
)

// The seven days, so a call site reads types.Monday rather than
// types.Weekday(time.Monday). The values are time.Weekday's.
const (
	Sunday    = Weekday(time.Sunday)
	Monday    = Weekday(time.Monday)
	Tuesday   = Weekday(time.Tuesday)
	Wednesday = Weekday(time.Wednesday)
	Thursday  = Weekday(time.Thursday)
	Friday    = Weekday(time.Friday)
	Saturday  = Weekday(time.Saturday)
)

// WeekdayOf converts a standard time.Weekday into this type. It is the
// opposite direction of the Std method.
func WeekdayOf(d time.Weekday) Weekday {
	return Weekday(d)
}

// weekdayNames holds the lowercase long names, indexed by time.Weekday.
var weekdayNames = [...]string{
	"sunday",
	"monday",
	"tuesday",
	"wednesday",
	"thursday",
	"friday",
	"saturday",
}

// ParseWeekday reads a weekday from its name, in any case and in the long or
// the three letter form: "Monday", "monday", "MON" and "mon" are the same day.
// Surrounding space is ignored.
func ParseWeekday(s string) (Weekday, error) {
	name := strings.ToLower(strings.TrimSpace(s))
	if name == "" {
		return 0, fmt.Errorf("types: weekday: %w: empty", ErrInvalid)
	}

	for i, weekdayName := range weekdayNames {
		if name == weekdayName || name == weekdayName[0:3] {
			return Weekday(i), nil
		}
	}

	return 0, fmt.Errorf("types: weekday: %w: %q", ErrInvalid, s)
}

// IsValid reports whether the weekday is one of the seven days.
func (d Weekday) IsValid() bool {
	return d >= Weekday(time.Sunday) && d <= Weekday(time.Saturday)
}

// Std returns the standard library type. It is named Std and not ToWeekday
// because a "ToWeekday" in the wild converts into a weekday type rather than
// out of one, and having both directions under one name during a migration is
// how the wrong one gets called. WeekdayOf is the other direction.
func (d Weekday) Std() time.Weekday {
	return time.Weekday(d)
}

// String returns the lowercase English name of the day ("monday"). An out of
// range value renders like the standard library's, so a bad value is visible
// in a log instead of silently reading as a Sunday.
func (d Weekday) String() string {
	if !d.IsValid() {
		return fmt.Sprintf("%%!Weekday(%d)", int(d))
	}

	return weekdayNames[d]
}

// MarshalJSONTo writes the day name straight into the encoder, which is the
// path ctx.JSON takes, with no intermediate byte slice.
func (d Weekday) MarshalJSONTo(enc *jsontext.Encoder) error {
	if !d.IsValid() {
		return fmt.Errorf("types: weekday: %w: %d", ErrInvalid, int(d))
	}

	return enc.WriteToken(jsontext.String(d.String()))
}

// MarshalJSON renders the day name for encoding/json (v1) interop.
func (d Weekday) MarshalJSON() ([]byte, error) {
	if !d.IsValid() {
		return nil, fmt.Errorf("types: weekday: %w: %d", ErrInvalid, int(d))
	}

	return []byte(`"` + d.String() + `"`), nil
}

// UnmarshalJSONFrom reads the day name from the decoder. A JSON null and an
// empty text leave the value as it is, matching the rest of this package.
func (d *Weekday) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == 'n' { // null.
		_, err := dec.ReadToken()
		return err
	}

	var s string
	if err := json.UnmarshalDecode(dec, &s); err != nil {
		return err
	}

	return d.parse(s)
}

// UnmarshalJSON reads the day name for encoding/json (v1) interop.
func (d *Weekday) UnmarshalJSON(data []byte) error {
	if isNull(data) {
		return nil
	}

	return d.parse(string(trimQuotes(data)))
}

// parse sets the weekday from its name, leaving it alone when the name is
// empty: an absent value is not a wrong one.
func (d *Weekday) parse(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	weekday, err := ParseWeekday(s)
	if err != nil {
		return err
	}

	*d = weekday
	return nil
}

// Value completes the driver.Valuer interface: the day is stored as its
// lowercase name, so the column reads the same as the JSON.
func (d Weekday) Value() (driver.Value, error) {
	if !d.IsValid() {
		return nil, fmt.Errorf("types: weekday: %w: %d", ErrInvalid, int(d))
	}

	return d.String(), nil
}

// Scan completes the sql.Scanner interface. A text column is parsed by name,
// an integer one by time.Weekday's numbering and a NULL leaves the value as it
// is, as an empty text does: an absent value is not a wrong one. Number, whose
// column is numeric, takes the opposite view and makes an empty text an error.
func (d *Weekday) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		return d.parse(v)
	case []byte: // text-mode drivers give the column value as raw bytes.
		return d.parse(string(v))
	case int64:
		weekday := Weekday(v)
		if !weekday.IsValid() {
			return fmt.Errorf("types: weekday: scan: %w: %d", ErrInvalid, v)
		}
		*d = weekday
	case time.Weekday:
		weekday := Weekday(v)
		if !weekday.IsValid() {
			return fmt.Errorf("types: weekday: scan: %w: %d", ErrInvalid, v)
		}
		*d = weekday
	default:
		return fmt.Errorf("types: weekday: scan: unknown type of: %T", src)
	}

	return nil
}
