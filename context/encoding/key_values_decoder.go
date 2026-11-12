package encoding

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/kataras/iris/v14/core/pkg/reflex"

	"uuid"
)

// KeyValuesDecoderOptions is the options for the url query decoder.
// See `NewKeyValuesDecoder` package-level function for more.
type KeyValuesDecoderOptions = EntriesDecoderOptions[[]string]

// KeyValuesDecoder is a url query decoder.
// It decodes the url query to the target struct value of T.
// It is safe for concurrent use.
// See `NewKeyValuesDecoder` package-level function for more.
type KeyValuesDecoder[T any] struct {
	*EntriesDecoder[T, []string]
}

// NewKeyValuesDecoder returns a new url query decoder.
// It returns an error if T is not a pointer to a struct.
// The options can be empty.
// See `QueryDecoderOptions` struct and its `DecodeKeyValues` method for more.
func NewKeyValuesDecoder[T any](opts KeyValuesDecoderOptions) (*KeyValuesDecoder[T], error) {
	entriesDecoder, err := NewEntriesDecoder[T](opts)
	if err != nil {
		return nil, err
	}

	dec := &KeyValuesDecoder[T]{
		EntriesDecoder: entriesDecoder,
	}
	return dec, nil
}

// DecodeKeyValues decodes the url query to the target struct value of T.
// Multiple values using the same key e.g. names=makis&names=george.
func (d *KeyValuesDecoder[T]) DecodeKeyValues(query map[string][]string, target *T) error {
	if d.options.Strict {
		if expectedFieldsLen, queryLen := len(d.fields), len(query); expectedFieldsLen != queryLen {
			if queryLen > expectedFieldsLen { // the expectedFieldsLen > queryLen actually is calculated above, with the required option forcely set to true if Strict was true.
				// extract the extra fields from query.
				extraFields := make([]string, 0, queryLen-expectedFieldsLen)
				extraFieldsValues := make([][]string, 0, queryLen-expectedFieldsLen)
				for paramName := range query {
					if _, ok := d.fields[paramName]; !ok {
						extraFields = append(extraFields, paramName)
						extraFieldsValues = append(extraFieldsValues, query[paramName])
					}
				}

				return newExtraFieldError(strings.Join(extraFields, ", "), extraFieldsValues)
			}
		}
	}

	targetPtr := unsafe.Pointer(target)
	for paramName, info := range d.fields {
		values, ok := query[paramName]
		if !ok || len(values) == 0 {
			if info.required {
				return newRequiredFieldError(paramName, values)
			}
			continue
		}

		fieldPtr := unsafe.Add(targetPtr, info.offset)
		if err := info.setter(info.field, fieldPtr, values); err != nil {
			return fmt.Errorf("%v->%s: %w", values, paramName, err)
		}
	}

	if err := d.validateTarget(target); err != nil {
		return err
	}

	return nil
}

// DefaultQueryFieldSetter is the default field setter for url query values.
var DefaultQueryFieldSetter = func(field *reflect.StructField, ptr unsafe.Pointer, values []string) error {
	switch field.Type {
	case reflex.StringType:
		*(*string)(ptr) = values[0] // values are always at least one here.
	case reflex.IntType:
		intVal, err := strconv.Atoi(values[0])
		if err != nil {
			return err
		}
		*(*int)(ptr) = intVal
	case reflex.Int8Type:
		intVal, err := strconv.ParseInt(values[0], 10, 8)
		if err != nil {
			return err
		}
		*(*int8)(ptr) = int8(intVal)
	case reflex.Int16Type:
		intVal, err := strconv.ParseInt(values[0], 10, 16)
		if err != nil {
			return err
		}
		*(*int16)(ptr) = int16(intVal)
	case reflex.Int32Type:
		intVal, err := strconv.ParseInt(values[0], 10, 32)
		if err != nil {
			return err
		}
		*(*int32)(ptr) = int32(intVal)
	case reflex.Int64Type:
		intVal, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil {
			return err
		}
		*(*int64)(ptr) = intVal
	case reflex.UintType:
		uintVal, err := strconv.ParseUint(values[0], 10, 64)
		if err != nil {
			return err
		}
		*(*uint)(ptr) = uint(uintVal)
	case reflex.Uint8Type:
		uintVal, err := strconv.ParseUint(values[0], 10, 8)
		if err != nil {
			return err
		}
		*(*uint8)(ptr) = uint8(uintVal)
	case reflex.Uint16Type:
		uintVal, err := strconv.ParseUint(values[0], 10, 16)
		if err != nil {
			return err
		}
		*(*uint16)(ptr) = uint16(uintVal)
	case reflex.Uint32Type:
		uintVal, err := strconv.ParseUint(values[0], 10, 32)
		if err != nil {
			return err
		}
		*(*uint32)(ptr) = uint32(uintVal)
	case reflex.Uint64Type:
		uintVal, err := strconv.ParseUint(values[0], 10, 64)
		if err != nil {
			return err
		}
		*(*uint64)(ptr) = uintVal
	case reflex.Float32Type:
		floatVal, err := strconv.ParseFloat(values[0], 32)
		if err != nil {
			return err
		}
		*(*float32)(ptr) = float32(floatVal)
	case reflex.Float64Type:
		floatVal, err := strconv.ParseFloat(values[0], 64)
		if err != nil {
			return err
		}
		*(*float64)(ptr) = floatVal
	case reflex.BoolType:
		boolVal, err := strconv.ParseBool(values[0])
		if err != nil {
			return err
		}
		*(*bool)(ptr) = boolVal
	case reflex.ArrayStringTyp:
		*(*[]string)(ptr) = values
	case reflex.ArrayIntegerTyp:
		intValues := make([]int, len(values))
		for i, v := range values {
			intVal, err := strconv.Atoi(v)
			if err != nil {
				return err
			}
			intValues[i] = intVal
		}
		*(*[]int)(ptr) = intValues
	case reflex.TimeType:
		t, err := time.Parse(time.RFC3339, values[0])
		if err != nil {
			return err
		}
		*(*time.Time)(ptr) = t
	case reflex.WeekdayType:
		weekday, err := strconv.Atoi(values[0])
		if err != nil {
			return err
		}
		*(*time.Weekday)(ptr) = time.Weekday(weekday)
	case reflex.UUIDType:
		id, err := uuid.Parse(values[0])
		if err != nil {
			return err
		}
		*(*uuid.UUID)(ptr) = id
	case reflex.DurationType:
		d, err := time.ParseDuration(values[0])
		if err != nil {
			return err
		}
		*(*time.Duration)(ptr) = d
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedType, field.Type.String())
	}

	return nil
}

/*
	setter := opts.FieldSetter
	if setter == nil {
		switch field.Type.Kind() {
		case reflect.String:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*string)(ptr) = value.(string)
				return nil
			}
		case reflect.Int:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*int)(ptr) = value.(int)
				return nil
			}
		case reflect.Int8:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*int8)(ptr) = value.(int8)
				return nil
			}
		case reflect.Int16:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*int16)(ptr) = value.(int16)
				return nil
			}
		case reflect.Int32:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*int32)(ptr) = value.(int32)
				return nil
			}
		case reflect.Int64:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*int64)(ptr) = value.(int64)
				return nil
			}
		case reflect.Uint:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*uint)(ptr) = value.(uint)
				return nil
			}
		case reflect.Uint8:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*uint8)(ptr) = value.(uint8)
				return nil
			}
		case reflect.Uint16:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*uint16)(ptr) = value.(uint16)
				return nil
			}
		case reflect.Uint32:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*uint32)(ptr) = value.(uint32)
				return nil
			}
		case reflect.Uint64:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*uint64)(ptr) = value.(uint64)
				return nil
			}
		case reflect.Float32:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*float32)(ptr) = value.(float32)
				return nil
			}
		case reflect.Float64:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*float64)(ptr) = value.(float64)
				return nil
			}
		case reflect.Bool:
			setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
				*(*bool)(ptr) = value.(bool)
				return nil
			}
		case reflect.Slice:
			switch field.Type.Elem().Kind() {
			case reflect.String:
				setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
					if str, ok := value.(string); ok {
						*(*[]string)(ptr) = strings.Split(str, "/")
						return nil
					}

					*(*[]string)(ptr) = value.([]string)
					return nil
				}
			case reflect.Int:
				setter = func(_ reflect.Kind, ptr unsafe.Pointer, value any) error {
					*(*[]int)(ptr) = value.([]int)
					return nil
				}
			}
		default:
			return nil, fmt.Errorf("unsupported type %s.%s", typ.String(), field.Type)
		}

		opts.FieldSetter = setter
	}
*/
/*
	^
	BenchmarkReadParams_SchemaDecodeVsKataras/kataras_param_decoder-20              42532648               290.9 ns/op           176 B/op          4 allocs/op
	VS
	BenchmarkReadParams_SchemaDecodeVsKataras/kataras_param_decoder-20              41677461               293.1 ns/op           176 B/op          4 allocs/op

	The difference between the above commented code and the "defaultSetter" one is too small, so we use the 2nd one which is easier to read:
*/
// Other way if the value was string:
/*
	var setter fieldSetter
	switch field.Type.Kind() {
	case reflect.String:
		setter = func(ptr unsafe.Pointer, value string) error {
			*(*string)(ptr) = value
			return nil
		}
	// case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	// 	setter = func(ptr unsafe.Pointer, value string) error {
	// 		intVal, err := strconv.ParseInt(value, 10, 64)
	// 		if err != nil {
	// 			return err
	// 		}
	//
	// 		// Go guarantees that the memory layout of integer types is consistent with their sizes. That is, an int8 occupies 1 byte,
	// 		// an int16 occupies 2 bytes, an int32 occupies 4 bytes, and an int64 occupies 8 bytes.
	// 		//
	// 		// When you assign an int64 value to a field of type int, you’re essentially copying the lower part of the int64 value into the int field. Since int in Go is either 32
	// 		// or 64 bits depending on the platform, it can safely hold the value of an int64 as long as the value doesn’t exceed the maximum value that an int can hold on that platform.
	// 		//
	// 		// But let's divide this part, we are not sure that the running system is 64bit, so we need to check if the int64 value is within the int range.
	// 		*(*int64)(ptr) = intVal
	// 		return nil
	// 	}
	case reflect.Int:
		setter = func(ptr unsafe.Pointer, value string) error {
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}

			*(*int)(ptr) = int(intVal)
			return nil
		}
	case reflect.Int8:
		setter = func(ptr unsafe.Pointer, value string) error {
			intVal, err := strconv.ParseInt(value, 10, 8)
			if err != nil {
				return err
			}

			*(*int8)(ptr) = int8(intVal)
			return nil
		}
	case reflect.Int16:
		setter = func(ptr unsafe.Pointer, value string) error {
			intVal, err := strconv.ParseInt(value, 10, 16)
			if err != nil {
				return err
			}

			*(*int16)(ptr) = int16(intVal)
			return nil
		}
	case reflect.Int32:
		setter = func(ptr unsafe.Pointer, value string) error {
			intVal, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return err
			}

			*(*int32)(ptr) = int32(intVal)
			return nil
		}
	case reflect.Int64:
		setter = func(ptr unsafe.Pointer, value string) error {
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}

			*(*int64)(ptr) = intVal
			return nil
		}

	case reflect.Uint:
		setter = func(ptr unsafe.Pointer, value string) error {
			uintVal, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}

			*(*uint)(ptr) = uint(uintVal)
			return nil
		}
	case reflect.Uint8:
		setter = func(ptr unsafe.Pointer, value string) error {
			uintVal, err := strconv.ParseUint(value, 10, 8)
			if err != nil {
				return err
			}

			*(*uint8)(ptr) = uint8(uintVal)
			return nil
		}
	case reflect.Uint16:
		setter = func(ptr unsafe.Pointer, value string) error {
			uintVal, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				return err
			}

			*(*uint16)(ptr) = uint16(uintVal)
			return nil
		}
	case reflect.Uint32:
		setter = func(ptr unsafe.Pointer, value string) error {
			uintVal, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return err
			}

			*(*uint32)(ptr) = uint32(uintVal)
			return nil
		}
	case reflect.Uint64:
		setter = func(ptr unsafe.Pointer, value string) error {
			uintVal, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}

			*(*uint64)(ptr) = uintVal
			return nil
		}

	case reflect.Float32:
		setter = func(ptr unsafe.Pointer, value string) error {
			floatVal, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return err
			}

			*(*float32)(ptr) = float32(floatVal)
			return nil
		}
	case reflect.Float64:
		setter = func(ptr unsafe.Pointer, value string) error {
			floatVal, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}

			*(*float64)(ptr) = floatVal
			return nil
		}

	case reflect.Bool:
		setter = func(ptr unsafe.Pointer, value string) error {
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}

			*(*bool)(ptr) = boolVal
			return nil
		}
	case reflect.Slice:
		switch field.Type.Elem().Kind() {
		case reflect.String:
			setter = func(ptr unsafe.Pointer, value string) error {
				*(*[]string)(ptr) = strings.Split(value, "/")
				return nil
			}
		case reflect.Int:
			setter = func(ptr unsafe.Pointer, value string) error {
				values := strings.Split(value, "/")
				intValues := make([]int, len(values))
				for i, v := range values {
					intVal, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						return err
					}
					intValues[i] = int(intVal)
				}

				*(*[]int)(ptr) = intValues
				return nil
			}
		}
	default:
		return nil, fmt.Errorf("unsupported type %s.%s", typ.String(), field.Type)
	}
*/
