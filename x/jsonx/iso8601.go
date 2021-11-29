package jsonx

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// ISO8601Layout holds the time layout for the the javascript iso time.
	// Read more at: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Date/toISOString.
	ISO8601Layout = "2006-01-02T15:04:05"
	// ISO8601ZLayout same as ISO8601Layout but with the timezone suffix.
	ISO8601ZLayout = "2006-01-02T15:04:05Z"
)

// ISO8601 describes a time compatible with javascript time format.
type ISO8601 time.Time

// ParseISO8601 reads from "s" and returns the ISO8601 time.
func ParseISO8601(s string) (ISO8601, error) {
	if s == "" || s == "null" {
		return ISO8601{}, nil
	}

	var (
		tt  time.Time
		err error
	)

	if s[len(s)-1] == 'Z' {
		tt, err = time.Parse(ISO8601ZLayout, s)
	} else {
		tt, err = time.Parse(ISO8601Layout, s)
	}

	if err != nil {
		return ISO8601{}, err
	}

	return ISO8601(tt.UTC()), nil
}

// UnmarshalJSON parses the "b" into ISO8601 time.
func (t *ISO8601) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	s := strings.Trim(string(b), `"`)
	tt, err := ParseISO8601(s)
	if err != nil {
		return err
	}

	*t = tt
	return nil
}

// MarshalJSON writes a quoted string in the ISO8601 time format.
func (t ISO8601) MarshalJSON() ([]byte, error) {
	if s := t.String(); s != "" {
		s = strconv.Quote(s)
		return []byte(s), nil
	}

	return nullLiteral, nil // Note: if the front-end wants an empty string instead I must change that.
}

// ToTime returns the unwrapped *t to time.Time.
func (t *ISO8601) ToTime() time.Time {
	tt := time.Time(*t)
	return tt
}

// IsZero reports whether "t" is zero time.
// It completes the pg.Zeroer interface.
func (t ISO8601) IsZero() bool {
	return time.Time(t).IsZero()
}

// String returns the text representation of the "t" using the ISO8601 time layout.
func (t ISO8601) String() string {
	tt := t.ToTime()
	if tt.IsZero() {
		return ""
	}

	return tt.Format(ISO8601Layout)
}

// Scan completes the sql driver.Scanner interface.
func (t *ISO8601) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time: // type was set to timestamp
		if v.IsZero() {
			return nil // don't set zero, ignore it.
		}
		*t = ISO8601(v)
	case string:
		tt, err := ParseISO8601(v)
		if err != nil {
			return err
		}
		*t = tt
	case nil:
		*t = ISO8601(time.Time{})
	default:
		return fmt.Errorf("ISO8601: unknown type of: %T", v)
	}

	return nil
}
