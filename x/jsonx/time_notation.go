package jsonx

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// TimeNotationDuration is a JSON representation of the standard Duration type in 00:00:00 (hour, minute seconds).
type TimeNotationDuration time.Duration

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02d:%02d:00", h, m) // Manos doesn't care about the seconds.
}

// ParseTimeNotationDuration parses a string to a time notation duration (00:00:00) hours:minutes:seconds.
func ParseTimeNotationDuration(s string) (TimeNotationDuration, error) {
	entries := strings.SplitN(s, ":", 3)
	if len(entries) < 3 {
		return TimeNotationDuration(0), fmt.Errorf("invalid duration format: expected hours:minutes:seconds (e.g. 01:05:00) but got: %s", s)
	}

	hours, err := strconv.Atoi(entries[0])
	if err != nil {
		return TimeNotationDuration(0), err
	}
	minutes, err := strconv.Atoi(entries[1])
	if err != nil {
		return TimeNotationDuration(0), err
	}

	// remove any .0000
	secondsStr := strings.TrimSuffix(entries[2], ".000000")
	seconds, err := strconv.Atoi(secondsStr)
	if err != nil {
		return TimeNotationDuration(0), err
	}

	format := fmt.Sprintf("%02dh%02dm%02ds", hours, minutes, seconds)
	v, err := time.ParseDuration(format)
	if err != nil {
		return TimeNotationDuration(0), err
	}

	return TimeNotationDuration(v), nil

}

func (d TimeNotationDuration) MarshalJSON() ([]byte, error) {
	v := d.ToDuration()

	format := fmtDuration(v)
	return []byte(strconv.Quote(format)), nil
}

func (d *TimeNotationDuration) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || bytes.Equal(b, nullLiteral) || bytes.Equal(b, emptyQuoteBytes) { // if null or empty don't throw an error.
		return nil
	}

	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = TimeNotationDuration(value)
		return nil
	case string:
		dv, err := ParseTimeNotationDuration(value)
		if err != nil {
			return err
		}
		*d = dv
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func (d TimeNotationDuration) ToDuration() time.Duration {
	return time.Duration(d)
}

func (d TimeNotationDuration) Value() (driver.Value, error) {
	return d.ToDuration(), nil
}

// Set sets the value of duration in nanoseconds.
func (d *TimeNotationDuration) Set(v float64) {
	if math.IsNaN(v) {
		return
	}

	*d = TimeNotationDuration(v)
}
