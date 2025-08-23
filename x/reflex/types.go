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
	StringType            = reflect.TypeFor[string]()
	BytesType             = reflect.TypeFor[[]byte]()
	IntType               = reflect.TypeFor[int]()
	Int16Type             = reflect.TypeFor[int16]()
	Int32Type             = reflect.TypeFor[int32]()
	Int64Type             = reflect.TypeFor[int64]()
	Float32Type           = reflect.TypeFor[float32]()
	Float64Type           = reflect.TypeFor[float64]()
	TimeType              = reflect.TypeFor[time.Time]()
	IpTyp                 = reflect.TypeFor[net.IP]()
	JSONNumberTyp         = reflect.TypeFor[json.Number]()
	StringerTyp           = reflect.TypeFor[fmt.Stringer]()
	ArrayIntegerTyp       = reflect.TypeFor[[]int]()
	ArrayStringTyp        = reflect.TypeFor[[]string]()
	DoubleArrayIntegerTyp = reflect.TypeFor[[][]int]()
	DoubleArrayStringTyp  = reflect.TypeFor[[][]string]()
	ErrTyp                = reflect.TypeFor[error]()
)
