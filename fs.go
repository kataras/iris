package iris

import (
	"fmt"
	"io"
	"mime"
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

	"github.com/kataras/go-errors"
)

// StaticHandler returns a new Handler which is ready
// to serve all kind of static files.
//
// Developers can wrap this handler using the `iris.StripPrefix`
// for a fixed static path when the result handler is being, finally, registered to a route.
//
//
// Usage:
// app := iris.New()
// ...
// fileserver := iris.StaticHandler("./static_files", false, false)
// h := iris.StripPrefix("/static", fileserver)
// /* http://mydomain.com/static/css/style.css */
// app.Get("/static", h)
// ...
//
func StaticHandler(systemPath string, showList bool, enableGzip bool, exceptRoutes ...RouteInfo) HandlerFunc {
	return NewStaticHandlerBuilder(systemPath).
		Listing(showList).
		Gzip(enableGzip).
		Except(exceptRoutes...).
		Build()
}

// StaticHandlerBuilder is the web file system's Handler builder
// use that or the iris.StaticHandler/StaticWeb methods
type StaticHandlerBuilder interface {
	Gzip(enable bool) StaticHandlerBuilder
	Listing(listDirectoriesOnOff bool) StaticHandlerBuilder
	Except(r ...RouteInfo) StaticHandlerBuilder
	Build() HandlerFunc
}

//  +------------------------------------------------------------+
//  |                                                            |
//  |                      Static Builder                        |
//  |                                                            |
//  +------------------------------------------------------------+

type fsHandler struct {
	// user options, only directory is required.
	directory       http.Dir
	gzip            bool
	listDirectories bool
	// these are init on the Build() call
	filesystem http.FileSystem
	once       sync.Once
	exceptions []RouteInfo
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

// abs calls filepath.Abs but ignores the error and
// returns the original value if any error occurred.
func abs(path string) string {
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
		directory: http.Dir(abs(dir)),
		// gzip is disabled by default
		gzip: false,
		// list directories disabled by default
		listDirectories: false,
	}
}

// Gzip if enable is true then gzip compression is enabled for this static directory
// Defaults to false
func (w *fsHandler) Gzip(enable bool) StaticHandlerBuilder {
	w.gzip = enable
	return w
}

// Listing turn on/off the 'show files and directories'.
// Defaults to false
func (w *fsHandler) Listing(listDirectoriesOnOff bool) StaticHandlerBuilder {
	w.listDirectories = listDirectoriesOnOff
	return w
}

// Except add a route exception,
// gives priority to that Route over the static handler.
func (w *fsHandler) Except(r ...RouteInfo) StaticHandlerBuilder {
	w.exceptions = append(w.exceptions, r...)
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
func (w *fsHandler) Build() HandlerFunc {
	// we have to ensure that Build is called ONLY one time,
	// one instance per one static directory.
	w.once.Do(func() {
		w.filesystem = w.directory

		fileserver := func(ctx *Context) {
			upath := ctx.Request.URL.Path
			if !strings.HasPrefix(upath, "/") {
				upath = "/" + upath
				ctx.Request.URL.Path = upath
			}

			// Note the request.url.path is changed but request.RequestURI is not
			// so on custom errors we use the requesturi instead.
			// this can be changed
			_, prevStatusCode := serveFile(ctx,
				w.filesystem,
				path.Clean(upath),
				false,
				w.listDirectories,
				(w.gzip && ctx.clientAllowsGzip()),
			)

			// check for any http errors after the file handler executed
			if prevStatusCode >= 400 { // error found (404 or 400 or 500 usually)
				if writer, ok := ctx.ResponseWriter.(*gzipResponseWriter); ok && writer != nil {
					writer.ResetBody()
					writer.Disable()
					// ctx.ResponseWriter.Header().Del(contentType) // application/x-gzip sometimes lawl
					// remove gzip headers
					headers := ctx.ResponseWriter.Header()
					headers[contentType] = nil
					headers["X-Content-Type-Options"] = nil
					headers[varyHeader] = nil
					headers[contentEncodingHeader] = nil
					headers[contentLength] = nil
				}
				// execute any custom error handler (per-party or global, if not found then it creates a new one and fires it)
				ctx.Framework().Router.Errors.Fire(prevStatusCode, ctx)
				return
			}

			// go to the next middleware
			if ctx.Pos < len(ctx.Middleware)-1 {
				ctx.Next()
			}
		}

		if len(w.exceptions) > 0 {
			middleware := make(Middleware, len(w.exceptions)+1)
			for i := range w.exceptions {
				middleware[i] = Prioritize(w.exceptions[i])
			}
			middleware[len(w.exceptions)] = HandlerFunc(fileserver)

			w.handler = func(ctx *Context) {
				ctx.Middleware = append(middleware, ctx.Middleware...)
				ctx.Do()
			}
			return
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
// h := iris.StripPrefix("/static", fileserver)
// app.Get("/static", h)
//
func StripPrefix(prefix string, h HandlerFunc) HandlerFunc {
	if prefix == "" {
		return h
	}
	// here we separate the path from the subdomain (if any), we care only for the path
	// fixes a bug when serving static files via a subdomain
	fixedPrefix := prefix
	if dotWSlashIdx := strings.Index(fixedPrefix, subdomainIndicator); dotWSlashIdx > 0 {
		fixedPrefix = fixedPrefix[dotWSlashIdx+1:]
	}
	fixedPrefix = toWebPath(fixedPrefix)

	return func(ctx *Context) {
		if p := strings.TrimPrefix(ctx.Request.URL.Path, fixedPrefix); len(p) < len(ctx.Request.URL.Path) {
			ctx.Request.URL.Path = p
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

func dirList(ctx *Context, f http.File) (string, int) {
	dirs, err := f.Readdir(-1)
	if err != nil {
		// TODO: log err.Error() to the Server.ErrorLog, once it's possible
		// for a handler to get at its Server via the http.ResponseWriter. See
		// Issue 12438.
		return "Error reading directory", StatusInternalServerError

	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	ctx.ResponseWriter.Header().Set(contentType, contentHTML+"; charset="+ctx.Framework().Config.Charset)
	fmt.Fprintf(ctx.ResponseWriter, "<pre>\n")
	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		// name may contain '?' or '#', which must be escaped to remain
		// part of the URL path, and not indicate the start of a query
		// string or fragment.
		url := url.URL{Path: name}
		fmt.Fprintf(ctx.ResponseWriter, "<a href=\"%s\">%s</a>\n", url.String(), htmlReplacer.Replace(name))
	}
	fmt.Fprintf(ctx.ResponseWriter, "</pre>\n")
	return "", StatusOK
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

// if name is empty, filename is unknown. (used for mime type, before sniffing)
// if modtime.IsZero(), modtime is unknown.
// content must be seeked to the beginning of the file.
// The sizeFunc is called at most once. Its error, if any, is sent in the HTTP response.
func serveContent(ctx *Context, name string, modtime time.Time, sizeFunc func() (int64, error), content io.ReadSeeker, gzip bool) (string, int) /* we could use the TransactionErrResult but prefer not to create new objects for each of the errors on static file handlers*/ {

	setLastModified(ctx, modtime)
	done, rangeReq := checkPreconditions(ctx, modtime)
	if done {
		return "", StatusNotModified
	}

	code := StatusOK

	// If Content-Type isn't set, use the file's extension to find it, but
	// if the Content-Type is unset explicitly, do not sniff the type.
	ctypes, haveType := ctx.ResponseWriter.Header()[contentType]
	var ctype string
	if !haveType {
		ctype = typeByExtension(filepath.Ext(name))
		if ctype == "" {
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			n, _ := io.ReadFull(content, buf[:])
			ctype = http.DetectContentType(buf[:n])
			_, err := content.Seek(0, io.SeekStart) // rewind to output whole file
			if err != nil {
				return "seeker can't seek", StatusInternalServerError

			}
		}
		ctx.ResponseWriter.Header().Set(contentType, ctype)
	} else if len(ctypes) > 0 {
		ctype = ctypes[0]
	}

	size, err := sizeFunc()
	if err != nil {
		return err.Error(), StatusInternalServerError
	}

	// handle Content-Range header.
	sendSize := size
	var sendContent io.Reader = content

	if gzip {
		// set the "Accept-Encoding" here in order to prevent the content-length header to be setted later on.
		ctx.SetHeader(contentEncodingHeader, "gzip")
		ctx.ResponseWriter.Header().Add(varyHeader, acceptEncodingHeader)
		gzipResWriter := acquireGzipResponseWriter(ctx.ResponseWriter)
		ctx.ResponseWriter = gzipResWriter
	}
	if size >= 0 {
		ranges, err := parseRange(rangeReq, size)
		if err != nil {
			if err == errNoOverlap {
				ctx.ResponseWriter.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
			}
			return err.Error(), StatusRequestedRangeNotSatisfiable

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
				return err.Error(), StatusRequestedRangeNotSatisfiable
			}
			sendSize = ra.length
			code = StatusPartialContent
			ctx.ResponseWriter.Header().Set("Content-Range", ra.contentRange(size))
		case len(ranges) > 1:
			sendSize = rangesMIMESize(ranges, ctype, size)
			code = StatusPartialContent

			pr, pw := io.Pipe()
			mw := multipart.NewWriter(pw)
			ctx.ResponseWriter.Header().Set(contentType, "multipart/byteranges; boundary="+mw.Boundary())
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
		ctx.ResponseWriter.Header().Set("Accept-Ranges", "bytes")
		if ctx.ResponseWriter.Header().Get(contentEncodingHeader) == "" {

			ctx.ResponseWriter.Header().Set(contentLength, strconv.FormatInt(sendSize, 10))
		}
	}

	ctx.SetStatusCode(code)

	if ctx.Method() != MethodHead {
		io.CopyN(ctx.ResponseWriter, sendContent, sendSize)
	}

	return "", code
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

func checkIfMatch(ctx *Context) condResult {
	im := ctx.Request.Header.Get("If-Match")
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
		if etagStrongMatch(etag, ctx.ResponseWriter.Header().Get("Etag")) {
			return condTrue
		}
		im = remain
	}

	return condFalse
}

func checkIfUnmodifiedSince(ctx *Context, modtime time.Time) condResult {
	ius := ctx.Request.Header.Get("If-Unmodified-Since")
	if ius == "" || isZeroTime(modtime) {
		return condNone
	}
	if t, err := http.ParseTime(ius); err == nil {
		// The Date-Modified header truncates sub-second precision, so
		// use mtime < t+1s instead of mtime <= t to check for unmodified.
		if modtime.Before(t.Add(1 * time.Second)) {
			return condTrue
		}
		return condFalse
	}
	return condNone
}

func checkIfNoneMatch(ctx *Context) condResult {
	inm := ctx.Request.Header.Get("If-None-Match")
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
		if etagWeakMatch(etag, ctx.ResponseWriter.Header().Get("Etag")) {
			return condFalse
		}
		buf = remain
	}
	return condTrue
}

func checkIfModifiedSince(ctx *Context, modtime time.Time) condResult {
	if ctx.Method() != MethodGet && ctx.Method() != MethodHead {
		return condNone
	}
	ims := ctx.Request.Header.Get("If-Modified-Since")
	if ims == "" || isZeroTime(modtime) {
		return condNone
	}
	t, err := http.ParseTime(ims)
	if err != nil {
		return condNone
	}
	// The Date-Modified header truncates sub-second precision, so
	// use mtime < t+1s instead of mtime <= t to check for unmodified.
	if modtime.Before(t.Add(1 * time.Second)) {
		return condFalse
	}
	return condTrue
}

func checkIfRange(ctx *Context, modtime time.Time) condResult {
	if ctx.Method() != MethodGet {
		return condNone
	}
	ir := ctx.Request.Header.Get("If-Range")
	if ir == "" {
		return condNone
	}
	etag, _ := scanETag(ir)
	if etag != "" {
		if etagStrongMatch(etag, ctx.ResponseWriter.Header().Get("Etag")) {
			return condTrue
		}
		return condFalse

	}
	// The If-Range value is typically the ETag value, but it may also be
	// the modtime date. See golang.org/issue/8367.
	if modtime.IsZero() {
		return condFalse
	}
	t, err := http.ParseTime(ir)
	if err != nil {
		return condFalse
	}
	if t.Unix() == modtime.Unix() {
		return condTrue
	}
	return condFalse
}

var unixEpochTime = time.Unix(0, 0)

// isZeroTime reports whether t is obviously unspecified (either zero or Unix()=0).
func isZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

func setLastModified(ctx *Context, modtime time.Time) {
	if !isZeroTime(modtime) {
		ctx.SetHeader("Last-Modified", modtime.UTC().Format(ctx.Framework().Config.TimeFormat))
	}
}

func writeNotModified(ctx *Context) {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g., Last-Modified might be useful if the
	// response does not have an ETag field).
	h := ctx.ResponseWriter.Header()
	delete(h, contentType)

	delete(h, contentLength)
	if h.Get("Etag") != "" {
		delete(h, "Last-Modified")
	}
	ctx.SetStatusCode(StatusNotModified)
}

// checkPreconditions evaluates request preconditions and reports whether a precondition
// resulted in sending StatusNotModified or StatusPreconditionFailed.
func checkPreconditions(ctx *Context, modtime time.Time) (done bool, rangeHeader string) {
	// This function carefully follows RFC 7232 section 6.
	ch := checkIfMatch(ctx)
	if ch == condNone {
		ch = checkIfUnmodifiedSince(ctx, modtime)
	}
	if ch == condFalse {

		ctx.SetStatusCode(StatusPreconditionFailed)
		return true, ""
	}
	switch checkIfNoneMatch(ctx) {
	case condFalse:
		if ctx.Method() == MethodGet || ctx.Method() == MethodHead {
			writeNotModified(ctx)
			return true, ""
		}
		ctx.SetStatusCode(StatusPreconditionFailed)
		return true, ""

	case condNone:
		if checkIfModifiedSince(ctx, modtime) == condFalse {
			writeNotModified(ctx)
			return true, ""
		}
	}

	rangeHeader = ctx.Request.Header.Get("Range")
	if rangeHeader != "" {
		if checkIfRange(ctx, modtime) == condFalse {
			rangeHeader = ""
		}
	}
	return false, rangeHeader
}

// name is '/'-separated, not filepath.Separator.
func serveFile(ctx *Context, fs http.FileSystem, name string, redirect bool, showList bool, gzip bool) (string, int) {
	const indexPage = "/index.html"

	// redirect .../index.html to .../
	// can't use Redirect() because that would make the path absolute,
	// which would be a problem running under StripPrefix
	if strings.HasSuffix(ctx.Request.URL.Path, indexPage) {
		localRedirect(ctx, "./")
		return "", StatusMovedPermanently
	}

	f, err := fs.Open(name)
	if err != nil {
		msg, code := toHTTPError(err)
		return msg, code
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		msg, code := toHTTPError(err)
		return msg, code

	}

	if redirect {
		// redirect to canonical path: / at end of directory url
		// ctx.Request.URL.Path always begins with /
		url := ctx.Request.URL.Path
		if d.IsDir() {
			if url[len(url)-1] != '/' {
				localRedirect(ctx, path.Base(url)+"/")
				return "", StatusMovedPermanently
			}
		} else {
			if url[len(url)-1] == '/' {
				localRedirect(ctx, "../"+path.Base(url))
				return "", StatusMovedPermanently
			}
		}
	}

	// redirect if the directory name doesn't end in a slash
	if d.IsDir() {
		url := ctx.Request.URL.Path
		if url[len(url)-1] != '/' {
			localRedirect(ctx, path.Base(url)+"/")
			return "", StatusMovedPermanently
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
				name = index
				d = dd
				f = ff
			}
		}
	}

	// Still a directory? (we didn't find an index.html file)
	if d.IsDir() {
		if !showList {
			return "", StatusForbidden
		}
		if checkIfModifiedSince(ctx, d.ModTime()) == condFalse {
			writeNotModified(ctx)
			return "", StatusNotModified
		}
		ctx.ResponseWriter.Header().Set("Last-Modified", d.ModTime().UTC().Format(ctx.Framework().Config.TimeFormat))
		return dirList(ctx, f)

	}

	// serveContent will check modification time
	sizeFunc := func() (int64, error) { return d.Size(), nil }
	return serveContent(ctx, d.Name(), d.ModTime(), sizeFunc, f, gzip)
}

// toHTTPError returns a non-specific HTTP error message and status code
// for a given non-nil error value. It's important that toHTTPError does not
// actually return err.Error(), since msg and httpStatus are returned to users,
// and historically Go's ServeContent always returned just "404 Not Found" for
// all errors. We don't want to start leaking information in error messages.
func toHTTPError(err error) (msg string, httpStatus int) {
	if os.IsNotExist(err) {
		return "404 page not found", StatusNotFound
	}
	if os.IsPermission(err) {
		return "403 Forbidden", StatusForbidden
	}
	// Default:
	return "500 Internal Server Error", StatusInternalServerError
}

// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like Redirect does.
func localRedirect(ctx *Context, newPath string) {
	if q := ctx.Request.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	ctx.ResponseWriter.Header().Set("Location", newPath)
	ctx.SetStatusCode(StatusMovedPermanently)
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

// typeByExtension returns the MIME type associated with the file extension ext.
// The extension ext should begin with a leading dot, as in ".html".
// When ext has no associated type, typeByExtension returns "".
//
// Extensions are looked up first case-sensitively, then case-insensitively.
//
// The built-in table is small but on unix it is augmented by the local
// system's mime.types file(s) if available under one or more of these
// names:
//
//   /etc/mime.types
//   /etc/apache2/mime.types
//   /etc/apache/mime.types
//
// On Windows, MIME types are extracted from the registry.
//
// Text types have the charset parameter set to "utf-8" by default.
func typeByExtension(fullfilename string) (t string) {
	ext := filepath.Ext(fullfilename)
	//these should be found by the windows(registry) and unix(apache) but on windows some machines have problems on this part.
	if t = mime.TypeByExtension(ext); t == "" {
		// no use of map here because we will have to lock/unlock it, by hand is better, no problem:
		if ext == ".json" {
			t = "application/json"
		} else if ext == ".js" {
			t = "application/javascript"
		} else if ext == ".zip" {
			t = "application/zip"
		} else if ext == ".3gp" {
			t = "video/3gpp"
		} else if ext == ".7z" {
			t = "application/x-7z-compressed"
		} else if ext == ".ace" {
			t = "application/x-ace-compressed"
		} else if ext == ".aac" {
			t = "audio/x-aac"
		} else if ext == ".ico" { // for any case
			t = "image/x-icon"
		} else if ext == ".png" {
			t = "image/png"
		} else {
			t = "application/octet-stream"
		}
		// mime.TypeByExtension returns as text/plain; | charset=utf-8 the static .js (not always)
	} else if t == "text/plain" || t == "text/plain; charset=utf-8" {
		if ext == ".js" {
			t = "application/javascript"
		}
	}
	return
}

// directoryExists returns true if a directory(or file) exists, otherwise false
func directoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}
