package jsonx

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math"
	"time"
)

// Duration is a JSON representation of the standard Duration type, until Go version 2 supports it under the hoods.
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(value)
		return nil
	case string:
		v, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(v)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

func (d Duration) Value() (driver.Value, error) {
	return int64(d), nil
}

// Set sets the value of duration in nanoseconds.
func (d *Duration) Set(v float64) {
	if math.IsNaN(v) {
		return
	}

	*d = Duration(v)
}
