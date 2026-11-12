// This file exists for its name.
//
// It holds the generic methods of *Options and ErrorCodeMap, and it is named
// so that it sorts after every other file of this package that declares a
// method on either type: today those are error_code_map.go, map_error_as.go
// and options.go. The Go type checker orders a named type's methods by the
// file they are declared in, and file names are sorted, so this file's name
// decides that the generic methods come last.
//
// Why that matters: go vet and staticcheck record what they learn about a
// method as a fact keyed by objectpath, and objectpath addresses a method by
// its index in the type's method list. On this toolchain a generic method
// declared before another method makes that index disagree between the
// package that writes the fact and the package that reads it back, so the
// fact lands on some other method of the type, in every program that imports
// this package. With the generic methods declared after all the others, no
// ordinary method has a generic one in front of it and the indexes agree.
//
// So: do not add an ordinary method of *Options or ErrorCodeMap to this file,
// do not declare a generic method of either type anywhere else, and do not
// rename this file to anything that sorts before options.go.
// TestOptionsGenericMethodsDeclaredLast fails on all three.
//
// The upstream bug is golang/go#81306 (export data loses method order). The
// context package carries the same workaround the other way round, see
// context/before_generic_methods_writef.go. Once go.mod requires a Go release
// with the fix, these methods can move next to MapErrorContains and this file
// and its test can go.
package errors

import "github.com/kataras/iris/v14/context"

// MapErrorAs registers a typed error mapping: an E is looked for anywhere in
// the error tree, as the standard errors.AsType does, and render turns the
// value found into the *Error to send. A nil render result means "not mine",
// so the next mapping gets a turn, exactly like MapErrorFunc returning nil.
//
// It chains like every other mapping method, and E is inferred from render:
//
//	opts := errors.NewOptions().
//		MapErrors(errors.NotFound, sql.ErrNoRows).
//		MapErrorAs(errors.ResourceExhausted, func(e *QuotaError) *errors.Error {
//			return &errors.Error{
//				Message: "quota exceeded",
//				Detail:  fmt.Sprintf("the limit is %d", e.Limit),
//			}
//		})
//
// A nil render classifies without transforming, the same as MapErrorTypes.
// Then E has to be named, because there is nothing left to infer it from:
//
//	opts.MapErrorAs[*QuotaError](errors.ResourceExhausted, nil)
//
// Prefer this over MapErrorContains whenever the error carries a type: the
// fields of E are available to the message and nothing depends on the wording
// of a driver's text.
func (opts *Options) MapErrorAs[E error](code ErrorCodeName, render func(E) *Error) *Options {
	if opts.codeMap == nil {
		opts.codeMap = make(ErrorCodeMap)
	}
	opts.codeMap.MapErrorAs(code, render)

	return opts
}

// MapErrorAs registers a typed error mapping.
// See Options.MapErrorAs for the full documentation.
func (m ErrorCodeMap) MapErrorAs[E error](code ErrorCodeName, render func(E) *Error) ErrorCodeMap {
	return m.MapErrorFuncCtx(code, func(_ *context.Context, err error) error {
		target, ok := AsType[E](err)
		if !ok {
			return nil
		}

		if render == nil {
			return err // classification only.
		}

		if rendered := render(target); rendered != nil {
			return rendered
		}

		// A typed nil *Error would read as a non-nil error to the caller:
		// return the untyped nil so the mapping falls through.
		return nil
	})
}
