package router

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/context"
)

// StaticEmbeddedHandler returns a Handler which can serve
// embedded into executable files.
//
//
// Examples: https://github.com/kataras/iris/tree/master/_examples/file-server
func StaticEmbeddedHandler(vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) context.Handler {
	// Depends on the command the user gave to the go-bindata
	// the assset path (names) may be or may not be prepended with a slash.
	// What we do: we remove the ./ from the vdir which should be
	// the same with the asset path (names).
	// we don't pathclean, because that will prepend a slash
	//					   go-bindata should give a correct path format.
	// On serve time we check the "paramName" (which is the path after the "requestPath")
	// so it has the first directory part missing, we use the "vdir" to complete it
	// and match with the asset path (names).
	if len(vdir) > 0 {
		if vdir[0] == '.' {
			vdir = vdir[1:]
		}
		if vdir[0] == '/' || vdir[0] == os.PathSeparator { // second check for /something, (or ./something if we had dot on 0 it will be removed
			vdir = vdir[1:]
		}
	}

	// collect the names we are care for,
	// because not all Asset used here, we need the vdir's assets.
	allNames := namesFn()

	var names []string
	for _, path := range allNames {
		// i.e: path = public/css/main.css

		// check if path is the path name we care for
		if !strings.HasPrefix(path, vdir) {
			continue
		}

		names = append(names, path)
	}

	modtime := time.Now()
	h := func(ctx context.Context) {
		reqPath := strings.TrimPrefix(ctx.Request().URL.Path, "/"+vdir)
		// i.e : /css/main.css

		for _, path := range names {
			// in order to map "/" as "/index.html"
			if path == "/index.html" && reqPath == "/" {
				reqPath = "/index.html"
			}

			if path != vdir+reqPath {
				continue
			}

			cType := TypeByFilename(path)

			buf, err := assetFn(path) // remove the first slash

			if err != nil {
				continue
			}
			ctx.ContentType(cType)
			if _, err := ctx.WriteWithExpiration(buf, modtime); err != nil {
				ctx.StatusCode(http.StatusInternalServerError)
				ctx.StopExecution()
			}
			return
		}

		// not found or error
		ctx.NotFound()

	}

	return h
}

// StaticHandler returns a new Handler which is ready
// to serve all kind of static files.
//
// Developers can wrap this handler using the `router.StripPrefix`
// for a fixed static path when the result handler is being, finally, registered to a route.
//
//
// Usage:
// app := iris.New()
// ...
// fileserver := iris.StaticHandler("./static_files", false, false)
// h := router.StripPrefix("/static", fileserver)
// /* http://mydomain.com/static/css/style.css */
// app.Get("/static", h)
// ...
//
func StaticHandler(systemPath string, showList bool, gzip bool) context.Handler {
	return NewStaticHandlerBuilder(systemPath).
		Gzip(gzip).
		Listing(showList).
		Build()
}

// StaticHandlerBuilder is the web file system's Handler builder
// use that or the iris.StaticHandler/StaticWeb methods.
type StaticHandlerBuilder interface {
	Gzip(enable bool) StaticHandlerBuilder
	Listing(listDirectoriesOnOff bool) StaticHandlerBuilder
	Build() context.Handler
}

//  +------------------------------------------------------------+
//  |                                                            |
//  |                      Static Builder                        |
//  |                                                            |
//  +------------------------------------------------------------+

type fsHandler struct {
	// user options, only directory is required.
	directory       http.Dir
	listDirectories bool
	// these are init on the Build() call
	filesystem http.FileSystem
	once       sync.Once
	handler    context.Handler
	begin      context.Handlers
}

func toWebPath(systemPath string) string {
	// winos slash to slash
	webpath := strings.Replace(systemPath, "\\", "/", -1)
	// double slashes to single
	webpath = strings.Replace(webpath, "//", "/", -1)
	// remove all dots
	webpath = strings.Replace(webpath, ".", "", -1)
	return webpath
}

// Abs calls filepath.Abs but ignores the error and
// returns the original value if any error occurred.
func Abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
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
	return &fsHandler{
		directory: http.Dir(Abs(dir)),
		// list directories disabled by default
		listDirectories: false,
	}
}

// Gzip if enable is true then gzip compression is enabled for this static directory
// Defaults to false
func (w *fsHandler) Gzip(enable bool) StaticHandlerBuilder {
	w.begin = append(w.begin, func(ctx context.Context) {
		ctx.Gzip(true)
		ctx.Next()
	})
	return w
}

// Listing turn on/off the 'show files and directories'.
// Defaults to false
func (w *fsHandler) Listing(listDirectoriesOnOff bool) StaticHandlerBuilder {
	w.listDirectories = listDirectoriesOnOff
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
func (w *fsHandler) Open(name string) (http.File, error) {
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
func (w *fsHandler) Build() context.Handler {
	// we have to ensure that Build is called ONLY one time,
	// one instance per one static directory.
	w.once.Do(func() {
		w.filesystem = w.directory

		fileserver := func(ctx context.Context) {
			upath := ctx.Request().URL.Path
			if !strings.HasPrefix(upath, "/") {
				upath = "/" + upath
				ctx.Request().URL.Path = upath
			}

			// Note the request.url.path is changed but request.RequestURI is not
			// so on custom errors we use the requesturi instead.
			// this can be changed

			_, gzipEnabled := ctx.ResponseWriter().(*context.GzipResponseWriter)
			_, prevStatusCode := serveFile(ctx,
				w.filesystem,
				path.Clean(upath),
				false,
				w.listDirectories,
				gzipEnabled)

			// check for any http errors after the file handler executed
			if context.StatusCodeNotSuccessful(prevStatusCode) { // error found (404 or 400 or 500 usually)
				if writer, ok := ctx.ResponseWriter().(*context.GzipResponseWriter); ok && writer != nil {
					writer.ResetBody()
					writer.Disable()
					// ctx.ResponseWriter.Header().Del(contentType) // application/x-gzip sometimes lawl
					// remove gzip headers
					// headers := ctx.ResponseWriter.Header()
					// headers[contentType] = nil
					// headers["X-Content-Type-Options"] = nil
					// headers[varyHeader] = nil
					// headers[contentEncodingHeader] = nil
					// headers[contentLength] = nil
				}
				// ctx.Application().Logger().Infof(errMsg)
				ctx.StatusCode(prevStatusCode)
				return
			}

			// go to the next middleware
			ctx.Next()
		}
		if len(w.begin) > 0 {
			handlers := append(w.begin[0:], fileserver)
			w.handler = func(ctx context.Context) {
				ctx.Do(handlers)
			}
		}
		w.handler = fileserver
	})

	return w.handler
}

// StripPrefix returns a handler that serves HTTP requests
// by removing the given prefix from the request URL's Path
// and invoking the handler h. StripPrefix handles a
// request for a path that doesn't begin with prefix by
// replying with an HTTP 404 not found error.
//
// Usage:
// fileserver := iris.StaticHandler("./static_files", false, false)
// h := router.StripPrefix("/static", fileserver)
// app.Get("/static", h)
//
func StripPrefix(prefix string, h context.Handler) context.Handler {
	if prefix == "" {
		return h
	}
	// here we separate the path from the subdomain (if any), we care only for the path
	// fixes a bug when serving static files via a subdomain
	fixedPrefix := prefix
	if dotWSlashIdx := strings.Index(fixedPrefix, SubdomainPrefix); dotWSlashIdx > 0 {
		fixedPrefix = fixedPrefix[dotWSlashIdx+1:]
	}
	fixedPrefix = toWebPath(fixedPrefix)

	return func(ctx context.Context) {
		if p := strings.TrimPrefix(ctx.Request().URL.Path, fixedPrefix); len(p) < len(ctx.Request().URL.Path) {
			ctx.Request().URL.Path = p
			h(ctx)
		} else {
			ctx.NotFound()
		}
	}
}

//  +------------------------------------------------------------+
//  |                                                            |
//  |                      serve file handler                    |
//  | edited from net/http/fs.go in order to support GZIP with   |
//  | custom iris http errors and fallback to non-compressed data|
//  | when not supported.                                        |
//  |                                                            |
//  +------------------------------------------------------------+

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

func dirList(ctx context.Context, f http.File) (string, int) {
	dirs, err := f.Readdir(-1)
	if err != nil {
		// TODO: log err.Error() to the Server.ErrorLog, once it's possible
		// for a handler to get at its Server via the http.ResponseWriter. See
		// Issue 12438.
		return "Error reading directory", http.StatusInternalServerError

	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	ctx.ContentType("text/html")
	fmt.Fprintf(ctx.ResponseWriter(), "<pre>\n")
	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		// name may contain '?' or '#', which must be escaped to remain
		// part of the URL path, and not indicate the start of a query
		// string or fragment.
		url := url.URL{Path: name}
		fmt.Fprintf(ctx.ResponseWriter(), "<a href=\"%s\">%s</a>\n", url.String(), htmlReplacer.Replace(name))
	}
	fmt.Fprintf(ctx.ResponseWriter(), "</pre>\n")
	return "", http.StatusOK
}

// errSeeker is returned by ServeContent's sizeFunc when the content
// doesn't seek properly. The underlying Seeker's error text isn't
// included in the sizeFunc reply so it's not sent over HTTP to end
// users.
var errSeeker = errors.New("seeker can't seek")

// errNoOverlap is returned by serveContent's parseRange if first-byte-pos of
// all of the byte-range-spec values is greater than the content size.
var errNoOverlap = errors.New("invalid range: failed to overlap")

// The algorithm uses at most sniffLen bytes to make its decision.
const sniffLen = 512

func detectOrWriteContentType(ctx context.Context, name string, content io.ReadSeeker) (string, error) {
	// If Content-Type isn't set, use the file's extension to find it, but
	// if the Content-Type is unset explicitly, do not sniff the type.
	ctypes, haveType := ctx.ResponseWriter().Header()["Content-Type"]
	var ctype string

	if !haveType {
		ctype = TypeByExtension(filepath.Ext(name))
		if ctype == "" {
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			n, _ := io.ReadFull(content, buf[:])
			ctype = http.DetectContentType(buf[:n])
			_, err := content.Seek(0, io.SeekStart) // rewind to output whole file
			if err != nil {
				return "", err
			}
		}

		ctx.ContentType(ctype)
	} else if len(ctypes) > 0 {
		ctype = ctypes[0]
	}

	return ctype, nil
}

// if name is empty, filename is unknown. (used for mime type, before sniffing)
// if modtime.IsZero(), modtime is unknown.
// content must be seeked to the beginning of the file.
// The sizeFunc is called at most once. Its error, if any, is sent in the HTTP response.
func serveContent(ctx context.Context, name string, modtime time.Time, sizeFunc func() (int64, error), content io.ReadSeeker) (string, int) /* we could use the TransactionErrResult but prefer not to create new objects for each of the errors on static file handlers*/ {
	ctx.SetLastModified(modtime)
	done, rangeReq := checkPreconditions(ctx, modtime)
	if done {
		return "", http.StatusNotModified
	}

	code := http.StatusOK

	// If Content-Type isn't set, use the file's extension to find it, but
	// if the Content-Type is unset explicitly, do not sniff the type.
	ctype, err := detectOrWriteContentType(ctx, name, content)
	if err != nil {
		return "while seeking", http.StatusInternalServerError
	}

	size, err := sizeFunc()
	if err != nil {
		return err.Error(), http.StatusInternalServerError
	}

	// handle Content-Range header.
	sendSize := size
	var sendContent io.Reader = content

	if size >= 0 {
		ranges, err := parseRange(rangeReq, size)
		if err != nil {
			if err == errNoOverlap {
				ctx.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
			}
			return err.Error(), http.StatusRequestedRangeNotSatisfiable

		}
		if sumRangesSize(ranges) > size {
			// The total number of bytes in all the ranges
			// is larger than the size of the file by
			// itself, so this is probably an attack, or a
			// dumb client. Ignore the range request.
			ranges = nil
		}
		switch {
		case len(ranges) == 1:
			// RFC 2616, Section 14.16:
			// "When an HTTP message includes the content of a single
			// range (for example, a response to a request for a
			// single range, or to a request for a set of ranges
			// that overlap without any holes), this content is
			// transmitted with a Content-Range header, and a
			// Content-Length header showing the number of bytes
			// actually transferred.
			// ...
			// A response to a request for a single range MUST NOT
			// be sent using the multipart/byteranges media type."
			ra := ranges[0]
			if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
				return err.Error(), http.StatusRequestedRangeNotSatisfiable
			}
			sendSize = ra.length
			code = http.StatusPartialContent
			ctx.Header("Content-Range", ra.contentRange(size))
		case len(ranges) > 1:
			sendSize = rangesMIMESize(ranges, ctype, size)
			code = http.StatusPartialContent

			pr, pw := io.Pipe()
			mw := multipart.NewWriter(pw)
			ctx.ContentType("multipart/byteranges; boundary=" + mw.Boundary())
			sendContent = pr
			defer pr.Close() // cause writing goroutine to fail and exit if CopyN doesn't finish.
			go func() {
				for _, ra := range ranges {
					part, err := mw.CreatePart(ra.mimeHeader(ctype, size))
					if err != nil {
						pw.CloseWithError(err)
						return
					}
					if _, err := content.Seek(ra.start, io.SeekStart); err != nil {
						pw.CloseWithError(err)
						return
					}
					if _, err := io.CopyN(part, content, ra.length); err != nil {
						pw.CloseWithError(err)
						return
					}
				}
				mw.Close()
				pw.Close()
			}()
		}
		ctx.Header("Accept-Ranges", "bytes")
		if ctx.ResponseWriter().Header().Get(contentEncodingHeaderKey) == "" {
			ctx.Header(contentLengthHeaderKey, strconv.FormatInt(sendSize, 10))
		}
	}

	ctx.StatusCode(code)

	if ctx.Method() != http.MethodHead {
		io.CopyN(ctx.ResponseWriter(), sendContent, sendSize)
	}

	return "", code
}

func etagEmptyOrStrongMatch(rangeValue string, etagValue string) bool {
	etag, _ := scanETag(rangeValue)
	if etag != "" {
		if etagStrongMatch(etag, etagValue) {
			return true
		}
		return false
	}
	return true
}

// scanETag determines if a syntactically valid ETag is present at s. If so,
// the ETag and remaining text after consuming ETag is returned. Otherwise,
// it returns "", "".
func scanETag(s string) (etag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, "W/") {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	// ETag is either W/"text" or "text".
	// See RFC 7232 2.3.
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		// Character values allowed in ETags.
		case c == 0x21 || c >= 0x23 && c <= 0x7E || c >= 0x80:
		case c == '"':
			return string(s[:i+1]), s[i+1:]
		default:
			break
		}
	}
	return "", ""
}

// etagStrongMatch reports whether a and b match using strong ETag comparison.
// Assumes a and b are valid ETags.
func etagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

// etagWeakMatch reports whether a and b match using weak ETag comparison.
// Assumes a and b are valid ETags.
func etagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, "W/") == strings.TrimPrefix(b, "W/")
}

// condResult is the result of an HTTP request precondition check.
// See https://tools.ietf.org/html/rfc7232 section 3.
type condResult int

const (
	condNone condResult = iota
	condTrue
	condFalse
)

func checkIfMatch(ctx context.Context) condResult {
	im := ctx.GetHeader("If-Match")
	if im == "" {
		return condNone
	}
	for {
		im = textproto.TrimString(im)
		if len(im) == 0 {
			break
		}
		if im[0] == ',' {
			im = im[1:]
			continue
		}
		if im[0] == '*' {
			return condTrue
		}
		etag, remain := scanETag(im)
		if etag == "" {
			break
		}
		if etagStrongMatch(etag, ctx.ResponseWriter().Header().Get("Etag")) {
			return condTrue
		}
		im = remain
	}

	return condFalse
}

func checkIfNoneMatch(ctx context.Context) condResult {
	inm := ctx.GetHeader("If-None-Match")
	if inm == "" {
		return condNone
	}
	buf := inm
	for {
		buf = textproto.TrimString(buf)
		if len(buf) == 0 {
			break
		}
		if buf[0] == ',' {
			buf = buf[1:]
		}
		if buf[0] == '*' {
			return condFalse
		}
		etag, remain := scanETag(buf)
		if etag == "" {
			break
		}
		if etagWeakMatch(etag, ctx.ResponseWriter().Header().Get("Etag")) {
			return condFalse
		}
		buf = remain
	}
	return condTrue
}

// checkPreconditions evaluates request preconditions and reports whether a precondition
// resulted in sending StatusNotModified or StatusPreconditionFailed.
func checkPreconditions(ctx context.Context, modtime time.Time) (done bool, rangeHeader string) {
	// This function carefully follows RFC 7232 section 6.
	ch := checkIfMatch(ctx)
	if ch == condNone {
		ch = checkIfUnmodifiedSince(ctx, modtime)
	}
	if ch == condFalse {

		ctx.StatusCode(http.StatusPreconditionFailed)
		return true, ""
	}
	switch checkIfNoneMatch(ctx) {
	case condFalse:
		if ctx.Method() == http.MethodGet || ctx.Method() == http.MethodHead {
			ctx.WriteNotModified()
			return true, ""
		}
		ctx.StatusCode(http.StatusPreconditionFailed)
		return true, ""

	case condNone:
		if modified, err := ctx.CheckIfModifiedSince(modtime); !modified && err == nil {
			ctx.WriteNotModified()
			return true, ""
		}
	}

	rangeHeader = ctx.GetHeader("Range")
	if rangeHeader != "" {
		if checkIfRange(ctx, etagEmptyOrStrongMatch, modtime) == condFalse {
			rangeHeader = ""
		}
	}
	return false, rangeHeader
}

func checkIfUnmodifiedSince(ctx context.Context, modtime time.Time) condResult {
	ius := ctx.GetHeader("If-Unmodified-Since")
	if ius == "" || context.IsZeroTime(modtime) {
		return condNone
	}
	if t, err := context.ParseTime(ctx, ius); err == nil {
		// The Date-Modified header truncates sub-second precision, so
		// use mtime < t+1s instead of mtime <= t to check for unmodified.
		if modtime.Before(t.Add(1 * time.Second)) {
			return condTrue
		}
		return condFalse
	}
	return condNone
}

func checkIfRange(ctx context.Context, etagEmptyOrStrongMatch func(ifRangeValue string, etagValue string) bool, modtime time.Time) condResult {
	if ctx.Method() != http.MethodGet {
		return condNone
	}
	ir := ctx.GetHeader("If-Range")
	if ir == "" {
		return condNone
	}

	if etagEmptyOrStrongMatch(ir, ctx.GetHeader("Etag")) {
		return condTrue
	}

	// The If-Range value is typically the ETag value, but it may also be
	// the modtime date. See golang.org/issue/8367.
	if modtime.IsZero() {
		return condFalse
	}
	t, err := context.ParseTime(ctx, ir)
	if err != nil {
		return condFalse
	}
	if t.Unix() == modtime.Unix() {
		return condTrue
	}
	return condFalse
}

// name is '/'-separated, not filepath.Separator.
func serveFile(ctx context.Context, fs http.FileSystem, name string, redirect bool, showList bool, gzip bool) (string, int) {
	const indexPage = "/index.html"

	// redirect .../index.html to .../
	// can't use Redirect() because that would make the path absolute,
	// which would be a problem running under StripPrefix
	if strings.HasSuffix(ctx.Request().URL.Path, indexPage) {
		localRedirect(ctx, "./")
		return "", http.StatusMovedPermanently
	}

	f, err := fs.Open(name)
	if err != nil {
		return err.Error(), 404
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		return err.Error(), 404
	}

	if redirect {
		// redirect to canonical path: / at end of directory url
		// ctx.Request.URL.Path always begins with /
		url := ctx.Request().URL.Path
		if d.IsDir() {
			if url[len(url)-1] != '/' {
				localRedirect(ctx, path.Base(url)+"/")
				return "", http.StatusMovedPermanently
			}
		} else {
			if url[len(url)-1] == '/' {
				localRedirect(ctx, "../"+path.Base(url))
				return "", http.StatusMovedPermanently
			}
		}
	}

	// redirect if the directory name doesn't end in a slash
	if d.IsDir() {
		url := ctx.Request().URL.Path
		if url[len(url)-1] != '/' {
			localRedirect(ctx, path.Base(url)+"/")
			return "", http.StatusMovedPermanently
		}
	}

	// use contents of index.html for directory, if present
	if d.IsDir() {
		index := strings.TrimSuffix(name, "/") + indexPage
		ff, err := fs.Open(index)
		if err == nil {
			defer ff.Close()
			dd, err := ff.Stat()
			if err == nil {
				d = dd
				f = ff
			}
		}
	}

	// Still a directory? (we didn't find an index.html file)
	if d.IsDir() {
		if !showList {
			return "", http.StatusForbidden
		}
		if modified, err := ctx.CheckIfModifiedSince(d.ModTime()); !modified && err == nil {
			ctx.WriteNotModified()
			return "", http.StatusNotModified
		}
		ctx.SetLastModified(d.ModTime())
		return dirList(ctx, f)
	}

	// if gzip disabled then continue using content byte ranges
	if !gzip {
		// serveContent will check modification time
		sizeFunc := func() (int64, error) { return d.Size(), nil }
		return serveContent(ctx, d.Name(), d.ModTime(), sizeFunc, f)
	}

	// else, set the last modified as "serveContent" does.
	ctx.SetLastModified(d.ModTime())

	// write the file to the response writer.
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		ctx.Application().Logger().Debugf("err reading file: %v", err)
		return "error reading the file", http.StatusInternalServerError
	}

	// Use `WriteNow` instead of `Write`
	// because we need to know the compressed written size before
	// the `FlushResponse`.
	_, err = ctx.GzipResponseWriter().Write(contents)
	if err != nil {
		ctx.Application().Logger().Debugf("short write: %v", err)
		return "short write", http.StatusInternalServerError
	}

	// try to find and send the correct content type based on the filename
	// and the binary data inside "f".
	detectOrWriteContentType(ctx, d.Name(), f)

	return "", http.StatusOK
}

// toHTTPError returns a non-specific HTTP error message and status code
// for a given non-nil error value. It's important that toHTTPError does not
// actually return err.Error(), since msg and httpStatus are returned to users,
// and historically Go's ServeContent always returned just "404 Not Found" for
// all errors. We don't want to start leaking information in error messages.
func toHTTPError(err error) (msg string, httpStatus int) {
	if os.IsNotExist(err) {
		return "404 page not found", http.StatusNotFound
	}
	if os.IsPermission(err) {
		return "403 Forbidden", http.StatusForbidden
	}
	// Default:
	return "500 Internal Server Error", http.StatusInternalServerError
}

// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like Redirect does.
func localRedirect(ctx context.Context, newPath string) {
	if q := ctx.Request().URL.RawQuery; q != "" {
		newPath += "?" + q
	}

	ctx.Header("Location", newPath)
	ctx.StatusCode(http.StatusMovedPermanently)
}

func containsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

// httpRange specifies the byte range to be sent to the client.
type httpRange struct {
	start, length int64
}

func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}

func (r httpRange) mimeHeader(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Range": {r.contentRange(size)},
		contentType:     {contentType},
	}
}

// parseRange parses a Range header string as per RFC 2616.
// errNoOverlap is returned if none of the ranges overlap.
func parseRange(s string, size int64) ([]httpRange, error) {
	if s == "" {
		return nil, nil // header not present
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, errors.New("invalid range")
	}
	var ranges []httpRange
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = strings.TrimSpace(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return nil, errors.New("invalid range")
		}
		start, end := strings.TrimSpace(ra[:i]), strings.TrimSpace(ra[i+1:])
		var r httpRange
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file.
			i, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				return nil, errors.New("invalid range")
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, errors.New("invalid range")
			}
			if i >= size {
				// If the range begins after the size of the content,
				// then it does not overlap.
				noOverlap = true
				continue
			}
			r.start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return nil, errors.New("invalid range")
				}
				if i >= size {
					i = size - 1
				}
				r.length = i - r.start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		return nil, errNoOverlap
	}
	return ranges, nil
}

// countingWriter counts how many bytes have been written to it.
type countingWriter int64

func (w *countingWriter) Write(p []byte) (n int, err error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

// rangesMIMESize returns the number of bytes it takes to encode the
// provided ranges as a multipart response.
func rangesMIMESize(ranges []httpRange, contentType string, contentSize int64) (encSize int64) {
	var w countingWriter
	mw := multipart.NewWriter(&w)
	for _, ra := range ranges {
		mw.CreatePart(ra.mimeHeader(contentType, contentSize))
		encSize += ra.length
	}
	mw.Close()
	encSize += int64(w)
	return
}

func sumRangesSize(ranges []httpRange) (size int64) {
	for _, ra := range ranges {
		size += ra.length
	}
	return
}

// DirectoryExists returns true if a directory(or file) exists, otherwise false
func DirectoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}
