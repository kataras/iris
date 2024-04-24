package timex

import "time"

// RangeDate returns a function which returns a time
// between "start" and "end". When the iteration finishes
// the returned time is zero.
func RangeDate(start, end time.Time) func() time.Time {
	y, m, d := start.Date()
	start = time.Date(y, m, d, 0, 0, 0, 0, start.Location())
	y, m, d = end.Date()
	end = time.Date(y, m, d, 0, 0, 0, 0, end.Location())

	return func() time.Time {
		if start.After(end) {
			return time.Time{}
		}
		date := start
		start = start.AddDate(0, 0, 1)
		return date
	}
}

type DateRangeType string

const (
	DayRange   DateRangeType = "day"
	MonthRange DateRangeType = "month"
	WeekRange  DateRangeType = "week"
	YearRange  DateRangeType = "year"
)

// Between returns the dates from "start" to "end".
func Between(start, end time.Time) []time.Time {
	var dates []time.Time
	for df := RangeDate(start, end); ; {
		d := df()
		if d.IsZero() {
			break
		}
		dates = append(dates, d)
	}
	return dates
}

// Backwards returns a list of dates between "end" and -n (years, months, weeks or days).
func Backwards(typ DateRangeType, end time.Time, n int) []time.Time {
	var start time.Time

	switch typ {
	case DayRange:
		n = n - 1 // -7 should be -6 to get the week from today.
		start = end.AddDate(0, 0, -n)
	case WeekRange:
		// 7 should be 6 to get the week.
		start = end.AddDate(0, 0, -n*6)
	case MonthRange:
		start = end.AddDate(0, -n, 0)
	case YearRange:
		start = end.AddDate(-n, 0, 0)
	}

	return Between(start, end)
}

// BackwardsN returns the dates from back to "n" years, months, weeks or days from today.
func BackwardsN(typ DateRangeType, n int) []time.Time {
	end := time.Now()
	return Backwards(typ, end, n)
}

// BackwardsToMonday returns the dates between "end" (including "end")
// until the previous monday of the current week (including monday).
func BackwardsToMonday(end time.Time) []time.Time {
	dates := []time.Time{end}
	for end.Weekday() != time.Monday {
		end = end.AddDate(0, 0, -1)
		dates = append(dates, end)
	}
	return dates
}

// GetWeekDate returns the date of the given weekday (monday, tuesday, etc.) of the current week.
func GetWeekDate(now time.Time, weekday, start, end time.Weekday) time.Time {
	dates := GetWeekdays(now, start, end)
	for _, d := range dates {
		if d.Weekday() == weekday {
			return d
		}
	}

	return time.Time{}
}

// GetWeekStart returns the date of the first week day (startWeekday) of the current now's week.
func GetWeekStart(now time.Time, startWeekday time.Weekday) time.Time {
	offset := (int(startWeekday) - int(now.Weekday()) - 7) % 7
	date := now.Add(time.Duration(offset*24) * time.Hour)
	return date
}

// GetWeekEnd returns the date of the last week day (endWeekday) of the current now's week.
func GetWeekEnd(now time.Time, endWeekday time.Weekday) time.Time {
	offset := (int(endWeekday) - int(now.Weekday()) + 7) % 7
	date := now.Add(time.Duration(offset*24) * time.Hour)
	return date
}

// GetWeekdays returns the range between "startWeekday" and "endWeekday" of the current week.
func GetWeekdays(now time.Time, startWeekday, endWeekday time.Weekday) (dates []time.Time) {
	return Between(GetWeekStart(now, startWeekday), GetWeekEnd(now, endWeekday))
}

// GetMonthStart returns the date of the first month day of the current now's month.
func GetMonthStart(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

// GetMonthEnd returns the date of the last month day of the current now's month.
func GetMonthEnd(now time.Time) time.Time {
	now = now.UTC()
	// Add one month to the current date and subtract one day
	return time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())
}

// GetMonthDays returns the range between first and last days the current month.
func GetMonthDays(now time.Time) (dates []time.Time) {
	return Between(GetMonthStart(now), GetMonthEnd(now))
}

// GetYearStart returns the date of the first year of the current now's year.
func GetYearStart(now time.Time) time.Time {
	return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
}
