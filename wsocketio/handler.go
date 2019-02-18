package wsocketio

import (
	"fmt"
	"reflect"
)

type funcHandler struct {
	argTypes []reflect.Type
	f        reflect.Value
}

func newEventFunc(f interface{}) *funcHandler {
	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		panic("event handler must be a func.")
	}
	ft := fv.Type()
	if ft.NumIn() < 1 || ft.In(0).Name() != "Conn" {
		panic("handler function should be like func(socketio.Conn, ...)")
	}
	argTypes := make([]reflect.Type, ft.NumIn()-1)
	for i := range argTypes {
		argTypes[i] = ft.In(i + 1)
	}
	if len(argTypes) == 0 {
		argTypes = nil
	}
	return &funcHandler{
		argTypes: argTypes,
		f:        fv,
	}
}

func newAckFunc(f interface{}) *funcHandler {
	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		panic("ack callback must be a func.")
	}
	ft := fv.Type()
	argTypes := make([]reflect.Type, ft.NumIn())
	for i := range argTypes {
		argTypes[i] = ft.In(i)
	}
	if len(argTypes) == 0 {
		argTypes = nil
	}
	return &funcHandler{
		argTypes: argTypes,
		f:        fv,
	}
}

func (h *funcHandler) Call(args []reflect.Value) (ret []reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("event call error: %s", r)
			}
		}
	}()
	ret = h.f.Call(args)
	return
}
