package hero

import (
	"fmt"
	"strings"

	"reflect"

	"github.com/kataras/iris/v12/context"
)

type (
	// DependencyHandler is the native function declaration which implementors should return a value match to an input.
	DependencyHandler = func(ctx *context.Context, input *Input) (reflect.Value, error)

	// DependencyMatchFunc type alias describes dependency
	// match function with an input (field or parameter).
	//
	// See "DependencyMatcher" too, which can be used on a Container to
	// change the way dependencies are matched to inputs for all dependencies.
	DependencyMatchFunc = func(in reflect.Type) bool

	// Dependency describes the design-time dependency to be injected at serve time.
	// Contains its source location, the dependency handler (provider) itself and information
	// such as static for static struct values or explicit to bind a value to its exact DestType and not if just assignable to it (interfaces).
	Dependency struct {
		OriginalValue interface{} // Used for debugging and for logging only.
		Source        Source
		Handle        DependencyHandler
		// It's the exact type of return to bind, if declared to return <T>, otherwise nil.
		DestType reflect.Type
		Static   bool
		// If true then input and dependency DestType should be indedical,
		// not just assiginable to each other.
		// Example of use case: depenendency like time.Time that we want to be bindable
		// only to time.Time inputs and not to a service with a `String() string` method that time.Time struct implements too.
		Explicit bool

		// Match holds the matcher. Defaults to the Container's one.
		Match DependencyMatchFunc

		// StructDependents if true then the Container will try to resolve
		// the fields of a struct value, if any, when it's a dependent struct value
		// based on the previous registered dependencies.
		//
		// Defaults to false.
		StructDependents bool
	}
)

// Explicitly sets Explicit option to true.
// See `Dependency.Explicit` field godoc for more.
//
// Returns itself.
func (d *Dependency) Explicitly() *Dependency {
	d.Explicit = true
	return d
}

// EnableStructDependents sets StructDependents to true.
func (d *Dependency) EnableStructDependents() *Dependency {
	d.StructDependents = true
	return d
}

func (d *Dependency) String() string {
	sourceLine := d.Source.String()
	val := d.OriginalValue
	if val == nil {
		val = d.Handle
	}
	return fmt.Sprintf("%s (%#+v)", sourceLine, val)
}

// NewDependency converts a function or a function which accepts other dependencies or static struct value to a *Dependency.
//
// See `Container.Handler` for more.
func NewDependency(dependency interface{}, funcDependencies ...*Dependency) *Dependency { // used only on tests.
	return newDependency(dependency, false, false, nil, funcDependencies...)
}

func newDependency(
	dependency interface{},
	disablePayloadAutoBinding bool,
	enableStructDependents bool,
	matchDependency DependencyMatcher,
	funcDependencies ...*Dependency,
) *Dependency {
	if dependency == nil {
		panic(fmt.Sprintf("bad value: nil: %T", dependency))
	}

	if d, ok := dependency.(*Dependency); ok {
		// already a *Dependency, do not continue (and most importatly do not call resolveDependency) .
		return d
	}

	v := valueOf(dependency)
	if !goodVal(v) {
		panic(fmt.Sprintf("bad value: %#+v", dependency))
	}

	if matchDependency == nil {
		matchDependency = DefaultDependencyMatcher
	}

	dest := &Dependency{
		Source:           newSource(v),
		OriginalValue:    dependency,
		StructDependents: enableStructDependents,
	}
	dest.Match = ToDependencyMatchFunc(dest, matchDependency)

	if !resolveDependency(v, disablePayloadAutoBinding, dest, funcDependencies...) {
		panic(fmt.Sprintf("bad value: could not resolve a dependency from: %#+v", dependency))
	}

	return dest
}

// DependencyResolver func(v reflect.Value, dest *Dependency) bool
// Resolver     DependencyResolver

func resolveDependency(v reflect.Value, disablePayloadAutoBinding bool, dest *Dependency, prevDependencies ...*Dependency) bool {
	return fromDependencyHandler(v, dest) ||
		fromBuiltinValue(v, dest) ||
		fromStructValueOrDependentStructValue(v, disablePayloadAutoBinding, dest, prevDependencies) ||
		fromFunc(v, dest) ||
		len(prevDependencies) > 0 && fromDependentFunc(v, disablePayloadAutoBinding, dest, prevDependencies)
}

func fromDependencyHandler(_ reflect.Value, dest *Dependency) bool {
	// It's already on the desired form, just return it.
	dependency := dest.OriginalValue
	handler, ok := dependency.(DependencyHandler)
	if !ok {
		handler, ok = dependency.(func(*context.Context, *Input) (reflect.Value, error))
		if !ok {
			// It's almost a handler, only the second `Input` argument is missing.
			if h, is := dependency.(func(*context.Context) (reflect.Value, error)); is {
				handler = func(ctx *context.Context, _ *Input) (reflect.Value, error) {
					return h(ctx)
				}
				ok = is
			}
		}
	}
	if !ok {
		return false
	}

	dest.Handle = handler
	return true
}

func fromBuiltinValue(v reflect.Value, dest *Dependency) bool {
	if !isBuiltinValue(v) {
		return false
	}

	// It's just a static builtin value.
	handler := func(*context.Context, *Input) (reflect.Value, error) {
		return v, nil
	}

	dest.DestType = v.Type()
	dest.Static = true
	dest.Handle = handler
	return true
}

func fromStructValue(v reflect.Value, dest *Dependency) bool {
	if !isStructValue(v) {
		return false
	}

	// It's just a static struct value.
	handler := func(*context.Context, *Input) (reflect.Value, error) {
		return v, nil
	}

	dest.DestType = v.Type()
	dest.Static = true
	dest.Handle = handler
	return true
}

func fromStructValueOrDependentStructValue(v reflect.Value, disablePayloadAutoBinding bool, dest *Dependency, prevDependencies []*Dependency) bool {
	if !isStructValue(v) {
		// It's not just a static struct value.
		return false
	}

	if len(prevDependencies) == 0 || !dest.StructDependents { // As a non depedent struct.
		// We must make this check so we can avoid the auto-filling of
		// the dependencies from Iris builtin dependencies.
		return fromStructValue(v, dest)
	}

	// Check if it's a builtin dependency (e.g an MVC Application (see mvc.go#newApp)),
	// if it's and registered without a Dependency wrapper, like the rest builtin dependencies,
	// then do NOT try to resolve its fields.
	//
	// Although EnableStructDependents is false by default, we must check if it's a builtin dependency for any case.
	if strings.HasPrefix(indirectType(v.Type()).PkgPath(), "github.com/kataras/iris/v12") {
		return fromStructValue(v, dest)
	}

	bindings := getBindingsForStruct(v, prevDependencies, false, disablePayloadAutoBinding, dest.StructDependents, DefaultDependencyMatcher, -1, nil)
	if len(bindings) == 0 {
		return fromStructValue(v, dest) // same as above.
	}

	// As a depedent struct, however we may need to resolve its dependencies first
	// so we can decide if it's really a depedent struct or not.
	var (
		handler = func(*context.Context, *Input) (reflect.Value, error) {
			return v, nil
		}
		isStatic = true
	)

	for _, binding := range bindings {
		if !binding.Dependency.Static {
			isStatic = false
			break
		}
	}

	handler = func(ctx *context.Context, _ *Input) (reflect.Value, error) { // Called once per dependency on build-time if the dependency is static.
		elem := v
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		for _, binding := range bindings {
			field := elem.FieldByIndex(binding.Input.StructFieldIndex)
			if !field.CanSet() || !field.IsZero() {
				continue // already set.
			}
			// if !binding.Dependency.Match(field.Type()) { A check already happen in getBindingsForStruct.
			// 	continue
			// }

			input, err := binding.Dependency.Handle(ctx, binding.Input)
			if err != nil {
				if err == ErrSeeOther {
					continue
				}

				return emptyValue, err
			}

			// fmt.Printf("binding %s to %#+v\n", field.String(), input)

			field.Set(input)
		}

		return v, nil
	}

	dest.DestType = v.Type()
	dest.Static = isStatic
	dest.Handle = handler
	return true
}

func fromFunc(v reflect.Value, dest *Dependency) bool {
	if !isFunc(v) {
		return false
	}

	typ := v.Type()
	numIn := typ.NumIn()
	numOut := typ.NumOut()

	if numIn == 0 {
		// it's an empty function, that must return a structure.
		if numOut != 1 {
			firstOutType := indirectType(typ.Out(0))
			if firstOutType.Kind() != reflect.Struct && firstOutType.Kind() != reflect.Interface {
				panic(fmt.Sprintf("bad value: function has zero inputs: empty input function must output a single value but got: length=%v, type[0]=%s", numOut, firstOutType.String()))
			}
		}

		// fallback to structure.
		return fromStructValue(v.Call(nil)[0], dest)
	}

	if numOut == 0 {
		panic("bad value: function has zero outputs")
	}

	if numOut == 2 && !isError(typ.Out(1)) {
		panic("bad value: second output should be an error")
	}

	if numOut > 2 {
		// - at least one output value
		// - maximum of two output values
		// - second output value should be a type of error.
		panic(fmt.Sprintf("bad value: function has invalid number of output arguments: %v", numOut))
	}

	var handler DependencyHandler

	firstIsContext := isContext(typ.In(0))
	secondIsInput := numIn == 2 && typ.In(1) == inputTyp
	onlyContext := (numIn == 1 && firstIsContext) || (numIn == 2 && firstIsContext && typ.IsVariadic())

	if onlyContext || (firstIsContext && secondIsInput) {
		handler = handlerFromFunc(v, typ)
	}

	if handler == nil {
		return false
	}

	dest.DestType = typ.Out(0)
	dest.Handle = handler
	return true
}

func handlerFromFunc(v reflect.Value, typ reflect.Type) DependencyHandler {
	// * func(Context, *Input) <T>, func(Context) <T>
	// * func(Context) <T>, func(Context) <T>
	// * func(Context, *Input) <T>, func(Context) (<T>, error)
	// * func(Context) <T>, func(Context) (<T>, error)

	hasErrorOut := typ.NumOut() == 2 // if two, always an error type here.
	hasInputIn := typ.NumIn() == 2 && typ.In(1) == inputTyp

	return func(ctx *context.Context, input *Input) (reflect.Value, error) {
		inputs := ctx.ReflectValue()
		if hasInputIn {
			inputs = append(inputs, input.selfValue)
		}
		results := v.Call(inputs)
		if hasErrorOut {
			return results[0], toError(results[1])
		}

		return results[0], nil
	}
}

func fromDependentFunc(v reflect.Value, disablePayloadAutoBinding bool, dest *Dependency, funcDependencies []*Dependency) bool {
	// * func(<D>...) returns <T>
	// * func(<D>...) returns error
	// * func(<D>...) returns <T>, error

	typ := v.Type()
	if !isFunc(v) {
		return false
	}

	bindings := getBindingsForFunc(v, funcDependencies, disablePayloadAutoBinding, -1 /* parameter bindings are disabled for depent dependencies */)

	numIn := typ.NumIn()
	numOut := typ.NumOut()

	// d1 = Logger
	// d2 = func(Logger) S1
	// d2 should be static: it accepts dependencies that are static
	// (note: we don't check the output argument(s) of this dependency).
	if numIn == len(bindings) {
		static := true
		for _, b := range bindings {
			if !b.Dependency.Static && b.Dependency.Match(typ.In(b.Input.Index)) {
				static = false
				break
			}
		}

		if static {
			dest.Static = static
		}
	}

	firstOutIsError := numOut == 1 && isError(typ.Out(0))
	secondOutIsError := numOut == 2 && isError(typ.Out(1))

	handler := func(ctx *context.Context, _ *Input) (reflect.Value, error) {
		inputs := make([]reflect.Value, numIn)

		for _, binding := range bindings {
			input, err := binding.Dependency.Handle(ctx, binding.Input)
			if err != nil {
				if err == ErrSeeOther {
					continue
				}

				return emptyValue, err
			}

			inputs[binding.Input.Index] = input
		}

		outputs := v.Call(inputs)
		if firstOutIsError {
			return emptyValue, toError(outputs[0])
		} else if secondOutIsError {
			return outputs[0], toError(outputs[1])
		}
		return outputs[0], nil
	}

	dest.DestType = typ.Out(0)
	dest.Handle = handler

	return true
}
