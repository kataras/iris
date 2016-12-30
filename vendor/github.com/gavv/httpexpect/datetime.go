package httpexpect

import (
	"time"
)

// DateTime provides methods to inspect attached time.Time value.
type DateTime struct {
	chain chain
	value time.Time
}

// NewDateTime returns a new DateTime object given a reporter used to report
// failures and time.Time value to be inspected.
//
// reporter should not be nil.
//
// Example:
//   dt := NewDateTime(reporter, time.Now())
//   dt.Le(time.Now())
//
//   time.Sleep(time.Second)
//   dt.Lt(time.Now())
func NewDateTime(reporter Reporter, value time.Time) *DateTime {
	return &DateTime{makeChain(reporter), value}
}

// Raw returns underlying time.Time value attached to DateTime.
// This is the value originally passed to NewDateTime.
//
// Example:
//  dt := NewDateTime(t, timestamp)
//  assert.Equal(t, timestamp, dt.Raw())
func (dt *DateTime) Raw() time.Time {
	return dt.value
}

// Equal succeeds if DateTime is equal to given value.
//
// Example:
//  dt := NewDateTime(t, time.Unix(0, 1))
//  dt.Equal(time.Unix(0, 1))
func (dt *DateTime) Equal(value time.Time) *DateTime {
	if !dt.value.Equal(value) {
		dt.chain.fail("\nexpected datetime equal to:\n %s\n\nbut got:\n %s",
			value, dt.value)
	}
	return dt
}

// NotEqual succeeds if DateTime is not equal to given value.
//
// Example:
//  dt := NewDateTime(t, time.Unix(0, 1))
//  dt.NotEqual(time.Unix(0, 2))
func (dt *DateTime) NotEqual(value time.Time) *DateTime {
	if dt.value.Equal(value) {
		dt.chain.fail("\nexpected datetime not equal to:\n %s", value)
	}
	return dt
}

// Gt succeeds if DateTime is greater than given value.
//
// Example:
//  dt := NewDateTime(t, time.Unix(0, 2))
//  dt.Gt(time.Unix(0, 1))
func (dt *DateTime) Gt(value time.Time) *DateTime {
	if !dt.value.After(value) {
		dt.chain.fail("\nexpected datetime > then:\n %s\n\nbut got:\n %s",
			value, dt.value)
	}
	return dt
}

// Ge succeeds if DateTime is greater than or equal to given value.
//
// Example:
//  dt := NewDateTime(t, time.Unix(0, 2))
//  dt.Ge(time.Unix(0, 1))
func (dt *DateTime) Ge(value time.Time) *DateTime {
	if !(dt.value.After(value) || dt.value.Equal(value)) {
		dt.chain.fail("\nexpected datetime >= then:\n %s\n\nbut got:\n %s",
			value, dt.value)
	}
	return dt
}

// Lt succeeds if DateTime is lesser than given value.
//
// Example:
//  dt := NewDateTime(t, time.Unix(0, 1))
//  dt.Lt(time.Unix(0, 2))
func (dt *DateTime) Lt(value time.Time) *DateTime {
	if !dt.value.Before(value) {
		dt.chain.fail("\nexpected datetime < then:\n %s\n\nbut got:\n %s",
			value, dt.value)
	}
	return dt
}

// Le succeeds if DateTime is lesser than or equal to given value.
//
// Example:
//  dt := NewDateTime(t, time.Unix(0, 1))
//  dt.Le(time.Unix(0, 2))
func (dt *DateTime) Le(value time.Time) *DateTime {
	if !(dt.value.Before(value) || dt.value.Equal(value)) {
		dt.chain.fail("\nexpected datetime <= then:\n %s\n\nbut got:\n %s",
			value, dt.value)
	}
	return dt
}

// InRange succeeds if DateTime is in given range [min; max].
//
// Example:
//  dt := NewDateTime(t, time.Unix(0, 2))
//  dt.InRange(time.Unix(0, 1), time.Unix(0, 3))
//  dt.InRange(time.Unix(0, 2), time.Unix(0, 2))
func (dt *DateTime) InRange(min, max time.Time) *DateTime {
	if !((dt.value.After(min) || dt.value.Equal(min)) &&
		(dt.value.Before(max) || dt.value.Equal(max))) {
		dt.chain.fail(
			"\nexpected datetime in range:\n min: %s\n max: %s\n\nbut got: %s",
			min, max, dt.value)
	}
	return dt
}
