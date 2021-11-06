package jsonx

import (
	"fmt"
	"strconv"
	"time"
)

// KitckenTimeLayout represents the "3:04 PM" Go time format, similar to time.Kitcken.
const KitckenTimeLayout = "3:04 PM"

// KitckenTime holds a json "3:04 PM" time.
type KitckenTime time.Time

// ParseKitchenTime reads from "s" and returns the KitckenTime time.
func ParseKitchenTime(s string) (KitckenTime, error) {
	if s == "" || s == "null" {
		return KitckenTime{}, nil
	}

	var (
		tt  time.Time
		err error
	)

	tt, err = time.Parse(KitckenTimeLayout, s)
	if err != nil {
		return KitckenTime{}, err
	}

	return KitckenTime(tt.UTC()), nil
}

// UnmarshalJSON binds the json "data" to "t" with the `KitckenTimeLayout`.
func (t *KitckenTime) UnmarshalJSON(data []byte) error {
	if isNull(data) {
		return nil
	}

	data = trimQuotes(data)

	if len(data) == 0 {
		return nil
	}

	tt, err := time.Parse(KitckenTimeLayout, string(data))
	if err != nil {
		return err
	}

	*t = KitckenTime(tt)
	return nil
}

// MarshalJSON returns the json representation of the "t".
func (t KitckenTime) MarshalJSON() ([]byte, error) {
	if s := t.String(); s != "" {
		s = strconv.Quote(s)
		return []byte(s), nil
	}

	return emptyQuoteBytes, nil
}

// IsZero reports whether "t" is zero time.
// It completes the pg.Zeroer interface.
func (t KitckenTime) IsZero() bool {
	return t.Value().IsZero()
}

// Value returns the standard time type.
func (t KitckenTime) Value() time.Time {
	return time.Time(t)
}

// String returns the text representation of the date
// formatted based on the `KitckenTimeLayout`.
// If date is zero it returns an empty string.
func (t KitckenTime) String() string {
	tt := t.Value()
	if tt.IsZero() {
		return ""
	}

	return tt.Format(KitckenTimeLayout)
}

// Scan completes the pg and native sql driver.Scanner interface
// reading functionality of a custom type.
func (t *KitckenTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time: // type was set to timestamp
		if v.IsZero() {
			return nil // don't set zero, ignore it.
		}
		*t = KitckenTime(v)
	case string:
		tt, err := ParseKitchenTime(v)
		if err != nil {
			return err
		}
		*t = tt
	default:
		return fmt.Errorf("KitckenTime: unknown type of: %T", v)
	}

	return nil
}
