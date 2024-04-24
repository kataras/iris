package context

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12/core/memstore"
)

// RequestParams is a key string - value string storage which
// context's request dynamic path params are being kept.
// Empty if the route is static.
type RequestParams struct {
	memstore.Store
}

// Set inserts a parameter value.
// See `Get` too.
func (r *RequestParams) Set(key, value string) {
	if ln := len(r.Store); cap(r.Store) > ln {
		r.Store = r.Store[:ln+1]
		p := &r.Store[ln]
		p.Key = key
		p.ValueRaw = value
		return
	}

	r.Store = append(r.Store, memstore.Entry{
		Key:      key,
		ValueRaw: value,
	})
}

// Get returns a path parameter's value based on its route's dynamic path key.
func (r *RequestParams) Get(key string) string {
	for i := range r.Store {
		if kv := r.Store[i]; kv.Key == key {
			if v, ok := kv.ValueRaw.(string); ok {
				return v // it should always be string here on :string parameter.
			}

			if v, ok := kv.ValueRaw.(fmt.Stringer); ok {
				return v.String()
			}

			return fmt.Sprintf("%v", kv.ValueRaw)
		}
	}

	return ""
}

// GetEntryAt will return the parameter's internal store's `Entry` based on the index.
// If not found it will return an emptry `Entry`.
func (r *RequestParams) GetEntryAt(index int) memstore.Entry {
	entry, _ := r.Store.GetEntryAt(index)
	return entry
}

// GetEntry will return the parameter's internal store's `Entry` based on its name/key.
// If not found it will return an emptry `Entry`.
func (r *RequestParams) GetEntry(key string) memstore.Entry {
	entry, _ := r.Store.GetEntry(key)
	return entry
}

// Visit accepts a visitor which will be filled
// by the key-value params.
func (r *RequestParams) Visit(visitor func(key string, value string)) {
	r.Store.Visit(func(k string, v interface{}) {
		visitor(k, fmt.Sprintf("%v", v)) // always string here.
	})
}

// GetTrim returns a path parameter's value without trailing spaces based on its route's dynamic path key.
func (r *RequestParams) GetTrim(key string) string {
	return strings.TrimSpace(r.Get(key))
}

// GetEscape returns a path parameter's double-url-query-escaped value based on its route's dynamic path key.
func (r *RequestParams) GetEscape(key string) string {
	return DecodeQuery(DecodeQuery(r.Get(key)))
}

// GetDecoded returns a path parameter's double-url-query-escaped value based on its route's dynamic path key.
// same as `GetEscape`.
func (r *RequestParams) GetDecoded(key string) string {
	return r.GetEscape(key)
}

// TrimParamFilePart is a middleware which replaces all route dynamic path parameters
// with values that do not contain any part after the last dot (.) character.
//
// Example Code:
//
//	package main
//
//	import (
//		"github.com/kataras/iris/v12"
//	)
//
//	func main() {
//		app := iris.New()
//		app.Get("/{uid:string regexp(^[0-9]{1,20}.html$)}", iris.TrimParamFilePart, handler)
//		// TrimParamFilePart can be registered as a middleware to a Party (group of routes) as well.
//		app.Listen(":8080")
//	}
//
//	func handler(ctx iris.Context) {
//		//
//		// The above line is useless now that we've registered the TrimParamFilePart middleware:
//		// uid := ctx.Params().GetTrimFileUint64("uid")
//		//
//
//		uid := ctx.Params().GetUint64Default("uid", 0)
//		ctx.Writef("Param value: %d\n", uid)
//	}
func TrimParamFilePart(ctx *Context) { // See #2024.
	params := ctx.Params()

	for i, param := range params.Store {
		if value, ok := param.ValueRaw.(string); ok {
			if idx := strings.LastIndexByte(value, '.'); idx > 1 /* at least .h */ {
				value = value[0:idx]
				param.ValueRaw = value
			}
		}

		params.Store[i] = param
	}

	ctx.Next()
}

// GetTrimFile returns a parameter value but without the last ".ANYTHING_HERE" part.
func (r *RequestParams) GetTrimFile(key string) string {
	value := r.Get(key)

	if idx := strings.LastIndexByte(value, '.'); idx > 1 /* at least .h */ {
		return value[0:idx]
	}

	return value
}

// GetTrimFileInt same as GetTrimFile but it returns the value as int.
func (r *RequestParams) GetTrimFileInt(key string) int {
	value := r.Get(key)

	if idx := strings.LastIndexByte(value, '.'); idx > 1 /* at least .h */ {
		value = value[0:idx]
	}

	v, _ := strconv.Atoi(value)
	return v
}

// GetTrimFileUint64 same as GetTrimFile but it returns the value as uint64.
func (r *RequestParams) GetTrimFileUint64(key string) uint64 {
	value := r.Get(key)

	if idx := strings.LastIndexByte(value, '.'); idx > 1 /* at least .h */ {
		value = value[0:idx]
	}

	v, err := strconv.ParseUint(value, 10, strconv.IntSize)
	if err != nil {
		return 0
	}

	return v
}

// GetTrimFileUint64 same as GetTrimFile but it returns the value as uint.
func (r *RequestParams) GetTrimFileUint(key string) uint {
	return uint(r.GetTrimFileUint64(key))
}

func (r *RequestParams) getRightTrimmed(key string, cutset string) string {
	return strings.TrimRight(strings.ToLower(r.Get(key)), cutset)
}

// GetTrimHTML returns a parameter value but without the last ".html" part.
func (r *RequestParams) GetTrimHTML(key string) string {
	return r.getRightTrimmed(key, ".html")
}

// GetTrimJSON returns a parameter value but without the last ".json" part.
func (r *RequestParams) GetTrimJSON(key string) string {
	return r.getRightTrimmed(key, ".json")
}

// GetTrimXML returns a parameter value but without the last ".xml" part.
func (r *RequestParams) GetTrimXML(key string) string {
	return r.getRightTrimmed(key, ".xml")
}

// GetIntUnslashed same as Get but it removes the first slash if found.
// Usage: Get an id from a wildcard path.
//
// Returns -1 and false if not path parameter with that "key" found.
func (r *RequestParams) GetIntUnslashed(key string) (int, bool) {
	v := r.Get(key)
	if v != "" {
		if len(v) > 1 {
			if v[0] == '/' {
				v = v[1:]
			}
		}

		vInt, err := strconv.Atoi(v)
		if err != nil {
			return -1, false
		}
		return vInt, true
	}

	return -1, false
}

// ParamResolvers is the global param resolution for a parameter type for a specific go std or custom type.
//
// Key is the specific type, which should be unique.
// The value is a function which accepts the parameter index
// and it should return the value as the parameter type evaluator expects it.
//
//	i.e [reflect.TypeOf("string")] = func(paramIndex int) interface{} {
//	    return func(ctx *Context) <T> {
//	        return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(<T>)
//	    }
//	}
//
// Read https://github.com/kataras/iris/tree/main/_examples/routing/macros for more details.
// Checks for total available request parameters length
// and parameter index based on the hero/mvc function added
// in order to support the MVC.HandleMany("GET", "/path/{ps}/{pssecond} /path/{ps}")
// when on the second requested path, the 'pssecond' should be empty.
var ParamResolvers = map[reflect.Type]func(paramIndex int) interface{}{
	reflect.TypeOf(""): func(paramIndex int) interface{} {
		return func(ctx *Context) string {
			if ctx.Params().Len() <= paramIndex {
				return ""
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(string)
		}
	},
	reflect.TypeOf(int(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) int {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			// v, _ := ctx.Params().GetEntryAt(paramIndex).IntDefault(0)
			// return v
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(int)
		}
	},
	reflect.TypeOf(int8(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) int8 {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(int8)
		}
	},
	reflect.TypeOf(int16(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) int16 {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(int16)
		}
	},
	reflect.TypeOf(int32(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) int32 {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(int32)
		}
	},
	reflect.TypeOf(int64(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) int64 {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(int64)
		}
	},
	reflect.TypeOf(uint(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) uint {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(uint)
		}
	},
	reflect.TypeOf(uint8(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) uint8 {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(uint8)
		}
	},
	reflect.TypeOf(uint16(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) uint16 {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(uint16)
		}
	},
	reflect.TypeOf(uint32(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) uint32 {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(uint32)
		}
	},
	reflect.TypeOf(uint64(1)): func(paramIndex int) interface{} {
		return func(ctx *Context) uint64 {
			if ctx.Params().Len() <= paramIndex {
				return 0
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(uint64)
		}
	},
	reflect.TypeOf(true): func(paramIndex int) interface{} {
		return func(ctx *Context) bool {
			if ctx.Params().Len() <= paramIndex {
				return false
			}
			return ctx.Params().GetEntryAt(paramIndex).ValueRaw.(bool)
		}
	},
	reflect.TypeOf(time.Time{}): func(paramIndex int) interface{} {
		return func(ctx *Context) time.Time {
			if ctx.Params().Len() <= paramIndex {
				return unixEpochTime
			}

			v, ok := ctx.Params().GetEntryAt(paramIndex).ValueRaw.(time.Time)
			if !ok {
				return unixEpochTime
			}

			return v
		}
	},
	reflect.TypeOf(time.Weekday(0)): func(paramIndex int) interface{} {
		return func(ctx *Context) time.Weekday {
			if ctx.Params().Len() <= paramIndex {
				return time.Sunday
			}

			v, ok := ctx.Params().GetEntryAt(paramIndex).ValueRaw.(time.Weekday)
			if !ok {
				return time.Sunday
			}

			return v
		}
	},
}

// ParamResolverByTypeAndIndex will return a function that can be used to bind path parameter's exact value by its Go std type
// and the parameter's index based on the registered path.
// Usage: nameResolver := ParamResolverByKindAndKey(reflect.TypeOf(""), 0)
// Inside a Handler:      nameResolver.Call(ctx)[0]
//
//	it will return the reflect.Value Of the exact type of the parameter(based on the path parameters and macros).
//
// It is only useful for dynamic binding of the parameter, it is used on "hero" package and it should be modified
// only when Macros are modified in such way that the default selections for the available go std types are not enough.
//
// Returns empty value and false if "k" does not match any valid parameter resolver.
func ParamResolverByTypeAndIndex(typ reflect.Type, paramIndex int) (reflect.Value, bool) {
	/* NO:
	// This could work but its result is not exact type, so direct binding is not possible.
	resolver := m.ParamResolver
	fn := func(ctx *context.Context) interface{} {
		entry, _ := ctx.Params().GetEntry(paramName)
		return resolver(entry)
	}
	//

	// This works but it is slower on serve-time.
	paramNameValue := []reflect.Value{reflect.ValueOf(paramName)}
	var fnSignature func(*context.Context) string
	return reflect.MakeFunc(reflect.ValueOf(&fnSignature).Elem().Type(), func(in []reflect.Value) []reflect.Value {
		return in[0].MethodByName("Params").Call(emptyIn)[0].MethodByName("Get").Call(paramNameValue)
		// return []reflect.Value{reflect.ValueOf(in[0].Interface().(*context.Context).Params().Get(paramName))}
	})
	//
	*/

	r, ok := ParamResolvers[typ]
	if !ok || r == nil {
		return reflect.Value{}, false
	}

	return reflect.ValueOf(r(paramIndex)), true
}
