// This file exists for its name.
//
// It holds Context.Writef, and it is named so that it sorts before every file
// of this package that declares a generic *Context method: today those are
// context_bind.go, context_bind_update.go, context_request.go and page.go, and
// the first of them, context_bind.go, is the one that matters. The Go type
// checker orders a named type's methods by the file they are declared in, and
// file names are sorted, so this file's name decides that Writef comes first.
//
// Why that matters: go vet's printf analyzer records "Writef is a printf
// wrapper" as a fact keyed by objectpath, and objectpath addresses a method by
// its index in the type's method list. On go1.27.0 a generic method declared
// before Writef makes that index disagree between the package that writes the
// fact and the package that reads it back, so the fact lands on some other
// method. Two things then go wrong at once, both confirmed from a scratch
// module that imports Iris:
//
//   - ctx.Writef("%d", "x") is not reported to any user of the framework,
//     although it is a real bug;
//   - whichever unrelated *Context method happens to occupy the shifted slot is
//     treated as a printf function, so a plain call like
//     ctx.QueryInt("limit", iris.QueryRequired()) is reported as "call has
//     arguments but no formatting directives".
//
// With Writef declared ahead of every generic method the two indexes agree
// again and both diagnostics come out right. So: do not move this declaration
// back into context_response.go, and do not rename this file to anything that
// sorts after context_bind.go. Either one silently removes format checking of
// ctx.Writef from every program that imports Iris, and hands an unrelated
// method a false positive its author cannot explain. A minimal reproduction is
// three files with no Iris in them: a type with a generic method, a Writef, and
// a consumer package.
//
// The upstream bug is golang/go#81306 (export data loses method order). Go
// 1.27.1 still has it; the fix is backported to Go 1.27.2. Once go.mod requires
// 1.27.2, Writef can move back and this file and its test can go.
package context

import "fmt"

// Writef formats according to a format specifier and writes to the response.
//
// Returns the number of bytes written and any write error encountered.
func (ctx *Context) Writef(format string, a ...any) (n int, err error) {
	/* if len(a) == 0 {
	 	return ctx.WriteString(format)
	} ^ No, let it complain about arguments, because go test will do even if the app is running.
	Users should use WriteString instead of (format, args)
	when format may contain go-sprintf reserved chars (e.g. %).*/

	return fmt.Fprintf(ctx.writer, format, a...)
}
