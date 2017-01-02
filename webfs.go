package iris

import (
	"net/http"
	"os"
	"strings"
	"sync"
)

// StaticHandlerBuilder is the web file system's Handler builder
// use that or the iris.StaticHandler/StaticWeb methods
type StaticHandlerBuilder interface {
	Path(requestRoutePath string) StaticHandlerBuilder
	Gzip(enable bool) StaticHandlerBuilder
	Listing(listDirectoriesOnOff bool) StaticHandlerBuilder
	StripPath(yesNo bool) StaticHandlerBuilder
	Build() HandlerFunc
}

type webfs struct {
	// user options, only directory is required.
	directory       http.Dir
	requestPath     string
	stripPath       bool
	gzip            bool
	listDirectories bool
	// these are init on the Build() call
	filesystem http.FileSystem
	once       sync.Once
	handler    HandlerFunc
}

func toWebPath(systemPath string) string {
	// winos slash to slash
	webpath := strings.Replace(systemPath, "\\", slash, -1)
	// double slashes to single
	webpath = strings.Replace(webpath, slash+slash, slash, -1)
	// remove all dots
	webpath = strings.Replace(webpath, ".", "", -1)
	return webpath
}

// NewStaticHandlerBuilder returns a new Handler which serves static files
// supports gzip, no listing and much more
// Note that, this static builder returns a Handler
// it doesn't cares about the rest of your iris configuration.
//
// Use the iris.StaticHandler/StaticWeb in order to serve static files on more automatic way
// this builder is used by people who have more complicated application
// structure and want a fluent api to work on.
func NewStaticHandlerBuilder(dir string) StaticHandlerBuilder {
	return &webfs{
		directory: http.Dir(dir),
		// default route path is the same as the directory
		requestPath: toWebPath(dir),
		// enable strip path by-default
		stripPath: true,
		// gzip is disabled by default
		gzip: false,
		// list directories disabled by default
		listDirectories: false,
	}
}

// Path sets the request path.
// Defaults to same as system path
func (w *webfs) Path(requestRoutePath string) StaticHandlerBuilder {
	w.requestPath = toWebPath(requestRoutePath)
	return w
}

// Gzip if enable is true then gzip compression is enabled for this static directory
// Defaults to false
func (w *webfs) Gzip(enable bool) StaticHandlerBuilder {
	w.gzip = enable
	return w
}

// Listing turn on/off the 'show files and directories'.
// Defaults to false
func (w *webfs) Listing(listDirectoriesOnOff bool) StaticHandlerBuilder {
	w.listDirectories = listDirectoriesOnOff
	return w
}

func (w *webfs) StripPath(yesNo bool) StaticHandlerBuilder {
	w.stripPath = yesNo
	return w
}

type (
	noListFile struct {
		http.File
	}
)

// Overrides the Readdir of the http.File in order to disable showing a list of the dirs/files
func (n noListFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

// Implements the http.Filesystem
// Do not call it.
func (w *webfs) Open(name string) (http.File, error) {
	info, err := w.filesystem.Open(name)

	if err != nil {
		return nil, err
	}

	if !w.listDirectories {
		return noListFile{info}, nil
	}

	return info, nil
}

// Build the handler (once) and returns it
func (w *webfs) Build() HandlerFunc {
	// we have to ensure that Build is called ONLY one time,
	// one instance per one static directory.
	w.once.Do(func() {
		w.filesystem = http.Dir(w.directory)

		// set the filesystem to itself in order to be recognised of listing property (can be change at runtime too)
		fileserver := http.FileServer(w)
		fsHandler := fileserver
		if w.stripPath {
			prefix := w.requestPath
			fsHandler = http.StripPrefix(prefix, fileserver)
		}

		w.handler = func(ctx *Context) {
			writer := ctx.ResponseWriter.ResponseWriter

			if w.gzip && ctx.clientAllowsGzip() {
				ctx.ResponseWriter.Header().Add(varyHeader, acceptEncodingHeader)
				ctx.SetHeader(contentEncodingHeader, "gzip")
				gzipResWriter := acquireGzipResponseWriter(ctx.ResponseWriter.ResponseWriter)
				writer = gzipResWriter
				defer releaseGzipResponseWriter(gzipResWriter)
			}

			fsHandler.ServeHTTP(writer, ctx.Request)
		}
	})

	return w.handler
}
