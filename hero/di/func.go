package di

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/v12/context"
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
		fn             reflect.Value
		typ            reflect.Type
		FallbackBinder FallbackBinder
		ErrorHandler   ErrorHandler

		inputs []*targetFuncInput
		// Length is the number of the valid, final binded input arguments.
		Length int
		// Valid is True when `Length` is > 0, it's statically set-ed for
		// performance reasons.
		Has bool

		lost []*missingInput // Author's note: don't change this to a map.
	}
)

type missingInput struct {
	index     int // the function's input argument's index.
	found     bool
	remaining Values
}

func (s *FuncInjector) miss(index int, remaining Values) {
	s.lost = append(s.lost, &missingInput{
		index:     index,
		remaining: remaining,
	})
}

// MakeFuncInjector returns a new func injector, which will be the object
// that the caller should use to bind input arguments of the "fn" function.
//
// The hijack and the goodFunc are optional, the "values" is the dependencies collection.
func MakeFuncInjector(fn reflect.Value, values ...reflect.Value) *FuncInjector {
	typ := IndirectType(fn.Type())
	s := &FuncInjector{
		fn:             fn,
		typ:            typ,
		FallbackBinder: DefaultFallbackBinder,
		ErrorHandler:   DefaultErrorHandler,
	}

	if !IsFunc(typ) {
		return s
	}

	defer s.refresh()

	n := typ.NumIn()

	for i := 0; i < n; i++ {
		inTyp := typ.In(i)

		if b, ok := tryBindContext(inTyp); ok {
			s.inputs = append(s.inputs, &targetFuncInput{
				InputIndex: i,
				Object:     b,
			})
			continue
		}

		matched := false

		for j, v := range values {
			if s.addValue(i, v) {
				matched = true
				// remove this value, so it will not try to get binded
				// again, a next value even with the same type is able to be
				// used to other input arg. One value per input argument, order
				// matters if same type of course.
				// if len(values) > j+1 {
				values = append(values[:j], values[j+1:]...)
				//}

				break
			}

			// TODO: (already working on it) clean up or even re-write the whole di, hero and some of the mvc,
			// this is a dirty but working-solution for #1449.
			// Limitations:
			// - last input argument
			// - not able to customize it other than DefaultFallbackBinder on MVC (on hero it can be customized)
			// - the "di" package is now depends on context package which is not an import-cycle issue, it's not imported there.
			if i == n-1 {
				if v.Type() == autoBindingTyp && s.FallbackBinder != nil {

					canFallback := true
					if k := inTyp.Kind(); k == reflect.Ptr {
						if inTyp.Elem().Kind() != reflect.Struct {
							canFallback = false
						}
					} else if k != reflect.Struct {
						canFallback = false
					}

					if canFallback {
						matched = true

						s.inputs = append(s.inputs, &targetFuncInput{
							InputIndex: i,
							Object: &BindObject{
								Type:     inTyp,
								BindType: Dynamic,
								ReturnValue: func(ctx context.Context) reflect.Value {
									value, err := s.FallbackBinder(ctx, OrphanInput{Type: inTyp})
									if err != nil {
										if s.ErrorHandler != nil {
											s.ErrorHandler.HandleError(ctx, err)
										}
									}

									return value
								},
							},
						})

						break
					}
				}
			}
		}

		if !matched {
			// if no binding for this input argument,
			// this will make the func injector invalid state,
			// but before this let's make a list of failed
			// inputs, so they can be used for a re-try
			// with different set of binding "values".
			s.miss(i, values) // send the remaining dependencies values.
		}
	}

	return s
}

func (s *FuncInjector) refresh() {
	s.Length = len(s.inputs)
	s.Has = s.Length > 0
}

// AutoBindingValue a fake type to expliclty set the return value of hero.AutoBinding.
type AutoBindingValue struct{}

var autoBindingTyp = reflect.TypeOf(AutoBindingValue{})

func (s *FuncInjector) addValue(inputIndex int, value reflect.Value) bool {
	defer s.refresh()

	if s.typ.NumIn() < inputIndex {
		return false
	}

	inTyp := s.typ.In(inputIndex)

	// the binded values to the func's inputs.
	b, err := MakeBindObject(value, s.ErrorHandler)
	if err != nil {
		return false
	}

	if b.IsAssignable(inTyp) {
		// fmt.Printf("binded input index: %d for type: %s and value: %v with dependency: %v\n",
		// 	inputIndex, b.Type.String(), inTyp.String(), b)
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
func (s *FuncInjector) Retry(retryFn func(inIndex int, inTyp reflect.Type, remainingValues Values) (reflect.Value, bool)) bool {
	for _, missing := range s.lost {
		if missing.found {
			continue
		}

		invalidIndex := missing.index

		inTyp := s.typ.In(invalidIndex)
		v, ok := retryFn(invalidIndex, inTyp, missing.remaining)
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
func (s *FuncInjector) Inject(ctx context.Context, in *[]reflect.Value) {
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
func (s *FuncInjector) Call(ctx context.Context) []reflect.Value {
	in := make([]reflect.Value, s.Length)
	s.Inject(ctx, &in)
	return s.fn.Call(in)
}
