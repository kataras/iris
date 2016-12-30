package fs

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	// TimeFormat default time format for any kind of datetime parsing
	TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
	// StaticCacheDuration expiration duration for INACTIVE file handlers
	StaticCacheDuration = 20 * time.Second
	// Charset the charset will be used to the Content-Type response header, if not given previously
	Charset = "utf-8"
)

var (
	// contentTypeHeader represents the header["Content-Type"]
	contentTypeHeader = "Content-Type"
	// contentLength represents the header["Content-Length"]
	contentLength = "Content-Length"
	// contentEncodingHeader represents the header["Content-Encoding"]
	contentEncodingHeader = "Content-Encoding"
	// varyHeader represents the header "Vary"
	varyHeader = "Vary"
	// acceptEncodingHeader represents the header key & value "Accept-Encoding"
	acceptEncodingHeader = "Accept-Encoding"
	// lastModified "Last-Modified"
	lastModified = "Last-Modified"
	// ifModifiedSince "If-Modified-Since"
	ifModifiedSince = "If-Modified-Since"
	// contentDisposition "Content-Disposition"
	contentDisposition = "Content-Disposition"
	// contentBinary header value for binary data.
	contentBinary = "application/octet-stream"
)

func setContentType(res http.ResponseWriter, contentTypeValue string, alternative string) {
	// check if contnet type value is empty
	if contentTypeValue == "" && res.Header().Get("Content-Type") == "" {
		// if it's empty, then set it to  alternative
		contentTypeValue = alternative
	}
	// check if charset part doesn't exists and the file is not binary form
	if !strings.Contains(contentTypeValue, ";charset=") && contentTypeValue != contentBinary {
		// if not, then add this to the value
		contentTypeValue += "; charset=" + Charset
	}
	// set the header
	res.Header().Set(contentTypeHeader, contentTypeValue)
}

func errorHandler(httpStatusCode int) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		line := http.StatusText(httpStatusCode)
		if line == "" {
			line = http.StatusText(http.StatusBadRequest)
		}
		http.Error(res, line, httpStatusCode)
	})
}

/*
  Note:
  If you want to be 100% compatible with http standars you have to put these handlers to both "GET" and "HEAD" HTTP Methods.
*/

// StaticContentHandler returns the net/http.Handler interface to handle raw binary data,
// normally the data parameter was read by custom file reader or by variable
func StaticContentHandler(data []byte, contentType string) http.Handler {
	if len(data) == 0 {
		return errorHandler(http.StatusNoContent)
	}
	modtime := time.Now()

	modtimeStr := modtime.UTC().Format(TimeFormat)
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if t, err := time.Parse(TimeFormat, req.Header.Get(ifModifiedSince)); err == nil && modtime.Before(t.Add(StaticCacheDuration)) {
			res.Header().Del(contentTypeHeader)
			res.Header().Del(contentLength)
			res.WriteHeader(http.StatusNotModified)
			return
		}
		setContentType(res, contentType, contentBinary)
		res.Header().Set(lastModified, modtimeStr)
		res.Write(data)
	})
}

// StaticFileHandler serves a static file such as css,js, favicons, static images
// it stores the file contents to the memory, doesn't supports seek because we read all-in-one the file, but seek is supported by net/http.ServeContent
func StaticFileHandler(filename string) http.Handler {
	fcontents, err := ioutil.ReadFile(filename) // cache the contents of the file, this is the difference from net/http's impl, this is used only for static files, like favicons, css and so on
	if err != nil {
		return errorHandler(http.StatusBadRequest)
	}
	return StaticContentHandler(fcontents, TypeByExtension(filename))
}

// SendStaticFileHandler sends a file for force-download to the client
// it stores the file contents to the memory, doesn't supports seek because we read all-in-one the file, but seek is supported by net/http.ServeContent
func SendStaticFileHandler(filename string) http.Handler {
	staticHandler := StaticFileHandler(filename)
	_, sendfilename := filepath.Split(filename)
	h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		staticHandler.ServeHTTP(res, req)
		res.Header().Set(contentDisposition, "attachment;filename="+sendfilename)
	})

	return h
}

// FaviconHandler receives the favicon path and serves the favicon
func FaviconHandler(favPath string) http.Handler {
	f, err := os.Open(favPath)
	if err != nil {
		panic(errFileOpen.Format(favPath, err.Error()))
	}
	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() { // if it's dir the try to get the favicon.ico
		fav := path.Join(favPath, "favicon.ico")
		f, err = os.Open(fav)
		if err != nil {
			//we try again with .png
			favPath = path.Join(favPath, "favicon.png")
			return FaviconHandler(favPath)
		}
		favPath = fav
		fi, _ = f.Stat()
	}

	cType := TypeByExtension(favPath)
	// copy the bytes here in order to cache and not read the ico on each request.
	cacheFav := make([]byte, fi.Size())
	if _, err = f.Read(cacheFav); err != nil {
		panic(errFileRead.Format(favPath, "Favicon: "+err.Error()))
	}

	return StaticContentHandler(cacheFav, cType)
}

const slash = "/"

// DirHandler serves a directory as web resource
// accepts a system Directory (string),
// a string which will be stripped off if not empty and
// Note 1: this is a dynamic dir handler, means that if a new file is added to the folder it will be served
// Note 2: it doesn't cache the system files, use it with your own risk, otherwise you can use the http.FileServer method, which is different of what I'm trying to do here.
// example:
// staticHandler := http.FileServer(http.Dir("static"))
// http.Handle("/static/", http.StripPrefix("/static/", staticHandler))
// converted to ->
// http.Handle("/static/", fs.DirHandler("./static", "/static/"))
func DirHandler(dir string, strippedPrefix string) http.Handler {
	if dir == "" {
		return errorHandler(http.StatusNoContent)
	}

	dir = strings.Replace(dir, "/", PathSeparator, -1)

	h := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		reqpath := req.URL.Path
		if !strings.HasPrefix(reqpath, "/") {
			reqpath = PathSeparator + reqpath
			req.URL.Path = reqpath
		}
		reqpath = path.Clean(reqpath)
		fpath := reqpath
		relpath, err := filepath.Rel(dir, reqpath)
		if err != nil {
			abspath, err := filepath.Abs(dir + reqpath)
			if err == nil {
				fpath = abspath
			}
		} else {
			fpath = relpath
		}
		http.ServeFile(res, req, fpath)
	})
	// the stripprefix handler checks for empty prefix so
	return http.StripPrefix(strippedPrefix, h)
}
