// Package memstore contains a store which is just
// a collection of key-value entries with immutability capabilities.
//
// Developers can use that storage to their own apps if they like its behavior.
// It's fast and in the same time you get read-only access (safety) when you need it.
package memstore

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/kataras/iris/core/errors"
)

type (
	// ValueSetter is the interface which can be accepted as a generic solution of RequestParams or memstore when Set is the only requirement,
	// i.e internally on macro/template/TemplateParam#Eval:paramChanger.
	ValueSetter interface {
		Set(key string, newValue interface{}) (Entry, bool)
	}
	// Entry is the entry of the context storage Store - .Values()
	Entry struct {
		Key       string
		ValueRaw  interface{}
		immutable bool // if true then it can't change by its caller.
	}

	// Store is a collection of key-value entries with immutability capabilities.
	Store []Entry
)

var _ ValueSetter = (*Store)(nil)

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
	if v == nil {
		return def
	}

	if vString, ok := v.(string); ok {
		return vString
	}

	val := fmt.Sprintf("%v", v)
	if val != "" {
		return val
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

var errFindParse = errors.New("unable to find the %s with key: %s")

// IntDefault returns the entry's value as int.
// If not found returns "def" and a non-nil error.
func (e Entry) IntDefault(def int) (int, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("int", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.Atoi(vv)
		if err != nil {
			return def, err
		}
		return val, nil
	case int:
		return vv, nil
	case int8:
		return int(vv), nil
	case int16:
		return int(vv), nil
	case int32:
		return int(vv), nil
	case int64:
		return int(vv), nil
	case uint:
		return int(vv), nil
	case uint8:
		return int(vv), nil
	case uint16:
		return int(vv), nil
	case uint32:
		return int(vv), nil
	case uint64:
		return int(vv), nil
	}

	return def, errFindParse.Format("int", e.Key)
}

// Int8Default returns the entry's value as int8.
// If not found returns "def" and a non-nil error.
func (e Entry) Int8Default(def int8) (int8, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("int8", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseInt(vv, 10, 8)
		if err != nil {
			return def, err
		}
		return int8(val), nil
	case int:
		return int8(vv), nil
	case int8:
		return vv, nil
	case int16:
		return int8(vv), nil
	case int32:
		return int8(vv), nil
	case int64:
		return int8(vv), nil
	}

	return def, errFindParse.Format("int8", e.Key)
}

// Int16Default returns the entry's value as int16.
// If not found returns "def" and a non-nil error.
func (e Entry) Int16Default(def int16) (int16, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("int16", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseInt(vv, 10, 16)
		if err != nil {
			return def, err
		}
		return int16(val), nil
	case int:
		return int16(vv), nil
	case int8:
		return int16(vv), nil
	case int16:
		return vv, nil
	case int32:
		return int16(vv), nil
	case int64:
		return int16(vv), nil
	}

	return def, errFindParse.Format("int16", e.Key)
}

// Int32Default returns the entry's value as int32.
// If not found returns "def" and a non-nil error.
func (e Entry) Int32Default(def int32) (int32, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("int32", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseInt(vv, 10, 32)
		if err != nil {
			return def, err
		}
		return int32(val), nil
	case int:
		return int32(vv), nil
	case int8:
		return int32(vv), nil
	case int16:
		return int32(vv), nil
	case int32:
		return vv, nil
	case int64:
		return int32(vv), nil
	}

	return def, errFindParse.Format("int32", e.Key)
}

// Int64Default returns the entry's value as int64.
// If not found returns "def" and a non-nil error.
func (e Entry) Int64Default(def int64) (int64, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("int64", e.Key)
	}

	switch vv := v.(type) {
	case string:
		return strconv.ParseInt(vv, 10, 64)
	case int64:
		return vv, nil
	case int32:
		return int64(vv), nil
	case int8:
		return int64(vv), nil
	case int:
		return int64(vv), nil
	}

	return def, errFindParse.Format("int64", e.Key)
}

// UintDefault returns the entry's value as uint.
// If not found returns "def" and a non-nil error.
func (e Entry) UintDefault(def uint) (uint, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("uint", e.Key)
	}

	x64 := strconv.IntSize == 64
	var maxValue uint64 = math.MaxUint32
	if x64 {
		maxValue = math.MaxUint64
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseUint(vv, 10, strconv.IntSize)
		if err != nil {
			return def, err
		}
		if val > uint64(maxValue) {
			return def, errFindParse.Format("uint", e.Key)
		}
		return uint(val), nil
	case uint:
		return vv, nil
	case uint8:
		return uint(vv), nil
	case uint16:
		return uint(vv), nil
	case uint32:
		return uint(vv), nil
	case uint64:
		if vv > uint64(maxValue) {
			return def, errFindParse.Format("uint", e.Key)
		}
		return uint(vv), nil
	case int:
		if vv < 0 || vv > int(maxValue) {
			return def, errFindParse.Format("uint", e.Key)
		}
		return uint(vv), nil
	}

	return def, errFindParse.Format("uint", e.Key)
}

// Uint8Default returns the entry's value as uint8.
// If not found returns "def" and a non-nil error.
func (e Entry) Uint8Default(def uint8) (uint8, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("uint8", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseUint(vv, 10, 8)
		if err != nil {
			return def, err
		}
		if val > math.MaxUint8 {
			return def, errFindParse.Format("uint8", e.Key)
		}
		return uint8(val), nil
	case uint:
		if vv > math.MaxUint8 {
			return def, errFindParse.Format("uint8", e.Key)
		}
		return uint8(vv), nil
	case uint8:
		return vv, nil
	case uint16:
		if vv > math.MaxUint8 {
			return def, errFindParse.Format("uint8", e.Key)
		}
		return uint8(vv), nil
	case uint32:
		if vv > math.MaxUint8 {
			return def, errFindParse.Format("uint8", e.Key)
		}
		return uint8(vv), nil
	case uint64:
		if vv > math.MaxUint8 {
			return def, errFindParse.Format("uint8", e.Key)
		}
		return uint8(vv), nil
	case int:
		if vv < 0 || vv > math.MaxUint8 {
			return def, errFindParse.Format("uint8", e.Key)
		}
		return uint8(vv), nil
	}

	return def, errFindParse.Format("uint8", e.Key)
}

// Uint16Default returns the entry's value as uint16.
// If not found returns "def" and a non-nil error.
func (e Entry) Uint16Default(def uint16) (uint16, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("uint16", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseUint(vv, 10, 16)
		if err != nil {
			return def, err
		}
		if val > math.MaxUint16 {
			return def, errFindParse.Format("uint16", e.Key)
		}
		return uint16(val), nil
	case uint:
		if vv > math.MaxUint16 {
			return def, errFindParse.Format("uint16", e.Key)
		}
		return uint16(vv), nil
	case uint8:
		return uint16(vv), nil
	case uint16:
		return vv, nil
	case uint32:
		if vv > math.MaxUint16 {
			return def, errFindParse.Format("uint16", e.Key)
		}
		return uint16(vv), nil
	case uint64:
		if vv > math.MaxUint16 {
			return def, errFindParse.Format("uint16", e.Key)
		}
		return uint16(vv), nil
	case int:
		if vv < 0 || vv > math.MaxUint16 {
			return def, errFindParse.Format("uint16", e.Key)
		}
		return uint16(vv), nil
	}

	return def, errFindParse.Format("uint16", e.Key)
}

// Uint32Default returns the entry's value as uint32.
// If not found returns "def" and a non-nil error.
func (e Entry) Uint32Default(def uint32) (uint32, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("uint32", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseUint(vv, 10, 32)
		if err != nil {
			return def, err
		}
		if val > math.MaxUint32 {
			return def, errFindParse.Format("uint32", e.Key)
		}
		return uint32(val), nil
	case uint:
		if vv > math.MaxUint32 {
			return def, errFindParse.Format("uint32", e.Key)
		}
		return uint32(vv), nil
	case uint8:
		return uint32(vv), nil
	case uint16:
		return uint32(vv), nil
	case uint32:
		return vv, nil
	case uint64:
		if vv > math.MaxUint32 {
			return def, errFindParse.Format("uint32", e.Key)
		}
		return uint32(vv), nil
	case int32:
		return uint32(vv), nil
	case int64:
		if vv < 0 || vv > math.MaxUint32 {
			return def, errFindParse.Format("uint32", e.Key)
		}
		return uint32(vv), nil
	}

	return def, errFindParse.Format("uint32", e.Key)
}

// Uint64Default returns the entry's value as uint64.
// If not found returns "def" and a non-nil error.
func (e Entry) Uint64Default(def uint64) (uint64, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("uint64", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseUint(vv, 10, 64)
		if err != nil {
			return def, err
		}
		if val > math.MaxUint64 {
			return def, errFindParse.Format("uint64", e.Key)
		}
		return uint64(val), nil
	case uint8:
		return uint64(vv), nil
	case uint16:
		return uint64(vv), nil
	case uint32:
		return uint64(vv), nil
	case uint64:
		return vv, nil
	case int64:
		return uint64(vv), nil
	case int:
		return uint64(vv), nil
	}

	return def, errFindParse.Format("uint64", e.Key)
}

// Float32Default returns the entry's value as float32.
// If not found returns "def" and a non-nil error.
func (e Entry) Float32Default(key string, def float32) (float32, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("float32", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseFloat(vv, 32)
		if err != nil {
			return def, err
		}
		if val > math.MaxFloat32 {
			return def, errFindParse.Format("float32", e.Key)
		}
		return float32(val), nil
	case float32:
		return vv, nil
	case float64:
		if vv > math.MaxFloat32 {
			return def, errFindParse.Format("float32", e.Key)
		}
		return float32(vv), nil
	case int:
		return float32(vv), nil
	}

	return def, errFindParse.Format("float32", e.Key)
}

// Float64Default returns the entry's value as float64.
// If not found returns "def" and a non-nil error.
func (e Entry) Float64Default(def float64) (float64, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("float64", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseFloat(vv, 64)
		if err != nil {
			return def, err
		}
		return val, nil
	case float32:
		return float64(vv), nil
	case float64:
		return vv, nil
	case int:
		return float64(vv), nil
	case int64:
		return float64(vv), nil
	case uint:
		return float64(vv), nil
	case uint64:
		return float64(vv), nil
	}

	return def, errFindParse.Format("float64", e.Key)
}

// BoolDefault returns the user's value as bool.
// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
// or "0" or "f" or "F" or "FALSE" or "false" or "False".
// Any other value returns an error.
//
// If not found returns "def" and a non-nil error.
func (e Entry) BoolDefault(def bool) (bool, error) {
	v := e.ValueRaw
	if v == nil {
		return def, errFindParse.Format("bool", e.Key)
	}

	switch vv := v.(type) {
	case string:
		val, err := strconv.ParseBool(vv)
		if err != nil {
			return def, err
		}
		return val, nil
	case bool:
		return vv, nil
	case int:
		if vv == 1 {
			return true, nil
		}
		return false, nil
	}

	return def, errFindParse.Format("bool", e.Key)
}

// Value returns the value of the entry,
// respects the immutable.
func (e Entry) Value() interface{} {
	if e.immutable {
		// take its value, no pointer even if setted with a reference.
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

var emptyEntry Entry

// GetEntry returns a pointer to the "Entry" found with the given "key"
// if nothing found then it returns an empty Entry and false.
func (r *Store) GetEntry(key string) (Entry, bool) {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		if kv := args[i]; kv.Key == key {
			return kv, true
		}
	}

	return emptyEntry, false
}

// GetEntryAt returns the internal Entry of the memstore based on its index,
// the stored index by the router.
// If not found then it returns a zero Entry and false.
func (r *Store) GetEntryAt(index int) (Entry, bool) {
	args := *r
	if len(args) > index {
		return args[index], true
	}
	return emptyEntry, false
}

// GetDefault returns the entry's value based on its key.
// If not found returns "def".
// This function checks for immutability as well, the rest don't.
func (r *Store) GetDefault(key string, def interface{}) interface{} {
	v, ok := r.GetEntry(key)
	if !ok || v.ValueRaw == nil {
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
	v, ok := r.GetEntry(key)
	if !ok {
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

// GetInt returns the entry's value as int, based on its key.
// If not found returns -1 and a non-nil error.
func (r *Store) GetInt(key string) (int, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return 0, errFindParse.Format("int", key)
	}
	return v.IntDefault(-1)
}

// GetIntDefault returns the entry's value as int, based on its key.
// If not found returns "def".
func (r *Store) GetIntDefault(key string, def int) int {
	if v, err := r.GetInt(key); err == nil {
		return v
	}

	return def
}

// GetInt8 returns the entry's value as int8, based on its key.
// If not found returns -1 and a non-nil error.
func (r *Store) GetInt8(key string) (int8, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return -1, errFindParse.Format("int8", key)
	}
	return v.Int8Default(-1)
}

// GetInt8Default returns the entry's value as int8, based on its key.
// If not found returns "def".
func (r *Store) GetInt8Default(key string, def int8) int8 {
	if v, err := r.GetInt8(key); err == nil {
		return v
	}

	return def
}

// GetInt16 returns the entry's value as int16, based on its key.
// If not found returns -1 and a non-nil error.
func (r *Store) GetInt16(key string) (int16, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return -1, errFindParse.Format("int16", key)
	}
	return v.Int16Default(-1)
}

// GetInt16Default returns the entry's value as int16, based on its key.
// If not found returns "def".
func (r *Store) GetInt16Default(key string, def int16) int16 {
	if v, err := r.GetInt16(key); err == nil {
		return v
	}

	return def
}

// GetInt32 returns the entry's value as int32, based on its key.
// If not found returns -1 and a non-nil error.
func (r *Store) GetInt32(key string) (int32, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return -1, errFindParse.Format("int32", key)
	}
	return v.Int32Default(-1)
}

// GetInt32Default returns the entry's value as int32, based on its key.
// If not found returns "def".
func (r *Store) GetInt32Default(key string, def int32) int32 {
	if v, err := r.GetInt32(key); err == nil {
		return v
	}

	return def
}

// GetInt64 returns the entry's value as int64, based on its key.
// If not found returns -1 and a non-nil error.
func (r *Store) GetInt64(key string) (int64, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return -1, errFindParse.Format("int64", key)
	}
	return v.Int64Default(-1)
}

// GetInt64Default returns the entry's value as int64, based on its key.
// If not found returns "def".
func (r *Store) GetInt64Default(key string, def int64) int64 {
	if v, err := r.GetInt64(key); err == nil {
		return v
	}

	return def
}

// GetUint returns the entry's value as uint, based on its key.
// If not found returns 0 and a non-nil error.
func (r *Store) GetUint(key string) (uint, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return 0, errFindParse.Format("uint", key)
	}
	return v.UintDefault(0)
}

// GetUintDefault returns the entry's value as uint, based on its key.
// If not found returns "def".
func (r *Store) GetUintDefault(key string, def uint) uint {
	if v, err := r.GetUint(key); err == nil {
		return v
	}

	return def
}

// GetUint8 returns the entry's value as uint8, based on its key.
// If not found returns 0 and a non-nil error.
func (r *Store) GetUint8(key string) (uint8, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return 0, errFindParse.Format("uint8", key)
	}
	return v.Uint8Default(0)
}

// GetUint8Default returns the entry's value as uint8, based on its key.
// If not found returns "def".
func (r *Store) GetUint8Default(key string, def uint8) uint8 {
	if v, err := r.GetUint8(key); err == nil {
		return v
	}

	return def
}

// GetUint16 returns the entry's value as uint16, based on its key.
// If not found returns 0 and a non-nil error.
func (r *Store) GetUint16(key string) (uint16, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return 0, errFindParse.Format("uint16", key)
	}
	return v.Uint16Default(0)
}

// GetUint16Default returns the entry's value as uint16, based on its key.
// If not found returns "def".
func (r *Store) GetUint16Default(key string, def uint16) uint16 {
	if v, err := r.GetUint16(key); err == nil {
		return v
	}

	return def
}

// GetUint32 returns the entry's value as uint32, based on its key.
// If not found returns 0 and a non-nil error.
func (r *Store) GetUint32(key string) (uint32, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return 0, errFindParse.Format("uint32", key)
	}
	return v.Uint32Default(0)
}

// GetUint32Default returns the entry's value as uint32, based on its key.
// If not found returns "def".
func (r *Store) GetUint32Default(key string, def uint32) uint32 {
	if v, err := r.GetUint32(key); err == nil {
		return v
	}

	return def
}

// GetUint64 returns the entry's value as uint64, based on its key.
// If not found returns 0 and a non-nil error.
func (r *Store) GetUint64(key string) (uint64, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return 0, errFindParse.Format("uint64", key)
	}
	return v.Uint64Default(0)
}

// GetUint64Default returns the entry's value as uint64, based on its key.
// If not found returns "def".
func (r *Store) GetUint64Default(key string, def uint64) uint64 {
	if v, err := r.GetUint64(key); err == nil {
		return v
	}

	return def
}

// GetFloat64 returns the entry's value as float64, based on its key.
// If not found returns -1 and a non nil error.
func (r *Store) GetFloat64(key string) (float64, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return -1, errFindParse.Format("float64", key)
	}
	return v.Float64Default(-1)
}

// GetFloat64Default returns the entry's value as float64, based on its key.
// If not found returns "def".
func (r *Store) GetFloat64Default(key string, def float64) float64 {
	if v, err := r.GetFloat64(key); err == nil {
		return v
	}

	return def
}

// GetBool returns the user's value as bool, based on its key.
// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
// or "0" or "f" or "F" or "FALSE" or "false" or "False".
// Any other value returns an error.
//
// If not found returns false and a non-nil error.
func (r *Store) GetBool(key string) (bool, error) {
	v, ok := r.GetEntry(key)
	if !ok {
		return false, errFindParse.Format("bool", key)
	}

	return v.BoolDefault(false)
}

// GetBoolDefault returns the user's value as bool, based on its key.
// a string which is "1" or "t" or "T" or "TRUE" or "true" or "True"
// or "0" or "f" or "F" or "FALSE" or "false" or "False".
//
// If not found returns "def".
func (r *Store) GetBoolDefault(key string, def bool) bool {
	if v, err := r.GetBool(key); err == nil {
		return v
	}

	return def
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
