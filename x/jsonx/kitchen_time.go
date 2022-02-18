package jsonx

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// KitchenTimeLayout represents the "3:04 PM" Go time format, similar to time.Kitchen.
const KitchenTimeLayout = "3:04 PM"

// KitchenTime holds a json "3:04 PM" time.
type KitchenTime time.Time

var ErrParseKitchenTimeColon = fmt.Errorf("parse kitchen time: missing ':' character")

func parseKitchenTime(s string) (KitchenTime, error) {
	// Remove any second,millisecond variable (probably given by postgres 00:00:00.000000).
	// required(00:00)remove(:00.000000)

	firstIndex := strings.IndexByte(s, ':')
	if firstIndex == -1 {
		return KitchenTime{}, ErrParseKitchenTimeColon
	} else {
		nextIndex := strings.LastIndexByte(s, ':')
		spaceIdx := strings.LastIndexByte(s, ' ')
		if nextIndex > firstIndex && spaceIdx > 0 {
			tmp := s[0:nextIndex]
			s = tmp + s[spaceIdx:]
		}
	}

	tt, err := time.Parse(KitchenTimeLayout, s)
	if err != nil {
		return KitchenTime{}, err
	}

	return KitchenTime(tt), nil
}

// ParseKitchenTime reads from "s" and returns the KitchenTime time.
func ParseKitchenTime(s string) (KitchenTime, error) {
	if s == "" || s == "null" {
		return KitchenTime{}, nil
	}

	return parseKitchenTime(s)
}

// UnmarshalJSON binds the json "data" to "t" with the `KitchenTimeLayout`.
func (t *KitchenTime) UnmarshalJSON(data []byte) error {
	if t == nil {
		return fmt.Errorf("kitchen time: dest is nil")
	}

	if isNull(data) {
		return nil
	}

	data = trimQuotes(data)

	if len(data) == 0 {
		return nil
	}

	tt, err := parseKitchenTime(string(data))
	if err != nil {
		return err
	}

	*t = KitchenTime(tt)
	return nil
}

// MarshalJSON returns the json representation of the "t".
func (t KitchenTime) MarshalJSON() ([]byte, error) {
	if s := t.String(); s != "" {
		s = strconv.Quote(s)
		return []byte(s), nil
	}

	return emptyQuoteBytes, nil
}

// IsZero reports whether "t" is zero time.
// It completes the pg.Zeroer interface.
func (t KitchenTime) IsZero() bool {
	return t.Value().IsZero()
}

// Value returns the standard time type.
func (t KitchenTime) Value() time.Time {
	return time.Time(t)
}

// String returns the text representation of the date
// formatted based on the `KitchenTimeLayout`.
// If date is zero it returns an empty string.
func (t KitchenTime) String() string {
	tt := t.Value()
	if tt.IsZero() {
		return ""
	}

	return tt.Format(KitchenTimeLayout)
}

// Scan completes the pg and native sql driver.Scanner interface
// reading functionality of a custom type.
func (t *KitchenTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time: // type was set to timestamp.
		if v.IsZero() {
			return nil // don't set zero, ignore it.
		}
		*t = KitchenTime(v)
	case string: // type was set to time, input example: 10:00:00.000000
		d, err := ParseTimeNotationDuration(v)
		if err != nil {
			return fmt.Errorf("kitchen time: convert to time notation first: %w", err)
		}

		s := kitchenTimeStringFromDuration(d.ToDuration())
		*t, err = ParseKitchenTime(s)
		return err
	case int64: // timestamp with integer.
		u := time.Unix(v/1000, v%1000)
		s := kitchenTimeStringFromHourAndMinute(u.Hour(), u.Minute())

		tt, err := ParseKitchenTime(s)
		if err != nil {
			return err
		}
		*t = tt
	case nil:
		*t = KitchenTime(time.Time{})
	default:
		return fmt.Errorf("KitchenTime: unknown type of: %T", v)
	}

	return nil
}

func kitchenTimeStringFromDuration(dt time.Duration) string {
	hour := int(dt.Hours())
	minute := 0
	if totalMins := dt.Minutes(); totalMins > 0 {
		minute := int(totalMins / 60)
		if minute < 0 {
			minute = 0
		}
	}

	return kitchenTimeStringFromHourAndMinute(hour, minute)
}

func kitchenTimeStringFromHourAndMinute(hour, minute int) string {
	ampm := "AM"
	if hour/12 == 1 {
		ampm = "PM"
	}
	th := hour % 12
	hh := strconv.Itoa(th)
	if th < 10 {
		hh = "0" + hh
	}
	tm := minute
	mm := strconv.Itoa(tm)
	if tm < 10 {
		mm = "0" + mm
	}
	return hh + ":" + mm + " " + ampm
}
