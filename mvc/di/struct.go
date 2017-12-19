package di

import (
	"fmt"
	"reflect"
)

type State uint8

const (
	Stateless State = iota
	Singleton
)

type (
	targetStructField struct {
		Object     *BindObject
		FieldIndex []int
	}

	StructInjector struct {
		initRef        reflect.Value
		initRefAsSlice []reflect.Value // useful when the struct is passed on a func as input args via reflection.
		elemType       reflect.Type
		//
		fields []*targetStructField
		// is true when contains bindable fields and it's a valid target struct,
		// it maybe 0 but struct may contain unexported fields or exported but no bindable (Stateless)
		// see `setState`.
		HasFields bool
		CanInject bool // if any bindable fields when the state is NOT singleton.
		State     State
	}
)

func (s *StructInjector) countBindType(typ BindType) (n int) {
	for _, f := range s.fields {
		if f.Object.BindType == typ {
			n++
		}
	}
	return
}

func MakeStructInjector(v reflect.Value, hijack Hijacker, goodFunc TypeChecker, values ...reflect.Value) *StructInjector {
	s := &StructInjector{
		initRef:        v,
		initRefAsSlice: []reflect.Value{v},
		elemType:       IndirectType(v.Type()),
	}

	fields := lookupFields(s.elemType, true, nil)
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

	s.HasFields = len(s.fields) > 0
	// set the overall state of this injector.
	s.setState()
	s.fillStruct()

	return s
}

// set the state, once.
func (s *StructInjector) setState() {
	// note for zero length of struct's fields:
	// if struct doesn't contain any field
	// so both of the below variables will be 0,
	// so it's a singleton.
	// At the other hand the `s.HasFields` maybe false
	// but the struct may contain UNEXPORTED fields or non-bindable fields (request-scoped on both cases)
	// so a new controller/struct at the caller side should be initialized on each request,
	// we should not depend on the `HasFields` for singleton or no, this is the reason I
	// added the `.State` now.

	staticBindingsFieldsLength := s.countBindType(Static)
	structFieldsLength := NumFields(s.elemType, false)

	// println("staticBindingsFieldsLength: ", staticBindingsFieldsLength)
	// println("structFieldsLength: ", structFieldsLength)

	// if the number of static values binded is equal to the
	// total struct's fields(including unexported fields this time) then set as singleton.
	if staticBindingsFieldsLength == structFieldsLength {
		s.State = Singleton
		return
	}

	s.CanInject = s.State == Stateless && s.HasFields
	// the default is `Stateless`, which means that a new instance should be created
	//  on each inject action by the caller.
}

// fill the static bindings values once.
func (s *StructInjector) fillStruct() {
	if !s.HasFields {
		return
	}
	// if field is Static then set it to the value that passed by the caller,
	// so will have the static bindings already and we can just use that value instead
	// of creating new instance.
	destElem := IndirectValue(s.initRef)
	for _, f := range s.fields {
		// if field is Static then set it to the value that passed by the caller,
		// so will have the static bindings already and we can just use that value instead
		// of creating new instance.
		if s.State == Singleton && f.Object.BindType == Static {
			destElem.FieldByIndex(f.FieldIndex).Set(f.Object.Value)
		}
	}
}

func (s *StructInjector) String() (trace string) {
	for i, f := range s.fields {
		elemField := s.elemType.FieldByIndex(f.FieldIndex)
		trace += fmt.Sprintf("[%d] %s binding: '%s' for field '%s %s'\n",
			i+1, bindTypeString(f.Object.BindType), f.Object.Type.String(),
			elemField.Name, elemField.Type.String())
	}

	return
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
			destElem.FieldByIndex(f.FieldIndex).Set(v)
		})
	}
}

func (s *StructInjector) New() reflect.Value {
	if s.State == Singleton {
		return s.initRef
	}
	return reflect.New(s.elemType)
}

func (s *StructInjector) NewAsSlice() []reflect.Value {
	if s.State == Singleton {
		return s.initRefAsSlice
	}
	return []reflect.Value{reflect.New(s.elemType)}
}
