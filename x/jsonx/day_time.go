package jsonx

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// DayTimeLayout holds the time layout for the the format of "hour:minute:second", hour can be 15, meaning 3 PM.
	DayTimeLayout = "15:04:05"
)

// DayTime describes a time compatible with DayTimeLayout.
type DayTime time.Time

// ParseDayTime reads from "s" and returns the DayTime time.
func ParseDayTime(s string) (DayTime, error) {
	if s == "" || s == "null" {
		return DayTime{}, nil
	}

	tt, err := time.Parse(DayTimeLayout, s)
	if err != nil {
		return DayTime{}, err
	}

	return DayTime(tt), nil
}

// UnmarshalJSON parses the "b" into DayTime time.
func (t *DayTime) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	s := strings.Trim(string(b), `"`)
	tt, err := ParseDayTime(s)
	if err != nil {
		return err
	}

	*t = tt
	return nil
}

// MarshalJSON writes a quoted string in the DayTime time format.
func (t DayTime) MarshalJSON() ([]byte, error) {
	if s := t.String(); s != "" {
		s = strconv.Quote(s)
		return []byte(s), nil
	}

	return nullLiteral, nil // Note: if the front-end wants an empty string instead I must change that.
}

// ToTime returns the unwrapped *t to time.Time.
func (t *DayTime) ToTime() time.Time {
	tt := time.Time(*t)
	return tt
}

// IsZero reports whether "t" is zero time.
func (t DayTime) IsZero() bool {
	return time.Time(t).IsZero()
}

// String returns the text representation of the "t" using the DayTime time layout.
func (t DayTime) String() string {
	tt := t.ToTime()
	if tt.IsZero() {
		return ""
	}

	return tt.Format(DayTimeLayout)
}

// Scan completes the sql driver.Scanner interface.
func (t *DayTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time: // type was set to timestamp
		if v.IsZero() {
			return nil // don't set zero, ignore it.
		}
		*t = DayTime(v)
	case string:
		tt, err := ParseDayTime(v)
		if err != nil {
			return err
		}
		*t = tt
	case nil:
		*t = DayTime(time.Time{})
	default:
		return fmt.Errorf("DayTime: unknown type of: %T", v)
	}

	return nil
}
