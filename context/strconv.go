package context

import (
	"fmt"
	"strconv"
	"time"
)

func strParseUint(value string) (uint, error) {
	result, err := strconv.ParseUint(value, 10, strconv.IntSize)
	if err != nil {
		return 0, err
	}

	return uint(result), nil
}

func strParseUint8(value string) (uint8, error) {
	result, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return 0, err
	}

	return uint8(result), nil
}

func strParseUint16(value string) (uint16, error) {
	result, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return 0, err
	}

	return uint16(result), nil
}

func strParseUint32(value string) (uint32, error) {
	result, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(result), nil
}

func strParseUint64(value string) (uint64, error) {
	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func strParseInt(value string) (int, error) {
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func strParseInt8(value string) (int8, error) {
	result, err := strconv.ParseInt(value, 10, 8)
	if err != nil {
		return 0, err
	}

	return int8(result), nil
}

func strParseInt16(value string) (int16, error) {
	result, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return 0, err
	}

	return int16(result), nil
}

func strParseInt32(value string) (int32, error) {
	result, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(result), nil
}

func strParseInt64(value string) (int64, error) {
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func strParseFloat32(value string) (float32, error) {
	result, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, err
	}

	return float32(result), nil
}

func strParseFloat64(value string) (float64, error) {
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func strParseComplex64(value string) (complex64, error) {
	result, err := strconv.ParseComplex(value, 64)
	if err != nil {
		return 0, err
	}

	return complex64(result), nil
}

func strParseComplex128(value string) (complex128, error) {
	result, err := strconv.ParseComplex(value, 128)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func strParseBool(value string) (bool, error) {
	result, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}

	return result, nil
}

var dayNames = map[string]time.Weekday{
	// longDayNames.
	"Sunday":    time.Sunday,
	"Monday":    time.Monday,
	"Tuesday":   time.Tuesday,
	"Wednesday": time.Wednesday,
	"Thursday":  time.Thursday,
	"Friday":    time.Friday,
	"Saturday":  time.Saturday,
	// longDayNames: lowercase.
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,

	// shortDayNames
	"Sun": time.Sunday,
	"Mon": time.Monday,
	"Tue": time.Tuesday,
	"Wed": time.Wednesday,
	"Thu": time.Thursday,
	"Fri": time.Friday,
	"Sat": time.Saturday,
	// shortDayNames: lowercase.
	"sun": time.Sunday,
	"mon": time.Monday,
	"tue": time.Tuesday,
	"wed": time.Wednesday,
	"thu": time.Thursday,
	"fri": time.Friday,
	"sat": time.Saturday,
}

func strParseWeekday(value string) (time.Weekday, error) {
	result, ok := dayNames[value]
	if !ok {
		return 0, ErrNotFound
	}

	return result, nil
}

func strParseTime(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

const (
	simpleDateLayout1 = "2006/01/02"
	simpleDateLayout2 = "2006-01-02"
)

func strParseSimpleDate(value string) (time.Time, error) {
	t1, err := strParseTime(simpleDateLayout1, value)
	if err != nil {
		t2, err2 := strParseTime(simpleDateLayout2, value)
		if err2 != nil {
			return time.Time{}, fmt.Errorf("%s, %w", err.Error(), err2)
		}

		return t2, nil
	}

	return t1, nil
}
