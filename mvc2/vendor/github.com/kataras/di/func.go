package di

import (
	"reflect"
)

type (
	targetFuncInput struct {
		Object     *BindObject
		InputIndex int
	}

	FuncInjector struct {
		// the original function, is being used
		// only the .Call, which is refering to the same function, always.
		fn reflect.Value

		inputs []*targetFuncInput
		Length int
		Valid  bool // is True when contains func inputs and it's a valid target func.
	}
)

func MakeFuncInjector(fn reflect.Value, hijack Hijacker, goodFunc TypeChecker, values ...reflect.Value) *FuncInjector {
	typ := indirectTyp(fn.Type())
	s := &FuncInjector{
		fn: fn,
	}

	if !isFunc(typ) {
		return s
	}

	n := typ.NumIn()

	// function input can have many values of the same types,
	// so keep track of them in order to not set a func input to a next bind value,
	// i.e (string, string) with two different binder funcs because of the different param's name.
	consumedValues := make(map[int]bool, n)

	for i := 0; i < n; i++ {
		inTyp := typ.In(i)

		if hijack != nil {
			if b, ok := hijack(inTyp); ok && b != nil {
				s.inputs = append(s.inputs, &targetFuncInput{
					InputIndex: i,
					Object:     b,
				})

				continue
			}
		}

		for valIdx, val := range values {
			if _, shouldSkip := consumedValues[valIdx]; shouldSkip {
				continue
			}
			inTyp := typ.In(i)

			// the binded values to the func's inputs.
			b, err := MakeBindObject(val, goodFunc)

			if err != nil {
				return s // if error stop here.
			}

			if b.IsAssignable(inTyp) {
				// fmt.Printf("binded input index: %d for type: %s and value: %v with pointer: %v\n",
				// 	i, b.Type.String(), val.String(), val.Pointer())
				s.inputs = append(s.inputs, &targetFuncInput{
					InputIndex: i,
					Object:     &b,
				})

				consumedValues[valIdx] = true
				break
			}
		}
	}

	s.Length = n
	s.Valid = len(s.inputs) > 0
	return s
}

func (s *FuncInjector) Inject(in *[]reflect.Value, ctx ...reflect.Value) {
	args := *in
	for _, input := range s.inputs {
		input.Object.Assign(ctx, func(v reflect.Value) {
			// fmt.Printf("assign input index: %d for value: %v\n",
			// 	input.InputIndex, v.String())
			args[input.InputIndex] = v
		})

	}

	*in = args
}

func (s *FuncInjector) Call(ctx ...reflect.Value) []reflect.Value {
	in := make([]reflect.Value, s.Length, s.Length)
	s.Inject(&in, ctx...)
	return s.fn.Call(in)
}
