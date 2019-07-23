package di

import (
	"fmt"
	"reflect"
)

type (
	targetFuncInput struct {
		Object     *BindObject
		InputIndex int
	}

	// FuncInjector keeps the data that are needed in order to do the binding injection
	// as fast as possible and with the best possible and safest way.
	FuncInjector struct {
		// the original function, is being used
		// only the .Call, which is referring to the same function, always.
		fn       reflect.Value
		typ      reflect.Type
		goodFunc TypeChecker

		inputs []*targetFuncInput
		// Length is the number of the valid, final binded input arguments.
		Length int
		// Valid is True when `Length` is > 0, it's statically set-ed for
		// performance reasons.
		Has bool

		trace string // for debug info.

		lost []*missingInput // Author's note: don't change this to a map.
	}
)

type missingInput struct {
	index int // the function's input argument's index.
	found bool
}

func (s *FuncInjector) miss(index int) {
	s.lost = append(s.lost, &missingInput{
		index: index,
	})
}

// MakeFuncInjector returns a new func injector, which will be the object
// that the caller should use to bind input arguments of the "fn" function.
//
// The hijack and the goodFunc are optional, the "values" is the dependencies collection.
func MakeFuncInjector(fn reflect.Value, hijack Hijacker, goodFunc TypeChecker, values ...reflect.Value) *FuncInjector {
	typ := IndirectType(fn.Type())
	s := &FuncInjector{
		fn:       fn,
		typ:      typ,
		goodFunc: goodFunc,
	}

	if !IsFunc(typ) {
		return s
	}

	defer s.refresh()

	n := typ.NumIn()

	for i := 0; i < n; i++ {
		inTyp := typ.In(i)

		if hijack != nil {
			b, ok := hijack(inTyp)

			if ok && b != nil {
				s.inputs = append(s.inputs, &targetFuncInput{
					InputIndex: i,
					Object:     b,
				})
				continue
			}
		}

		matched := false

		for j, v := range values {
			if s.addValue(i, v) {
				matched = true
				// remove this value, so it will not try to get binded
				// again, a next value even with the same type is able to be
				// used to other input arg. One value per input argument, order
				// matters if same type of course.
				//if len(values) > j+1 {
				values = append(values[:j], values[j+1:]...)
				//}

				break
			}
		}

		if !matched {
			// if no binding for this input argument,
			// this will make the func injector invalid state,
			// but before this let's make a list of failed
			// inputs, so they can be used for a re-try
			// with different set of binding "values".
			s.miss(i)
		}

	}

	return s
}

func (s *FuncInjector) refresh() {
	s.Length = len(s.inputs)
	s.Has = s.Length > 0
}

func (s *FuncInjector) addValue(inputIndex int, value reflect.Value) bool {
	defer s.refresh()

	if s.typ.NumIn() < inputIndex {
		return false
	}

	inTyp := s.typ.In(inputIndex)

	// the binded values to the func's inputs.
	b, err := MakeBindObject(value, s.goodFunc)

	if err != nil {
		return false
	}

	if b.IsAssignable(inTyp) {
		// fmt.Printf("binded input index: %d for type: %s and value: %v with pointer: %v\n",
		// 	i, b.Type.String(), inTyp.String(), inTyp.Pointer())
		s.inputs = append(s.inputs, &targetFuncInput{
			InputIndex: inputIndex,
			Object:     &b,
		})
		return true
	}

	return false
}

// Retry used to add missing dependencies, i.e path parameter builtin bindings if not already exists
// in the `hero.Handler`, once, only for that func injector.
func (s *FuncInjector) Retry(retryFn func(inIndex int, inTyp reflect.Type) (reflect.Value, bool)) bool {
	for _, missing := range s.lost {
		if missing.found {
			continue
		}

		invalidIndex := missing.index

		inTyp := s.typ.In(invalidIndex)
		v, ok := retryFn(invalidIndex, inTyp)
		if !ok {
			continue
		}

		if !s.addValue(invalidIndex, v) {
			continue
		}

		// if this value completes an invalid index
		// then remove this from the invalid input indexes.
		missing.found = true
	}

	return s.Length == s.typ.NumIn()
}

// String returns a debug trace text.
func (s *FuncInjector) String() (trace string) {
	for i, in := range s.inputs {
		bindmethodTyp := bindTypeString(in.Object.BindType)
		typIn := s.typ.In(in.InputIndex)
		// remember: on methods that are part of a struct (i.e controller)
		// the input index  = 1 is the begggining instead of the 0,
		// because the 0 is the controller receiver pointer of the method.
		trace += fmt.Sprintf("[%d] %s binding: '%s' for input position: %d and type: '%s'\n",
			i+1, bindmethodTyp, in.Object.Type.String(), in.InputIndex, typIn.String())
	}
	return
}

// Inject accepts an already created slice of input arguments
// and fills them, the "ctx" is optional and it's used
// on the dependencies that depends on one or more input arguments, these are the "ctx".
func (s *FuncInjector) Inject(in *[]reflect.Value, ctx ...reflect.Value) {
	args := *in
	for _, input := range s.inputs {
		input.Object.Assign(ctx, func(v reflect.Value) {
			// fmt.Printf("assign input index: %d for value: %v of type: %s\n",
			// 	input.InputIndex, v.String(), v.Type().Name())
			args[input.InputIndex] = v
		})

	}

	*in = args
}

// Call calls the "Inject" with a new slice of input arguments
// that are computed by the length of the input argument from the MakeFuncInjector's "fn" function.
//
// If the function needs a receiver, so
// the caller should be able to in[0] = receiver before injection,
// then the `Inject` method should be used instead.
func (s *FuncInjector) Call(ctx ...reflect.Value) []reflect.Value {
	in := make([]reflect.Value, s.Length, s.Length)
	s.Inject(&in, ctx...)
	return s.fn.Call(in)
}
