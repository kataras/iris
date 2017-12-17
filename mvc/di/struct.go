package di

import (
	"fmt"
	"reflect"
)

type (
	targetStructField struct {
		Object     *BindObject
		FieldIndex []int
	}

	StructInjector struct {
		elemType reflect.Type
		//
		fields []*targetStructField
		Valid  bool   // is true when contains fields and it's a valid target struct.
		trace  string // for debug info.
	}
)

func MakeStructInjector(v reflect.Value, hijack Hijacker, goodFunc TypeChecker, values ...reflect.Value) *StructInjector {
	s := &StructInjector{
		elemType: IndirectType(v.Type()),
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

	if s.Valid {
		for i, f := range s.fields {
			bindmethodTyp := "Static"
			if f.Object.BindType == Dynamic {
				bindmethodTyp = "Dynamic"
			}
			elemField := s.elemType.FieldByIndex(f.FieldIndex)
			s.trace += fmt.Sprintf("[%d] %s binding: '%s' for field '%s %s'\n", i+1, bindmethodTyp, f.Object.Type.String(), elemField.Name, elemField.Type.String())
		}
	}

	return s
}

func (s *StructInjector) String() string {
	return s.trace
}

func (s *StructInjector) Inject(dest interface{}, ctx ...reflect.Value) {
	if dest == nil {
		return
	}

	v := IndirectValue(ValueOf(dest))
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

func (s *StructInjector) InjectElemStaticOnly(destElem reflect.Value) (n int) {
	for _, f := range s.fields {
		if f.Object.BindType != Static {
			continue
		}
		destElem.FieldByIndex(f.FieldIndex).Set(f.Object.Value)
		n++
	}
	return
}

func (s *StructInjector) New(ctx ...reflect.Value) reflect.Value {
	dest := reflect.New(s.elemType)
	s.InjectElem(dest, ctx...)
	return dest
}
