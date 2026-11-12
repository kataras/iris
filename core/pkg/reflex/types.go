package reflex

import (
	stdContext "context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"reflect"
	"time"

	"uuid"
)

// Common reflect types for go standard data types.
var (
	StringType     = reflect.TypeFor[string]()
	BytesType      = reflect.TypeFor[[]byte]()
	IntType        = reflect.TypeFor[int]()
	Int8Type       = reflect.TypeFor[int8]()
	Int16Type      = reflect.TypeFor[int16]()
	Int32Type      = reflect.TypeFor[int32]()
	Int64Type      = reflect.TypeFor[int64]()
	UintType       = reflect.TypeFor[uint]()
	Uint8Type      = reflect.TypeFor[uint8]()
	Uint16Type     = reflect.TypeFor[uint16]()
	Uint32Type     = reflect.TypeFor[uint32]()
	Uint64Type     = reflect.TypeFor[uint64]()
	Float32Type    = reflect.TypeFor[float32]()
	Float64Type    = reflect.TypeFor[float64]()
	BoolType       = reflect.TypeFor[bool]()
	TimeType       = reflect.TypeFor[time.Time]()
	WeekdayType    = reflect.TypeFor[time.Weekday]()
	DurationType   = reflect.TypeFor[time.Duration]()
	UUIDType       = reflect.TypeFor[uuid.UUID]()
	NetipAddrType  = reflect.TypeFor[netip.Addr]()
	StdContextType = reflect.TypeFor[stdContext.Context]()
	IpTyp          = reflect.TypeFor[net.IP]()
	JSONNumberTyp  = reflect.TypeFor[json.Number]() // gojson.Number is an alias of this.

	JSONRawMessageTyp = reflect.TypeFor[json.RawMessage]() // gojson.RawMessage is an alias of this.

	StringerTyp           = reflect.TypeFor[fmt.Stringer]()
	ArrayIntegerTyp       = reflect.TypeFor[[]int]()
	ArrayStringTyp        = reflect.TypeFor[[]string]()
	DoubleArrayIntegerTyp = reflect.TypeFor[[][]int]()
	DoubleArrayStringTyp  = reflect.TypeFor[[][]string]()
	ErrTyp                = reflect.TypeFor[error]()
)
