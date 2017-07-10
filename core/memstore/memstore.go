// Package memstore contains a store which is just
// a collection of key-value entries with immutability capabilities.
//
// Developers can use that storage to their own apps if they like its behavior.
// It's fast and in the same time you get read-only access (safety) when you need it.
package memstore

import (
	"reflect"
	"strconv"

	"github.com/kataras/iris/core/errors"
)

type (
	// Entry is the entry of the context storage Store - .Values()
	Entry struct {
		Key       string
		value     interface{}
		immutable bool // if true then it can't change by its caller.
	}

	// Store is a collection of key-value entries with immutability capabilities.
	Store []Entry
)

// Value returns the value of the entry,
// respects the immutable.
func (e Entry) Value() interface{} {
	if e.immutable {
		// take its value, no pointer even if setted with a rreference.
		vv := reflect.Indirect(reflect.ValueOf(e.value))

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
	return e.value
}

// the id is immutable(true or false)+key
// so the users will be able to use the same key
// to store two different entries (one immutable and other mutable).
// or no? better no, that will confuse and maybe result on unexpected results.
// I will just replace the value and the immutable bool value when Set if
// a key is already exists.
// func (e Entry) identifier() string {}

func (r *Store) save(key string, value interface{}, immutable bool) {
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
				kv.value = value
				kv.immutable = immutable
			} else if kv.immutable == false {
				// if it was not immutable then user can alt it via `Set` and `SetImmutable`
				kv.value = value
				kv.immutable = immutable
			}
			// else it was immutable and called by `Set` then disallow the update
			return
		}
	}

	// expand slice to add it
	c := cap(args)
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.Key = key
		kv.value = value
		kv.immutable = immutable
		*r = args
		return
	}

	// add
	kv := Entry{
		Key:       key,
		value:     value,
		immutable: immutable,
	}
	*r = append(args, kv)
}

// Set saves a value to the key-value storage.
// See `SetImmutable` and `Get`.
func (r *Store) Set(key string, value interface{}) {
	r.save(key, value, false)
}

// SetImmutable saves a value to the key-value storage.
// Unlike `Set`, the output value cannot be changed by the caller later on (when .Get OR .Set)
//
// An Immutable entry should be only changed with a `SetImmutable`, simple `Set` will not work
// if the entry was immutable, for your own safety.
//
// Use it consistently, it's far slower than `Set`.
// Read more about muttable and immutable go types: https://stackoverflow.com/a/8021081
func (r *Store) SetImmutable(key string, value interface{}) {
	r.save(key, value, true)
}

// Get returns the entry's value based on its key.
func (r *Store) Get(key string) interface{} {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.Key == key {
			return kv.Value()
		}
	}

	return nil
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

// GetString returns the entry's value as string, based on its key.
func (r *Store) GetString(key string) string {
	if v, ok := r.Get(key).(string); ok {
		return v
	}

	return ""
}

// ErrIntParse returns an error message when int parse failed
// it's not statical error, it depends on the failed value.
var ErrIntParse = errors.New("unable to find or parse the integer, found: %#v")

// GetInt returns the entry's value as int, based on its key.
func (r *Store) GetInt(key string) (int, error) {
	v := r.Get(key)
	if vint, ok := v.(int); ok {
		return vint, nil
	} else if vstring, sok := v.(string); sok {
		return strconv.Atoi(vstring)
	}

	return -1, ErrIntParse.Format(v)
}

// GetInt64 returns the entry's value as int64, based on its key.
func (r *Store) GetInt64(key string) (int64, error) {
	return strconv.ParseInt(r.GetString(key), 10, 64)
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
