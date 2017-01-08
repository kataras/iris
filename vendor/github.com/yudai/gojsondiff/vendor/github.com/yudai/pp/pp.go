package pp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/mattn/go-colorable"
)

var (
	out     io.Writer
	outLock sync.Mutex

	defaultOut = colorable.NewColorableStdout()
)

func init() {
	out = defaultOut
}

// Print prints given arguments.
func Print(a ...interface{}) (n int, err error) {
	return fmt.Fprint(out, formatAll(a)...)
}

// Printf prints a given format.
func Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(out, format, formatAll(a)...)
}

// Println prints given arguments with newline.
func Println(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(out, formatAll(a)...)
}

// Sprint formats given arguemnts and returns the result as string.
func Sprint(a ...interface{}) string {
	return fmt.Sprint(formatAll(a)...)
}

// Sprintf formats with pretty print and returns the result as string.
func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, formatAll(a)...)
}

// Sprintln formats given arguemnts with newline and returns the result as string.
func Sprintln(a ...interface{}) string {
	return fmt.Sprintln(formatAll(a)...)
}

// Fprint prints given arguments to a given writer.
func Fprint(w io.Writer, a ...interface{}) (n int, err error) {
	return fmt.Fprint(w, formatAll(a)...)
}

// Fprintf prints format to a given writer.
func Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(w, format, formatAll(a)...)
}

// Fprintln prints given arguments to a given writer with newline.
func Fprintln(w io.Writer, a ...interface{}) (n int, err error) {
	return fmt.Fprintln(w, formatAll(a)...)
}

// Errorf formats given arguments and returns it as error type.
func Errorf(format string, a ...interface{}) error {
	return errors.New(Sprintf(format, a...))
}

// Fatal prints given arguments and finishes execution with exit status 1.
func Fatal(a ...interface{}) {
	fmt.Fprint(out, formatAll(a)...)
	os.Exit(1)
}

// Fatalf prints a given format and finishes execution with exit status 1.
func Fatalf(format string, a ...interface{}) {
	fmt.Fprintf(out, format, formatAll(a)...)
	os.Exit(1)
}

// Fatalln prints given arguments with newline and finishes execution with exit status 1.
func Fatalln(a ...interface{}) {
	fmt.Fprintln(out, formatAll(a)...)
	os.Exit(1)
}

// Change Print* functions' output to a given writer.
// For example, you can limit output by ENV.
//
//	func init() {
//		if os.Getenv("DEBUG") == "" {
//			pp.SetDefaultOutput(ioutil.Discard)
//		}
//	}
func SetDefaultOutput(o io.Writer) {
	outLock.Lock()
	out = o
	outLock.Unlock()
}

// GetDefaultOutput returns pp's default output.
func GetDefaultOutput() io.Writer {
	return out
}

// Change Print* functions' output to default one.
func ResetDefaultOutput() {
	outLock.Lock()
	out = defaultOut
	outLock.Unlock()
}

func formatAll(objects []interface{}) []interface{} {
	results := []interface{}{}
	for _, object := range objects {
		results = append(results, format(object))
	}
	return results
}
