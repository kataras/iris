package binder

import (
	"reflect"
)

type StructBinding struct {
	Field StructValue
	Func  FuncResultValue
}

func (b *StructBinding) AddSource(dest reflect.Value, source ...reflect.Value) {
	typ := indirectTyp(dest.Type()) //indirectTyp(reflect.TypeOf(dest))
	if typ.Kind() != reflect.Struct {
		return
	}

	fields := lookupFields(typ, -1)
	for _, f := range fields {
		for _, s := range source {
			if s.Type().Kind() == reflect.Func {
				returnValue, outType, err := makeReturnValue(s)
				if err != nil {
					continue
				}
				gotTyp = outType
				service.ReturnValue = returnValue
			}

			gotTyp := s.Type()

			v := StructValue{
				Type:       gotTyp,
				Value:      s,
				FieldIndex: f.Index,
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
