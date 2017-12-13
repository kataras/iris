// Package memstore contains a store which is just
// a collection of key-value entries with immutability capabilities.
//
// Developers can use that storage to their own apps if they like its behavior.
// It's fast and in the same time you get read-only access (safety) when you need it.
package memstore

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/kataras/iris/core/errors"
)

type (
	// Entry is the entry of the context storage Store - .Values()
	Entry struct {
		Key       string
		ValueRaw  interface{}
		immutable bool // if true then it can't change by its caller.
	}

	// Store is a collection of key-value entries with immutability capabilities.
	Store []Entry
)

// GetByKindOrNil will try to get this entry's value of "k" kind,
// if value is not that kind it will NOT try to convert it the "k", instead
// it will return nil, except if boolean; then it will return false
// even if the value was not bool.
//
// If the "k" kind is not a string or int or int64 or bool
// then it will return the raw value of the entry as it's.
func (e Entry) GetByKindOrNil(k reflect.Kind) interface{} {
	switch k {
	case reflect.String:
		v := e.StringDefault("__$nf")
		if v == "__$nf" {
			return nil
		}
		return v
	case reflect.Int:
		v, err := e.IntDefault(-1)
		if err != nil || v == -1 {
			return nil
		}
		return v
	case reflect.Int64:
		v, err := e.Int64Default(-1)
		if err != nil || v == -1 {
			return nil
		}
		return v
	case reflect.Bool:
		v, err := e.BoolDefault(false)
		if err != nil {
			return nil
		}
		return v
	default:
		return e.ValueRaw
	}
}

// StringDefault returns the entry's value as string.
// If not found returns "def".
func (e Entry) StringDefault(def string) string {
	v := e.ValueRaw

	if vString, ok := v.(string); ok {
		return vString
	}

	return def
}

// String returns the entry's value as string.
func (e Entry) String() string {
	return e.StringDefault("")
}

// StringTrim returns the entry's string value without trailing spaces.
func (e Entry) StringTrim() string {
	return strings.TrimSpace(e.String())
}

// ErrIntParse returns an error message when int parse failed
// it's not statical error, it depends on the failed value.
var ErrIntParse = errors.New("unable to find or parse the integer, found: %#v")

// IntDefault returns the entry's value as int.
// If not found returns "def".
func (e Entry) IntDefault(def int) (int, error) {
	v := e.ValueRaw
	if v == nil {
		return def, nil
	}
	if vint, ok := v.(int); ok {
		return vint, nil
	} else if vstring, sok := v.(string); sok {
		if vstring == "" {
			return def, nil
		}
		return strconv.Atoi(vstring)
	}

	return def, nil
}

// Int64Default returns the entry's value as int64.
// If not found returns "def".
func (e Entry) Int64Default(def int64) (int64, error) {
	v := e.ValueRaw
	if v == nil {
		return def, nil
	}
	if vint64, ok := v.(int64); ok {
		return vint64, nil
	} else if vstring, sok := v.(string); sok {
		if vstring == "" {
			return def, nil
		}
		return strconv.ParseInt(vstring, 10, 64)
	}

	return def, nil
}

// Float64Default returns the entry's value as float64.
// If not found returns "def".
func (e Entry) Float64Default(def float64) (float64, error) {
	v := e.ValueRaw
	if v == nil {
		return def, nil
	}
	if vfloat64, ok := v.(float64); ok {
		return vfloat64, nil
	} else if vstring, sok := v.(string); sok {
		if vstring == "" {
			return def, nil
		}
		return strconv.ParseFloat(vstring, 64)
	}

	return def, nil
}

// BoolDefault returns the user's value as bool.
// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
// or "0" or "f" or "F" or "FALSE" or "false" or "False".
// Any other value returns an error.
//
// If not found returns "def".
func (e Entry) BoolDefault(def bool) (bool, error) {
	v := e.ValueRaw
	if v == nil {
		return def, nil
	}

	if vBoolean, ok := v.(bool); ok {
		return vBoolean, nil
	}

	if vString, ok := v.(string); ok {
		return strconv.ParseBool(vString)
	}

	if vInt, ok := v.(int); ok {
		if vInt == 1 {
			return true, nil
		}
		return false, nil
	}

	return def, nil
}

// Value returns the value of the entry,
// respects the immutable.
func (e Entry) Value() interface{} {
	if e.immutable {
		// take its value, no pointer even if setted with a rreference.
		vv := reflect.Indirect(reflect.ValueOf(e.ValueRaw))

		// return copy of that slice
		if vv.Type().Kind() == reflect.Slice {
			newSlice := reflect.MakeSlice(vv.Type(), vv.Len(), vv.Cap())
			reflect.Copy(newSlice, vv)
			return newSlice.Interface()
		}
		// return a copy of that map
		if vv.Type().Kind() == reflect.Map {
			newMap := reflect.MakeMap(vv.Type())
			for _, k := range vv.MapKeys() {
				newMap.SetMapIndex(k, vv.MapIndex(k))
			}
			return newMap.Interface()
		}
		// if was *value it will return value{}.
		return vv.Interface()
	}
	return e.ValueRaw
}

// Save same as `Set`
// However, if "immutable" is true then saves it as immutable (same as `SetImmutable`).
//
//
// Returns the entry and true if it was just inserted, meaning that
// it will return the entry and a false boolean if the entry exists and it has been updated.
func (r *Store) Save(key string, value interface{}, immutable bool) (Entry, bool) {
	args := *r
	n := len(args)

	// replace if we can, else just return
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.Key == key {
			if immutable && kv.immutable {
				// if called by `SetImmutable`
				// then allow the update, maybe it's a slice that user wants to update by SetImmutable method,
				// we should allow this
				kv.ValueRaw = value
				kv.immutable = immutable
			} else if kv.immutable == false {
				// if it was not immutable then user can alt it via `Set` and `SetImmutable`
				kv.ValueRaw = value
				kv.immutable = immutable
			}
			// else it was immutable and called by `Set` then disallow the update
			return *kv, false
		}
	}

	// expand slice to add it
	c := cap(args)
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.Key = key
		kv.ValueRaw = value
		kv.immutable = immutable
		*r = args
		return *kv, true
	}

	// add
	kv := Entry{
		Key:       key,
		ValueRaw:  value,
		immutable: immutable,
	}
	*r = append(args, kv)
	return kv, true
}

// Set saves a value to the key-value storage.
// Returns the entry and true if it was just inserted, meaning that
// it will return the entry and a false boolean if the entry exists and it has been updated.
//
// See `SetImmutable` and `Get`.
func (r *Store) Set(key string, value interface{}) (Entry, bool) {
	return r.Save(key, value, false)
}

// SetImmutable saves a value to the key-value storage.
// Unlike `Set`, the output value cannot be changed by the caller later on (when .Get OR .Set)
//
// An Immutable entry should be only changed with a `SetImmutable`, simple `Set` will not work
// if the entry was immutable, for your own safety.
//
// Returns the entry and true if it was just inserted, meaning that
// it will return the entry and a false boolean if the entry exists and it has been updated.
//
// Use it consistently, it's far slower than `Set`.
// Read more about muttable and immutable go types: https://stackoverflow.com/a/8021081
func (r *Store) SetImmutable(key string, value interface{}) (Entry, bool) {
	return r.Save(key, value, true)
}

// GetEntry returns a pointer to the "Entry" found with the given "key"
// if nothing found then it returns nil, so be careful with that,
// it's not supposed to be used by end-developers.
func (r *Store) GetEntry(key string) *Entry {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.Key == key {
			return kv
		}
	}

	return nil
}

// GetDefault returns the entry's value based on its key.
// If not found returns "def".
// This function checks for immutability as well, the rest don't.
func (r *Store) GetDefault(key string, def interface{}) interface{} {
	v := r.GetEntry(key)
	if v == nil || v.ValueRaw == nil {
		return def
	}
	vv := v.Value()
	if vv == nil {
		return def
	}
	return vv
}

// Get returns the entry's value based on its key.
// If not found returns nil.
func (r *Store) Get(key string) interface{} {
	return r.GetDefault(key, nil)
}

// Visit accepts a visitor which will be filled
// by the key-value objects.
func (r *Store) Visit(visitor func(key string, value interface{})) {
	args := *r
	for i, n := 0, len(args); i < n; i++ {
		kv := args[i]
		visitor(kv.Key, kv.Value())
	}
}

// GetStringDefault returns the entry's value as string, based on its key.
// If not found returns "def".
func (r *Store) GetStringDefault(key string, def string) string {
	v := r.GetEntry(key)
	if v == nil {
		return def
	}

	return v.StringDefault(def)
}

// GetString returns the entry's value as string, based on its key.
func (r *Store) GetString(key string) string {
	return r.GetStringDefault(key, "")
}

// GetStringTrim returns the entry's string value without trailing spaces.
func (r *Store) GetStringTrim(name string) string {
	return strings.TrimSpace(r.GetString(name))
}

// GetIntDefault returns the entry's value as int, based on its key.
// If not found returns "def".
func (r *Store) GetIntDefault(key string, def int) (int, error) {
	v := r.GetEntry(key)
	if v == nil {
		return def, nil
	}
	return v.IntDefault(def)
}

// GetInt returns the entry's value as int, based on its key.
// If not found returns 0.
func (r *Store) GetInt(key string) (int, error) {
	return r.GetIntDefault(key, 0)
}

// GetInt64Default returns the entry's value as int64, based on its key.
// If not found returns "def".
func (r *Store) GetInt64Default(key string, def int64) (int64, error) {
	v := r.GetEntry(key)
	if v == nil {
		return def, nil
	}
	return v.Int64Default(def)
}

// GetInt64 returns the entry's value as int64, based on its key.
// If not found returns 0.0.
func (r *Store) GetInt64(key string) (int64, error) {
	return r.GetInt64Default(key, 0.0)
}

// GetFloat64Default returns the entry's value as float64, based on its key.
// If not found returns "def".
func (r *Store) GetFloat64Default(key string, def float64) (float64, error) {
	v := r.GetEntry(key)
	if v == nil {
		return def, nil
	}
	return v.Float64Default(def)
}

// GetFloat64 returns the entry's value as float64, based on its key.
// If not found returns 0.0.
func (r *Store) GetFloat64(key string) (float64, error) {
	return r.GetFloat64Default(key, 0.0)
}

// GetBoolDefault returns the user's value as bool, based on its key.
// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
// or "0" or "f" or "F" or "FALSE" or "false" or "False".
// Any other value returns an error.
//
// If not found returns "def".
func (r *Store) GetBoolDefault(key string, def bool) (bool, error) {
	v := r.GetEntry(key)
	if v == nil {
		return def, nil
	}
	return v.BoolDefault(def)
}

// GetBool returns the user's value as bool, based on its key.
// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
// or "0" or "f" or "F" or "FALSE" or "false" or "False".
// Any other value returns an error.
//
// If not found returns false.
func (r *Store) GetBool(key string) (bool, error) {
	return r.GetBoolDefault(key, false)
}

// Remove deletes an entry linked to that "key",
// returns true if an entry is actually removed.
func (r *Store) Remove(key string) bool {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.Key == key {
			// we found the index,
			// let's remove the item by appending to the temp and
			// after set the pointer of the slice to this temp args
			args = append(args[:i], args[i+1:]...)
			*r = args
			return true
		}
	}
	return false
}

// Reset clears all the request entries.
func (r *Store) Reset() {
	*r = (*r)[0:0]
}

// Len returns the full length of the entries.
func (r *Store) Len() int {
	args := *r
	return len(args)
}

// Serialize returns the byte representation of the current Store.
func (r Store) Serialize() []byte { // note: no pointer here, ignore linters if shows up.
	b, _ := GobSerialize(r)
	return b
}
