package timex

import (
	"fmt"
	"testing"
	"time"
)

const ISO8601Layout = "2006-01-02T15:04:05"

func TestMonthAndYearStart(t *testing.T) {
	now, err := time.Parse(ISO8601Layout, "2021-04-21T00:00:00")
	if err != nil {
		t.Fatal(err)
	}

	startMonthDate := GetMonthStart(now)
	if expected, got := "2021-04-01 00:00:00 +0000 UTC", startMonthDate.String(); expected != got {
		t.Logf("start of the current month: expected: %s but got: %s", expected, got)
	}

	startYearDate := GetYearStart(now)
	if expected, got := "2021-01-01 00:00:00 +0000 UTC", startYearDate.String(); expected != got {
		t.Logf("start of the current year: expected: %s but got: %s", expected, got)
	}
}

func TestGetWeekEnd(t *testing.T) {
	var tests = []struct {
		End             time.Weekday
		Dates           []string
		ExpectedDateEnd string
	}{
		{ // 1. Test sunday as end.
			End: time.Sunday,
			Dates: []string{
				"2022-01-17T00:00:00", // 1.
				"2022-01-18T00:00:00", // 2.
				"2022-01-19T00:00:00", // 3.
				"2022-01-20T00:00:00", // 4.
				"2022-01-21T00:00:00", // 5.
				"2022-01-22T00:00:00", // 6.
				"2022-01-23T00:00:00", // 7.
			},
			ExpectedDateEnd: "2022-01-23T00:00:00",
		},
		{ // 1. Test saturday as end.
			End: time.Saturday,
			Dates: []string{
				"2022-01-23T00:00:00", // Sunday.
				"2022-01-24T00:00:00",
				"2022-01-25T00:00:00",
				"2022-01-26T00:00:00",
				"2022-01-27T00:00:00",
				"2022-01-28T00:00:00",
				"2022-01-29T00:00:00",
			},
			ExpectedDateEnd: "2022-01-29T00:00:00",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(fmt.Sprintf("%s[%d]", t.Name(), i+1), func(t *testing.T) {
			for j, date := range tt.Dates {
				now, err := time.Parse(ISO8601Layout, date)
				if err != nil {
					t.Fatal(err)
				}

				weekEndDate := GetWeekEnd(now, tt.End)
				if got := weekEndDate.Format(ISO8601Layout); got != tt.ExpectedDateEnd {
					t.Fatalf("[%d] expected week end date: %s but got: %s ", j+1, tt.ExpectedDateEnd, got)
				}
			}
		})
	}
}

func TestGetWeekDate(t *testing.T) {
	now, err := time.Parse(ISO8601Layout, "2022-02-10T00:00:00")
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		Now          time.Time
		Start        time.Weekday
		End          time.Weekday
		Weekday      time.Weekday
		ExpectedDate string
	}{
		{
			Now:          now,
			Start:        time.Monday,
			End:          time.Sunday,
			Weekday:      time.Monday,
			ExpectedDate: "2022-02-07T00:00:00",
		},
		{
			Now:          now,
			Start:        time.Monday,
			End:          time.Sunday,
			Weekday:      time.Tuesday,
			ExpectedDate: "2022-02-08T00:00:00",
		},
		{
			Now:          now,
			Start:        time.Monday,
			End:          time.Sunday,
			Weekday:      time.Wednesday,
			ExpectedDate: "2022-02-09T00:00:00",
		},
		{
			Now:          now,
			Start:        time.Monday,
			End:          time.Sunday,
			Weekday:      time.Thursday,
			ExpectedDate: "2022-02-10T00:00:00",
		},
		{
			Now:          now,
			Start:        time.Monday,
			End:          time.Sunday,
			Weekday:      time.Friday,
			ExpectedDate: "2022-02-11T00:00:00",
		},
		{
			Now:          now,
			Start:        time.Monday,
			End:          time.Sunday,
			Weekday:      time.Saturday,
			ExpectedDate: "2022-02-12T00:00:00",
		},
		{
			Now:          now,
			Start:        time.Monday,
			End:          time.Sunday,
			Weekday:      time.Sunday,
			ExpectedDate: "2022-02-13T00:00:00",
		},
		// Test sunday to saturday.
		{
			Now:          now,
			Start:        time.Sunday,
			End:          time.Saturday,
			Weekday:      time.Wednesday,
			ExpectedDate: "2022-02-09T00:00:00",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(fmt.Sprintf("%s[%s]", t.Name(), tt.Weekday.String()), func(t *testing.T) {
			weekDate := GetWeekDate(tt.Now, tt.Weekday, tt.Start, tt.End)
			if got := weekDate.Format(ISO8601Layout); got != tt.ExpectedDate {
				t.Fatalf("[%d] expected week date: %s but got: %s ", i+1, tt.ExpectedDate, got)
			}
		})
	}
}

func TestGetMonthDays(t *testing.T) {
	now, err := time.Parse(ISO8601Layout, "2023-12-12T00:00:00")
	if err != nil {
		t.Fatal(err)
	}

	dates := GetMonthDays(now)
	expectedDates := []string{
		"2023-12-01 00:00:00 +0000 UTC",
		"2023-12-02 00:00:00 +0000 UTC",
		"2023-12-03 00:00:00 +0000 UTC",
		"2023-12-04 00:00:00 +0000 UTC",
		"2023-12-05 00:00:00 +0000 UTC",
		"2023-12-06 00:00:00 +0000 UTC",
		"2023-12-07 00:00:00 +0000 UTC",
		"2023-12-08 00:00:00 +0000 UTC",
		"2023-12-09 00:00:00 +0000 UTC",
		"2023-12-10 00:00:00 +0000 UTC",
		"2023-12-11 00:00:00 +0000 UTC",
		"2023-12-12 00:00:00 +0000 UTC",
		"2023-12-13 00:00:00 +0000 UTC",
		"2023-12-14 00:00:00 +0000 UTC",
		"2023-12-15 00:00:00 +0000 UTC",
		"2023-12-16 00:00:00 +0000 UTC",
		"2023-12-17 00:00:00 +0000 UTC",
		"2023-12-18 00:00:00 +0000 UTC",
		"2023-12-19 00:00:00 +0000 UTC",
		"2023-12-20 00:00:00 +0000 UTC",
		"2023-12-21 00:00:00 +0000 UTC",
		"2023-12-22 00:00:00 +0000 UTC",
		"2023-12-23 00:00:00 +0000 UTC",
		"2023-12-24 00:00:00 +0000 UTC",
		"2023-12-25 00:00:00 +0000 UTC",
		"2023-12-26 00:00:00 +0000 UTC",
		"2023-12-27 00:00:00 +0000 UTC",
		"2023-12-28 00:00:00 +0000 UTC",
		"2023-12-29 00:00:00 +0000 UTC",
		"2023-12-30 00:00:00 +0000 UTC",
		"2023-12-31 00:00:00 +0000 UTC",
	}

	for i, d := range dates {
		if expectedDates[i] != d.String() {
			t.Fatalf("expected: %s but got: %s", expectedDates[i], d.String())
		}
	}
}

func TestGetWeekdays(t *testing.T) {
	var tests = []struct {
		Date          string
		ExpectedDates []string
	}{
		{
			Date: "2021-02-04T00:00:00",
			ExpectedDates: []string{
				"2021-02-01 00:00:00 +0000 UTC",
				"2021-02-02 00:00:00 +0000 UTC",
				"2021-02-03 00:00:00 +0000 UTC",
				"2021-02-04 00:00:00 +0000 UTC",
				"2021-02-05 00:00:00 +0000 UTC",
				"2021-02-06 00:00:00 +0000 UTC",
				"2021-02-07 00:00:00 +0000 UTC",
			},
		},
		{ // It's monday.
			Date: "2022-01-17T00:00:00",
			ExpectedDates: []string{
				"2022-01-17 00:00:00 +0000 UTC",
				"2022-01-18 00:00:00 +0000 UTC",
				"2022-01-19 00:00:00 +0000 UTC",
				"2022-01-20 00:00:00 +0000 UTC",
				"2022-01-21 00:00:00 +0000 UTC",
				"2022-01-22 00:00:00 +0000 UTC",
				"2022-01-23 00:00:00 +0000 UTC",
			},
		},
		{ // Test all other days by order.
			Date: "2022-01-18T00:00:00",
			ExpectedDates: []string{
				"2022-01-17 00:00:00 +0000 UTC",
				"2022-01-18 00:00:00 +0000 UTC",
				"2022-01-19 00:00:00 +0000 UTC",
				"2022-01-20 00:00:00 +0000 UTC",
				"2022-01-21 00:00:00 +0000 UTC",
				"2022-01-22 00:00:00 +0000 UTC",
				"2022-01-23 00:00:00 +0000 UTC",
			},
		},
		{
			Date: "2022-01-19T00:00:00",
			ExpectedDates: []string{
				"2022-01-17 00:00:00 +0000 UTC",
				"2022-01-18 00:00:00 +0000 UTC",
				"2022-01-19 00:00:00 +0000 UTC",
				"2022-01-20 00:00:00 +0000 UTC",
				"2022-01-21 00:00:00 +0000 UTC",
				"2022-01-22 00:00:00 +0000 UTC",
				"2022-01-23 00:00:00 +0000 UTC",
			},
		},
		{
			Date: "2022-01-20T00:00:00",
			ExpectedDates: []string{
				"2022-01-17 00:00:00 +0000 UTC",
				"2022-01-18 00:00:00 +0000 UTC",
				"2022-01-19 00:00:00 +0000 UTC",
				"2022-01-20 00:00:00 +0000 UTC",
				"2022-01-21 00:00:00 +0000 UTC",
				"2022-01-22 00:00:00 +0000 UTC",
				"2022-01-23 00:00:00 +0000 UTC",
			},
		},
		{
			Date: "2022-01-21T00:00:00",
			ExpectedDates: []string{
				"2022-01-17 00:00:00 +0000 UTC",
				"2022-01-18 00:00:00 +0000 UTC",
				"2022-01-19 00:00:00 +0000 UTC",
				"2022-01-20 00:00:00 +0000 UTC",
				"2022-01-21 00:00:00 +0000 UTC",
				"2022-01-22 00:00:00 +0000 UTC",
				"2022-01-23 00:00:00 +0000 UTC",
			},
		},
		{
			Date: "2022-01-22T00:00:00",
			ExpectedDates: []string{
				"2022-01-17 00:00:00 +0000 UTC",
				"2022-01-18 00:00:00 +0000 UTC",
				"2022-01-19 00:00:00 +0000 UTC",
				"2022-01-20 00:00:00 +0000 UTC",
				"2022-01-21 00:00:00 +0000 UTC",
				"2022-01-22 00:00:00 +0000 UTC",
				"2022-01-23 00:00:00 +0000 UTC",
			},
		},
		{ // Sunday.
			Date: "2022-01-23T00:00:00",
			ExpectedDates: []string{
				"2022-01-17 00:00:00 +0000 UTC",
				"2022-01-18 00:00:00 +0000 UTC",
				"2022-01-19 00:00:00 +0000 UTC",
				"2022-01-20 00:00:00 +0000 UTC",
				"2022-01-21 00:00:00 +0000 UTC",
				"2022-01-22 00:00:00 +0000 UTC",
				"2022-01-23 00:00:00 +0000 UTC",
			},
		},
		{ // Test 1st Jenuary (Saturday) .
			Date: "2022-01-01T00:00:00",
			ExpectedDates: []string{
				"2021-12-27 00:00:00 +0000 UTC", // monday.
				"2021-12-28 00:00:00 +0000 UTC",
				"2021-12-29 00:00:00 +0000 UTC",
				"2021-12-30 00:00:00 +0000 UTC",
				"2021-12-31 00:00:00 +0000 UTC",
				"2022-01-01 00:00:00 +0000 UTC",
				"2022-01-02 00:00:00 +0000 UTC", // sunday.
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(fmt.Sprintf("%s[%d]", t.Name(), i+1), func(t *testing.T) {
			now, err := time.Parse(ISO8601Layout, tt.Date)
			if err != nil {
				t.Fatal(err)
			}

			dates := GetWeekdays(now, time.Monday, time.Sunday)
			checkDates(t, "", tt.ExpectedDates, dates)
		})
	}

	// t.Logf("[%s] Current day of the week: %s", now.String(), now.Weekday().String())
}

func TestBackwardsToMonday(t *testing.T) {
	end, err := time.Parse(ISO8601Layout, "2021-04-05T00:00:00")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"2021-04-05 00:00:00 +0000 UTC",
	}

	// Test when when today is monday.
	dates := BackwardsToMonday(end)
	checkDates(t, "", expected, dates)

	// Test when today is tuesday.
	end, err = time.Parse(ISO8601Layout, "2021-04-06T00:00:00")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string{
		"2021-04-06 00:00:00 +0000 UTC",
		"2021-04-05 00:00:00 +0000 UTC",
	}

	dates = BackwardsToMonday(end)
	checkDates(t, "", expected, dates)

	// Test when today is thursday.
	end, err = time.Parse(ISO8601Layout, "2021-04-08T00:00:00")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string{
		"2021-04-08 00:00:00 +0000 UTC",
		"2021-04-07 00:00:00 +0000 UTC",
		"2021-04-06 00:00:00 +0000 UTC",
		"2021-04-05 00:00:00 +0000 UTC",
	}

	dates = BackwardsToMonday(end)
	checkDates(t, "", expected, dates)

	// Test when today is sunday.
	end, err = time.Parse(ISO8601Layout, "2021-04-10T00:00:00")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string{
		"2021-04-10 00:00:00 +0000 UTC",
		"2021-04-09 00:00:00 +0000 UTC",
		"2021-04-08 00:00:00 +0000 UTC",
		"2021-04-07 00:00:00 +0000 UTC",
		"2021-04-06 00:00:00 +0000 UTC",
		"2021-04-05 00:00:00 +0000 UTC",
	}

	dates = BackwardsToMonday(end)
	checkDates(t, "", expected, dates)
}

func checkDates(t *testing.T, typ DateRangeType, expected []string, dates []time.Time) {
	t.Helper()

	t.Logf("[%s] length of days: %d", typ, len(dates))

	if expectedLength, gotLength := len(expected), len(dates); expectedLength != gotLength {
		t.Logf("[%s] expected days length: %d but got: %d", typ, expectedLength, gotLength)

		if gotLength > expectedLength {
			t.Logf("Got %d extra date(s), list of all dates we've got:", gotLength-expectedLength)
			for i, gotDate := range dates {
				t.Logf("[%d] %s ", i, gotDate.String())
			}
		}

		t.FailNow()
	}

	for i, date := range dates {
		//	t.Logf("[%d] %s", i, date.String())
		if expected, got := expected[i], date.String(); expected != got {
			t.Fatalf("[%d] [%s] expected date: %s but got: %s", i, typ, expected, got)
		}
	}
}

func TestBetweenAndBackwardsN(t *testing.T) {
	start, err := time.Parse(ISO8601Layout, "2021-03-26T00:00:00")
	if err != nil {
		t.Fatal(err)
	}

	end, err := time.Parse(ISO8601Layout, "2021-04-01T00:00:00")
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"2021-03-26 00:00:00 +0000 UTC",
		"2021-03-27 00:00:00 +0000 UTC",
		"2021-03-28 00:00:00 +0000 UTC",
		"2021-03-29 00:00:00 +0000 UTC",
		"2021-03-30 00:00:00 +0000 UTC",
		"2021-03-31 00:00:00 +0000 UTC",
		"2021-04-01 00:00:00 +0000 UTC",
	}

	dates := Between(start, end)
	checkDates(t, "", expected, dates)

	dates = Backwards(DayRange, end, 7)
	checkDates(t, DayRange, expected, dates)

	dates = Backwards(WeekRange, end, 1)
	checkDates(t, WeekRange, expected, dates)

	dates = Backwards(MonthRange, end, 2)
	expectedMonthDates := []string{
		"2021-02-01 00:00:00 +0000 UTC",
		"2021-02-02 00:00:00 +0000 UTC",
		"2021-02-03 00:00:00 +0000 UTC",
		"2021-02-04 00:00:00 +0000 UTC",
		"2021-02-05 00:00:00 +0000 UTC",
		"2021-02-06 00:00:00 +0000 UTC",
		"2021-02-07 00:00:00 +0000 UTC",
		"2021-02-08 00:00:00 +0000 UTC",
		"2021-02-09 00:00:00 +0000 UTC",
		"2021-02-10 00:00:00 +0000 UTC",
		"2021-02-11 00:00:00 +0000 UTC",
		"2021-02-12 00:00:00 +0000 UTC",
		"2021-02-13 00:00:00 +0000 UTC",
		"2021-02-14 00:00:00 +0000 UTC",
		"2021-02-15 00:00:00 +0000 UTC",
		"2021-02-16 00:00:00 +0000 UTC",
		"2021-02-17 00:00:00 +0000 UTC",
		"2021-02-18 00:00:00 +0000 UTC",
		"2021-02-19 00:00:00 +0000 UTC",
		"2021-02-20 00:00:00 +0000 UTC",
		"2021-02-21 00:00:00 +0000 UTC",
		"2021-02-22 00:00:00 +0000 UTC",
		"2021-02-23 00:00:00 +0000 UTC",
		"2021-02-24 00:00:00 +0000 UTC",
		"2021-02-25 00:00:00 +0000 UTC",
		"2021-02-26 00:00:00 +0000 UTC",
		"2021-02-27 00:00:00 +0000 UTC",
		"2021-02-28 00:00:00 +0000 UTC",
		"2021-03-01 00:00:00 +0000 UTC",
		"2021-03-02 00:00:00 +0000 UTC",
		"2021-03-03 00:00:00 +0000 UTC",
		"2021-03-04 00:00:00 +0000 UTC",
		"2021-03-05 00:00:00 +0000 UTC",
		"2021-03-06 00:00:00 +0000 UTC",
		"2021-03-07 00:00:00 +0000 UTC",
		"2021-03-08 00:00:00 +0000 UTC",
		"2021-03-09 00:00:00 +0000 UTC",
		"2021-03-10 00:00:00 +0000 UTC",
		"2021-03-11 00:00:00 +0000 UTC",
		"2021-03-12 00:00:00 +0000 UTC",
		"2021-03-13 00:00:00 +0000 UTC",
		"2021-03-14 00:00:00 +0000 UTC",
		"2021-03-15 00:00:00 +0000 UTC",
		"2021-03-16 00:00:00 +0000 UTC",
		"2021-03-17 00:00:00 +0000 UTC",
		"2021-03-18 00:00:00 +0000 UTC",
		"2021-03-19 00:00:00 +0000 UTC",
		"2021-03-20 00:00:00 +0000 UTC",
		"2021-03-21 00:00:00 +0000 UTC",
		"2021-03-22 00:00:00 +0000 UTC",
		"2021-03-23 00:00:00 +0000 UTC",
		"2021-03-24 00:00:00 +0000 UTC",
		"2021-03-25 00:00:00 +0000 UTC",
		"2021-03-26 00:00:00 +0000 UTC",
		"2021-03-27 00:00:00 +0000 UTC",
		"2021-03-28 00:00:00 +0000 UTC",
		"2021-03-29 00:00:00 +0000 UTC",
		"2021-03-30 00:00:00 +0000 UTC",
		"2021-03-31 00:00:00 +0000 UTC",
		"2021-04-01 00:00:00 +0000 UTC",
	}

	checkDates(t, MonthRange, expectedMonthDates, dates)
}
