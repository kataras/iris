package jsonx

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/kataras/iris/v12/x/timex"
)

const (
	// SimpleDateLayout represents the "year-month-day" Go time format.
	SimpleDateLayout         = "2006-01-02"
	simpleDateLayoutPostgres = "2006-1-2"
)

// SimpleDate holds a json "year-month-day" time.
type SimpleDate time.Time

var _ Exampler = (*SimpleDate)(nil)

// SimpleDateFromTime accepts a "t" Time and returns
// a SimpleDate. If format fails, it returns the zero value of time.Time.
func SimpleDateFromTime(t time.Time) SimpleDate {
	date, _ := ParseSimpleDate(t.Format(SimpleDateLayout))
	return date
}

// ParseSimpleDate reads from "s" and returns the SimpleDate time.
//
// The function supports the following formats:
//   - "2024-01-01"
//   - "2024-1-1"
func ParseSimpleDate(s string) (SimpleDate, error) {
	if s == "" || s == "null" {
		return SimpleDate{}, nil
	}

	var (
		tt  time.Time
		err error
	)

	tt, err = time.Parse(SimpleDateLayout, s)
	if err != nil {
		// After v5.0.0-alpha.3 of pgx this is coming as "1993-1-1" instead of the stored
		// value "1993-01-01".
		var err2 error
		tt, err2 = time.Parse(simpleDateLayoutPostgres, s)
		if err2 != nil {
			return SimpleDate{}, fmt.Errorf("%s: %w", err2.Error(), err)
		}
	}

	return SimpleDate(tt), nil
}

// UnmarshalJSON binds the json "data" to "t" with the `SimpleDateLayout`.
func (t *SimpleDate) UnmarshalJSON(data []byte) error {
	if isNull(data) {
		return nil
	}

	data = trimQuotes(data)
	dataStr := string(data)
	if len(dataStr) == 0 {
		return nil // do not allow empty "" on simple dates.
	}

	tt, err := time.Parse(SimpleDateLayout, dataStr)
	if err != nil {
		return err
	}

	*t = SimpleDate(tt)
	return nil
}

// MarshalJSON returns the json representation of the "t".
func (t SimpleDate) MarshalJSON() ([]byte, error) {
	if s := t.String(); s != "" {
		s = strconv.Quote(s)
		return []byte(s), nil
	}

	return emptyQuoteBytes, nil
}

// Examples returns a list of example values.
func (t SimpleDate) ListExamples() any {
	return []string{
		"2024-01-01",
		"2024-1-1",
	}
}

// IsZero reports whether "t" is zero time.
// It completes the pg.Zeroer interface.
func (t SimpleDate) IsZero() bool {
	return t.ToTime().IsZero()
}

// Add returns the date of "t" plus "d".
func (t SimpleDate) Add(d time.Duration) SimpleDate {
	return SimpleDateFromTime(t.ToTime().Add(d))
}

// CountPastDays returns the count of days between "t" and "pastDate".
func (t SimpleDate) CountPastDays(pastDate SimpleDate) int {
	t1, t2 := t.ToTime(), pastDate.ToTime()
	return int(t1.Sub(t2).Hours() / 24)
}

// Equal reports back if "t" and "d" equals to the same date.
func (t SimpleDate) Equal(d SimpleDate) bool {
	return t.String() == d.String()
}

// After reports whether the time instant t is after u.
func (t SimpleDate) After(d2 SimpleDate) bool {
	t1, t2 := t.ToTime(), d2.ToTime()
	return t1.Truncate(24 * time.Hour).After(t2.Truncate(24 * time.Hour))
}

// Before reports whether the time instant t is before u.
func (t SimpleDate) Before(d2 SimpleDate) bool {
	t1, t2 := t.ToTime(), d2.ToTime()
	return t1.Truncate(24 * time.Hour).Before(t2.Truncate(24 * time.Hour))
	// OR: compare year and year's day.
}

// ToTime returns the standard time type.
func (t SimpleDate) ToTime() time.Time {
	return time.Time(t)
}

// Value completes the pg and native sql driver.Valuer interface.
func (t SimpleDate) Value() (driver.Value, error) {
	return t.String(), nil
}

// String returns the text representation of the date
// formatted based on the `SimpleDateLayout`.
// If date is zero it returns an empty string.
func (t SimpleDate) String() string {
	tt := t.ToTime()
	if tt.IsZero() {
		return ""
	}

	return tt.Format(SimpleDateLayout)
}

// Scan completes the pg and native sql driver.Scanner interface
// reading functionality of a custom type.
func (t *SimpleDate) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time: // type was set to timestamp
		if v.IsZero() {
			return nil // don't set zero, ignore it.
		}
		*t = SimpleDate(v)
	case string:
		tt, err := ParseSimpleDate(v)
		if err != nil {
			return err
		}
		*t = tt
	case nil:
		*t = SimpleDate(time.Time{})
	default:
		return fmt.Errorf("SimpleDate: unknown type of: %T", v)
	}

	return nil
}

// Slice of SimpleDate.
type SimpleDates []SimpleDate

// First returns the first element of the date slice.
func (t SimpleDates) First() SimpleDate {
	if len(t) == 0 {
		return SimpleDate{}
	}

	return t[0]
}

// Last returns the last element of the date slice.
func (t SimpleDates) Last() SimpleDate {
	if len(t) == 0 {
		return SimpleDate{}
	}

	return t[len(t)-1]
}

// DateStrings returns a slice of string representation of the dates.
func (t SimpleDates) DateStrings() []string {
	list := make([]string, 0, len(t))
	for _, d := range t {
		list = append(list, d.String())
	}

	return list
}

// Scan completes the pg and native sql driver.Scanner interface.
func (t *SimpleDates) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("simple dates: scan: invalid type of: %T", src)
	}

	err := json.Unmarshal(data, t)
	return err
}

// Value completes the pg and native sql driver.Valuer interface.
func (t SimpleDates) Value() (driver.Value, error) {
	if len(t) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(t)
	return b, err
}

// Contains reports if the "date" exists inside "t".
func (t SimpleDates) Contains(date SimpleDate) bool {
	for _, v := range t {
		if v.Equal(date) {
			return true
		}
	}

	return false
}

// DateRangeType is the type of the date range.
type DateRangeType string

const (
	// DayRange is the date range type of a day.
	DayRange DateRangeType = "day"
	// MonthRange is the date range type of a month.
	MonthRange DateRangeType = "month"
	// WeekRange is the date range type of a week.
	WeekRange DateRangeType = "week"
	// YearRange is the date range type of a year.
	YearRange DateRangeType = "year"
)

// GetSimpleDateRange returns a slice of SimpleDate between "start" and "end" pf "date"
// based on given "typ" (WeekRange, MonthRange...).
//
// Example Code:
// date := jsonx.SimpleDateFromTime(time.Now())
// dates := jsonx.GetSimpleDateRange(date, jsonx.WeekRange, time.Monday, time.Sunday)
func GetSimpleDateRange(date SimpleDate, typ DateRangeType, startWeekday, endWeekday time.Weekday) SimpleDates {
	var dates []time.Time
	switch typ {
	case WeekRange:
		dates = timex.GetWeekdays(date.ToTime(), startWeekday, endWeekday)
	case MonthRange:
		dates = timex.GetMonthDays(date.ToTime())
	default:
		panic(fmt.Sprintf("invalid DateRangeType given: %s", typ))
	}

	simpleDates := make(SimpleDates, len(dates))
	for i, t := range dates {
		simpleDates[i] = SimpleDateFromTime(t)
	}

	return simpleDates
}
