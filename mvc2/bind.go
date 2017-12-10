package mvc2

import "reflect"

type bindType uint32

const (
	objectType         bindType = iota // simple assignable value.
	functionResultType                 // dynamic value, depends on the context.
)

type bindObject struct {
	Type  reflect.Type // the Type of 'Value' or the type of the returned 'ReturnValue' .
	Value reflect.Value

	BindType    bindType
	ReturnValue func(ctx []reflect.Value) reflect.Value
}

// makeReturnValue takes any function
// that accept a context and returns something
// and returns a binder function, which accepts the context as slice of reflect.Value
// and returns a reflect.Value for that.
// Iris uses to
// resolve and set the input parameters when a handler is executed.
//
// The "fn" can have the following form:
// `func(iris.Context) UserViewModel`.
//
// The return type of the "fn" should be a value instance, not a pointer, for your own protection.
// The binder function should return only one value and
// it can accept only one input argument,
// the Iris' Context (`context.Context` or `iris.Context`).
func makeReturnValue(fn reflect.Value) (func([]reflect.Value) reflect.Value, reflect.Type, error) {
	typ := indirectTyp(fn.Type())

	// invalid if not a func.
	if typ.Kind() != reflect.Func {
		return nil, typ, errBad
	}

	// invalid if not returns one single value.
	if typ.NumOut() != 1 {
		return nil, typ, errBad
	}

	// invalid if input args length is not one.
	if typ.NumIn() != 1 {
		return nil, typ, errBad
	}

	// invalid if that single input arg is not a typeof context.Context.
	if !isContext(typ.In(0)) {
		return nil, typ, errBad
	}

	outTyp := typ.Out(0)
	zeroOutVal := reflect.New(outTyp).Elem()

	bf := func(ctxValue []reflect.Value) reflect.Value {
		results := fn.Call(ctxValue) // ctxValue is like that because of; read makeHandler.
		if len(results) == 0 {
			return zeroOutVal
		}

		v := results[0]
		if !v.IsValid() {
			return zeroOutVal
		}
		return v
	}

	return bf, outTyp, nil
}

func makeBindObject(v reflect.Value) (b bindObject, err error) {
	if isFunc(v) {
		b.BindType = functionResultType
		b.ReturnValue, b.Type, err = makeReturnValue(v)
	} else {
		b.BindType = objectType
		b.Type = v.Type()
		b.Value = v
	}

	return
}

func (b *bindObject) IsAssignable(to reflect.Type) bool {
	return equalTypes(b.Type, to)
}

func (b *bindObject) Assign(ctx []reflect.Value, toSetter func(reflect.Value)) {
	if b.BindType == functionResultType {
		toSetter(b.ReturnValue(ctx))
		return
	}
	toSetter(b.Value)
}

type (
	targetField struct {
		Object     *bindObject
		FieldIndex []int
	}
	targetFuncInput struct {
		Object     *bindObject
		InputIndex int
	}
)

type targetStruct struct {
	Fields []*targetField
	Valid  bool // is True when contains fields and it's a valid target struct.
}

func newTargetStruct(v reflect.Value, bindValues ...reflect.Value) *targetStruct {
	typ := indirectTyp(v.Type())
	s := &targetStruct{}

	fields := lookupFields(typ, nil)
	for _, f := range fields {
		for _, val := range bindValues {
			// the binded values to the struct's fields.
			b, err := makeBindObject(val)

			if err != nil {
				return s // if error stop here.
			}

			if b.IsAssignable(f.Type) {
				// fmt.Printf("bind the object to the field: %s at index: %#v and type: %s\n", f.Name, f.Index, f.Type.String())
				s.Fields = append(s.Fields, &targetField{
					FieldIndex: f.Index,
					Object:     &b,
				})
				break
			}

		}
	}

	s.Valid = len(s.Fields) > 0
	return s
}

func (s *targetStruct) Fill(destElem reflect.Value, ctx ...reflect.Value) {
	for _, f := range s.Fields {
		f.Object.Assign(ctx, func(v reflect.Value) {
			// defer func() {
			// 	if err := recover(); err != nil {
			// 		fmt.Printf("for index: %#v on: %s where num fields are: %d\n",
			// 			f.FieldIndex, f.Object.Type.String(), destElem.NumField())
			// 	}
			// }()
			destElem.FieldByIndex(f.FieldIndex).Set(v)
		})
	}
}

type targetFunc struct {
	Inputs []*targetFuncInput
	Valid  bool // is True when contains func inputs and it's a valid target func.
}

func newTargetFunc(fn reflect.Value, bindValues ...reflect.Value) *targetFunc {
	typ := indirectTyp(fn.Type())
	s := &targetFunc{
		Valid: false,
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

		// if it's context then bind it directly here and continue to the next func's input arg.
		if isContext(inTyp) {
			s.Inputs = append(s.Inputs, &targetFuncInput{
				InputIndex: i,
				Object: &bindObject{
					Type:     contextTyp,
					BindType: functionResultType,
					ReturnValue: func(ctxValue []reflect.Value) reflect.Value {
						return ctxValue[0]
					},
				},
			})
			continue
		}

		for valIdx, val := range bindValues {
			if _, shouldSkip := consumedValues[valIdx]; shouldSkip {
				continue
			}
			inTyp := typ.In(i)

			// the binded values to the func's inputs.
			b, err := makeBindObject(val)

			if err != nil {
				return s // if error stop here.
			}

			if b.IsAssignable(inTyp) {
				// fmt.Printf("binded input index: %d for type: %s and value: %v with pointer: %v\n",
				// 	i, b.Type.String(), val.String(), val.Pointer())
				s.Inputs = append(s.Inputs, &targetFuncInput{
					InputIndex: i,
					Object:     &b,
				})

				consumedValues[valIdx] = true
				break
			}
		}
	}

	s.Valid = len(s.Inputs) > 0
	return s
}

func (s *targetFunc) Fill(in *[]reflect.Value, ctx ...reflect.Value) {
	args := *in
	for _, input := range s.Inputs {
		input.Object.Assign(ctx, func(v reflect.Value) {
			// fmt.Printf("assign input index: %d for value: %v\n",
			// 	input.InputIndex, v.String())
			args[input.InputIndex] = v
		})

	}

	*in = args
}
