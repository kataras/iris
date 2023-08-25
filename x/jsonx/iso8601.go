package jsonx

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	// To load all system and embedded locations by name:
	// _ "time/tzdata" 	// OR build with: -tags timetzdata
)

var fixedEastUTCLocations = make(map[int]*time.Location)

// RegisterFixedLocation should be called on initialization of the program.
// It registers a fixed location to the time parser.
//
// E.g. for input of 2023-02-04T09:48:14+03:00 to result a time string of 2023-02-04 09:48:14 +0300 EEST
// you have to RegisterFixedLocation("EEST", 10800) otherwise it will result to: 2023-02-04 09:48:14 +0300 +0300.
func RegisterFixedLocation(name string, secondsFromUTC int) {
	loc := time.FixedZone(name, secondsFromUTC)
	fixedEastUTCLocations[secondsFromUTC] = loc
}

func init() {
	RegisterFixedLocation("EEST", 3*60*60) // + 3 hours.
	RegisterFixedLocation("UTC", 0)
}

const (
	// ISO8601Layout holds the time layout for the the javascript iso time.
	// Read more at: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Date/toISOString.
	ISO8601Layout = "2006-01-02T15:04:05"
	// ISO8601ZLayout same as ISO8601Layout but with the timezone suffix.
	ISO8601ZLayout = "2006-01-02T15:04:05Z"
	// ISO8601ZUTCOffsetLayout ISO 8601 format, with full time and zone with UTC offset.
	// Example: 2022-08-10T03:21:00.000000+03:00, 2023-02-04T09:48:14+00:00, 2022-08-09T00:00:00.000000.
	ISO8601ZUTCOffsetLayout = "2006-01-02T15:04:05.999999Z07:00" // -07:00
)

// ISO8601 describes a time compatible with javascript time format.
type ISO8601 time.Time

var _ Exampler = (*ISO8601)(nil)

// ParseISO8601 reads from "s" and returns the ISO8601 time.
func ParseISO8601(s string) (ISO8601, error) {
	if s == "" || s == "null" {
		return ISO8601{}, nil
	}

	var (
		tt  time.Time
		err error
	)

	if idx := strings.LastIndexFunc(s, startUTCOffsetIndexFunc); idx > 18 /* should have some distance, with and without milliseconds */ {
		length := parseSignedOffset(s[idx:])

		if idx+1 > idx+length || len(s) <= idx+length+1 {
			return ISO8601{}, fmt.Errorf("ISO8601: invalid timezone format: %s", s[idx:])
		}

		offsetText := s[idx+1 : idx+length]
		offset, parseErr := strconv.Atoi(offsetText)
		if parseErr != nil {
			return ISO8601{}, fmt.Errorf("ISO8601: %w", parseErr)
		}

		// E.g. offset of +0300 is returned as 10800 which is - (3 * 60 * 60).
		secondsEastUTC := offset * 60 * 60

		if loc, ok := fixedEastUTCLocations[secondsEastUTC]; ok { // Specific (fixed) zone.
			tt, err = time.ParseInLocation(ISO8601ZUTCOffsetLayout, s, loc)
		} else { // Local or UTC.
			tt, err = time.Parse(ISO8601ZUTCOffsetLayout, s)
		}
	} else if s[len(s)-1] == 'Z' {
		tt, err = time.Parse(ISO8601ZLayout, s)
	} else {
		tt, err = time.Parse(ISO8601Layout, s)
	}

	if err != nil {
		return ISO8601{}, fmt.Errorf("ISO8601: %w", err)
	}

	return ISO8601(tt), nil
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

// Examples returns a list of example values.
func (t ISO8601) ListExamples() any {
	return []string{
		"2006-01-02T15:04:05",
		"2022-08-09T00:00:00.000000",
		"2022-08-10T03:21:00.000000+03:00",
		"2023-02-04T09:48:14+00:00",
	}
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

// ToSimpleDate converts the current ISO8601 "t" to SimpleDate in specific location.
func (t ISO8601) ToSimpleDate(in *time.Location) SimpleDate {
	if in == nil {
		in = time.UTC
	}

	return SimpleDateFromTime(t.ToTime().In(in))
}

// Value returns the database value of time.Time.
func (t ISO8601) Value() (driver.Value, error) {
	return time.Time(t), nil
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
	case []byte:
		return t.Scan(string(v))
	case nil:
		*t = ISO8601(time.Time{})
	default:
		return fmt.Errorf("ISO8601: unknown type of: %T", v)
	}

	return nil
}

// parseSignedOffset parses a signed timezone offset (e.g. "+03" or "-04").
// The function checks for a signed number in the range -23 through +23 excluding zero.
// Returns length of the found offset string or 0 otherwise.
//
// Language internal function.
func parseSignedOffset(value string) int {
	sign := value[0]
	if sign != '-' && sign != '+' {
		return 0
	}
	x, rem, err := leadingInt(value[1:])

	// fail if nothing consumed by leadingInt
	if err != nil || value[1:] == rem {
		return 0
	}
	if x > 23 {
		return 0
	}
	return len(value) - len(rem)
}

var errLeadingInt = errors.New("ISO8601: time: bad [0-9]*") // never printed.

// leadingInt consumes the leading [0-9]* from s.
//
// Language internal function.
func leadingInt(s string) (x uint64, rem string, err error) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > 1<<63/10 {
			// overflow
			return 0, "", errLeadingInt
		}
		x = x*10 + uint64(c) - '0'
		if x > 1<<63 {
			// overflow
			return 0, "", errLeadingInt
		}
	}
	return x, s[i:], nil
}

func startUTCOffsetIndexFunc(char rune) bool {
	return char == '+' || char == '-'
}
