package router

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
)

const indexName = "/index.html"

// DirOptions contains the optional settings that
// `FileServer` and `Party#HandleDir` can use to serve files and assets.
type DirOptions struct {
	// Defaults to "/index.html", if request path is ending with **/*/$IndexName
	// then it redirects to **/*(/) which another handler is handling it,
	// that another handler, called index handler, is auto-registered by the framework
	// if end developer does not managed to handle it by hand.
	IndexName string
	// When files should served under compression.
	Gzip bool

	// List the files inside the current requested directory if `IndexName` not found.
	ShowList bool
	// If `ShowList` is true then this function will be used instead of the default one to show the list of files of a current requested directory(dir).
	DirList func(ctx context.Context, dirName string, dir http.File) error

	// When embedded.
	Asset      func(name string) ([]byte, error)      // we need this to make it compatible os.File.
	AssetInfo  func(name string) (os.FileInfo, error) // we need this for range support on embedded files.
	AssetNames func() []string                        // called once.

	// Optional validator that loops through each requested resource.
	AssetValidator func(ctx context.Context, name string) bool
}

func getDirOptions(opts ...DirOptions) (options DirOptions) {
	if len(opts) > 0 {
		options = opts[0]
	}

	if options.IndexName == "" {
		options.IndexName = indexName
	} else {
		options.IndexName = prefix(options.IndexName, "/")
	}

	return
}

type embeddedFile struct {
	os.FileInfo
	io.ReadSeeker
}

var _ http.File = (*embeddedFile)(nil)

func (f *embeddedFile) Close() error {
	return nil
}

// func (f *embeddedFile) Readdir(count int) ([]os.FileInfo, error) {
// 	// this should never happen, show dirs is already checked on the handler level before this call.
// 	if count != -1 {
// 		return nil, nil
// 	}

// 	list := make([]os.FileInfo, len(f.dir.assetNames))
// 	var err error
// 	for i, name := range f.dir.assetNames {
// 		list[i], err = f.dir.assetInfo(name)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
// 	return list, nil
// }

func (f *embeddedFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil // should never happen, read directories is done by `embeddedDir`.
}

func (f *embeddedFile) Stat() (os.FileInfo, error) {
	return f.FileInfo, nil
}

// func (f *embeddedFile) Name() string {
// 	return strings.TrimPrefix(f.vdir, f.FileInfo.Name())
// }

type embeddedFileSystem struct {
	vdir     string
	dirNames map[string]*embeddedDir // embedded tools doesn't give that info, so we initialize it in order to support `ShowList` on embedded files as well.

	asset     func(name string) ([]byte, error)
	assetInfo func(name string) (os.FileInfo, error)
}

var _ http.FileSystem = (*embeddedFileSystem)(nil)

func (fs *embeddedFileSystem) Open(name string) (http.File, error) {
	// name = fs.vdir + name <- no need, check the TrimPrefix(name, vdir) on names loop and the asset and assetInfo redefined on `HandleDir`.
	if d, ok := fs.dirNames[name]; ok {
		return d, nil
	}

	info, err := fs.assetInfo(name)
	if err != nil {
		return nil, err
	}
	b, err := fs.asset(name)
	if err != nil {
		return nil, err
	}
	return &embeddedFile{
		FileInfo:   info,
		ReadSeeker: bytes.NewReader(b),
	}, nil
}

type embeddedBaseFileInfo struct {
	baseName string
	os.FileInfo
}

func (info *embeddedBaseFileInfo) Name() string {
	return info.baseName
}

type embeddedDir struct {
	name          string
	modTimeUnix   int64
	list          []os.FileInfo
	*bytes.Reader // never used, will always be nil.
}

var _ http.File = (*embeddedDir)(nil)

func (f *embeddedDir) Close() error               { return nil }
func (f *embeddedDir) Name() string               { return f.name }
func (f *embeddedDir) Size() int64                { return 0 }
func (f *embeddedDir) Mode() os.FileMode          { return os.ModeDir }
func (f *embeddedDir) ModTime() time.Time         { return time.Unix(f.modTimeUnix, 0) }
func (f *embeddedDir) IsDir() bool                { return true }
func (f *embeddedDir) Sys() interface{}           { return f }
func (f *embeddedDir) Stat() (os.FileInfo, error) { return f, nil }

func (f *embeddedDir) Readdir(count int) ([]os.FileInfo, error) {
	// this should never happen, show dirs is already checked on the handler level before this call.
	if count != -1 {
		return nil, nil
	}

	return f.list, nil
}

// FileServer returns a Handler which serves files from a specific system, phyisical, directory
// or an embedded one.
// The first parameter is the directory, relative to the executable program.
// The second optional parameter is any optional settings that the caller can use.
//
// See `Party#HandleDir` too.
// Examples can be found at: https://github.com/kataras/iris/tree/master/_examples/file-server
func FileServer(directory string, opts ...DirOptions) context.Handler {
	if directory == "" {
		panic("FileServer: directory is empty. The directory parameter should point to a physical system directory or to an embedded one")
	}

	options := getDirOptions(opts...)

	// `embeddedFileSystem` (if AssetInfo, Asset and AssetNames are defined) or `http.Dir`.
	var fs http.FileSystem = http.Dir(directory)

	if options.Asset != nil && options.AssetInfo != nil && options.AssetNames != nil {
		// Depends on the command the user gave to the go-bindata
		// the assset path (names) may be or may not be prepended with a slash.
		// What we do: we remove the ./ from the vdir which should be
		// the same with the asset path (names).
		// we don't pathclean, because that will prepend a slash
		//					   go-bindata should give a correct path format.
		// On serve time we check the "paramName" (which is the path after the "requestPath")
		// so it has the first directory part missing, we use the "vdir" to complete it
		// and match with the asset path (names).
		vdir := directory

		if vdir[0] == '.' {
			vdir = vdir[1:]
		}

		// second check for /something, (or ./something if we had dot on 0 it will be removed)
		if vdir[0] == '/' || vdir[0] == os.PathSeparator {
			vdir = vdir[1:]
		}

		// check for trailing slashes because new users may be do that by mistake
		// although all examples are showing the correct way but you never know
		// i.e "./assets/" is not correct, if was inside "./assets".
		// remove last "/".
		if trailingSlashIdx := len(vdir) - 1; vdir[trailingSlashIdx] == '/' {
			vdir = vdir[0:trailingSlashIdx]
		}

		// select only the paths that we care;
		// that have prefix of the directory and
		// skip any unnecessary the end-dev or the 3rd party tool may set.
		var names []string
		for _, name := range options.AssetNames() {
			// i.e: name = static/css/main.css (including the directory, see `embeddedFileSystem.vdir`)

			if !strings.HasPrefix(name, vdir) {
				continue
			}

			names = append(names, strings.TrimPrefix(name, vdir))
		}

		if len(names) == 0 {
			panic("FileServer: zero embedded files")
		}

		asset := func(name string) ([]byte, error) {
			return options.Asset(vdir + name)
		}

		assetInfo := func(name string) (os.FileInfo, error) {
			return options.AssetInfo(vdir + name)
		}

		dirNames := make(map[string]*embeddedDir)

		// sort filenames by smaller path.
		sort.Slice(names, func(i, j int) bool {
			return strings.Count(names[j], "/") > strings.Count(names[i], "/")
		})

		for _, name := range names {
			dirName := path.Dir(name)
			d, ok := dirNames[dirName]

			if !ok {
				d = &embeddedDir{
					name:        dirName,
					modTimeUnix: time.Now().Unix(),
				}
				dirNames[dirName] = d
			}

			info, err := assetInfo(name)
			if err != nil {
				panic(fmt.Sprintf("FileServer: report as bug: file info: %s not found in: %s", name, dirName))
			}
			d.list = append(d.list, &embeddedBaseFileInfo{path.Base(name), info})
		}

		fs = &embeddedFileSystem{
			vdir:     vdir,
			dirNames: dirNames,

			asset:     asset,
			assetInfo: assetInfo,
		}
	}
	// Let it for now.
	// else if !DirectoryExists(directory) {
	// 	panic("FileServer: system directory: " + directory + " does not exist")
	// }

	plainStatusCode := func(ctx context.Context, statusCode int) {
		if writer, ok := ctx.ResponseWriter().(*context.GzipResponseWriter); ok && writer != nil {
			writer.ResetBody()
			writer.Disable()
		}
		ctx.StatusCode(statusCode)
	}

	htmlReplacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		// "&#34;" is shorter than "&quot;".
		`"`, "&#34;",
		// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
		"'", "&#39;",
	)

	dirList := options.DirList
	if dirList == nil {
		dirList = func(ctx context.Context, dirName string, dir http.File) error {
			dirs, err := dir.Readdir(-1)
			if err != nil {
				return err
			}

			// dst, _ := dir.Stat()
			// dirName := dst.Name()

			sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

			ctx.ContentType(context.ContentHTMLHeaderValue)
			_, err = ctx.WriteString("<pre>\n")
			if err != nil {
				return err
			}
			for _, d := range dirs {
				name := d.Name()
				if d.IsDir() {
					name += "/"
				}
				// name may contain '?' or '#', which must be escaped to remain
				// part of the URL path, and not indicate the start of a query
				// string or fragment.
				url := url.URL{Path: joinPath("./"+dirName, name)} // edit here to redirect correctly, standard library misses that.
				_, err = ctx.Writef("<a href=\"%s\">%s</a>\n", url.String(), htmlReplacer.Replace(name))
				if err != nil {
					return err
				}
			}
			_, err = ctx.WriteString("</pre>\n")
			return err
		}
	}

	h := func(ctx context.Context) {
		name := prefix(ctx.Request().URL.Path, "/")
		ctx.Request().URL.Path = name

		gzip := options.Gzip
		if !gzip {
			// if false then check if the dev did something like `ctx.Gzip(true)`.
			_, gzip = ctx.ResponseWriter().(*context.GzipResponseWriter)
		}

		f, err := fs.Open(name)
		if err != nil {
			plainStatusCode(ctx, http.StatusNotFound)
			return
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			plainStatusCode(ctx, http.StatusNotFound)
			return
		}

		// use contents of index.html for directory, if present
		if info.IsDir() && options.IndexName != "" {
			// Note that, in contrast of the default net/http mechanism;
			// here different handlers may serve the indexes
			// if manually then this will block will never fire,
			// if index handler are automatically registered by the framework
			// then this block will be fired on indexes because the static site routes are registered using the static route's handler.
			//
			// End-developers must have the chance to register different logic and middlewares
			// to an index file, useful on Single Page Applications.

			index := strings.TrimSuffix(name, "/") + options.IndexName
			fIndex, err := fs.Open(index)
			if err == nil {
				defer fIndex.Close()
				infoIndex, err := fIndex.Stat()
				if err == nil {
					info = infoIndex
					f = fIndex
				}
			}
		}

		// Still a directory? (we didn't find an index.html file)
		if info.IsDir() {
			if !options.ShowList {
				plainStatusCode(ctx, http.StatusNotFound)
				return
			}
			if modified, err := ctx.CheckIfModifiedSince(info.ModTime()); !modified && err == nil {
				ctx.WriteNotModified()
				ctx.StatusCode(http.StatusNotModified)
				ctx.Next()
				return
			}
			ctx.SetLastModified(info.ModTime())
			err = dirList(ctx, info.Name(), f)
			if err != nil {
				plainStatusCode(ctx, http.StatusInternalServerError)
				return
			}

			ctx.Next()
			return
		}

		// index requested, send a moved permanently status
		// and navigate back to the route without the index suffix.
		if strings.HasSuffix(name, options.IndexName) {
			localRedirect(ctx, "./")
			return
		}

		if options.AssetValidator != nil {
			if !options.AssetValidator(ctx, info.Name()) {
				errCode := ctx.GetStatusCode()
				if ctx.ResponseWriter().Written() <= context.StatusCodeWritten {
					// if nothing written as body from the AssetValidator but 200 status code(which is the default),
					// then we assume that the end-developer just returned false expecting this to be not found.
					if errCode == http.StatusOK {
						errCode = http.StatusNotFound
					}
					plainStatusCode(ctx, errCode)
				}
				return
			}
		}

		// try to find and send the correct content type based on the filename
		// and the binary data inside "f".
		detectOrWriteContentType(ctx, info.Name(), f)

		if gzip {
			// set the last modified as "serveContent" does.
			ctx.SetLastModified(info.ModTime())

			// write the file to the response writer.
			contents, err := ioutil.ReadAll(f)
			if err != nil {
				ctx.Application().Logger().Debugf("err reading file: %v", err)
				plainStatusCode(ctx, http.StatusInternalServerError)
				return
			}

			// Use `WriteNow` instead of `Write`
			// because we need to know the compressed written size before
			// the `FlushResponse`.
			_, err = ctx.GzipResponseWriter().Write(contents)
			if err != nil {
				ctx.Application().Logger().Debugf("short write: %v", err)
				plainStatusCode(ctx, http.StatusInternalServerError)
				return
			}
			return
		}

		http.ServeContent(ctx.ResponseWriter(), ctx.Request(), info.Name(), info.ModTime(), f)
		if serveCode := ctx.GetStatusCode(); context.StatusCodeNotSuccessful(serveCode) {
			plainStatusCode(ctx, serveCode)
			return
		}

		ctx.Next() // fire any middleware, if any.
	}

	return h
}

// StripPrefix returns a handler that serves HTTP requests
// by removing the given prefix from the request URL's Path
// and invoking the handler h. StripPrefix handles a
// request for a path that doesn't begin with prefix by
// replying with an HTTP 404 not found error.
//
// Usage:
// fileserver := FileServer("./static_files", DirOptions {...})
// h := StripPrefix("/static", fileserver)
// app.Get("/static/{file:path}", h)
// app.Head("/static/{file:path}", h)
func StripPrefix(prefix string, h context.Handler) context.Handler {
	if prefix == "" {
		return h
	}
	// here we separate the path from the subdomain (if any), we care only for the path
	// fixes a bug when serving static files via a subdomain
	canonicalPrefix := prefix
	if dotWSlashIdx := strings.Index(canonicalPrefix, SubdomainPrefix); dotWSlashIdx > 0 {
		canonicalPrefix = canonicalPrefix[dotWSlashIdx+1:]
	}
	canonicalPrefix = toWebPath(canonicalPrefix)

	return func(ctx context.Context) {
		if p := strings.TrimPrefix(ctx.Request().URL.Path, canonicalPrefix); len(p) < len(ctx.Request().URL.Path) {
			ctx.Request().URL.Path = p
			h(ctx)
		} else {
			ctx.NotFound()
		}
	}
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

// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like Redirect does.
func localRedirect(ctx context.Context, newPath string) {
	if q := ctx.Request().URL.RawQuery; q != "" {
		newPath += "?" + q
	}

	ctx.Header("Location", newPath)
	ctx.StatusCode(http.StatusMovedPermanently)
}

// DirectoryExists returns true if a directory(or file) exists, otherwise false
func DirectoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}
