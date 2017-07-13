package errors

import (
	"sync"
)

// StackError contains the Stack method.
type StackError interface {
	Stack() []Error
	Error() string
}

// PrintAndReturnErrors prints the "err" to the given "printer",
// printer will be called multiple times if the "err" is a StackError, where it contains more than one error.
func PrintAndReturnErrors(err error, printer func(string, ...interface{})) error {
	if err == nil || err.Error() == "" {
		return nil
	}

	if stackErr, ok := err.(StackError); ok {
		if len(stackErr.Stack()) == 0 {
			return nil
		}

		stack := stackErr.Stack()

		for _, e := range stack {
			if e.HasStack() {
				for _, es := range e.Stack {
					printer("%v", es)
				}
				continue
			}
			printer("%v", e)
		}

		return stackErr
	}

	printer("%v", err)
	return err
}

// Reporter is a helper structure which can
// stack errors and prints them to a printer of func(string).
type Reporter struct {
	mu      sync.Mutex
	wrapper Error
}

// NewReporter returns a new empty error reporter.
func NewReporter() *Reporter {
	return &Reporter{wrapper: New("")}
}

// AddErr adds an error to the error stack.
// if "err" is a StackError then
// each of these errors will be printed as individual.
func (r *Reporter) AddErr(err error) {
	if err == nil {
		return
	}

	if stackErr, ok := err.(StackError); ok {
		r.addStack(stackErr.Stack())
		return
	}

	r.mu.Lock()
	r.wrapper = r.wrapper.AppendErr(err)
	r.mu.Unlock()
}

// Add adds a formatted message as an error to the error stack.
func (r *Reporter) Add(format string, a ...interface{}) {
	//  usually used as:  "module: %v", err so
	// check if the first argument is error and if that error is empty then don't add it.
	if len(a) > 0 {
		f := a[0]
		if e, ok := f.(interface {
			Error() string
		}); ok {
			if e.Error() == "" {
				return
			}
		}
	}

	r.mu.Lock()
	r.wrapper = r.wrapper.Append(format, a...)
	r.mu.Unlock()
}

// Describe same as `Add` but if "err" is nil then it does nothing.
func (r *Reporter) Describe(format string, err error) {
	if err == nil {
		return
	}
	if stackErr, ok := err.(StackError); ok {
		r.addStack(stackErr.Stack())
		return
	}

	r.Add(format, err)
}

// PrintStack prints all the errors to the given "printer".
// Returns itself in order to be used as printer and return the full error in the same time.
func (r *Reporter) PrintStack(printer func(string, ...interface{})) error {
	return PrintAndReturnErrors(r, printer)
}

// Stack returns the list of the errors in the stack.
func (r *Reporter) Stack() []Error {
	return r.wrapper.Stack
}

func (r *Reporter) addStack(stack []Error) {
	for _, e := range stack {
		if e.Error() == "" {
			continue
		}
		r.mu.Lock()
		r.wrapper = r.wrapper.AppendErr(e)
		r.mu.Unlock()
	}
}

// Error implements the error, returns the full error string.
func (r *Reporter) Error() string {
	return r.wrapper.Error()
}

// Return returns nil if the error is empty, otherwise returns the full error.
func (r *Reporter) Return() error {
	if r.Error() == "" {
		return nil
	}

	return r
}
