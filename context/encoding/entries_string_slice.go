package encoding

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/rest/types"
)

// StringSliceEntries is a read-only map[string][]string.
// It is used to represent a map[string][]string in a sorted and safe way.
// It contains a lot of helper methods to convert each value.
// It is used to represent URL query parameters or headers or form data.
type StringSliceEntries struct {
	StringEntries `json:"entries" yaml:"Entries" toml:"Entries" ini:"entries" env:"entries"` // no pointer, leave it read-only as it is used for request values and we also want to reest it without affecting the original values or having to nil the values.
	originalMap   map[string][]string
	sep           string // separator used to join values in the map.
}

// NewStringSliceEntries returns a new sorted StringSliceEntries.
// It converts each of the map values to a single string using the valuesJoinSeparator.
// The valuesJoinSeparator is used to join the values of the map.
// The same separator is used on StringSlice afterwards to get back the slice value.
// If valuesJoinSeparator is empty, "," is used.
func NewStringSliceEntries(m map[string][]string, valuesJoinSeparator string) StringSliceEntries /* read-only*/ {
	if len(m) == 0 { // nil or empty.
		return StringSliceEntries{}
	}

	if valuesJoinSeparator == "" {
		valuesJoinSeparator = DefaultQueryEntrySep
	}

	// Convert all to single strings, so all methods of the existing Entries work as expected.
	entries := make(StringEntries, 0, len(m))

	for _, k := range slices.Sorted(maps.Keys(m)) {
		v := m[k]
		entry := StringEntry{
			Key:   k,
			Value: strings.Join(v, valuesJoinSeparator),
			Sep:   valuesJoinSeparator,
		}
		entries.AddEntry(entry)
	}

	return StringSliceEntries{
		StringEntries: entries,
		originalMap:   m,
		sep:           valuesJoinSeparator,
	}
}

// GetSeparator returns the separator used to join values in the map.
func (e StringSliceEntries) GetSeparator() string {
	return e.sep
}

// IsEmpty returns true if the original map is empty.
func (e StringSliceEntries) IsEmpty() bool {
	return len(e.originalMap) == 0
}

// ToQuery returns the original map as url.Values.
func (e StringSliceEntries) ToQuery() url.Values { // override the entries.ToQuery for faster access.
	return e.originalMap
}

// ToHeader returns the original map as http.Header.
func (e StringSliceEntries) ToHeader() http.Header {
	return e.originalMap
}

// ToMap returns the original map.
func (e StringSliceEntries) ToMap() map[string][]string { // override the entries.ToMap for faster access.
	return e.originalMap
}

// Exists returns true if the key exists in the original map.
func (e StringSliceEntries) Exists(key string) bool { // override the original entries.Exists for faster access.
	_, found := e.originalMap[key]
	return found
}

// GetSlice returns the slice of values of a key from the original map.
func (e StringSliceEntries) GetSlice(key string) []string {
	return e.originalMap[key]
}

// GetSimpleDate returns the first non-empty value of the given keys parsed as a
// [types.SimpleDate] ("2006-01-02", or "2006-1-2"). Keys are tried in order
// and default to "date" when none is given; a value is trimmed and a blank one
// counts as absent.
//
// It reports whether any key was present: a missing key returns
// (zero, false, nil) so the caller decides the default (today, an error, no
// filter), and an unparsable value returns (zero, true, err) so a client typo
// is not mistaken for an absent parameter. It renders nothing; the caller
// writes the response.
//
//	date, found, err := ctx.Query().GetSimpleDate("date", "day")
//	if err != nil {
//		errors.InvalidArgument.Detailf(ctx, "invalid date query parameter", "%v", err)
//		return
//	}
//	if !found {
//		date = types.SimpleDateFromTime(time.Now().UTC())
//	}
func (e StringSliceEntries) GetSimpleDate(keys ...string) (types.SimpleDate, bool, error) {
	if len(keys) == 0 {
		keys = []string{"date"}
	}

	for _, key := range keys {
		values := e.GetSlice(key)
		if len(values) == 0 {
			continue
		}

		value := strings.TrimSpace(values[0])
		if value == "" {
			continue
		}

		date, err := types.ParseSimpleDate(value)
		if err != nil {
			return types.SimpleDate{}, true, err
		}

		return date, true, nil
	}

	return types.SimpleDate{}, false, nil
}
