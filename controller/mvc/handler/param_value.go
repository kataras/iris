package handler

import (
	"math"
	"strconv"
	"uuid"
)

// ConvertPathParamValue converts a raw path parameter value, as the route macro
// evaluator stored it, into the type the handler declared for it.
//
// The macro and the handler signature are chosen independently by the
// end-developer, and they routinely disagree on the Go type:
//
//   - the "uuid" macro stores a uuid.UUID, while a handler declares a string id
//     (the default id type, which MVC maps to the uuid macro),
//   - an untyped "{id}" stores a string, while a handler declares uuid.UUID or int,
//   - the ":int" macro stores an int, while a handler declares int64.
//
// Without this conversion those handlers reject every request with 400, which
// made the whole CRUD surface unusable with a uuid.UUID id.
//
// It reports whether the conversion succeeded. It never panics: the value is
// client-controlled, so an unexpected shape has to be an error, not a crash.
func ConvertPathParamValue[Param any](value any) (Param, bool) {
	// Fast path: the stored value is already the declared type. This is a single
	// type assertion, exactly what the code did before the conversion existed.
	if v, ok := value.(Param); ok {
		return v, true
	}

	var out Param

	switch any(out).(type) {
	case uuid.UUID:
		if s, ok := value.(string); ok {
			if u, err := uuid.Parse(s); err == nil {
				return any(u).(Param), true
			}
		}
	case string:
		// A typed macro stores a number or a UUID; the handler wants its textual form.
		switch v := value.(type) {
		case uuid.UUID:
			return any(v.String()).(Param), true
		case int:
			return any(strconv.Itoa(v)).(Param), true
		case int64:
			return any(strconv.FormatInt(v, 10)).(Param), true
		case uint64:
			return any(strconv.FormatUint(v, 10)).(Param), true
		}
	case int:
		if n, ok := paramToInt64(value); ok && n >= math.MinInt && n <= math.MaxInt {
			return any(int(n)).(Param), true
		}
	case int64:
		if n, ok := paramToInt64(value); ok {
			return any(n).(Param), true
		}
	case uint64:
		if n, ok := paramToInt64(value); ok && n >= 0 {
			return any(uint64(n)).(Param), true
		}
		if s, ok := value.(string); ok {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				return any(n).(Param), true
			}
		}
	}

	return out, false
}

// paramToInt64 normalises the numeric and textual forms a macro can store.
func paramToInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case uint64:
		if v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	}

	return 0, false
}
