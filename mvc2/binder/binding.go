package binder

import (
	"reflect"
)

type Binding interface {
	AddSource(v reflect.Value, source ...reflect.Value)
}

type StructValue struct {
	Type  reflect.Type
	Value reflect.Value
}

type FuncResultValue struct {
	Type        reflect.Type
	ReturnValue func(ctx []reflect.Value) reflect.Value
}
