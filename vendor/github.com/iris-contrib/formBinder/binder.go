package formbinder

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

const tagName = "form"

var timeFormat = []string{
	"2006-01-02",
	time.ANSIC,
	time.UnixDate,
	time.RubyDate,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC3339,
	time.RFC3339Nano,
	time.Kitchen,
	time.Stamp,
	time.StampMilli,
	time.StampMicro,
	time.StampNano,
}

// A pathMap holds the values of a map with its key and values correspondent
type pathMap struct {
	ma    reflect.Value
	key   string
	value reflect.Value

	path string
}

// a pathMaps holds the values for each key
type pathMaps []*pathMap

// find find and get the value by the given key
func (ma pathMaps) find(id reflect.Value, key string) *pathMap {
	for _, v := range ma {
		if v.ma == id && v.key == key {
			return v
		}
	}
	return nil
}

// DecodeCustomTypeFunc is a function that indicate how should to decode a custom type
type DecodeCustomTypeFunc func([]string) (interface{}, error)

// DecodeCustomTypeField is a function registered for a specific field of the struct passed to the Decoder
type DecodeCustomTypeField struct {
	field reflect.Value
	fun   DecodeCustomTypeFunc
}

// DecodeCustomType fields for custom types
type DecodeCustomType struct {
	fun    DecodeCustomTypeFunc
	fields []DecodeCustomTypeField
}

// Decoder the main to decode the values
type Decoder struct {
	main       reflect.Value
	formValues url.Values
	opts       *DecoderOptions

	curr   reflect.Value
	value  string
	values []string

	path    string
	field   string
	bracket string
	isKey   bool

	maps pathMaps

	customTypes map[reflect.Type]*DecodeCustomType
}

// DecoderOptions options for decoding the values
type DecoderOptions struct {
	TagName string
}

// RegisterCustomType It is the method responsible for register functions for decoding custom types
func (dec *Decoder) RegisterCustomType(fn DecodeCustomTypeFunc, types []interface{}, fields []interface{}) *Decoder {
	if dec.customTypes == nil {
		dec.customTypes = make(map[reflect.Type]*DecodeCustomType)
	}
	for i := range types {
		typ := reflect.TypeOf(types[i])
		if dec.customTypes[typ] == nil {
			dec.customTypes[typ] = new(DecodeCustomType)
		}
		if len(fields) > 0 {
			for j := range fields {
				va := reflect.ValueOf(fields[j])
				f := DecodeCustomTypeField{field: va, fun: fn}
				dec.customTypes[typ].fields = append(dec.customTypes[typ].fields, f)
			}
		} else {
			dec.customTypes[typ].fun = fn
		}
	}
	return dec
}

// NewDecoder creates a new instance of Decoder
func NewDecoder(opts *DecoderOptions) *Decoder {
	dec := &Decoder{opts: opts}
	if dec.opts == nil {
		dec.opts = &DecoderOptions{}
	}
	if dec.opts.TagName == "" {
		dec.opts.TagName = tagName
	}
	return dec
}

// Decode decodes the url.Values into a element that must be a pointer to a type provided by argument
func (dec *Decoder) Decode(vs url.Values, dst interface{}) error {
	main := reflect.ValueOf(dst)
	if main.Kind() != reflect.Ptr {
		return newError(fmt.Errorf("form: the value passed for decode is not a pointer but a %v", main.Kind()))
	}
	dec.main = main.Elem()
	dec.formValues = vs
	return dec.prepare()
}

// Decode decodes the url.Values into a element that must be a pointer to a type provided by argument
func Decode(vs url.Values, dst interface{}) error {
	main := reflect.ValueOf(dst)
	if main.Kind() != reflect.Ptr {
		return newError(fmt.Errorf("form: the value passed for decode is not a pointer but a %v", main.Kind()))
	}
	dec := &Decoder{
		main:       main.Elem(),
		formValues: vs,
		opts: &DecoderOptions{
			TagName: tagName,
		},
	}
	return dec.prepare()
}

func (dec *Decoder) prepare() error {
	// iterate over the form's values and decode it
	for k, v := range dec.formValues {
		dec.path = k
		dec.field = k
		dec.values = v
		dec.value = v[0]
		dec.curr = dec.main
		if dec.value != "" {
			if err := dec.begin(); err != nil {
				return err
			}
		}
	}
	// set values of maps
	for _, v := range dec.maps {
		key := v.ma.Type().Key()
		// check if the key implements the UnmarshalText interface
		var val reflect.Value
		if key.Kind() == reflect.Ptr {
			val = reflect.New(key.Elem())
		} else {
			val = reflect.New(key).Elem()
		}
		// decode key
		dec.path = v.path
		dec.field = v.path
		dec.values = []string{v.key}
		dec.curr = val
		dec.value = v.key
		dec.isKey = true
		if err := dec.decode(); err != nil {
			return err
		}
		// set key with its value
		v.ma.SetMapIndex(dec.curr, v.value)
	}
	dec.maps = make(pathMaps, 0)
	return nil
}

// begin analyzes the current path to walk through it
func (dec *Decoder) begin() (err error) {
	inBracket := false
	valBracket := ""
	bracketClosed := false
	lastPos := 0
	tmp := dec.field

	// parse path
	for i, char := range tmp {
		if char == '[' && inBracket == false {
			// found an opening bracket
			bracketClosed = false
			inBracket = true
			dec.field = tmp[lastPos:i]
			lastPos = i + 1
			/*
				if err = dec.walk(); err != nil {
					return
				}
			*/
			continue
		} else if inBracket {
			// it is inside of bracket, so get its value
			if char == ']' {
				/*
					nextChar := tmp[i+1:]
					if nextChar != "" {
						t := nextChar[:1]
						if t != "[" && t != "." {
							valBracket += string(char)
							continue
						}
					}
				*/
				// found an closing bracket, so it will be recently close, so put as true the bracketClosed
				// and put as false inBracket and pass the value of bracket to dec.key
				inBracket = false
				bracketClosed = true
				dec.bracket = valBracket
				lastPos = i + 1
				if err = dec.walk(); err != nil {
					return
				}
				valBracket = ""
			} else {
				// still inside the bracket, so follow getting its value
				valBracket += string(char)
			}
			continue
		} else if !inBracket {
			// not found any bracket, so try found a field
			if char == '.' {
				// found a field, we need to know if the field is next of a closing bracket,
				// for example: [0].Field
				if bracketClosed {
					bracketClosed = false
					lastPos = i + 1
					continue
				}
				// found a field, but is not next of a closing bracket, for example: Field1.Field2
				dec.field = tmp[lastPos:i]
				//dec.field = tmp[:i]
				lastPos = i + 1
				if err = dec.walk(); err != nil {
					return
				}
			}
			continue
		}
	}
	// last field of path
	dec.field = tmp[lastPos:]

	return dec.end()
}

// walk traverses the current path until to the last field
func (dec *Decoder) walk() error {
	// check if there is field, if is so, then it should be struct or map (access by .)
	if dec.field != "" {
		// check if is a struct or map
		switch dec.curr.Kind() {
		case reflect.Struct:
			if err := dec.findStructField(); err != nil {
				return err
			}
		case reflect.Map:
			dec.walkInMap(dec.field)
		}
	}
	// check if is a interface and it is not nil. This mean that the interface
	// has a struct, map or slice as value
	if dec.curr.Kind() == reflect.Interface && !dec.curr.IsNil() {
		dec.curr = dec.curr.Elem()
	}
	// check if it is a pointer
	if dec.curr.Kind() == reflect.Ptr {
		if dec.curr.IsNil() {
			dec.curr.Set(reflect.New(dec.curr.Type().Elem()))
		}
		dec.curr = dec.curr.Elem()
	}
	// check if there is access to slice/array or map (access by [])
	if dec.bracket != "" {
		switch dec.curr.Kind() {
		case reflect.Array:
			index, err := strconv.Atoi(dec.bracket)
			if err != nil {
				return newError(fmt.Errorf("form: the index of array is not a number in the field \"%v\" of path \"%v\"", dec.field, dec.path))
			}
			dec.curr = dec.curr.Index(index)
		case reflect.Slice:
			index, err := strconv.Atoi(dec.bracket)
			if err != nil {
				return newError(fmt.Errorf("form: the index of slice is not a number in the field \"%v\" of path \"%v\"", dec.field, dec.path))
			}
			if dec.curr.Len() <= index {
				dec.expandSlice(index + 1)
			}
			dec.curr = dec.curr.Index(index)
		case reflect.Map:
			dec.walkInMap(dec.bracket)
		default:
			return newError(fmt.Errorf("form: the field \"%v\" in path \"%v\" has a index for array but it is a %v", dec.field, dec.path, dec.curr.Kind()))
		}
	}
	dec.field = ""
	dec.bracket = ""
	return nil
}

// walkMap puts in d.curr the map concrete for decode the current value
func (dec *Decoder) walkInMap(key string) {
	n := dec.curr.Type()
	takeAndAppend := func() {
		m := reflect.New(n.Elem()).Elem()
		dec.maps = append(dec.maps, &pathMap{dec.curr, key, m, dec.path})
		dec.curr = m
	}
	if dec.curr.IsNil() {
		dec.curr.Set(reflect.MakeMap(n))
		takeAndAppend()
	} else if a := dec.maps.find(dec.curr, key); a == nil {
		takeAndAppend()
	} else {
		dec.curr = a.value
	}
}

// end finds the last field for decode its value correspondent
func (dec *Decoder) end() error {
	switch dec.curr.Kind() {
	case reflect.Struct:
		if err := dec.findStructField(); err != nil {
			return err
		}
	case reflect.Map:
		// leave backward compatibility for access to maps by .
		dec.walkInMap(dec.field)
	}
	if dec.value == "" {
		return nil
	}
	return dec.decode()
}

// decode sets the value in the field
func (dec *Decoder) decode() error {
	// has registered a custom type? If so, then decode by it
	if ok, err := dec.checkCustomType(); ok || err != nil {
		return err
	}
	// implements UnmarshalText interface? If so, then decode by it
	if ok, err := checkUnmarshalText(dec.curr, dec.value); ok || err != nil {
		return err
	}

	switch dec.curr.Kind() {
	case reflect.Array:
		if dec.bracket == "" {
			// not has index, so to decode all values in the array
			tmp := dec.curr
			for i, v := range dec.values {
				dec.curr = tmp.Index(i)
				dec.value = v
				if err := dec.decode(); err != nil {
					return err
				}
			}
		} else {
			// has index, so to decode value by index indicated
			index, err := strconv.Atoi(dec.bracket)
			if err != nil {
				return newError(fmt.Errorf("form: the index of array is not a number in the field \"%v\" of path \"%v\"", dec.field, dec.path))
			}
			dec.curr = dec.curr.Index(index)
			return dec.decode()
		}
	case reflect.Slice:
		if dec.bracket == "" {
			// not has index, so to decode all values in the slice/array
			dec.expandSlice(len(dec.values))
			tmp := dec.curr
			for i, v := range dec.values {
				dec.curr = tmp.Index(i)
				dec.value = v
				if err := dec.decode(); err != nil {
					return err
				}
			}
		} else {
			// has index, so to decode value by index indicated
			index, err := strconv.Atoi(dec.bracket)
			if err != nil {
				return newError(fmt.Errorf("form: the index of slice is not a number in the field \"%v\" of path \"%v\"", dec.field, dec.path))
			}
			if dec.curr.Len() <= index {
				dec.expandSlice(index + 1)
			}
			dec.curr = dec.curr.Index(index)
			return dec.decode()
		}
	case reflect.String:
		dec.curr.SetString(dec.value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if num, err := strconv.ParseInt(dec.value, 10, 64); err != nil {
			return newError(fmt.Errorf("form: the value of field \"%v\" in path \"%v\" should be a valid signed integer number", dec.field, dec.path))
		} else {
			dec.curr.SetInt(num)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if num, err := strconv.ParseUint(dec.value, 10, 64); err != nil {
			return newError(fmt.Errorf("form: the value of field \"%v\" in path \"%v\" should be a valid unsigned integer number", dec.field, dec.path))
		} else {
			dec.curr.SetUint(num)
		}
	case reflect.Float32, reflect.Float64:
		if num, err := strconv.ParseFloat(dec.value, dec.curr.Type().Bits()); err != nil {
			return newError(fmt.Errorf("form: the value of field \"%v\" in path \"%v\" should be a valid float number", dec.field, dec.path))
		} else {
			dec.curr.SetFloat(num)
		}
	case reflect.Bool:
		switch dec.value {
		case "true", "on", "1":
			dec.curr.SetBool(true)
		case "false", "off", "0":
			dec.curr.SetBool(false)
		default:
			return newError(fmt.Errorf("form: the value of field \"%v\" in path \"%v\" is not a valid boolean", dec.field, dec.path))
		}
	case reflect.Interface:
		dec.curr.Set(reflect.ValueOf(dec.value))
	case reflect.Ptr:
		dec.curr.Set(reflect.New(dec.curr.Type().Elem()))
		dec.curr = dec.curr.Elem()
		return dec.decode()
	case reflect.Struct:
		switch dec.curr.Interface().(type) {
		case time.Time:
			var t time.Time
			var err error
			for _, layout := range timeFormat {
				t, err = time.Parse(layout, dec.value)
				if err == nil {
					break
				}
			}
			if err != nil {
				return newError(fmt.Errorf("form: the value of field \"%v\" in path \"%v\" is not a valid datetime", dec.field, dec.path))
			}
			dec.curr.Set(reflect.ValueOf(t))
		case url.URL:
			u, err := url.Parse(dec.value)
			if err != nil {
				return newError(fmt.Errorf("form: the value of field \"%v\" in path \"%v\" is not a valid url", dec.field, dec.path))
			}
			dec.curr.Set(reflect.ValueOf(*u))
		default:
			/*
				if dec.isKey {
					tmp := dec.curr
					dec.field = dec.value
					if err := dec.begin(); err != nil {
						return err
					}
					dec.curr = tmp
					return nil
				}
			*/
			return newError(fmt.Errorf("form: not supported type for field \"%v\" in path \"%v\"", dec.field, dec.path))
		}
	default:
		return newError(fmt.Errorf("form: not supported type for field \"%v\" in path \"%v\"", dec.field, dec.path))
	}

	return nil
}

// IsErrPath reports whether the incoming error is type of `ErrPath`, which can be ignored
// when server allows unknown post values to be sent by the client.
func IsErrPath(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(ErrPath)
	return ok
}

// ErrPath describes an error that can be ignored if server allows unknown post values to be sent on server.
type ErrPath struct {
	field string
}

func (err ErrPath) Error() string {
	return fmt.Sprintf("form: not found the field \"%s\"", err.field)
}

// Field returns the unknown posted request field.
func (err ErrPath) Field() string {
	return err.field
}

// findField finds a field by its name, if it is not found,
// then retry the search examining the tag "form" of every field of struct
func (dec *Decoder) findStructField() error {
	var anon reflect.Value

	num := dec.curr.NumField()
	for i := 0; i < num; i++ {
		field := dec.curr.Type().Field(i)
		if field.Name == dec.field {
			// check if the field's name is equal
			dec.curr = dec.curr.Field(i)
			return nil
		} else if field.Anonymous {
			// if the field is a anonymous struct, then iterate over its fields
			tmp := dec.curr
			dec.curr = dec.curr.FieldByIndex(field.Index)
			if err := dec.findStructField(); err != nil {
				dec.curr = tmp
				continue
			}
			// field in anonymous struct is found,
			// but first it should found the field in the rest of struct
			// (a field with same name in the current struct should have preference over anonymous struct)
			anon = dec.curr
			dec.curr = tmp
		} else if dec.field == field.Tag.Get(dec.opts.TagName) {
			// is not found yet, then retry by its tag name "form"
			dec.curr = dec.curr.Field(i)
			return nil
		}
	}
	if anon.IsValid() {
		dec.curr = anon
		return nil
	}

	return ErrPath{field: dec.field}
}

// expandSlice expands the length and capacity of the current slice
func (dec *Decoder) expandSlice(length int) {
	n := reflect.MakeSlice(dec.curr.Type(), length, length)
	reflect.Copy(n, dec.curr)
	dec.curr.Set(n)
}

// checkCustomType checks if the value to decode has a custom type registered
func (dec *Decoder) checkCustomType() (bool, error) {
	if dec.customTypes == nil {
		return false, nil
	}
	if v, ok := dec.customTypes[dec.curr.Type()]; ok {
		if len(v.fields) > 0 {
			for i := range v.fields {
				if v.fields[i].field.Elem() == dec.curr {
					va, err := v.fields[i].fun(dec.values)
					if err != nil {
						return true, err
					}
					dec.curr.Set(reflect.ValueOf(va))
					return true, nil
				}
			}
			if v.fun != nil {
				va, err := v.fun(dec.values)
				if err != nil {
					return true, err
				}
				dec.curr.Set(reflect.ValueOf(va))
				return true, nil
			}
		} else {
			va, err := v.fun(dec.values)
			if err != nil {
				return true, err
			}
			dec.curr.Set(reflect.ValueOf(va))
			return true, nil
		}
	}
	return false, nil
}
