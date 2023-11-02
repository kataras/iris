package context

import (
	"errors"
	"reflect"
	"sync"
)

// ErrInvalidArgs fires when the `Context.CallFunc`
// is called with invalid number of arguments.
var ErrInvalidArgs = errors.New("invalid arguments")

// Func represents a function registered by the Context.
// See its `buildMeta` and `call` internal methods.
type Func struct {
	RegisterName    string        // the name of which this function is registered, for information only.
	Raw             interface{}   // the Raw function, can be used for custom casting.
	PersistenceArgs []interface{} // the persistence input arguments given on registration.

	once sync.Once // guards build once, on first call.
	// Available after the first call.
	Meta *FuncMeta
}

func newFunc(name string, fn interface{}, persistenceArgs ...interface{}) *Func {
	return &Func{
		RegisterName:    name,
		Raw:             fn,
		PersistenceArgs: persistenceArgs,
	}
}

// FuncMeta holds the necessary information about a registered
// context function. Built once by the Func.
type FuncMeta struct {
	Handler            Handler              // when it's just a handler.
	HandlerWithErr     func(*Context) error // when it's just a handler which returns an error.
	RawFunc            func()               // when it's just a func.
	RawFuncWithErr     func() error         // when it's just a func which returns an error.
	RawFuncArgs        func(...interface{})
	RawFuncArgsWithErr func(...interface{}) error

	Value                   reflect.Value
	Type                    reflect.Type
	ExpectedArgumentsLength int
	PersistenceInputs       []reflect.Value
	AcceptsContext          bool // the Context, if exists should be always first argument.
	ReturnsError            bool // when the function's last output argument is error.
}

func (f *Func) buildMeta() {
	switch fn := f.Raw.(type) {
	case Handler:
		f.Meta = &FuncMeta{Handler: fn}
		return
	// case func(*Context):
	// 	f.Meta = &FuncMeta{Handler: fn}
	// 	return
	case func(*Context) error:
		f.Meta = &FuncMeta{HandlerWithErr: fn}
		return
	case func():
		f.Meta = &FuncMeta{RawFunc: fn}
		return
	case func() error:
		f.Meta = &FuncMeta{RawFuncWithErr: fn}
		return
	case func(...interface{}):
		f.Meta = &FuncMeta{RawFuncArgs: fn}
		return
	case func(...interface{}) error:
		f.Meta = &FuncMeta{RawFuncArgsWithErr: fn}
		return
	}

	fn := f.Raw

	meta := FuncMeta{}
	if val, ok := fn.(reflect.Value); ok {
		meta.Value = val
	} else {
		meta.Value = reflect.ValueOf(fn)
	}

	meta.Type = meta.Value.Type()

	if meta.Type.Kind() != reflect.Func {
		return
	}

	meta.ExpectedArgumentsLength = meta.Type.NumIn()

	skipInputs := len(meta.PersistenceInputs)
	if meta.ExpectedArgumentsLength > skipInputs {
		meta.AcceptsContext = isContext(meta.Type.In(skipInputs))
	}

	if numOut := meta.Type.NumOut(); numOut > 0 {
		// error should be the last output.
		if isError(meta.Type.Out(numOut - 1)) {
			meta.ReturnsError = true
		}
	}

	persistenceArgs := f.PersistenceArgs
	if len(persistenceArgs) > 0 {
		inputs := make([]reflect.Value, 0, len(persistenceArgs))
		for _, arg := range persistenceArgs {
			if in, ok := arg.(reflect.Value); ok {
				inputs = append(inputs, in)
			} else {
				inputs = append(inputs, reflect.ValueOf(in))
			}
		}

		meta.PersistenceInputs = inputs
	}

	f.Meta = &meta
}

func (f *Func) call(ctx *Context, args ...interface{}) ([]reflect.Value, error) {
	f.once.Do(f.buildMeta)
	meta := f.Meta

	if meta.Handler != nil {
		meta.Handler(ctx)
		return nil, nil
	}

	if meta.HandlerWithErr != nil {
		return nil, meta.HandlerWithErr(ctx)
	}

	if meta.RawFunc != nil {
		meta.RawFunc()
		return nil, nil
	}

	if meta.RawFuncWithErr != nil {
		return nil, meta.RawFuncWithErr()
	}

	if meta.RawFuncArgs != nil {
		meta.RawFuncArgs(args...)
		return nil, nil
	}

	if meta.RawFuncArgsWithErr != nil {
		return nil, meta.RawFuncArgsWithErr(args...)
	}

	inputs := make([]reflect.Value, 0, f.Meta.ExpectedArgumentsLength)
	inputs = append(inputs, f.Meta.PersistenceInputs...)
	if f.Meta.AcceptsContext {
		inputs = append(inputs, reflect.ValueOf(ctx))
	}

	for _, arg := range args {
		if in, ok := arg.(reflect.Value); ok {
			inputs = append(inputs, in)
		} else {
			inputs = append(inputs, reflect.ValueOf(arg))
		}
	}

	// keep it here, the inptus may contain the context.
	if f.Meta.ExpectedArgumentsLength != len(inputs) {
		return nil, ErrInvalidArgs
	}

	outputs := f.Meta.Value.Call(inputs)
	return outputs, getError(outputs)
}

var contextType = reflect.TypeOf((*Context)(nil))

// isContext returns true if the "typ" is a type of Context.
func isContext(typ reflect.Type) bool {
	return typ == contextType
}

var errTyp = reflect.TypeOf((*error)(nil)).Elem()

// isError returns true if "typ" is type of `error`.
func isError(typ reflect.Type) bool {
	return typ.Implements(errTyp)
}

func getError(outputs []reflect.Value) error {
	if n := len(outputs); n > 0 {
		lastOut := outputs[n-1]
		if isError(lastOut.Type()) {
			if lastOut.IsNil() {
				return nil
			}

			return lastOut.Interface().(error)
		}
	}

	return nil
}
