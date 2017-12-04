package mvc2

import (
	"fmt"
	"reflect"
)

type service struct {
	Type             reflect.Type
	Value            reflect.Value
	StructFieldIndex []int

	// for func input.
	ReturnValue           func(ctx []reflect.Value) reflect.Value
	FuncInputIndex        int
	FuncInputContextIndex int
}

type services []*service

func (serv *services) AddSource(dest reflect.Value, source ...reflect.Value) {
	fmt.Println("--------------AddSource------------")
	if len(source) == 0 {
		return
	}

	typ := indirectTyp(dest.Type()) //indirectTyp(reflect.TypeOf(dest))
	_serv := *serv

	if typ.Kind() == reflect.Func {
		n := typ.NumIn()
		for i := 0; i < n; i++ {

			inTyp := typ.In(i)
			if isContext(inTyp) {
				_serv = append(_serv, &service{FuncInputContextIndex: i})
				continue
			}

			for _, s := range source {
				gotTyp := s.Type()

				service := service{
					Type:                  gotTyp,
					Value:                 s,
					FuncInputIndex:        i,
					FuncInputContextIndex: -1,
				}

				if s.Type().Kind() == reflect.Func {
					fmt.Printf("Source is Func\n")
					returnValue, outType, err := makeReturnValue(s)
					if err != nil {
						fmt.Printf("Err on makeReturnValue: %v\n", err)
						continue
					}
					gotTyp = outType
					service.ReturnValue = returnValue
				}

				fmt.Printf("Types: In=%s vs Got=%s\n", inTyp.String(), gotTyp.String())
				if equalTypes(gotTyp, inTyp) {
					service.Type = gotTyp
					fmt.Printf("Bind In=%s->%s for func\n", inTyp.String(), gotTyp.String())
					_serv = append(_serv, &service)

					break
				}
			}
		}
		fmt.Printf("[1] Bind %d for %s\n", len(_serv), typ.String())
		*serv = _serv

		return
	}

	if typ.Kind() == reflect.Struct {
		fields := lookupFields(typ, -1)
		for _, f := range fields {
			for _, s := range source {
				gotTyp := s.Type()

				service := service{
					Type:                  gotTyp,
					Value:                 s,
					StructFieldIndex:      f.Index,
					FuncInputContextIndex: -1,
				}

				if s.Type().Kind() == reflect.Func {
					returnValue, outType, err := makeReturnValue(s)
					if err != nil {
						continue
					}
					gotTyp = outType
					service.ReturnValue = returnValue
				}

				if equalTypes(gotTyp, f.Type) {
					service.Type = gotTyp
					_serv = append(_serv, &service)
					fmt.Printf("[2] Bind In=%s->%s for struct field[%d]\n", f.Type, gotTyp.String(), f.Index)
					break
				}
			}
		}
		fmt.Printf("[2] Bind %d for %s\n", len(_serv), typ.String())
		*serv = _serv

		return
	}
}

func (serv services) FillStructStaticValues(elem reflect.Value) {
	if len(serv) == 0 {
		return
	}

	for _, s := range serv {
		if len(s.StructFieldIndex) > 0 {
			// fmt.Printf("FillStructStaticValues for index: %d\n", s.StructFieldIndex)
			elem.FieldByIndex(s.StructFieldIndex).Set(s.Value)
		}
	}
}

func (serv services) FillStructDynamicValues(elem reflect.Value, ctx []reflect.Value) {
	if len(serv) == 0 {
		return
	}

	for _, s := range serv {
		if len(s.StructFieldIndex) > 0 {
			elem.FieldByIndex(s.StructFieldIndex).Set(s.ReturnValue(ctx))
		}
	}
}

func (serv services) FillFuncInput(ctx []reflect.Value, destIn *[]reflect.Value) {
	if len(serv) == 0 {
		return
	}

	in := *destIn
	for _, s := range serv {
		if s.ReturnValue != nil {
			in[s.FuncInputIndex] = s.ReturnValue(ctx)
			continue
		}

		in[s.FuncInputIndex] = s.Value
		if s.FuncInputContextIndex >= 0 {
			in[s.FuncInputContextIndex] = ctx[0]
		}
	}

	*destIn = in
}

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
		// []reflect.Value{reflect.ValueOf(ctx)}
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

func getServicesFor(dest reflect.Value, source []reflect.Value) (s services) {
	s.AddSource(dest, source...)
	return s
}
