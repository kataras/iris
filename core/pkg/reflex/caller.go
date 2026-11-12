package reflex

import (
	"runtime"
	"slices"
	"strconv"
	"strings"
)

const pkgName = "github.com/kataras/iris/v14"

var skipFrameFunctionPrefixes = []string{
	pkgName + "/core/pkg/reflex",
	// Continue here as it will get this method and .err and .logErr methods of this subpackage.
	pkgName + "/rest",
	// To allow getting stack traces that are coming from a wrapped iris handler, e.g.:
	/*
	   func returnHandler() iris.Handler {
	   	return func(ctx iris.Context) {
	   		if err := serviceMethodThatReturnsDBError(); err != nil {
	   			errors.Internal.LogErr(ctx, err)
	   			return
	   		}
	   	}
	   }
	*/
	pkgName + "/context",
}

var skipFrameFileSuffixes = []string{
	// Skip the stack trace of the errors.go file of this package. We need to do it by filename for internal debugging.
	"rest/errors/errors.go",
	"core/pkg/reflex/caller.go",
}

var stopFrameFunctionPrefixes = []string{
	// But stop here, it's probably the routing or other core implementation,
	// because it should already get the end-dev code, we don't need to print the stack trace of the Iris or the std library itself.
	pkgName + "/core",
}

func prefixed(slice []string, s string) bool {
	return slices.ContainsFunc(slice, func(prefix string) bool {
		return strings.HasPrefix(s, prefix)
	})
}

func suffixed(slice []string, s string) bool {
	return slices.ContainsFunc(slice, func(suffix string) bool {
		return strings.HasSuffix(s, suffix)
	})
}

// Frame represents a single frame of a stack trace.
type Frame struct {
	// Function name.
	Function string
	// File name.
	File string
	// Line number.
	Line int
}

func NewFrame(f runtime.Frame) Frame {
	return Frame{
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}

// FileLine returns the file and line number of the frame as a single string separated by a colon.
func (f Frame) FileLine() string {
	if f.File == "" {
		return "???:0"
	}

	return f.File + ":" + strconv.Itoa(f.Line)
}

// String returns the frame as a string in the format "Function (File:Line)".
func (f Frame) String() string {
	return f.Function + " (" + f.FileLine() + ")"
}

// GetStackTrace returns the stack trace of the caller, without the Iris internal frames.
// This is a very very simple implementation.
// See `Context.HandlerName` package-level function too.
func GetStackTrace() []Frame {
	// Create a buffer to hold the stack trace, we use a small buffer here to avoid large stack traces inside the loop,
	// 16 spots before Iris is more than enough, we can change it if requested.
	pcs := make([]uintptr, 16)
	n := runtime.Callers(4, pcs)
	if n == 0 {
		return nil
	}

	frames := make([]Frame, 0, len(pcs))
	callersFrames := runtime.CallersFrames(pcs)

	for {
		callerFrame, more := callersFrames.Next()

		if prefixed(skipFrameFunctionPrefixes, callerFrame.Function) {
			continue
		}

		if suffixed(skipFrameFileSuffixes, callerFrame.File) {
			continue
		}

		if prefixed(stopFrameFunctionPrefixes, callerFrame.Function) {
			return frames
		}

		frame := NewFrame(callerFrame)
		frames = append(frames, frame)

		if !more {
			break
		}
	}

	return frames
}

// GetInternalStackTrace returns all the stack trace of the caller, including Iris components.
func GetInternalStackTrace() []Frame {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(4, pcs)
	if n == 0 {
		return nil
	}

	frames := make([]Frame, 0, len(pcs))
	callersFrames := runtime.CallersFrames(pcs)

	for {
		callerFrame, more := callersFrames.Next()
		frame := NewFrame(callerFrame)
		frames = append(frames, frame)

		if !more {
			break
		}
	}

	return frames
}
