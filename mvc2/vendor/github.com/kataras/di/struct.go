package di

import "reflect"

type (
	targetStructField struct {
		Object     *BindObject
		FieldIndex []int
	}

	StructInjector struct {
		elemType reflect.Type
		//
		fields []*targetStructField
		Valid  bool // is True when contains fields and it's a valid target struct.
	}
)

func MakeStructInjector(v reflect.Value, hijack Hijacker, goodFunc TypeChecker, values ...reflect.Value) *StructInjector {
	s := &StructInjector{
		elemType: indirectTyp(v.Type()),
	}

	fields := lookupFields(s.elemType, nil)
	for _, f := range fields {

		if hijack != nil {
			if b, ok := hijack(f.Type); ok && b != nil {
				s.fields = append(s.fields, &targetStructField{
					FieldIndex: f.Index,
					Object:     b,
				})

				continue
			}
		}

		for _, val := range values {
			// the binded values to the struct's fields.
			b, err := MakeBindObject(val, goodFunc)

			if err != nil {
				return s // if error stop here.
			}

			if b.IsAssignable(f.Type) {
				// fmt.Printf("bind the object to the field: %s at index: %#v and type: %s\n", f.Name, f.Index, f.Type.String())
				s.fields = append(s.fields, &targetStructField{
					FieldIndex: f.Index,
					Object:     &b,
				})
				break
			}

		}
	}

	s.Valid = len(s.fields) > 0
	return s
}

func (s *StructInjector) Inject(dest interface{}, ctx ...reflect.Value) {
	if dest == nil {
		return
	}

	v := indirectVal(valueOf(dest))
	s.InjectElem(v, ctx...)
}

func (s *StructInjector) InjectElem(destElem reflect.Value, ctx ...reflect.Value) {
	for _, f := range s.fields {
		f.Object.Assign(ctx, func(v reflect.Value) {
			// fmt.Printf("%s for %s at index: %d\n", destElem.Type().String(), f.Object.Type.String(), f.FieldIndex)
			destElem.FieldByIndex(f.FieldIndex).Set(v)
		})
	}
}

func (s *StructInjector) New(ctx ...reflect.Value) reflect.Value {
	dest := reflect.New(s.elemType)
	s.InjectElem(dest, ctx...)
	return dest
}
