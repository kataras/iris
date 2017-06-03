package context

import (
	"strconv"

	"github.com/kataras/iris/core/errors"
)

type (
	// RequestValue is the entry of the context storage RequestValues - .Values()
	RequestValue struct {
		key   string
		value interface{}
	}

	// RequestValues is just a key-value storage which context's request values should implement.
	RequestValues []RequestValue
)

// RequestValuesReadOnly the request values with read-only access.
type RequestValuesReadOnly struct {
	RequestValues
}

// Set does nothing.
func (r RequestValuesReadOnly) Set(string, interface{}) {}

// Set sets a value to the key-value context storage, can be familiar as "User Values".
func (r *RequestValues) Set(key string, value interface{}) {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.key == key {
			kv.value = value
			return
		}
	}

	c := cap(args)
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.key = key
		kv.value = value
		*r = args
		return
	}

	kv := RequestValue{}
	kv.key = key
	kv.value = value
	*r = append(args, kv)
}

// Get returns the user's value based on its key.
func (r *RequestValues) Get(key string) interface{} {
	args := *r
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.key == key {
			return kv.value
		}
	}
	return nil
}

// Visit accepts a visitor which will be filled
// by the key-value objects, the caller should not try to change the underline values.
func (r *RequestValues) Visit(visitor func(key string, value interface{})) {
	args := *r
	for i, n := 0, len(args); i < n; i++ {
		visitor(args[i].key, args[i].value)
	}
}

// GetString returns the user's value as string, based on its key.
func (r *RequestValues) GetString(key string) string {
	if v, ok := r.Get(key).(string); ok {
		return v
	}

	return ""
}

var errIntParse = errors.New("unable to find or parse the integer, found: %#v")

// GetInt returns the user's value as int, based on its key.
func (r *RequestValues) GetInt(key string) (int, error) {
	v := r.Get(key)
	if vint, ok := v.(int); ok {
		return vint, nil
	} else if vstring, sok := v.(string); sok {
		return strconv.Atoi(vstring)
	}

	return -1, errIntParse.Format(v)
}

// GetInt64 returns the user's value as int64, based on its key.
func (r *RequestValues) GetInt64(key string) (int64, error) {
	return strconv.ParseInt(r.GetString(key), 10, 64)
}

// Reset clears all the request values.
func (r *RequestValues) Reset() {
	*r = (*r)[0:0]
}

// ReadOnly returns a new request values with read-only access.
func (r *RequestValues) ReadOnly() RequestValuesReadOnly {
	args := *r
	values := make(RequestValues, len(args))
	copy(values, args)
	return RequestValuesReadOnly{values}
}

// Len returns the full length of the values.
func (r *RequestValues) Len() int {
	args := *r
	return len(args)
}

// RequestParam is the entry of RequestParams, request's url named parameters are storaged here.
type RequestParam struct {
	Key   string
	Value string
}

// RequestParams is a key string - value string storage which context's request params should implement.
// RequestValues is for communication between middleware, RequestParams cannot be changed, are setted at the routing
// time, stores the dynamic named parameters, can be empty if the route is static.
type RequestParams []RequestParam

// Get returns the param's value based on its key.
func (r RequestParams) Get(key string) string {
	for _, p := range r {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// Visit accepts a visitor which will be filled
// by the key and value.
// The caller should not try to change the underline values.
func (r RequestParams) Visit(visitor func(key string, value string)) {
	for i, n := 0, len(r); i < n; i++ {
		visitor(r[i].Key, r[i].Value)
	}
}

// GetInt returns the user's value as int, based on its key.
func (r RequestParams) GetInt(key string) (int, error) {
	v := r.Get(key)
	return strconv.Atoi(v)
}

// GetInt64 returns the user's value as int64, based on its key.
func (r RequestParams) GetInt64(key string) (int64, error) {
	return strconv.ParseInt(r.Get(key), 10, 64)
}

// GetDecoded returns the url-query-decoded user's value based on its key.
func (r RequestParams) GetDecoded(key string) string {
	return DecodeQuery(DecodeQuery(r.Get(key)))
}

// GetIntUnslashed same as Get but it removes the first slash if found.
// Usage: Get an id from a wildcard path.
//
// Returns -1 with an error if the parameter couldn't be found.
func (r RequestParams) GetIntUnslashed(key string) (int, error) {
	v := r.Get(key)
	if v != "" {
		if len(v) > 1 {
			if v[0] == '/' {
				v = v[1:]
			}
		}
		return strconv.Atoi(v)

	}

	return -1, errIntParse.Format(v)
}

// Len returns the full length of the params.
func (r RequestParams) Len() int {
	return len(r)
}
