package reflex

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"time"
)

// Common reflect types for go standard data types.
var (
	StringType            = reflect.TypeOf("")
	BytesType             = reflect.TypeOf([]byte{})
	IntType               = reflect.TypeOf(int(0))
	Int16Type             = reflect.TypeOf(int16(0))
	Int32Type             = reflect.TypeOf(int32(0))
	Int64Type             = reflect.TypeOf(int64(0))
	Float32Type           = reflect.TypeOf(float32(0))
	Float64Type           = reflect.TypeOf(float64(0))
	TimeType              = reflect.TypeOf(time.Time{})
	IpTyp                 = reflect.TypeOf(net.IP{})
	JSONNumberTyp         = reflect.TypeOf(json.Number(""))
	StringerTyp           = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	ArrayIntegerTyp       = reflect.TypeOf([]int{})
	ArrayStringTyp        = reflect.TypeOf([]string{})
	DoubleArrayIntegerTyp = reflect.TypeOf([][]int{})
	DoubleArrayStringTyp  = reflect.TypeOf([][]string{})
	ErrTyp                = reflect.TypeOf((*error)(nil)).Elem()
)
