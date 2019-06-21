package router

import (
	"runtime"
	"strings"

	"github.com/kataras/iris/context"
)

/*
	Relative to deprecation:
	- party.go#L138-154
	- deprecated_example_test.go
*/

// https://golang.org/doc/go1.9#callersframes
func getCaller() (string, int) {
	var pcs [32]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()

		if (!strings.Contains(frame.File, "github.com/kataras/iris") ||
			strings.Contains(frame.File, "github.com/kataras/iris/_examples") ||
			strings.Contains(frame.File, "github.com/iris-contrib/examples") ||
			(strings.Contains(frame.File, "github.com/kataras/iris/core/router") && !strings.Contains(frame.File, "deprecated.go"))) &&
			!strings.HasSuffix(frame.Func.Name(), ".getCaller") && !strings.Contains(frame.File, "/go/src/testing") {
			return frame.File, frame.Line
		}

		if !more {
			break
		}
	}

	return "?", 0
}

// StaticWeb is DEPRECATED. Use HandleDir(requestPath, directory) instead.
func (api *APIBuilder) StaticWeb(requestPath string, directory string) *Route {
	file, line := getCaller()
	api.reporter.Add(`StaticWeb is DEPRECATED and it will be removed eventually.
Source: %s:%d
Use .HandleDir("%s", "%s") instead.`, file, line, requestPath, directory)

	return nil
}

// StaticHandler is DEPRECATED.
// Use iris.FileServer(directory, iris.DirOptions{ShowList: true, Gzip: true}) instead.
//
// Example https://github.com/kataras/iris/tree/master/_examples/file-server/basic
func (api *APIBuilder) StaticHandler(directory string, showList bool, gzip bool) context.Handler {
	file, line := getCaller()
	api.reporter.Add(`StaticHandler is DEPRECATED and it will be removed eventually.
Source: %s:%d
Use iris.FileServer("%s", iris.DirOptions{ShowList: %v, Gzip: %v}) instead.`, file, line, directory, showList, gzip)
	return FileServer(directory, DirOptions{ShowList: showList, Gzip: gzip})
}

// StaticEmbedded is DEPRECATED.
// Use HandleDir(requestPath, directory, iris.DirOptions{Asset: Asset, AssetInfo: AssetInfo, AssetNames: AssetNames}) instead.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/file-server/embedding-files-into-app
func (api *APIBuilder) StaticEmbedded(requestPath string, directory string, assetFn func(name string) ([]byte, error), namesFn func() []string) *Route {
	file, line := getCaller()
	api.reporter.Add(`StaticEmbedded is DEPRECATED and it will be removed eventually.
It is also miss the AssetInfo bindata function, which is required now.
Source: %s:%d
Use .HandleDir("%s", "%s", iris.DirOptions{Asset: Asset, AssetInfo: AssetInfo, AssetNames: AssetNames}) instead.`, file, line, requestPath, directory)

	return nil
}

// StaticEmbeddedGzip is DEPRECATED.
// Use HandleDir(requestPath, directory, iris.DirOptions{Gzip: true, Asset: Asset, AssetInfo: AssetInfo, AssetNames: AssetNames}) instead.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/file-server/embedding-gziped-files-into-app
func (api *APIBuilder) StaticEmbeddedGzip(requestPath string, directory string, assetFn func(name string) ([]byte, error), namesFn func() []string) *Route {
	file, line := getCaller()
	api.reporter.Add(`StaticEmbeddedGzip is DEPRECATED and it will be removed eventually.
It is also miss the AssetInfo bindata function, which is required now.
Source: %s:%d
Use .HandleDir("%s", "%s", iris.DirOptions{Gzip: true, Asset: Asset, AssetInfo: AssetInfo, AssetNames: AssetNames}) instead.`, file, line, requestPath, directory)

	return nil
}
