package hero

import (
	"fmt"

	"reflect"

	"github.com/kataras/iris/v12/context"
)

type (
	// DependencyHandler is the native function declaration which implementors should return a value match to an input.
	DependencyHandler func(ctx *context.Context, input *Input) (reflect.Value, error)
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
		// If true then input and dependnecy DestType should be indedical,
		// not just assiginable to each other.
		// Example of use case: depenendency like time.Time that we want to be bindable
		// only to time.Time inputs and not to a service with a `String() string` method that time.Time struct implements too.
		Explicit bool
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
func NewDependency(dependency interface{}, funcDependencies ...*Dependency) *Dependency {
	if dependency == nil {
		panic(fmt.Sprintf("bad value: nil: %T", dependency))
	}

	if d, ok := dependency.(*Dependency); ok {
		// already a *Dependency.
		return d
	}

	v := valueOf(dependency)
	if !goodVal(v) {
		panic(fmt.Sprintf("bad value: %#+v", dependency))
	}

	dest := &Dependency{
		Source:        newSource(v),
		OriginalValue: dependency,
	}

	if !resolveDependency(v, dest, funcDependencies...) {
		panic(fmt.Sprintf("bad value: could not resolve a dependency from: %#+v", dependency))
	}

	return dest
}

// DependencyResolver func(v reflect.Value, dest *Dependency) bool
// Resolver     DependencyResolver

func resolveDependency(v reflect.Value, dest *Dependency, funcDependencies ...*Dependency) bool {
	return fromDependencyHandler(v, dest) ||
		fromStructValue(v, dest) ||
		fromFunc(v, dest) ||
		len(funcDependencies) > 0 && fromDependentFunc(v, dest, funcDependencies)
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

func fromStructValue(v reflect.Value, dest *Dependency) bool {
	if !isFunc(v) {
		// It's just a static value.
		handler := func(*context.Context, *Input) (reflect.Value, error) {
			return v, nil
		}

		dest.DestType = v.Type()
		dest.Static = true
		dest.Handle = handler
		return true
	}

	return false
}

func fromFunc(v reflect.Value, dest *Dependency) bool {
	if !isFunc(v) {
		return false
	}

	typ := v.Type()
	numIn := typ.NumIn()
	numOut := typ.NumOut()

	if numIn == 0 {
		panic("bad value: function has zero inputs")
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

func fromDependentFunc(v reflect.Value, dest *Dependency, funcDependencies []*Dependency) bool {
	// * func(<D>...) returns <T>
	// * func(<D>...) returns error
	// * func(<D>...) returns <T>, error

	typ := v.Type()
	if !isFunc(v) {
		return false
	}

	bindings := getBindingsForFunc(v, funcDependencies, -1 /* parameter bindings are disabled for depent dependencies */)

	numIn := typ.NumIn()
	numOut := typ.NumOut()

	// d1 = Logger
	// d2 = func(Logger) S1
	// d2 should be static: it accepts dependencies that are static
	// (note: we don't check the output argument(s) of this dependnecy).
	if numIn == len(bindings) {
		static := true
		for _, b := range bindings {
			if !b.Dependency.Static && matchDependency(b.Dependency, typ.In(b.Input.Index)) {
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
