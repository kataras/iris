package encoding

import (
	"fmt"
	"net/netip"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"uuid"
)

// Entry is a key string - value any storage.
type Entry[V any] struct {
	// Key is the key of the entry.
	Key string `json:"key" yaml:"Key" toml:"Key" ini:"key" env:"key"` // we could accept a K,V and constraint the K with comparable but we don't want to force the user to use a specific type for the key.
	// Value is the value of the entry.
	Value V `json:"value" yaml:"Value" toml:"Value" ini:"value" env:"value"` // the Value may be converted from and to specific type, usually by a TypeEvaluator on the TemplateParam of macros. Until then it's usually string unless context.Values.
	// Immutable indicates if the value is immutable.
	Immutable bool `json:"immutable" yaml:"Immutable" toml:"Immutable" ini:"immutable" env:"immutable"` // if true then it can't be changed by SetImmutable, Set or Replace.

	// Sep is optional separator.
	// It is used to split the string value to []string when the underline value is not a []string.
	// It is used on ToMap() of the Entries structure too.
	// It defaults to '/'.
	Sep string
}

const (
	defaultEntrySep = "/"
	// DefaultQueryEntrySep is the default separator used to join values as single string from the map.
	// See Entries.ToQuery.
	DefaultQueryEntrySep = ","
)

// getSep returns the separator.
func (e *Entry[V]) getSep() string { // make sure to use this instead of ".Sep" to avoid string split by letter.
	if e.Sep == "" {
		return defaultEntrySep
	}
	return e.Sep
}

const defaultPropertyEntryKeyPrefix = "iris."

// IsDefault reports whether the entry is an entry constructed by the framework itself and no by the end-developer.
func (e *Entry[V]) IsDefault() bool {
	return strings.HasPrefix(e.Key, defaultPropertyEntryKeyPrefix)
}

// String returns the value as string.
// It uses fmt.Sprintf to convert the value to string.
// Completes the fmt.Stringer interface.
func (e *Entry[V]) String() string {
	if value, ok := e.GetString(); ok {
		return value
	}

	return fmt.Sprintf("%v", e.Value)
}

// GetString returns the value as string.
// It returns false if the value is not a string.
func (e *Entry[V]) GetString() (string, bool) {
	v, ok := any(e.Value).(string)
	return v, ok
}

// MustString returns the value as string.
// It panics if the value is not a string.
func (e *Entry[V]) MustString() string {
	return any(e.Value).(string)
}

// StringSlice returns the value as []string.
// A string value is split by the entry's separator.
// It returns nil for any other type.
func (e *Entry[V]) StringSlice() []string {
	if value, ok := e.GetStringSlice(); ok {
		return value
	}

	// assume it's a string and try to convert it to []string.
	if value, ok := e.GetString(); ok {
		return strings.Split(value, e.getSep()) // macro: path.
		/*
			splitFunc := defaultPathParamTypeSplitFieldFunc
			switch len(separators) {
			case 0:
				// keep the default.
			case 1:
				if separators[0] == ',' {
					splitFunc = defaultQuerySplitFieldFunc
				}
			default:
				// a custom one was passed.
				splitFunc = func(r rune) bool {
					for _, sep := range separators {
						if r == sep {
							return true
						}
					}
					return false
				}
			}

			return strings.FieldsFunc(value, splitFunc)
		*/
	}

	return nil
}

func (e *Entry[V]) GetStringSlice() ([]string, bool) {
	v, ok := any(e.Value).([]string)
	return v, ok
}

// MustStringSlice returns the value as []string.
// It panics if the value is not a []string.
func (e *Entry[V]) MustStringSlice() []string {
	return any(e.Value).([]string)
}

// Int returns the value as int.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Int() int {
	value, _ := toNumber[int](any(e.Value))
	return value
}

func (e *Entry[V]) GetInt() (int, bool) {
	v, ok := any(e.Value).(int)
	return v, ok
}

// MustInt returns the value as int.
// It panics if the value is not an int.
func (e *Entry[V]) MustInt() int {
	return any(e.Value).(int)
}

// Int8 returns the value as int8.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Int8() int8 {
	value, _ := toNumber[int8](any(e.Value))
	return value
}

func (e *Entry[V]) GetInt8() (int8, bool) {
	v, ok := any(e.Value).(int8)
	return v, ok
}

// MustInt8 returns the value as int8.
// It panics if the value is not an int8.
func (e *Entry[V]) MustInt8() int8 {
	return any(e.Value).(int8)
}

// Int16 returns the value as int16.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Int16() int16 {
	value, _ := toNumber[int16](any(e.Value))
	return value
}

func (e *Entry[V]) GetInt16() (int16, bool) {
	v, ok := any(e.Value).(int16)
	return v, ok
}

// MustInt16 returns the value as int16.
// It panics if the value is not an int16.
func (e *Entry[V]) MustInt16() int16 {
	return any(e.Value).(int16)
}

// Int32 returns the value as int32.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Int32() int32 {
	value, _ := toNumber[int32](any(e.Value))
	return value
}

func (e *Entry[V]) GetInt32() (int32, bool) {
	v, ok := any(e.Value).(int32)
	return v, ok
}

// MustInt32 returns the value as int32.
// It panics if the value is not an int32.
func (e *Entry[V]) MustInt32() int32 {
	return any(e.Value).(int32)
}

// Int64 returns the value as int64.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Int64() int64 {
	value, _ := toNumber[int64](any(e.Value))
	return value
}

func (e *Entry[V]) GetInt64() (int64, bool) {
	v, ok := any(e.Value).(int64)
	return v, ok
}

// MustInt64 returns the value as int64.
// It panics if the value is not an int64.
func (e *Entry[V]) MustInt64() int64 {
	return any(e.Value).(int64)
}

// Uint returns the value as uint.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Uint() uint {
	value, _ := toNumber[uint](any(e.Value))
	return value
}

func (e *Entry[V]) GetUint() (uint, bool) {
	v, ok := any(e.Value).(uint)
	return v, ok
}

// MustUint returns the value as uint.
// It panics if the value is not a uint.
func (e *Entry[V]) MustUint() uint {
	return any(e.Value).(uint)
}

// Uint8 returns the value as uint8.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Uint8() uint8 {
	value, _ := toNumber[uint8](any(e.Value))
	return value
}

func (e *Entry[V]) GetUint8() (uint8, bool) {
	v, ok := any(e.Value).(uint8)
	return v, ok
}

// MustUint8 returns the value as uint8.
// It panics if the value is not a uint8.
func (e *Entry[V]) MustUint8() uint8 {
	return any(e.Value).(uint8)
}

// Uint16 returns the value as uint16.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Uint16() uint16 {
	value, _ := toNumber[uint16](any(e.Value))
	return value
}

func (e *Entry[V]) GetUint16() (uint16, bool) {
	v, ok := any(e.Value).(uint16)
	return v, ok
}

// MustUint16 returns the value as uint16.
// It panics if the value is not a uint16.
func (e *Entry[V]) MustUint16() uint16 {
	return any(e.Value).(uint16)
}

// Uint32 returns the value as uint32.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Uint32() uint32 {
	value, _ := toNumber[uint32](any(e.Value))
	return value
}

func (e *Entry[V]) GetUint32() (uint32, bool) {
	v, ok := any(e.Value).(uint32)
	return v, ok
}

// MustUint32 returns the value as uint32.
// It panics if the value is not a uint32.
func (e *Entry[V]) MustUint32() uint32 {
	return any(e.Value).(uint32)
}

// Uint64 returns the value as uint64.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Uint64() uint64 {
	value, _ := toNumber[uint64](any(e.Value))
	return value
}

func (e *Entry[V]) GetUint64() (uint64, bool) {
	v, ok := any(e.Value).(uint64)
	return v, ok
}

// MustUint64 returns the value as uint64.
// It panics if the value is not a uint64.
func (e *Entry[V]) MustUint64() uint64 {
	return any(e.Value).(uint64)
}

// Float32 returns the value as float32.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Float32() float32 {
	value, _ := toNumber[float32](any(e.Value))
	return value
}

func (e *Entry[V]) GetFloat32() (float32, bool) {
	v, ok := any(e.Value).(float32)
	return v, ok
}

// MustFloat32 returns the value as float32.
// It panics if the value is not a float32.
func (e *Entry[V]) MustFloat32() float32 {
	return any(e.Value).(float32)
}

// Float64 returns the value as float64.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Float64() float64 {
	value, _ := toNumber[float64](any(e.Value))
	return value
}

func (e *Entry[V]) GetFloat64() (float64, bool) {
	v, ok := any(e.Value).(float64)
	return v, ok
}

// MustFloat64 returns the value as float64.
// It panics if the value is not a float64.
func (e *Entry[V]) MustFloat64() float64 {
	return any(e.Value).(float64)
}

// Bool returns the value as bool.
// A string value is parsed by strconv.ParseBool.
// It returns false for any other type and for a string that does not parse.
func (e *Entry[V]) Bool() bool {
	if value, ok := e.GetBool(); ok {
		return value
	}

	// assume it's a string and try to convert it to bool.
	if value, ok := e.GetString(); ok {
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}

	return false
}

func (e *Entry[V]) GetBool() (bool, bool) {
	v, ok := any(e.Value).(bool)
	return v, ok
}

// MustBool returns the value as bool.
// It panics if the value is not a bool.
func (e *Entry[V]) MustBool() bool {
	return any(e.Value).(bool)
}

// Byte returns the value as byte.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Byte() byte {
	value, _ := toNumber[byte](any(e.Value))
	return value
}

func (e *Entry[V]) GetByte() (byte, bool) {
	v, ok := any(e.Value).(byte)
	return v, ok
}

// MustByte returns the value as byte.
// It panics if the value is not a byte.
func (e *Entry[V]) MustByte() byte {
	return any(e.Value).(byte)
}

// Rune returns the value as rune.
// A value stored as any other Go numeric is converted when it survives the
// conversion intact, and a numeric string is parsed. It returns 0 otherwise.
func (e *Entry[V]) Rune() rune {
	value, _ := toNumber[rune](any(e.Value))
	return value
}

func (e *Entry[V]) GetRune() (rune, bool) {
	v, ok := any(e.Value).(rune)
	return v, ok
}

// MustRune returns the value as rune.
// It panics if the value is not a rune.
func (e *Entry[V]) MustRune() rune {
	return any(e.Value).(rune)
}

var zeroDate time.Time

const (
	simpleDateLayout  = "2006/01/02"
	simpleDateLayout2 = "2006-01-02"
)

// ParseDate is a helper function that can be used to parse a date string.
var ParseDate = func(paramValue string) (time.Time, error) {
	tt, err := time.Parse(simpleDateLayout, paramValue)
	if err != nil {
		tt2, err2 := time.Parse(simpleDateLayout2, paramValue)
		if err2 != nil {
			return zeroDate, fmt.Errorf("%s: failed to parse time (2006/01/02, 2006-01-02): %w, %w", paramValue, err, err2)
		}

		tt = tt2
	}

	return tt, nil
}

// Date returns the value as time.Time (year/month/day).
// A string value is passed through ParseDate first.
// It returns the zero time.Time if the value is neither a time.Time nor a parsable date string.
func (e *Entry[V]) Date() time.Time {
	if value, ok := e.GetDate(); ok {
		return value
	}

	// assume it's a string and try to convert it to time.time.
	if value, ok := e.GetString(); ok {
		tt, err := ParseDate(value)
		if err != nil {
			return zeroDate
		}
		return tt
	}

	return zeroDate
}

// GetDate returns the value as time.Time.
// It returns false if the value is not a time.Time.
func (e *Entry[V]) GetDate() (time.Time, bool) {
	v, ok := any(e.Value).(time.Time)
	return v, ok
}

// MustDate returns the value as time.Time.
// It panics if the value is not a time.Time.
func (e *Entry[V]) MustDate() time.Time {
	return any(e.Value).(time.Time)
}

var (
	// ErrParamNotWeekday is fired when the parameter value is not a form of a time.Weekday.
	ErrParamNotWeekday = fmt.Errorf("parameter is not a valid weekday")

	longDayNames = map[string]time.Weekday{
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

	// ParseWeekday is a helper function that can be used to parse a weekday string.
	ParseWeekday = func(paramValue string) (time.Weekday, error) {
		d, ok := longDayNames[paramValue]
		if !ok {
			// try parse from integer.
			n, err := strconv.Atoi(paramValue)
			if err != nil {
				return -1, fmt.Errorf("%s: %w", paramValue, err)
			}

			if n < 0 || n > 6 {
				return -1, fmt.Errorf("%s: %w", paramValue, ErrParamNotWeekday)
			}

			return time.Weekday(n), nil
		}

		return d, nil
	}
)

// Weekday returns the value as time.Weekday.
// A string value is parsed by ParseWeekday.
// It returns -1 for any other type and for a string that does not parse.
func (e *Entry[V]) Weekday() time.Weekday {
	if value, ok := e.GetWeekday(); ok {
		return value
	}

	// assume it's a string and try to convert it to time.Weekday.
	if value, ok := e.GetString(); ok {
		wd, err := ParseWeekday(value)
		if err != nil {
			return -1
		}
		return wd
	}

	return -1
}

// GetWeekday returns the value as time.Weekday.
// It returns false if the value is not a time.Weekday.
func (e *Entry[V]) GetWeekday() (time.Weekday, bool) {
	v, ok := any(e.Value).(time.Weekday)
	return v, ok
}

// MustWeekday returns the value as time.Weekday.
// It panics if the value is not a time.Weekday.
func (e *Entry[V]) MustWeekday() time.Weekday {
	return any(e.Value).(time.Weekday)
}

// UUID returns the value as uuid.UUID.
// A string value is passed through uuid.Parse, which accepts the canonical
// hyphenated form, the braced and the urn:uuid: forms and the bare 32 hex
// digits, in any letter case.
// It returns the zero UUID (uuid.Nil()) if the value is neither a uuid.UUID
// nor a parsable string.
func (e *Entry[V]) UUID() uuid.UUID {
	if value, ok := e.GetUUID(); ok {
		return value
	}

	// assume it's a string and try to convert it to uuid.UUID.
	if value, ok := e.GetString(); ok {
		if id, err := uuid.Parse(value); err == nil {
			return id
		}
	}

	return uuid.UUID{}
}

// GetUUID returns the value as uuid.UUID.
// It returns false if the value is not a uuid.UUID.
func (e *Entry[V]) GetUUID() (uuid.UUID, bool) {
	v, ok := any(e.Value).(uuid.UUID)
	return v, ok
}

// MustUUID returns the value as uuid.UUID.
// It panics if the value is not a uuid.UUID.
func (e *Entry[V]) MustUUID() uuid.UUID {
	return any(e.Value).(uuid.UUID)
}

// Duration returns the value as time.Duration.
// A string value is passed through time.ParseDuration ("30s", "1h30m").
// A number is not a duration, because it carries no unit.
// It returns 0 if the value is neither a time.Duration nor a parsable string.
func (e *Entry[V]) Duration() time.Duration {
	if value, ok := e.GetDuration(); ok {
		return value
	}

	// assume it's a string and try to convert it to time.Duration.
	if value, ok := e.GetString(); ok {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}

	return 0
}

// GetDuration returns the value as time.Duration.
// It returns false if the value is not a time.Duration.
func (e *Entry[V]) GetDuration() (time.Duration, bool) {
	v, ok := any(e.Value).(time.Duration)
	return v, ok
}

// MustDuration returns the value as time.Duration.
// It panics if the value is not a time.Duration.
func (e *Entry[V]) MustDuration() time.Duration {
	return any(e.Value).(time.Duration)
}

// ParseTime is a helper function that can be used to parse a time string.
// It accepts the RFC 3339 form ("2006-01-02T15:04:05Z07:00"), with or without
// fractional seconds. Replace it to accept other layouts through Entry.Time.
var ParseTime = func(paramValue string) (time.Time, error) {
	return time.Parse(time.RFC3339, paramValue)
}

// Time returns the value as time.Time, a point in time.
// A string value is passed through ParseTime (RFC 3339). The calendar-day
// layouts ("2006/01/02", "2006-01-02") belong to Date, not here.
// It returns the zero time.Time if the value is neither a time.Time nor a
// parsable string.
func (e *Entry[V]) Time() time.Time {
	if value, ok := e.GetTime(); ok {
		return value
	}

	// assume it's a string and try to convert it to time.Time.
	if value, ok := e.GetString(); ok {
		if tt, err := ParseTime(value); err == nil {
			return tt
		}
	}

	return zeroDate
}

// GetTime returns the value as time.Time.
// It returns false if the value is not a time.Time.
func (e *Entry[V]) GetTime() (time.Time, bool) {
	v, ok := any(e.Value).(time.Time)
	return v, ok
}

// MustTime returns the value as time.Time.
// It panics if the value is not a time.Time.
func (e *Entry[V]) MustTime() time.Time {
	return any(e.Value).(time.Time)
}

// IP returns the value as netip.Addr.
// A string value is passed through netip.ParseAddr ("10.0.0.1", "::1",
// "fe80::1%eth0"). An address with a port or a host name is not an IP address.
// It returns the zero netip.Addr, for which IsValid reports false, if the
// value is neither a netip.Addr nor a parsable string.
func (e *Entry[V]) IP() netip.Addr {
	if value, ok := e.GetIP(); ok {
		return value
	}

	// assume it's a string and try to convert it to netip.Addr.
	if value, ok := e.GetString(); ok {
		if addr, err := netip.ParseAddr(value); err == nil {
			return addr
		}
	}

	return netip.Addr{}
}

// GetIP returns the value as netip.Addr.
// It returns false if the value is not a netip.Addr.
func (e *Entry[V]) GetIP() (netip.Addr, bool) {
	v, ok := any(e.Value).(netip.Addr)
	return v, ok
}

// MustIP returns the value as netip.Addr.
// It panics if the value is not a netip.Addr.
func (e *Entry[V]) MustIP() netip.Addr {
	return any(e.Value).(netip.Addr)
}

// GetTrim returns the value as string but it trims the spaces.
func (e *Entry[V]) GetTrim() string {
	return strings.TrimSpace(e.String())
}

// DecodeQuery returns the uri parameter as url (string)
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
// it uses just the url.QueryUnescape
func DecodeQuery(path string) string {
	if path == "" {
		return ""
	}
	encodedPath, err := url.QueryUnescape(path)
	if err != nil {
		return path
	}
	return encodedPath
}

// Escape returns the value as string but it decodes it first as url query.
func (e *Entry[V]) Escape() string {
	return DecodeQuery(DecodeQuery(e.String()))
}

// TrimFile returns a value without the last ".ext" part.
func (e *Entry[V]) TrimFile() string {
	value := e.String()

	if before, _, ok := strings.CutLast(value, "."); ok && len(before) > 1 {
		return before
	}

	return value
}

// TrimFileInt same as TrimFile but it returns the value as int.
func (e *Entry[V]) TrimFileInt() int {
	value := e.String()

	if before, _, ok := strings.CutLast(value, "."); ok && len(before) > 1 {
		value = before
	}

	v, _ := strconv.Atoi(value)
	return v
}

// trimSuffixFold returns the value without the given suffix. The suffix is
// matched without regard to letter case, and the rest of the value comes back
// as it was stored.
//
// It is a suffix and not a cutset: strings.TrimRight(value, ".html") strips
// every trailing '.', 'h', 't', 'm' and 'l', which turns "math.html" into "ma".
func (e *Entry[V]) trimSuffixFold(suffix string) string {
	value := e.String()

	if n := len(value) - len(suffix); n >= 0 && strings.EqualFold(value[n:], suffix) {
		return value[:n]
	}

	return value
}

// TrimHTML returns a value without the last ".html" part.
// The extension is matched in any letter case.
func (e *Entry[V]) TrimHTML() string {
	return e.trimSuffixFold(".html")
}

// TrimJSON returns a value without the last ".json" part.
// The extension is matched in any letter case.
func (e *Entry[V]) TrimJSON() string {
	return e.trimSuffixFold(".json")
}

// TrimXML returns a value without the last ".xml" part.
// The extension is matched in any letter case.
func (e *Entry[V]) TrimXML() string {
	return e.trimSuffixFold(".xml")
}

// IntUnslashed returns the value as int but without the first slash if found.
// It returns -1 if the value is empty or is not a valid int.
func (e *Entry[V]) IntUnslashed() int {
	v := e.String()
	if v != "" {
		if len(v) > 1 {
			if v[0] == '/' {
				v = v[1:]
			}
		}

		vInt, err := strconv.Atoi(v)
		if err != nil {
			return -1
		}
		return vInt
	}

	return -1
}

// ErrEntryNotFound may be returned from entry methods if
// a key (with a certain kind) was not found.
// Usage:
//
// var e *ErrEntryNotFound
// errors.As(err, &e)
// To check for specific key error:
// errors.As(err, &e) && e.Key == "key"
// To check for specific key-kind error:
// errors.As(err, &e) && e.Key == "key" && e.Kind == reflect.Int
//
// The target must always be a **ErrEntryNotFound (the address of a pointer variable);
// passing a struct literal makes errors.As panic, as the value type does not
// implement the error interface.
type ErrEntryNotFound struct {
	Key  string       // the entry's key.
	Kind reflect.Kind // i.e bool, int, string...
	Type reflect.Type // i.e time.Time{} or custom struct.
}

func (e *ErrEntryNotFound) Error() string {
	if e.Type == nil {
		// A probe built by hand, as the As documentation shows, carries no Type.
		return fmt.Sprintf("not found: %s as %s", e.Key, e.Kind.String())
	}

	return fmt.Sprintf("not found: %s as %s (%s)", e.Key, e.Kind.String(), e.Type.String())
}

// As can be used to manually check if the error came from the entry
// is a not found entry, the key must much in order to return true.
// The target must be a *ErrEntryNotFound probe; its non-empty Key and
// its non-Invalid Kind must match this error's ones.
// Usage:
// e.As(&ErrEntryNotFound{Key: "key", Kind: reflect.Int})
//
// Do NOT use this method directly, prefer` errors.As` method as explained above.
//
// Implements: go/src/errors/wrap.go#84
func (e *ErrEntryNotFound) As(target any) bool {
	v, ok := target.(*ErrEntryNotFound)
	if !ok {
		return false
	}

	if v.Key != "" && v.Key != e.Key {
		return false
	}

	if v.Kind != reflect.Invalid && v.Kind != e.Kind {
		return false
	}

	return true
}

// NotFound is a helper function that returns a new ErrEntryNotFound.
// It's useful when you want to return a not found error from entries.
func (e Entry[V]) NotFound(typ reflect.Type) *ErrEntryNotFound {
	return &ErrEntryNotFound{Key: e.Key, Kind: typ.Kind(), Type: typ}
}
