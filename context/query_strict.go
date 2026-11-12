package context

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kataras/iris/v14/rest/types"
)

// queryOptions is the resolved option set of the strict query readers.
type queryOptions struct {
	required     bool
	defaultValue string
	hasDefault   bool
	zeroValues   []string
}

// QueryOption configures a strict query reader (QueryDate, QueryInt,
// QueryBool). See QueryRequired, QueryDefault and QueryZeroValues.
type QueryOption func(*queryOptions)

// resolveQueryOptions applies the given options in order and returns the
// resolved set.
func resolveQueryOptions(opts []QueryOption) queryOptions {
	var options queryOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return options
}

// QueryRequired makes an absent (or blank) query value a 400 Bad Request
// naming the key (or every accepted key), instead of the zero value.
//
// QueryDefault wins over it: a reader that has a value to fall back on is never
// missing one, so the combination is not an error, it is simply the default.
func QueryRequired() QueryOption {
	return func(opts *queryOptions) {
		opts.required = true
	}
}

// QueryDefault sets the raw value to read when the key, or every accepted key,
// is absent. It is parsed
// exactly like a value the client sent, so the default is stated in the same
// wire form as the parameter itself ("2020-01-02", "25", "true"), and it is
// trimmed the same way.
//
// A blank raw value is no default at all: a blank query value already reads as
// an absent one, so the reader falls back to QueryRequired or to the zero
// value instead of parsing the blank. Without that rule the three readers
// would disagree on the same option, QueryInt panicking where QueryDate
// answered a zero date.
//
// An unparsable default is a programmer error and panics on the first request.
func QueryDefault(raw string) QueryOption {
	return func(opts *queryOptions) {
		if raw = strings.TrimSpace(raw); raw == "" {
			return
		}

		opts.defaultValue = raw
		opts.hasDefault = true
	}
}

// QueryZeroValues lists the raw values that mean "unset", the placeholders
// older clients send instead of omitting the key: "0", "zero", "0000-00-00",
// "null". Such a value reads exactly like an absent one, so it takes the
// QueryDefault value when there is one and is a 400 under QueryRequired.
func QueryZeroValues(values ...string) QueryOption {
	return func(opts *queryOptions) {
		opts.zeroValues = append(opts.zeroValues, values...)
	}
}

// QueryDate returns the value of a URL query key parsed as a [types.SimpleDate]
// ("2006-01-02", or "2006-1-2").
//
// A present but unparsable value is always a 400 Bad Request naming the key and
// the shape expected: a garbage parameter must not read as a silent zero, which
// every hand-written date filter does today and answers 200 with the wrong rows.
// The second result reports whether the caller may continue; false means a
// response was already written.
//
// The key may name several alternatives separated by a space, tried in order,
// so an endpoint that answers to more than one spelling states them once:
//
//	date, ok := ctx.QueryDate("date day from")
//
// A URL query key cannot contain a space, so a key without one is a single key
// and reads exactly as before.
func (ctx *Context) QueryDate(key string, opts ...QueryOption) (types.SimpleDate, bool) {
	return readQueryValue(ctx, key, "a date of the 2006-01-02 form", parseQueryDate, resolveQueryOptions(opts))
}

// QueryInt returns the value of a URL query key parsed as an int.
//
// A present but unparsable value is always a 400 Bad Request naming the key,
// never a silent zero. See QueryDate for the options and the second result.
func (ctx *Context) QueryInt(key string, opts ...QueryOption) (int, bool) {
	return readQueryValue(ctx, key, "an integer", strconv.Atoi, resolveQueryOptions(opts))
}

// QueryBool returns the value of a URL query key parsed as a bool. It accepts
// what strconv.ParseBool accepts plus yes/no/on/off, case-insensitively.
//
// A present but unparsable value is always a 400 Bad Request naming the key,
// never a silent false. See QueryDate for the options and the second result.
func (ctx *Context) QueryBool(key string, opts ...QueryOption) (bool, bool) {
	return readQueryValue(ctx, key, "a boolean (true, false, 1, 0, yes, no, on, off)", parseQueryBool, resolveQueryOptions(opts))
}

// parseQueryDate parses a query value as a date. It rejects the raw values the
// date parser itself maps to the zero date without an error, because what a
// client may write to mean "unset" is the caller's decision through
// QueryZeroValues, not the parser's.
func parseQueryDate(raw string) (types.SimpleDate, error) {
	if strings.EqualFold(raw, "null") {
		return types.SimpleDate{}, fmt.Errorf("not a date: %s", raw)
	}

	return types.ParseSimpleDate(raw)
}

// parseQueryBool parses the strconv.ParseBool forms plus the yes/no/on/off
// pairs clients and dashboards send for a checkbox.
func parseQueryBool(raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "yes", "on":
		return true, nil
	case "no", "off":
		return false, nil
	}

	return strconv.ParseBool(raw)
}

// readQueryValue is the one path of every strict query reader: it resolves the
// raw value (the query value, a zero placeholder, the default, or none), then
// parses it and renders the 400 itself.
func readQueryValue[T any](ctx *Context, key, expected string, parse func(string) (T, error), options queryOptions) (T, bool) {
	var zero T

	raw, matched := firstQueryValue(ctx, key)
	if raw != "" && isQueryZeroValue(raw, options.zeroValues) {
		raw = "" // a placeholder reads as an absent value.
	}

	if raw == "" {
		if options.hasDefault {
			value, err := parse(options.defaultValue)
			if err != nil {
				panic(fmt.Sprintf("iris: query %s: default value %q is not %s: %v", key, options.defaultValue, expected, err))
			}

			return value, true
		}

		if options.required {
			// No key carried a value, so the message names every one the
			// reader looked at: the client has to be told what to send.
			restErrorRenderer.HandleInvalidArgument(ctx, "missing %s query parameter: expected %s", queryKeyNames(key), expected)
			return zero, false
		}

		return zero, true
	}

	value, err := parse(raw)
	if err != nil {
		// One key carried the bad value; name that one and not its
		// alternatives, which the client did not send.
		restErrorRenderer.HandleInvalidArgument(ctx, "invalid %s query parameter: expected %s", matched, expected)
		return zero, false
	}

	return value, true
}

// queryKeys splits a key into the alternatives a strict reader accepts. A key
// holding several names separated by whitespace ("date day from") is read as
// the list of keys to try in order; a URL query key cannot contain a space, so
// there is nothing to disambiguate. A single name is a single key.
func queryKeys(key string) []string {
	keys := strings.Fields(key)
	if len(keys) == 0 {
		return []string{key}
	}

	return keys
}

// queryKeyNames renders the accepted keys for an error message: "date",
// "date or day", "date, day or from".
func queryKeyNames(key string) string {
	keys := queryKeys(key)
	switch len(keys) {
	case 1:
		return keys[0]
	case 2:
		return keys[0] + " or " + keys[1]
	default:
		return strings.Join(keys[:len(keys)-1], ", ") + " or " + keys[len(keys)-1]
	}
}

// firstQueryValue returns the first trimmed value of the first key that
// carries one, together with that key. It answers an empty raw value when
// every key is absent or blank; the key returned is then the first one, so a
// caller always has a name to report.
func firstQueryValue(ctx *Context, key string) (raw string, matched string) {
	keys := queryKeys(key)

	for _, name := range keys {
		values := ctx.Query().GetSlice(name)
		if len(values) == 0 {
			continue
		}

		if value := strings.TrimSpace(values[0]); value != "" {
			return value, name
		}
	}

	return "", keys[0]
}

// isQueryZeroValue reports whether the raw value is one of the placeholders
// QueryZeroValues declared, compared case-insensitively.
func isQueryZeroValue(raw string, zeroValues []string) bool {
	for _, zeroValue := range zeroValues {
		if strings.EqualFold(raw, zeroValue) {
			return true
		}
	}

	return false
}
