package encoding

import (
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"uuid"
)

// Entries is a slice of Entry.
type Entries[V any] []Entry[V] // no pointer to Entry, as they are small and we don't want to allocate memory for each of them.

// Reset clears all the request entries.
func (e *Entries[V]) Reset() {
	*e = (*e)[0:0]
}

// Set sets a key-value pair to the request entries.
// If the key already exists and it is not immutable, it replaces the value;
// an entry stored through SetImmutable is left untouched and no error is reported.
// If the key does not exist, it appends the key-value pair.
func (e *Entries[V]) Set(key string, value V) {
	e.set(key, value, false, "")
}

// SetImmutable sets a key-value pair to the request entries.
// If the key already exists and it is not already immutable, it replaces the value;
// an entry which is already immutable is left untouched and no error is reported.
// If the key does not exist, it appends the key-value pair.
// The stored value cannot be changed by Set, SetImmutable or Replace afterwards,
// it can only be removed (see Remove and Pop) or overwritten by its index
// (see SetByIndex and SetEntryByIndex).
func (e *Entries[V]) SetImmutable(key string, value V) {
	e.set(key, value, true, "")
}

func (e *Entries[V]) set(key string, value V, immutable bool, sep string) {
	list := *e

	// Replace if already exists and it was not immutable.
	for i := range list {
		entry := &list[i]
		if entry.Key == key {
			if entry.Immutable {
				// If the entry is immutable, do not replace it.
				return
			}

			entry.Value = value
			entry.Immutable = immutable
			entry.Sep = sep
			*e = list
			return
		}
	}

	if n := len(list); cap(list) > n {
		list = list[:n+1]
		entry := &list[n]
		entry.Key = key
		entry.Value = value
		entry.Immutable = immutable
		entry.Sep = sep
		*e = list
		return
	}

	list = append(list, Entry[V]{
		Key:       key,
		Value:     value,
		Immutable: immutable,
	})
	*e = list
}

// Add adds a key-value pair to the entries.
func (e *Entries[V]) Add(key string, value V) {
	e.AddEntry(Entry[V]{Key: key, Value: value})
}

// AddEntry adds a key-value pair (a Entry) to the entries.
func (e *Entries[V]) AddEntry(entryToAdd Entry[V]) {
	list := *e
	if n := len(list); cap(list) > n {
		list = list[:n+1]
		entry := &list[n]
		entry.Key = entryToAdd.Key
		entry.Value = entryToAdd.Value
		entry.Immutable = entryToAdd.Immutable
		entry.Sep = entryToAdd.Sep
		*e = list
		return
	}

	list = append(list, entryToAdd)
	*e = list
}

// SetByIndex sets the value of a key by its index.
// Replaces any existing value, even if it was immutable.
func (e *Entries[V]) SetByIndex(index int, value V) {
	(*e)[index].Value = value
}

// SetEntryByIndex sets a key-value pair (a Entry) to the entries by its index.
// Replaces any existing value, even if it was immutable.
func (e *Entries[V]) SetEntryByIndex(index int, entry Entry[V]) {
	(*e)[index] = entry
}

// Replace replaces the value of a key with a new value.
// Reports whether the key was found and the value was replaced.
func (e *Entries[V]) Replace(key string, value V) bool {
	for i, entry := range *e {
		if entry.Key == key {
			if entry.Immutable {
				// If the entry is immutable, do not replace it.
				return false
			}

			(*e)[i].Value = value
			return true
		}
	}

	return false
}

// Remove deletes a key from the entries, order is not changed.
func (e *Entries[V]) Remove(key string) bool {
	for i, entry := range *e {
		if entry.Key == key {
			*e = append((*e)[:i], (*e)[i+1:]...)
			return true
		}
	}

	return false

	/* Same as:
	list := *e
	for i, kv := range list {
		if kv.Key == key {
			copy(list[i:], list[i+1:])
			list = list[:len(list)-1]
			break
		}
	}
	*e = list
	*/
}

func (e Entries[V]) emptyValue() (v V) {
	return
}

// Exists reports whether the key exists in the request entries.
func (e Entries[V]) Exists(key string) bool {
	for _, entry := range e {
		if entry.Key == key {
			return true
		}
	}

	return false
}

// WithoutDefaults returns a new Entries without the default (builtin by the framework) key-values.
func (e Entries[V]) WithoutDefaults() Entries[V] {
	entries := make(Entries[V], 0, len(e))
	for _, entry := range e {
		if !entry.IsDefault() {
			entries = append(entries, entry)
		}
	}

	return entries
}

// Get returns the value of a key.
//
// Use `GetEntry` if you want to get the key-value pair.
// Use `Lookup` if you want to retrieve the raw value and check if the key exists.
// Use `GetString` if you want to get the value as string.
func (e Entries[V]) Get(key string) V {
	v, _ := e.Lookup(key)
	return v
}

// Bind binds the value of a key to a pointer.
func (e Entries[V]) Bind(key string, ptr *V) bool {
	if v, ok := e.Lookup(key); ok {
		*ptr = v
		return true
	}

	return false
}

// GetPointer returns the pointer of a key or nil.
func (e Entries[V]) GetPointer(key string) unsafe.Pointer {
	for _, entry := range e {
		if entry.Key == key {
			// return unsafe.Pointer((*uintptr)(unsafe.Pointer(&entry.Value)))
			// return unsafe.Pointer(any(entry.Value).(uintptr))
			// value := any(entry.Value).(*uintptr)
			// return unsafe.Pointer(value)
			return any(entry.Value).(unsafe.Pointer)
		}
	}

	return nil
}

// PopPointer deletes an entry based on the given "key" from the entries and
// returns the unsafe.Pointer of the deleted entry.
func (e *Entries[V]) PopPointer(key string) unsafe.Pointer {
	for i, entry := range *e {
		if entry.Key == key {
			*e = append((*e)[:i], (*e)[i+1:]...)
			return any(entry.Value).(unsafe.Pointer)
		}
	}

	return nil
}

// Pop deletes an entry based on the given "key" from the entries and
// returns the value of the deleted entry.
func (e *Entries[V]) Pop(key string) V {
	for i, entry := range *e {
		if entry.Key == key {
			*e = append((*e)[:i], (*e)[i+1:]...)
			return entry.Value
		}
	}

	return e.emptyValue()
}

// Lookup returns the value of a key.
// It returns false if the key does not exist.
func (e Entries[V]) Lookup(key string) (V, bool) {
	for _, entry := range e {
		if entry.Key == key {
			return entry.Value, true
		}
	}

	return e.emptyValue(), false
}

// LookupByIndex returns the value of a key by its index.
// It returns false if the key does not exist.
func (e Entries[V]) LookupByIndex(index int) (V, bool) {
	if index < 0 || index >= len(e) {
		return e.emptyValue(), false
	}
	return e[index].Value, true
}

// GetEntry returns the key-value of a key.
func (e Entries[V]) GetEntry(key string) (Entry[V], bool) {
	for _, entry := range e {
		if entry.Key == key {
			return entry, true
		}
	}

	return Entry[V]{}, false
}

// GetEntryByIndex returns the key-value pair by its index.
// It returns false if the index is out of range.
func (e Entries[V]) GetEntryByIndex(index int) (Entry[V], bool) {
	if index < 0 || index >= len(e) {
		return Entry[V]{}, false
	}
	return e[index], true
}

// GetLastEntry returns the last key-value pair.
// It returns false if the request entries are empty.
func (e Entries[V]) GetLastEntry() (Entry[V], bool) {
	if len(e) == 0 {
		return Entry[V]{}, false
	}
	return e[len(e)-1], true
}

// GetByIndex returns the value as string of a key by its index.
func (e Entries[V]) GetByIndex(index int) (string, bool) {
	if index < 0 || index >= len(e) {
		return "", false
	}
	return e[index].String(), true
}

// ForEach traverses the request entries and calls the given function.
func (e Entries[V]) ForEach(fn func(key string, value V)) {
	for _, entry := range e {
		fn(entry.Key, entry.Value)
	}
}

// ForEachString traverses the request entries and calls the given function.
// It converts the value to string before calling the function.
func (e Entries[V]) ForEachString(fn func(key string, value string)) {
	for _, entry := range e {
		fn(entry.Key, entry.String())
	}
}

// ToQuery converts the request entries to a url.Values.
// Uses the "," separator.
func (e Entries[V]) ToQuery() url.Values {
	n := len(e)
	if n == 0 {
		return nil
	}

	values := make(url.Values, n)
	for _, entry := range e {
		// []string on path entryeter, e.g.
		// /.../{tail:path}
		// Tail []string `entry:"tail"`
		switch v := any(entry.Value).(type) {
		case []string:
			values[entry.Key] = v
		case string:
			values[entry.Key] = strings.Split(v, DefaultQueryEntrySep)
		case int:
			values[entry.Key] = []string{strconv.Itoa(v)}
		default:
			values[entry.Key] = []string{entry.String()}
		}
	}

	return values
}

// ToMap converts the request entries to a map[string][]string.
// Uses the separator given for each entry.
func (e Entries[V]) ToMap() map[string][]string {
	n := len(e)
	if n == 0 {
		return nil
	}

	values := make(map[string][]string, n)
	for _, entry := range e {
		// []string on path entryeter, e.g.
		// /.../{tail:path}
		// Tail []string `entry:"tail"`
		switch v := any(entry.Value).(type) {
		case []string:
			values[entry.Key] = v
		case string:
			values[entry.Key] = strings.Split(v, entry.getSep())
		case int:
			values[entry.Key] = []string{strconv.Itoa(v)}
		default:
			values[entry.Key] = []string{entry.String()}
		}
	}

	return values
}

// MergeQuery merges the request entries with an extra url.Values.
func (e Entries[V]) MergeQuery(extra url.Values) url.Values {
	values := e.ToQuery()
	if len(extra) == 0 {
		return values
	}

	if values == nil {
		// An empty store: ToQuery has no map to give.
		values = make(url.Values, len(extra))
	}

	for key, extraValues := range extra {
		// Clip first, so that append copies. A []string the store still holds
		// may have spare capacity, and writing into it would leak this merge
		// into the next one.
		values[key] = append(slices.Clip(values[key]), extraValues...)
	}

	return values
}

// Clone returns a new copy of the request entries.
func (e Entries[V]) Clone() Entries[V] {
	return slices.Clone(e)
}

// Len returns the length of the request entries.
func (e Entries[V]) Len() int {
	return len(e)
}

// Other Go types.
//
// Each of these reads through the matching Entry accessor, so a value stored as
// any Go numeric is converted when it survives the conversion intact, and a
// numeric string is parsed. See Entry.Int.
//
// They report a missing key with a sentinel, and the sentinel differs by type
// because the types differ in what they can express: -1 for the signed and
// floating-point ones, 0 for the unsigned, "" for a string and false for a bool.
// A -1 is therefore only ever a hint. A key that is present but holds something
// unconvertible returns the zero value, not the sentinel, so 0 from GetInt means
// "present and not a number" while -1 means "absent" — and for GetUint both
// collapse to 0. Use GetEntry when the difference matters:
//
//	entry, ok := values.GetEntry("id") // ok reports presence, and nothing else does
//
// The getters for the richer standard-library types answer the zero value in
// both cases, absent and unconvertible: GetDate and GetTime the zero time.Time,
// GetUUID the zero UUID (uuid.Nil()), GetDuration 0 and GetIP an invalid
// netip.Addr. GetWeekday is the exception and keeps -1.

// GetString returns the value of a key as string.
// It returns an empty string if the key does not exist.
func (e Entries[V]) GetString(key string) string {
	if entry, ok := e.GetEntry(key); ok {
		return entry.String()
	}

	return ""
}

// GetInt returns the value of a key as int.
func (e Entries[V]) GetInt(key string) int {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Int()
	}
	return -1
}

// GetInt8 returns the value of a key as int8.
func (e Entries[V]) GetInt8(key string) int8 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Int8()
	}
	return -1
}

// GetInt16 returns the value of a key as int16.
func (e Entries[V]) GetInt16(key string) int16 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Int16()
	}
	return -1
}

// GetInt32 returns the value of a key as int32.
func (e Entries[V]) GetInt32(key string) int32 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Int32()
	}
	return -1
}

// GetInt64 returns the value of a key as int64.
func (e Entries[V]) GetInt64(key string) int64 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Int64()
	}
	return -1
}

// GetUint returns the value of a key as uint.
func (e Entries[V]) GetUint(key string) uint {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Uint()
	}
	return 0
}

// GetUint8 returns the value of a key as uint8.
func (e Entries[V]) GetUint8(key string) uint8 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Uint8()
	}
	return 0
}

// GetUint16 returns the value of a key as uint16.
func (e Entries[V]) GetUint16(key string) uint16 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Uint16()
	}
	return 0
}

// GetUint32 returns the value of a key as uint32.
func (e Entries[V]) GetUint32(key string) uint32 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Uint32()
	}
	return 0
}

// GetUint64 returns the value of a key as uint64.
func (e Entries[V]) GetUint64(key string) uint64 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Uint64()
	}
	return 0
}

// GetFloat32 returns the value of a key as float32.
func (e Entries[V]) GetFloat32(key string) float32 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Float32()
	}
	return -1
}

// GetFloat64 returns the value of a key as float64.
func (e Entries[V]) GetFloat64(key string) float64 {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Float64()
	}
	return -1
}

// GetBool returns the value of a key as bool.
func (e Entries[V]) GetBool(key string) bool {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Bool()
	}
	return false
}

// GetByte returns the value of a key as byte.
func (e Entries[V]) GetByte(key string) byte {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Byte()
	}
	return 0
}

// GetRune returns the value of a key as rune.
func (e Entries[V]) GetRune(key string) rune {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Rune()
	}
	return -1
}

// GetStringSlice returns the value of a key as []string.
// It returns nil if the key does not exist.
func (e Entries[V]) GetStringSlice(key string) []string {
	if entry, ok := e.GetEntry(key); ok {
		return entry.StringSlice()
	}

	return nil
}

// GetDate returns the value of a key as time.Time.
func (e Entries[V]) GetDate(key string) time.Time {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Date()
	}
	return zeroDate
}

// GetWeekday returns the value of a key as time.Weekday.
func (e Entries[V]) GetWeekday(key string) time.Weekday {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Weekday()
	}
	return -1
}

// GetUUID returns the value of a key as uuid.UUID.
// A string value is parsed with uuid.Parse. It returns the zero UUID if the
// key does not exist or its value is not a UUID.
func (e Entries[V]) GetUUID(key string) uuid.UUID {
	if entry, ok := e.GetEntry(key); ok {
		return entry.UUID()
	}
	return uuid.UUID{}
}

// GetDuration returns the value of a key as time.Duration.
// A string value is parsed with time.ParseDuration ("30s", "1h30m"). It
// returns 0 if the key does not exist or its value is not a duration.
func (e Entries[V]) GetDuration(key string) time.Duration {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Duration()
	}
	return 0
}

// GetTime returns the value of a key as time.Time, a point in time.
// A string value is parsed with ParseTime (RFC 3339, "2006-01-02T15:04:05Z07:00").
// For a calendar day see GetDate. It returns the zero time.Time if the key
// does not exist or its value is not a time.
func (e Entries[V]) GetTime(key string) time.Time {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Time()
	}
	return zeroDate
}

// GetIP returns the value of a key as netip.Addr.
// A string value is parsed with netip.ParseAddr. It returns the zero
// netip.Addr, for which IsValid reports false, if the key does not exist or
// its value is not an IP address.
func (e Entries[V]) GetIP(key string) netip.Addr {
	if entry, ok := e.GetEntry(key); ok {
		return entry.IP()
	}
	return netip.Addr{}
}

// GetTrim returns the value as string but it trims the spaces.
func (e Entries[V]) GetTrim(key string) string {
	if entry, ok := e.GetEntry(key); ok {
		return entry.GetTrim()
	}

	return ""
}

// GetEscape returns the value as string but it decodes it first as url query.
func (e Entries[V]) GetEscape(key string) string {
	if entry, ok := e.GetEntry(key); ok {
		return entry.Escape()
	}

	return ""
}

// GetTrimFile returns a value without the last ".ext" part.
func (e Entries[V]) GetTrimFile(key string) string {
	if entry, ok := e.GetEntry(key); ok {
		return entry.TrimFile()
	}

	return ""
}

// GetTrimFileInt same as GetTrimFile but it returns the value as int.
func (e Entries[V]) GetTrimFileInt(key string) int {
	if entry, ok := e.GetEntry(key); ok {
		return entry.TrimFileInt()
	}

	return -1
}

// GetTrimHTML returns a value without the last ".html" part.
func (e Entries[V]) GetTrimHTML(key string) string {
	if entry, ok := e.GetEntry(key); ok {
		return entry.TrimHTML()
	}

	return ""
}

// GetTrimJSON returns a value without the last ".json" part.
func (e Entries[V]) GetTrimJSON(key string) string {
	if entry, ok := e.GetEntry(key); ok {
		return entry.TrimJSON()
	}

	return ""
}

// GetTrimXML returns a value without the last ".xml" part.
func (e Entries[V]) GetTrimXML(key string) string {
	if entry, ok := e.GetEntry(key); ok {
		return entry.TrimXML()
	}

	return ""
}

// GetIntUnslashed returns the value as int but without the first slash if found.
// It returns -1 if the value wasn't found or it is not a valid int.
func (e Entries[V]) GetIntUnslashed(key string) int {
	if entry, ok := e.GetEntry(key); ok {
		return entry.IntUnslashed()
	}

	return -1
}
