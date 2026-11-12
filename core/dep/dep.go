package dep

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/kataras/iris/v14/core/pkg/reflex"
) // It could live on core/pkg/dep as well.

// Container is a dependency injection container.
// It is used to register and resolve dependencies.
// The dependencies are resolved in reverse order they are registered, starting from last to the first.
// See `RegisterDependency` and `Invoke` methods for more information.
type Container struct {
	dependencies []*Dependency // holds any registered dependencies.
}

// NewContainer creates a new DI Container.
// The container is used to register and resolve dependencies.
func NewContainer() *Container {
	return &Container{}
}

// RegisterDependency registers a dependency with the container.
// The dependency can be a function, a standard Go value or a pointer to a struct value.
// The dependency can have any number of input and output arguments, but only one error output argument.
// The dependencies are resolved in reverse order they are registered, starting from last to the first.
// The resolved dependencies are passed as input arguments (or fields if struct) to the dep.
// The output arguments (or struct value) are stored and can be used by other dependencies.
// As an exception if dep is request dependency, then it's resolved only when it's called by the caller.
// A request dependency is when a function accepts at least one request type as input argument.
// The request types are: `context.Context`, `iris.Context`, `time.Time` and `*http.Request`,
// plus the routing and controller types registered by the core/router package.
// See `RegisterRequestDependencyType` to add more custom types.
//
// If a dependency is not found, an error is returned.
// If a dependency is found, but the type does not match, an error is returned.
// If a dependency is found, but the value cannot be converted to the type, an error is returned.
func (c *Container) RegisterDependency(deps ...any) error {
	for _, dep := range deps {
		if dep == nil {
			return errors.New("dependency is nil")
		}

		// Check if "dep" is created through NewDependency.
		// It's not a common way to registere dependencies, but let's give the opportunity to the end-developer to use a custom resolver too.
		dependency, ok := dep.(*Dependency)
		if !ok {
			// If it's not let's create a new dependency with the given value.
			var err error
			dependency, err = NewDependency(dep, c.dependencies)
			if err != nil {
				return err
			}
		}

		c.dependencies = append(c.dependencies, dependency)
	}

	return nil
}

// ResolveValue resolves a dependency by type.
// If the type cannot be resolved by the registered dependencies, it returns false.
// If the dependency is found, it returns true and the value of the dependency.
//
// Prefer the typed `Resolve` method when the type is known at compile-time;
// ResolveValue remains for callers that only hold a reflect.Type or an untyped value.
func (c *Container) ResolveValue(typeOrValue any) (reflect.Value, bool) {
	typ := reflex.TypeOf(typeOrValue)

	for _, dep := range slices.Backward(c.dependencies) {
		if output, ok := dep.Resolve(typ); ok {
			return output, true
		}
	}

	return reflect.Value{}, false
}

// Resolve resolves the dependency registered for the type T and returns it as T.
// It reports false when no registered dependency output matches T, when the
// resolved value cannot be converted to an interface or when it does not
// assert to T. Use `ResolveValue` for reflect-typed access.
func (c *Container) Resolve[T any]() (T, bool) {
	var zero T

	output, ok := c.ResolveValue(reflect.TypeFor[T]())
	if !ok {
		return zero, false
	}

	if !output.CanInterface() {
		return zero, false
	}

	value, ok := output.Interface().(T)
	if !ok {
		return zero, false
	}

	return value, true
}

// InvokeReturningFirst same as `Invoke` but it returns only the first result.
func (c *Container) InvokeReturningFirst(value any) (any, error) {
	results, err := c.Invoke(value)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results[0], nil
}

// Invoke resolves and injects the given value.
// The value can be a function, a standard Go value or a pointer to a struct value.
// The value is resolved by the registered dependencies.
// If the value is a function, the input arguments are resolved by the registered dependencies.
// If the value is a standard Go value, it is returned as is.
// If the value is a pointer to a struct value, the struct value is resolved by the registered dependencies.
// If no matched dependencies are found, an error is returned.
func (c *Container) Invoke(value any) ([]any, error) {
	outputs, err := c.invoke(value)
	if err != nil {
		return nil, err
	}

	if len(outputs) == 0 {
		return nil, nil
	}

	results := make([]any, 0, len(outputs))
	for _, output := range outputs {
		if !output.CanInterface() {
			return nil, fmt.Errorf("value of type %s cannot be converted to interface", output.Type().String())
		}

		results = append(results, output.Interface())
	}

	output := outputs[0]
	if !output.CanInterface() {
		return nil, fmt.Errorf("value of type %s cannot be converted to interface", output.Type().String())
	}

	return results, nil
}

func (c *Container) invoke(value any) ([]reflect.Value, error) {
	if value == nil {
		return nil, errors.New("value is nil")
	}

	val := reflex.ValueOf(value)

	f, err := newFunctionDependency(val, c.dependencies...)
	if err != nil {
		return nil, err
	}

	return f.outputValues, nil
}

// BindStructFields binds the fields of the given struct value.
// The struct value's fields are resolved by the registered dependencies.
// If a field is not found, an error is returned.
func (c *Container) BindStructFields(val reflect.Value) error {
	typ := val.Type()

	// unsetUnexportedFields := 0
	if indirectType := reflex.IndirectType(typ); indirectType.Kind() == reflect.Struct {
		// If it's a structure, check its exported and zero fields that need to be binded as input arguments of the final function dependency.
		structFieldSetter := func(field reflect.StructField) (bool, error) { // Bind any fields here, so the end func just returns its final value.
			elem := val
			if elem.Kind() == reflect.Pointer {
				elem = val.Elem()
			}

			if field.Tag.Get("dep") == "-" || field.Tag.Get("mvc") == "-" {
				return true, nil
			}

			fieldValue := elem.FieldByIndex(field.Index)
			if !fieldValue.IsZero() {
				return true, nil // it's non-zero, already filled-up, don't do anything.
			}

			wasSet := len(c.dependencies) == 0                    // if zero dependencies, then we don't have to fire an error on not-set.
			for _, dep := range slices.Backward(c.dependencies) { // reverse, from last to first.
				if output, ok := dep.Resolve(field.Type); ok {
					// Pointer to a field using unsafe:
					// fieldPtr := (*int)(unsafe.Pointer(uintptr(structPtr) + field.Offset))
					// *fieldPtr = 42
					fieldValue.Set(output)
					// fmt.Printf("field %s of type %s was binded with %s\n", field.Name, field.Type, output.Type())
					wasSet = true
					break
				}
			}

			// Check if all exported fields of that structure are filled up, if not, fire an error here so the end-developer knows that it may fail on serve-time.
			if !wasSet {
				return true, fmt.Errorf("%w: %s: field %q of type %s was not binded", ErrDependencyNotFound, typ.String(), field.Name, field.Type)
			}

			return true, nil // always return true as we don't want to include any fields.
		}

		lookupOpts := &reflex.LookupFieldsOptions{
			SkipUnexported:  true,
			SkipStructTypes: []reflect.Type{reflex.TimeType},
			SkipField:       structFieldSetter, // just walk, ignore all, we don't care about the result of the lookup here, we just want to loop over the fields.
		}
		_, err := reflex.LookupFieldsWithOptions(lookupOpts, indirectType)
		return err
	}

	return nil
}

// Clone creates a new container with the same dependencies.
func (c *Container) Clone() *Container {
	clone := NewContainer()
	clone.dependencies = slices.Clone(c.dependencies)
	return clone
}

// Dependencies returns the dependencies of the container.
func (c *Container) Dependencies() []*Dependency {
	return c.dependencies
}

// VisitSingletons calls fn once for every resolved singleton output value of
// the registered dependencies, in registration order. Request-scoped
// dependencies are skipped: their values do not exist before a request.
// Comparable values (pointers included) are deduplicated, so a value that is
// registered directly and also returned by a constructor is visited once.
//
// The type parameter T filters the visit: only values whose dynamic type
// asserts to T invoke fn. Pass a func(v any) to visit every singleton
// (T is inferred as any), or an interface-typed T to visit only the
// singletons implementing it.
func (c *Container) VisitSingletons[T any](fn func(T)) {
	visited := make(map[any]struct{})

	for _, d := range c.dependencies {
		resolver, ok := d.DependencyResolver.(*functionDependency)
		if !ok || resolver.dependencyType != DependencyTypeSingleton {
			continue
		}

		for _, out := range resolver.outputValues {
			if !out.IsValid() || !out.CanInterface() {
				continue
			}

			value := out.Interface()
			if value == nil {
				continue
			}

			if reflect.TypeOf(value).Comparable() {
				if _, seen := visited[value]; seen {
					continue
				}
				visited[value] = struct{}{}
			}

			typed, ok := value.(T)
			if !ok {
				continue
			}

			fn(typed)
		}
	}
}

// DependencyType represents the type of a dependency.
// It is used to determine how a dependency is resolved.
// A dependency can be a singleton or a request dependency.
type DependencyType uint8

const (
	// DependencyTypeSingleton is a dependency that is created once and shared
	// across all other dependencies and requests.
	DependencyTypeSingleton DependencyType = iota
	// DependencyTypeRequest is a dependency that is created once per request.
	// This is useful for creating a new instance of a dependency for each
	// request, but sharing that instance across all other dependencies.
	DependencyTypeRequest
)

// DependencyResolver is an interface that resolves a dependency by type.
// It is used to resolve a dependency by given type(s).
type DependencyResolver interface {
	// DependencyType returns the type of the dependency.
	DependencyType() DependencyType
	// Resolve resolves a dependency by type.
	Resolve(toInput reflect.Type) (reflect.Value, bool)
}

// Dependency represents a dependency.
// It holds the original value of the dependency and its resolver.
type Dependency struct {
	// Source is the original value of the dependency.
	Source reflect.Value
	// DependencyResolver is the resolver of the dependency.
	// It is used to resolve the dependency by given type(s).
	DependencyResolver
}

// NewDependency creates a new dependency with the given value and dependencies.
// The value can be a function, a standard Go value or a pointer to a struct value.
// The resolved dependencies are passed as input arguments (or fields if struct) to the dep.
// The output arguments (or struct value) are stored and can be used by other dependencies.
func NewDependency(dep any, dependencies []*Dependency) (*Dependency, error) {
	if dependency, ok := dep.(*Dependency); ok {
		return dependency, nil
	}

	val := reflex.ValueOf(dep)

	resolver, err := newFunctionDependency(val, dependencies...)
	if err != nil {
		return nil, err
	}

	dependency := &Dependency{
		Source:             val,
		DependencyResolver: resolver,
	}
	return dependency, nil
}

type (
	// funcTypeValue represents a map of function type to value.
	// A cache of input arguments and output arguments dependencies
	// that are used as dependencies for the next dependencies in the chain.
	funcTypeValue        map[reflect.Type]reflect.Value
	funcTypeValueElement struct {
		Type  reflect.Type
		Value reflect.Value
	}
	// functionDependency represents a function dependency.
	// It implements the DependencyResolver interface.
	functionDependency struct {
		fn             reflect.Value
		fnType         reflect.Type
		dependencyType DependencyType

		inputs                   funcTypeValue
		outputs                  funcTypeValue
		outputsSortedByMethodLen []*funcTypeValueElement

		outputValues []reflect.Value
	}
)

var _ DependencyResolver = (*functionDependency)(nil)

// ErrDependencyNotFound is returned when a dependency is not found.
var ErrDependencyNotFound = errors.New("dependency not found")

// newFunctionDependency creates a new function dependency.
// It resolves the input arguments of the function or the struct's fields based on the given dependencies.
func newFunctionDependency[T DependencyResolver](val reflect.Value, dependencies ...T) (*functionDependency, error) {
	typ := val.Type()

	if !reflex.IsFunc(typ) { // if it's not a function but a std value or struct value, then create a function value in runtime and call newFunctionDependency with that.
		return convertStructToFunctionDependency(val, typ, dependencies...)
	}

	inputValues, inputs, err := resolveFuncInputs(typ, dependencies...)
	if err != nil {
		file, line := reflex.FuncFileLine(val)
		return nil, fmt.Errorf("%w: %s:%d", err, file, line)
	}

	wasRequestDep := inputValues == nil && inputs == nil && err == nil
	if wasRequestDep { // func input or struct field as always - structs are converted to functions (see convertStructToFunctionDependency above).
		return convertRequestFuncToFunctionDependency(val, typ, dependencies...)
	}

	outputValues, outputs, err := resolveFuncOutputs(val, typ, inputValues) // if the output is a type of error and it's not nil, then this "err" will be returned.
	if err != nil {
		return nil, err //No: return the raw impementation's error! fmt.Errorf("%w: %s:%d", err, file, line)
	}

	// create a sorted list of the output values based on the number of methods that the type has.
	elements := make([]*funcTypeValueElement, 0, len(outputValues))
	for _, out := range outputValues {
		outType := out.Type()

		// store only structs. NOTE: Or No? e.g. a custom type that is not a struct but has methods.
		// if reflex.IndirectType(outType).Kind() != reflect.Struct {
		// 	continue
		// }

		element := &funcTypeValueElement{
			Type:  outType,
			Value: out,
		}
		elements = append(elements, element)
	}
	slices.SortStableFunc(elements, func(a, b *funcTypeValueElement) int {
		return cmp.Compare(b.Type.NumMethod(), a.Type.NumMethod()) // descending. NOTE: maybe exclude the methods that are not exported?
	})

	resolver := &functionDependency{
		fn:             val,
		fnType:         typ,
		dependencyType: DependencyTypeSingleton,

		inputs:                   inputs,
		outputs:                  outputs,
		outputValues:             outputValues,
		outputsSortedByMethodLen: elements,
	}

	return resolver, nil
}

// convertStructToFunctionDependency converts a struct value to a function dependency.
func convertStructToFunctionDependency[T DependencyResolver](val reflect.Value, typ reflect.Type, dependencies ...T) (*functionDependency, error) {
	/* Don't bind struct fields. End-developer should use functions to bind them instead.
	if indirectType := reflex.IndirectType(typ); indirectType.Kind() == reflect.Struct {
		// If it's a structure, check its exported and zero fields that need to be binded as input arguments of the final function dependency.
		structFieldSetter := func(field reflect.StructField) (bool, error) { // Bind any fields here, so the end func just returns its final value.
			elem := val
			if elem.Kind() == reflect.Pointer {
				elem = val.Elem()
			}

			fieldValue := elem.FieldByIndex(field.Index)
			if !fieldValue.IsZero() {
				return true, nil // it's non-zero, already filled-up, don't do anything.
			}

			wasSet := false || len(dependencies) == 0     // if zero dependencies, then we don't have to fire an error on not-set.
			for j := len(dependencies) - 1; j >= 0; j-- { // reverse, from last to first.
				dep := dependencies[j]
				if output, ok := dep.Resolve(field.Type); ok {
					// Pointer to a field using unsafe:
					// fieldPtr := (*int)(unsafe.Pointer(uintptr(structPtr) + field.Offset))
					// *fieldPtr = 42
					fieldValue.Set(output)
					wasSet = true
					break
				}
			}

			// Check if all exported fields of that structure are filled up, if not, fire an error here so the end-developer knows that it may fail on serve-time.
			if !wasSet {
				file, line := reflex.FuncFileLine(val)
				return true, fmt.Errorf("%w: %s: field %s of type %s is not binded: %s:%d", ErrDependencyNotFound, typ.String(), field.Name, field.Type, file, line)
			}

			return true, nil // always return true as we don't want to include any fields.
		}

		lookupOpts := &reflex.LookupFieldsOptions{
			SkipUnexported:  true,
			SkipStructTypes: []reflect.Type{reflex.TimeType},
			SkipField:       structFieldSetter, // just walk, ignore all, we don't care about the result of the lookup here, we just want to loop over the fields.
		}
		_, err := reflex.LookupFieldsWithOptions(lookupOpts, indirectType)
		if err != nil {
			return nil, err
		}
	}
	*/

	// Else is a standard type or a pointer to a standard type. Just return this on the final function dependency.
	// Do not resolve struct fields automatically, let the end-developer use functions to bind them instead (this code is used for dependency injection of dependencies as well).
	funcType := reflect.FuncOf([]reflect.Type{}, []reflect.Type{typ}, false)
	// Use MakeFunc to create a function that returns value.
	makeFunc := reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{val}
	})

	return newFunctionDependency(makeFunc, dependencies...)
}

// convertRequestFuncToFunctionDependency converts a function value to a request function dependency.
func convertRequestFuncToFunctionDependency[T DependencyResolver](val reflect.Value, typ reflect.Type, dependencies ...T) (*functionDependency, error) {
	// Create a function that returns the function value itself, bind it as it's.
	// The only functions that are allowed to be binded as request dependencies are the
	// ones that are called by the caller, its input arguments are not resolved by previous dependencies and
	// its output arguments are not going to be binded to any next dependency.
	funcType := reflect.FuncOf([]reflect.Type{}, []reflect.Type{typ}, false)
	makeFunc := reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{val}
	})

	d, err := newFunctionDependency(makeFunc, dependencies...)
	if err != nil {
		return nil, err
	}
	d.dependencyType = DependencyTypeRequest
	return d, nil
}

func resolveFuncInputs[T DependencyResolver](typ reflect.Type, dependencies ...T) ([]reflect.Value, funcTypeValue, error) {
	numIn := typ.NumIn()
	callIn := make([]reflect.Value, 0, numIn)

	inputs := make(funcTypeValue, numIn)

	resolvedDependencyIdx := -1 // in order to not resolve the same one if more than one of the same input type is expected.
	for i := range numIn {
		in := typ.In(i)

		if isRequestInputDependency(in) {
			// just don't input and output values but don't return an error, it's probably a function which accepts request-types,
			// so let them unbinded, they'll probably be used as just a field->func in.
			return nil, nil, nil
		} else {
			// Check again for request dependency but this time inside its own struct fields (if "in" is a struct)
			// e.g func(*service) string which *service contains a field of type *Context.
			// We don't want it to go over the below !inResolved condition later on and try to resolve the
			// struct automatically as it depends on request-scoped struct type dependencies and not registered funcs.
			if reflex.IsStructPtr(in) {
				requestScopedFields, _ := reflex.LookupFieldsWithOptions(&reflex.LookupFieldsOptions{
					SkipUnexported: true,
					SkipField: func(f reflect.StructField) (bool, error) {
						// Return false to skip non-request scoped. Collect all request inputs, if they are > 0 then this is a request dependency.
						return !isRequestInputDependency(f.Type), nil
					},
				}, reflex.IndirectType(in))

				isRequestInput := len(requestScopedFields) > 0
				if isRequestInput {
					// see context/dep_test.go#TestRegisterRequestDependencyType_Struct.
					return nil, nil, fmt.Errorf("%s: request-scoped field: %s found inside a struct, please use a function for request dependencies instead", typ.String(), in.String())
				}
			}
		}

		// reverse loop, from end to start in order to count only the last dependencies.
		inResolved := false
		for j, dependency := range slices.Backward(dependencies) {
			if j == resolvedDependencyIdx {
				continue
			}

			// The explicit interface conversion is load-bearing: calling
			// Resolve directly on the type parameter goes through the generic
			// dictionary, and when T satisfies DependencyResolver via an
			// embedded interface (*Dependency does), the Go 1.27.0 linker
			// prunes the promoted wrapper method and the call dies with
			// "fatal error: unreachable method called. linker bug?".
			// Converting to the interface first keeps the wrapper reachable.
			// Still present in Go 1.27.1; golang/go#81353 carries the fix to
			// Go 1.27.2. Drop the conversion once go.mod requires 1.27.2.
			var resolver DependencyResolver = dependency
			if result, ok := resolver.Resolve(in); ok {
				resolvedDependencyIdx = j

				callIn = append(callIn, result)

				resultType := result.Type()
				inputs[resultType] = result
				inResolved = true
				break
			}
		}

		if !inResolved { // if not resolved by any known dependencies, try to make a new value for it and resolve it automatically. This can take more improvements, e.g. options.
			if reflex.IsStructPtr(in) {
				// It's a pointer to struct and it's not resolved by any known dependencies,
				// try to resolve its fields and create a new value and bind it automatically.

				// Create a new instance of the input and resolve its fields dependencies through newFunctionDependency.
				result := reflect.New(in.Elem())
				_, err := newFunctionDependency(result, dependencies...)
				if err != nil {
					return nil, nil, fmt.Errorf("%s: missing dependency, automatic registration failed : %w", typ.String(), err) // return the error by the 1st type that it tried to resolve.
				}

				// No need to resolve it again, "result" is already resolved.
				callIn = append(callIn, result)
				inputs[in] = result

				inResolved = true // for now here it's not used, but it's good to have it for future use.
				continue
			}
		}
	}

	if numIn != len(callIn) {
		return nil, nil, fmt.Errorf("%w: %s: expected %d input but got %d", ErrDependencyNotFound, typ.String(), numIn, len(callIn))
	}

	return callIn, inputs, nil
}

func resolveFuncOutputs(val reflect.Value, typ reflect.Type, inputValues []reflect.Value) ([]reflect.Value, funcTypeValue, error) {
	numOut := typ.NumOut()

	outputValues := make([]reflect.Value, 0, numOut)
	outputs := make(funcTypeValue, numOut)

	callOut := val.Call(inputValues)
	for _, out := range callOut {
		outputType := out.Type()
		if reflex.IsError(outputType) {
			// Return the error as it is, don't store it.
			if !out.IsNil() {
				return nil, nil, out.Interface().(error)
			}

			continue // do not put the error to the output values.
		}
		// store an ordered list of the raw output.
		outputValues = append(outputValues, out)
		outputs[out.Type()] = out
	}

	return outputValues, outputs, nil
}

// DependencyType returns the type of the dependency.
// It completes the DependencyResolver interface.
func (d *functionDependency) DependencyType() DependencyType {
	return d.dependencyType
}

// Resolve resolves a dependency by type.
// It completes the DependencyResolver interface.
func (d *functionDependency) Resolve(toInput reflect.Type) (reflect.Value, bool) {
	// if strings.Contains(toInput.String(), "CustomControllerAsService") {
	// 	for _, out := range d.outputs {
	// 		fmt.Println(out.Type().String())
	// 	}
	// }

	result, ok := d.outputs[toInput]
	if !ok {
		// If not found by exact type, then try to find by interface type.
		if toInput.Kind() == reflect.Interface {
			// Search by the longest method set.
			for _, element := range d.outputsSortedByMethodLen {
				if element.Type.Implements(toInput) {
					return element.Value, true
				}
			}
			// The above ^ is safer as it will not return a value that may be compatible but another one may be even more compatible with the interface.
			// for resultType, result := range d.outputs {
			// 	if resultType.Implements(toInput) {
			// 		return result, true
			// 	}
			// }
		}

		return reflect.Value{}, false
	}

	return result, true
}
