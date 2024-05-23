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
	// ISO8601LayoutWithTimezone same as ISO8601Layout but with the timezone suffix.
	ISO8601LayoutWithTimezone = "2006-01-02T15:04:05Z"

	// To match Go’s standard time layout that pads zeroes for microseconds, you can use the format 2006-01-02T15:04:05.000000Z07:00.
	// This layout uses 0s instead of 9s for the fractional second part, which ensures that the microseconds are
	// always represented with six digits, padding with leading zeroes if necessary.
	// ISO8601ZUTCOffsetLayoutWithMicroseconds = "2006-01-02T15:04:05.000000Z07:00"
	// ISO8601ZUTCOffsetLayoutWithMicroseconds ISO 8601 format, with full time and zone with UTC offset.
	// Example: 2022-08-10T03:21:00.000000+03:00, 2023-02-04T09:48:14+00:00, 2022-08-09T00:00:00.000000.
	ISO8601ZUTCOffsetLayoutWithMicroseconds = "2006-01-02T15:04:05.999999Z07:00"
	// ISO8601ZUTCOffsetLayoutWithoutMicroseconds ISO 8601 format, with full time and zone with UTC offset without microsecond precision.
	ISO8601ZUTCOffsetLayoutWithoutMicroseconds = "2006-01-02T15:04:05Z07:00"
	/*
		The difference between the two time layouts "2006-01-02T15:04:05Z07:00" and "2006-01-02T15:04:05-07:00" is the presence of the Z character:

		"2006-01-02T15:04:05Z07:00": The Z indicates that the time is in UTC (Coordinated Universal Time) if there’s no offset specified.
		When an offset is present, as in +03:00, it indicates the time is in a timezone that is 3 hours ahead of UTC.
		The Z is combined with the offset (07:00), which can be positive or negative to represent the timezone difference from UTC.

		"2006-01-02T15:04:05-07:00": This layout does not have the Z character and directly uses the offset (-07:00).
		It’s more straightforward and indicates that the time is in a timezone that is 7 hours behind UTC.
		In summary, the Z in the first layout serves as a placeholder for UTC and is used when the time might be in UTC or might have an offset.
		The second layout is used when you’re directly specifying the offset without any reference to UTC.
		Both layouts can parse the timestamp "2024-04-08T04:47:10+03:00" correctly, as they include placeholders for the timezone offset.
	*/

	// ISO8601UnconventionalOffsetLayout is the layout for the unconventional offset.
	// Custom offset layout, e.g., 2024-05-21T18:06:07.000000-04:01:19.
	ISO8601UnconventionalOffsetLayout = "2006-01-02T15:04:05.000000"
)

// ISO8601 describes a time compatible with javascript time format.
type ISO8601 time.Time

var _ Exampler = (*ISO8601)(nil)

// ParseISO8601 reads from "s" and returns the ISO8601 time.
//
// The function supports the following formats:
//   - 2024-01-02T15:04:05.999999Z
//   - 2024-01-02T15:04:05+07:00
//   - 2024-04-08T08:05:04.830140+00:00
//   - 2024-01-02T15:04:05Z
//   - 2024-04-08T08:05:04.830140
//   - 2024-01-02T15:04:05
//   - 2024-05-21T18:06:07.000000-04:01:19
func ParseISO8601(s string) (ISO8601, error) {
	if s == "" || s == "null" {
		return ISO8601{}, nil
	}

	var (
		tt  time.Time
		err error
	)

	/*
		// Check if the string contains a timezone offset after the 'T' character.
		hasOffset := strings.Contains(s, "Z") || (strings.Index(s, "+") > strings.Index(s, "T")) || (strings.Index(s, "-") > strings.Index(s, "T"))

		switch {
		case strings.HasSuffix(s, "Z"):
			tt, err = time.Parse(ISO8601LayoutWithTimezone, s)
		case hasOffset && strings.Contains(s, "."):
			tt, err = time.Parse(ISO8601ZUTCOffsetLayoutWithMicroseconds, s)
		case hasOffset:
			tt, err = parseWithOffset(s)
		default:
			tt, err = time.Parse(ISO8601Layout, s)
		}

		if err != nil {
			return ISO8601{}, fmt.Errorf("ISO8601: %w", err)
		}

		return ISO8601(tt), nil
	*/

	if idx := strings.LastIndexFunc(s, startUTCOffsetIndexFunc); idx > 18 { // should have some distance, with and without milliseconds
		length := parseSignedOffset(s[idx:])

		// Check if the offset is unconventional, e.g., -04:01:19
		if offset := s[idx:]; isUnconventionalOffset(offset) {
			mainPart := s[:idx]
			tt, err = time.Parse("2006-01-02T15:04:05.000000", mainPart)
			if err != nil {
				return ISO8601{}, fmt.Errorf("ISO8601: %w", err)
			}

			adjustedTime, parseErr := adjustForUnconventionalOffset(tt, offset)
			if parseErr != nil {
				return ISO8601{}, fmt.Errorf("ISO8601: %w", parseErr)
			}
			return ISO8601(adjustedTime), nil
		}

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

		// fmt.Printf("parsing %s with offset %s, secondsEastUTC: %d, using time layout: %s\n", s, offsetText, secondsEastUTC, ISO8601ZUTCOffsetLayoutWithMicroseconds)
		if loc, ok := fixedEastUTCLocations[secondsEastUTC]; ok { // Specific (fixed) zone.
			if strings.Contains(s, ".") {
				tt, err = time.ParseInLocation(ISO8601ZUTCOffsetLayoutWithMicroseconds, s, loc)
			} else {
				tt, err = time.ParseInLocation(ISO8601ZUTCOffsetLayoutWithoutMicroseconds, s, loc)
			}
		} else { // Local or UTC.
			if strings.Contains(s, ".") {
				tt, err = time.Parse(ISO8601ZUTCOffsetLayoutWithMicroseconds, s)
			} else {
				tt, err = time.Parse(ISO8601ZUTCOffsetLayoutWithoutMicroseconds, s)
			}
		}
	} else if s[len(s)-1] == 'Z' {
		tt, err = time.Parse(ISO8601LayoutWithTimezone, s)
	} else {
		tt, err = time.Parse(ISO8601Layout, s)
	}

	if err != nil {
		return ISO8601{}, fmt.Errorf("ISO8601: %w", err)
	}
	return ISO8601(tt), nil
}

func parseWithOffset(s string) (time.Time, error) {
	idx := strings.LastIndexFunc(s, startUTCOffsetIndexFunc)
	if idx == -1 {
		return time.Time{}, fmt.Errorf("ISO8601: missing timezone offset")
	}

	offsetText := s[idx:]
	secondsEastUTC, err := parseOffsetToSeconds(offsetText)
	if err != nil {
		return time.Time{}, err
	}

	loc, ok := fixedEastUTCLocations[secondsEastUTC]
	if !ok {
		loc = time.FixedZone("", secondsEastUTC)
	}

	return time.ParseInLocation(ISO8601ZUTCOffsetLayoutWithoutMicroseconds, s, loc)
}

func parseOffsetToSeconds(offsetText string) (int, error) {
	if len(offsetText) < 6 {
		return 0, fmt.Errorf("ISO8601: invalid timezone offset length: %s", offsetText)
	}

	sign := offsetText[0]
	if sign != '-' && sign != '+' {
		return 0, fmt.Errorf("ISO8601: invalid timezone offset sign: %c", sign)
	}

	hours, err := strconv.Atoi(offsetText[1:3])
	if err != nil {
		return 0, fmt.Errorf("ISO8601: %w", err)
	}

	minutes, err := strconv.Atoi(offsetText[4:6])
	if err != nil {
		return 0, fmt.Errorf("ISO8601: %w", err)
	}

	secondsEastUTC := (hours*60 + minutes) * 60
	if sign == '-' {
		secondsEastUTC = -secondsEastUTC
	}

	return secondsEastUTC, nil
}

func isUnconventionalOffset(offset string) bool {
	parts := strings.Split(offset, ":")
	return len(parts) == 3
}

func adjustForUnconventionalOffset(t time.Time, offset string) (time.Time, error) {
	sign := 1
	if offset[0] == '-' {
		sign = -1
	}
	offset = offset[1:]

	offsetParts := strings.Split(offset, ":")
	if len(offsetParts) != 3 {
		return time.Time{}, fmt.Errorf("invalid offset format: %s", offset)
	}

	hours, err := strconv.Atoi(offsetParts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing offset hours: %s: %w", offset, err)
	}

	if hours > 24 {
		return time.Time{}, fmt.Errorf("invalid offset hours: %d: %s", hours, offset)
	}

	minutes, err := strconv.Atoi(offsetParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing offset minutes: %s: %w", offset, err)
	}
	if minutes > 60 {
		return time.Time{}, fmt.Errorf("invalid offset minutes: %d: %s", minutes, offset)
	}

	seconds, err := strconv.Atoi(offsetParts[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing offset seconds: %s: %w", offset, err)
	}

	if seconds > 60 {
		return time.Time{}, fmt.Errorf("invalid offset seconds: %d: %s", seconds, offset)
	}

	totalOffset := time.Duration(sign) * (time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second)
	return t.Add(-totalOffset), nil
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
		"2024-01-02T15:04:05.999999Z",
		"2024-01-02T15:04:05+07:00",
		"2024-04-08T08:05:04.830140+00:00",
		"2024-01-02T15:04:05Z",
		"2024-04-08T08:05:04.830140",
		"2024-01-02T15:04:05",
	}
}

// ToTime returns the unwrapped *t to time.Time.
func (t ISO8601) ToTime() time.Time {
	return time.Time(t)
}

// IsZero reports whether "t" is zero time.
// It completes the pg.Zeroer interface.
func (t ISO8601) IsZero() bool {
	return time.Time(t).IsZero()
}

// After reports whether the time instant "t" is after "u".
func (t ISO8601) After(u ISO8601) bool {
	return t.ToTime().After(u.ToTime())
}

// Equal reports whether the time instant "t" is equal to "u".
func (t ISO8601) Equal(u ISO8601) bool {
	return t.ToTime().Equal(u.ToTime())
}

// Add returns the time "t" with the duration added.
func (t ISO8601) Add(d time.Duration) ISO8601 {
	return ISO8601(t.ToTime().Add(d))
}

// Sub returns the duration between "t" and "u".
func (t ISO8601) Sub(u ISO8601) time.Duration {
	return t.ToTime().Sub(u.ToTime())
}

// String returns the text representation of the "t" using the ISO8601 time layout.
func (t ISO8601) String() string {
	tt := t.ToTime()
	if tt.IsZero() {
		return ""
	}

	return tt.Format(ISO8601Layout)
}

// To24Hour returns the 24-hour representation of the time.
func (t ISO8601) To24Hour() string {
	tt := t.ToTime()
	if tt.IsZero() {
		return ""
	}

	return tt.Format("15:04")
}

// ToSimpleDate converts the current ISO8601 "t" to SimpleDate.
func (t ISO8601) ToSimpleDate() SimpleDate {
	return SimpleDateFromTime(t.ToTime())
}

// ToSimpleDateIn converts the current ISO8601 "t" to SimpleDate in specific location.
func (t ISO8601) ToSimpleDateIn(in *time.Location) SimpleDate {
	if in == nil {
		in = time.UTC
	}

	return SimpleDateFromTime(t.ToTime().In(in))
}

// ToDayTime converts the current ISO8601 "t" to DayTime.
func (t ISO8601) ToDayTime() DayTime {
	return DayTime(t.ToTime())
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
