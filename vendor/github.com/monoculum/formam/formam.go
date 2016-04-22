// Package formam implements functions to decode values of a html form.
package formam

import (
	"encoding"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const TAG_NAME = "formam"

// A pathMap holds the values of a map with its key and values correspondent
type pathMap struct {
	m reflect.Value

	key   string
	value reflect.Value

	path string
}

// a pathMaps holds the values for each key
type pathMaps []*pathMap

// find find and get the value by the given key
func (ma pathMaps) find(id reflect.Value, key string) *pathMap {
	for _, v := range ma {
		if v.m == id && v.key == key {
			return v
		}
	}
	return nil
}

// A decoder holds the values from form, the 'reflect' value of main struct
// and the 'reflect' value of current path
type decoder struct {
	main reflect.Value

	curr reflect.Value
	typ  reflect.Type

	maps pathMaps

	path  string
	field string
	value string
	index int
}

// Decode decodes the url.Values into a element that must be a pointer to a type provided by argument
func Decode(vs url.Values, dst interface{}) error {
	main := reflect.ValueOf(dst)
	if main.Kind() != reflect.Ptr {
		return fmt.Errorf("formam: the value passed for decode is not a pointer but a %v", main.Kind())
	}
	d := &decoder{main: main.Elem()}
	for k, v := range vs {
		d.path = k
		d.field = k
		d.value = v[0]
		if d.value != "" {
			if err := d.begin(); err != nil {
				return err
			}
		}
	}
	for _, v := range d.maps {
		key := v.m.Type().Key()
		switch key.Kind() {
		case reflect.String:
			// the key is a string
			v.m.SetMapIndex(reflect.ValueOf(v.key), v.value)
		default:
			// must to implement the TextUnmarshaler interface for to can to decode the map's key
			var vv reflect.Value

			if key.Kind() == reflect.Ptr {
				vv = reflect.New(key.Elem())
			} else {
				vv = reflect.New(key).Elem()
			}

			d.value = v.key
			ok, err := d.unmarshalText(vv)
			if !ok {
				return fmt.Errorf("formam: the key with %s type (%v) in the path %v should implements the TextUnmarshaler interface for to can decode it", key, v.m.Type(), v.path)
			}
			if err != nil {
				return fmt.Errorf("formam: an error has occured in the UnmarshalText method for type %s: %s", key, err)
			}

			v.m.SetMapIndex(vv, v.value)
		}
	}
	d.maps = []*pathMap{}
	return nil
}

// begin prepare the current path to walk through it
func (d *decoder) begin() (err error) {
	d.curr = d.main
	fields := strings.Split(d.field, ".")
	for i, field := range fields {
		b := strings.IndexAny(field, "[")
		if b != -1 {
			// is a array
			e := strings.IndexAny(field, "]")
			if e == -1 {
				return errors.New("formam: bad syntax array")
			}
			d.field = field[:b]
			if d.index, err = strconv.Atoi(field[b+1 : e]); err != nil {
				return errors.New("formam: the index of array is not a number")
			}
			if len(fields) == i+1 {
				return d.end()
			}
			if err = d.walk(); err != nil {
				return
			}
		} else {
			// not is a array
			d.field = field
			d.index = -1
			if len(fields) == i+1 {
				return d.end()
			}
			if err = d.walk(); err != nil {
				return
			}
		}
	}
	return
}

// walk traverses the current path until to the last field
func (d *decoder) walk() error {
	// check if is a struct or map
	switch d.curr.Kind() {
	case reflect.Struct:
		if err := d.findField(); err != nil {
			return err
		}
	case reflect.Map:
		d.currentMap()
	}
	// check if the struct or map is a interface
	if d.curr.Kind() == reflect.Interface {
		d.curr = d.curr.Elem()
	}
	// check if the struct or map is a pointer
	if d.curr.Kind() == reflect.Ptr {
		if d.curr.IsNil() {
			d.curr.Set(reflect.New(d.curr.Type().Elem()))
		}
		d.curr = d.curr.Elem()
	}
	// finally, check if there are access to slice/array or not...
	if d.index != -1 {
		switch d.curr.Kind() {
		case reflect.Slice, reflect.Array:
			if d.curr.Len() <= d.index {
				d.expandSlice()
			}
			d.curr = d.curr.Index(d.index)
		default:
			return fmt.Errorf("formam: the field \"%v\" in path \"%v\" has a index for array but it is not", d.field, d.path)
		}
	}
	return nil
}

// end finds the last field for decode its value correspondent
func (d *decoder) end() error {
	if d.curr.Kind() == reflect.Struct {
		if err := d.findField(); err != nil {
			return err
		}
	}
	if d.value == "" {
		return nil
	}
	return d.decode()
}

// decode sets the value in the last field found by end function
func (d *decoder) decode() error {
	ok, err := d.unmarshalText(d.curr)
	if ok {
		return err
	} else if err != nil {
		return err
	}

	switch d.curr.Kind() {
	case reflect.Map:
		d.currentMap()
		return d.decode()
	case reflect.Slice, reflect.Array:
		if d.curr.Len() <= d.index {
			d.expandSlice()
		}
		d.curr = d.curr.Index(d.index)
		return d.decode()
	case reflect.String:
		d.curr.SetString(d.value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if num, err := strconv.ParseInt(d.value, 10, 64); err != nil {
			return fmt.Errorf("formam: the value of field \"%v\" in path \"%v\" should be a valid signed integer number", d.field, d.path)
		} else {
			d.curr.SetInt(num)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if num, err := strconv.ParseUint(d.value, 10, 64); err != nil {
			return fmt.Errorf("formam: the value of field \"%v\" in path \"%v\" should be a valid unsigned integer number", d.field, d.path)
		} else {
			d.curr.SetUint(num)
		}
	case reflect.Float32, reflect.Float64:
		if num, err := strconv.ParseFloat(d.value, d.curr.Type().Bits()); err != nil {
			return fmt.Errorf("formam: the value of field \"%v\" in path \"%v\" should be a valid float number", d.field, d.path)
		} else {
			d.curr.SetFloat(num)
		}
	case reflect.Bool:
		switch d.value {
		case "true", "on", "1":
			d.curr.SetBool(true)
		case "false", "off", "0":
			d.curr.SetBool(false)
		default:
			return fmt.Errorf("formam: the value of field \"%v\" in path \"%v\" is not a valid boolean", d.field, d.path)
		}
	case reflect.Interface:
		d.curr.Set(reflect.ValueOf(d.value))
	case reflect.Ptr:
		d.curr.Set(reflect.New(d.curr.Type().Elem()))
		d.curr = d.curr.Elem()
		return d.decode()
	case reflect.Struct:
		switch d.curr.Interface().(type) {
		case time.Time:
			t, err := time.Parse("2006-01-02", d.value)
			if err != nil {
				return fmt.Errorf("formam: the value of field \"%v\" in path \"%v\" is not a valid datetime", d.field, d.path)
			}
			d.curr.Set(reflect.ValueOf(t))
		case url.URL:
			u, err := url.Parse(d.value)
			if err != nil {
				return fmt.Errorf("formam: the value of field \"%v\" in path \"%v\" is not a valid url", d.field, d.path)
			}
			d.curr.Set(reflect.ValueOf(*u))
		default:
			return fmt.Errorf("formam: not supported type for field \"%v\" in path \"%v\"", d.field, d.path)
		}
	default:
		return fmt.Errorf("formam: not supported type for field \"%v\" in path \"%v\"", d.field, d.path)
	}

	return nil
}

// findField finds a field by its name, if it is not found,
// then retry the search examining the tag "formam" of every field of struct
func (d *decoder) findField() error {
	var anon reflect.Value

	num := d.curr.NumField()
	for i := 0; i < num; i++ {
		field := d.curr.Type().Field(i)
		if field.Name == d.field {
			// check if the field's name is equal
			d.curr = d.curr.Field(i)
			return nil
		} else if field.Anonymous {
			// if the field is a anonymous struct, then iterate over its fields
			tmp := d.curr
			d.curr = d.curr.FieldByIndex(field.Index)
			if err := d.findField(); err != nil {
				d.curr = tmp
				continue
			}
			// field in anonymous struct is found,
			// but first it should found the field in the rest of struct
			// (a field with same name in the current struct should have preference over anonymous struct)
			anon = d.curr
			d.curr = tmp
		} else if d.field == field.Tag.Get(TAG_NAME) {
			// is not found yet, then retry by its tag name "formam"
			d.curr = d.curr.Field(i)
			return nil
		}
	}

	if anon.IsValid() {
		d.curr = anon
		return nil
	}

	return fmt.Errorf("formam: not found the field \"%v\" in the path \"%v\"", d.field, d.path)
}

// expandSlice expands the length and capacity of the current slice
func (d *decoder) expandSlice() {
	sli := reflect.MakeSlice(d.curr.Type(), d.index+1, d.index+1)
	reflect.Copy(sli, d.curr)
	d.curr.Set(sli)
}

// currentMap gets in d.curr the map concrete for decode the current value
func (d *decoder) currentMap() {
	typ := d.curr.Type()
	if d.curr.IsNil() {
		d.curr.Set(reflect.MakeMap(typ))
		v := reflect.New(typ.Elem()).Elem()
		d.maps = append(d.maps, &pathMap{d.curr, d.field, v, d.path})
		d.curr = v
	} else if a := d.maps.find(d.curr, d.field); a == nil {
		v := reflect.New(typ.Elem()).Elem()
		d.maps = append(d.maps, &pathMap{d.curr, d.field, v, d.path})
		d.curr = v
	} else {
		d.curr = a.value
	}
}

var (
	timeType  = reflect.TypeOf(time.Time{})
	timePType = reflect.TypeOf(&time.Time{})
)

// unmarshalText returns a boolean and error. The boolean is true if the
// value implements TextUnmarshaler, and false if not.
func (d *decoder) unmarshalText(v reflect.Value) (bool, error) {
	// skip if the type is time.Time
	typ := v.Type()
	if typ.ConvertibleTo(timeType) || typ.ConvertibleTo(timePType) {
		return false, nil
	}
	// check if implements the interface
	t, ok := v.Interface().(encoding.TextUnmarshaler)
	canAddr := v.CanAddr()
	if !ok && !canAddr {
		return false, nil
	} else if canAddr {
		return d.unmarshalText(v.Addr())
	}
	// return result
	err := t.UnmarshalText([]byte(d.value))
	return true, err
}
