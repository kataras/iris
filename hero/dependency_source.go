package hero

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// Source describes where a dependency is located at the source code itself.
type Source struct {
	File   string
	Line   int
	Caller string
}

func newSource(fn reflect.Value) Source {
	var (
		callerFileName   string
		callerLineNumber int
		callerName       string
	)

	switch fn.Kind() {
	case reflect.Func, reflect.Chan, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Slice:
		pc := fn.Pointer()
		fpc := runtime.FuncForPC(pc)
		if fpc != nil {
			callerFileName, callerLineNumber = fpc.FileLine(pc)
			callerName = fpc.Name()
		}

		fallthrough
	default:
		if callerFileName == "" {
			callerFileName, callerLineNumber = GetCaller()
		}
	}

	wd, _ := os.Getwd()
	if relFile, err := filepath.Rel(wd, callerFileName); err == nil {
		if !strings.HasPrefix(relFile, "..") {
			// Only if it's relative to this path, not parent.
			callerFileName = "./" + relFile
		}
	}

	return Source{
		File:   filepath.ToSlash(callerFileName),
		Line:   callerLineNumber,
		Caller: callerName,
	}
}

func getSource() Source {
	filename, line := GetCaller()
	return Source{
		File: filename,
		Line: line,
	}
}

func (s Source) String() string {
	return fmt.Sprintf("%s:%d", s.File, s.Line)
}

// https://golang.org/doc/go1.9#callersframes
func GetCaller() (string, int) {
	var pcs [32]uintptr
	n := runtime.Callers(4, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		file := frame.File

		if strings.Contains(file, "go/src/runtime/") {
			continue
		}

		// funcName is something like "github.com/kataras/iris.SomeFunc"
		funcName := frame.Function
		if !strings.HasPrefix(funcName, "github.com/kataras/iris/v12") {
			return file, frame.Line
		}

		if !more {
			break
		}
	}

	return "???", 0
}
