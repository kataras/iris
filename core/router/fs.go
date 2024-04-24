package router

import (
	"bytes"
	stdContext "context"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v12/context"
)

const indexName = "/index.html"

// DirListFunc is the function signature for customizing directory and file listing.
// See `DirList` and `DirListRich` functions for its implementations.
type DirListFunc func(ctx *context.Context, dirOptions DirOptions, dirName string, dir http.File) error

// Attachments options for files to be downloaded and saved locally by the client.
// See `DirOptions`.
type Attachments struct {
	// Set to true to enable the files to be downloaded and
	// saved locally by the client, instead of serving the file.
	Enable bool
	// Options to send files with a limit of bytes sent per second.
	Limit float64
	Burst int
	// Use this function to change the sent filename.
	NameFunc func(systemName string) (attachmentName string)
}

// DirCacheOptions holds the options for the cached file system.
// See `DirOptions`structure for more.
type DirCacheOptions struct {
	// Enable or disable cache.
	Enable bool
	// Minimum content size for compression in bytes.
	CompressMinSize int64
	// Ignore compress files that match this pattern.
	CompressIgnore *regexp.Regexp
	// The available sever's encodings to be negotiated with the client's needs,
	// common values: gzip, br.
	Encodings []string

	// If greater than zero then prints information about cached files to the stdout.
	// If it's 1 then it prints only the total cached and after-compression reduced file sizes
	// If it's 2 then it prints it per file too.
	Verbose uint8
}

// DirOptions contains the settings that `FileServer` can use to serve files.
// See `DefaultDirOptions`.
type DirOptions struct {
	// Defaults to "/index.html", if request path is ending with **/*/$IndexName
	// then it redirects to **/*(/).
	// That index handler is registered automatically
	// by the framework unless but it can be overridden.
	IndexName string
	// PushTargets filenames (map's value) to
	// be served without additional client's requests (HTTP/2 Push)
	// when a specific request path (map's key WITHOUT prefix)
	// is requested and it's not a directory (it's an `IndexFile`).
	//
	// Example:
	// 	"/": {
	// 		"favicon.ico",
	// 		"js/main.js",
	// 		"css/main.css",
	// 	}
	PushTargets map[string][]string
	// PushTargetsRegexp like `PushTargets` but accepts regexp which
	// is compared against all files under a directory (recursively).
	// The `IndexName` should be set.
	//
	// Example:
	// "/": regexp.MustCompile("((.*).js|(.*).css|(.*).ico)$")
	// See `iris.MatchCommonAssets` too.
	PushTargetsRegexp map[string]*regexp.Regexp

	// Cache to enable in-memory cache and pre-compress files.
	Cache DirCacheOptions
	// When files should served under compression.
	Compress bool

	// List the files inside the current requested
	// directory if `IndexName` not found.
	ShowList bool
	// If `ShowList` is true then this function will be used instead
	// of the default one to show the list of files
	// of a current requested directory(dir).
	// See `DirListRich` package-level function too.
	DirList DirListFunc

	// Show hidden files or directories or not when `ShowList` is true.
	ShowHidden bool

	// Files downloaded and saved locally.
	Attachments Attachments

	// Optional validator that loops through each requested resource.
	AssetValidator func(ctx *context.Context, name string) bool
	// If enabled then the router will render the index file on any not-found file
	// instead of firing the 404 error code handler.
	// Make sure the `IndexName` field is set.
	//
	// Usage:
	//  app.HandleDir("/", iris.Dir("./public"), iris.DirOptions{
	// 	 IndexName: "index.html",
	// 	 SPA:       true,
	//  })
	SPA bool
}

// DefaultDirOptions holds the default settings for `FileServer`.
var DefaultDirOptions = DirOptions{
	IndexName:         indexName,
	PushTargets:       make(map[string][]string),
	PushTargetsRegexp: make(map[string]*regexp.Regexp),
	Cache: DirCacheOptions{
		// Disable by-default.
		Enable: false,
		// Don't compress files smaller than 300 bytes.
		CompressMinSize: 300,
		// Gzip, deflate, br(brotli), snappy.
		Encodings: context.AllEncodings,
		// Log to the stdout (no iris logger) the total reduced file size.
		Verbose: 1,
	},
	Compress: true,
	ShowList: false,
	DirList: DirListRich(DirListRichOptions{
		Tmpl:     DirListRichTemplate,
		TmplName: "dirlist",
	}),
	Attachments: Attachments{
		Enable: false,
		Limit:  0,
		Burst:  0,
	},
	AssetValidator: nil,
	SPA:            false,
}

// FileServer returns a Handler which serves files from a specific file system.
// The first parameter is the file system,
// if it's a `http.Dir` the files should be located near the executable program.
// The second parameter is the settings that the caller can use to customize the behavior.
//
// See `Party#HandleDir` too.
// Examples can be found at: https://github.com/kataras/iris/tree/main/_examples/file-server
func FileServer(fs http.FileSystem, options DirOptions) context.Handler {
	if fs == nil {
		panic("FileServer: fs is nil. The fs parameter should point to a file system of physical system directory or to an embedded one")
	}

	// Make sure index name starts with a slash.
	if options.IndexName != "" {
		options.IndexName = prefix(options.IndexName, "/")
	}

	// Make sure PushTarget's paths are in the proper form.
	for path, filenames := range options.PushTargets {
		for idx, filename := range filenames {
			filenames[idx] = filepath.ToSlash(filename)
		}
		options.PushTargets[path] = filenames
	}

	if !options.Attachments.Enable {
		// make sure rate limiting is not used when attachments are not.
		options.Attachments.Limit = 0
		options.Attachments.Burst = 0
	}

	plainStatusCode := func(ctx *context.Context, statusCode int) {
		if writer, ok := ctx.ResponseWriter().(*context.CompressResponseWriter); ok {
			writer.Disabled = true
		}
		ctx.StatusCode(statusCode)
	}

	dirList := options.DirList
	if dirList == nil {
		dirList = DirList
	}

	open := fsOpener(fs, options.Cache) // We only need its opener, the "fs" is NOT used below.

	h := func(ctx *context.Context) {
		r := ctx.Request()
		name := prefix(r.URL.Path, "/")
		r.URL.Path = name

		var (
			indexFound bool
			noRedirect bool
		)

		f, err := open(name, r)
		if err != nil {
			if options.SPA && name != options.IndexName {
				oldname := name
				name = prefix(options.IndexName, "/") // to match push targets.
				r.URL.Path = name
				f, err = open(name, r) // try find the main index.
				if err != nil {
					r.URL.Path = oldname
					plainStatusCode(ctx, http.StatusNotFound)
					return
				}

				indexFound = true // to support push targets.
				noRedirect = true // to disable redirecting back to /.
			} else {
				plainStatusCode(ctx, http.StatusNotFound)
				return
			}
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
			fIndex, err := open(index, r)
			if err == nil {
				defer fIndex.Close()
				infoIndex, err := fIndex.Stat()
				if err == nil {
					indexFound = true
					f = fIndex
					info = infoIndex
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
			err = dirList(ctx, options, info.Name(), f)
			if err != nil {
				ctx.Application().Logger().Errorf("FileServer: dirList: %v", err)
				plainStatusCode(ctx, http.StatusInternalServerError)
				return
			}

			ctx.Next()
			return
		}

		// index requested, send a moved permanently status
		// and navigate back to the route without the index suffix.
		if !noRedirect && options.IndexName != "" && strings.HasSuffix(name, options.IndexName) {
			localRedirect(ctx, "./")
			return
		}

		if options.AssetValidator != nil {
			if !options.AssetValidator(ctx, name) {
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

		// if not index file and attachments should be force-sent:
		if !indexFound && options.Attachments.Enable {
			destName := info.Name()
			// diposition := "attachment"
			if nameFunc := options.Attachments.NameFunc; nameFunc != nil {
				destName = nameFunc(destName)
			}

			ctx.ResponseWriter().Header().Set(context.ContentDispositionHeaderKey, context.MakeDisposition(destName))
		}

		// the encoding saved from the negotiation.
		encoding, isCached := getFileEncoding(f)
		if isCached {
			// if it's cached and its settings didnt allow this file to be compressed
			// then don't try to compress it on the fly, even if the options.Compress was set to true.
			if encoding != "" {
				if ctx.ResponseWriter().Header().Get(context.ContentEncodingHeaderKey) != "" {
					// disable any compression writer if that header exist,
					// note that, we don't directly check for CompressResponseWriter type
					// because it may be a ResponseRecorder.
					ctx.CompressWriter(false)
				}
				// Set the response header we need, the data are already compressed.
				context.AddCompressHeaders(ctx.ResponseWriter().Header(), encoding)
			}
		} else if options.Compress {
			ctx.CompressWriter(true)
		}

		if indexFound && !options.Attachments.Enable {
			if indexAssets, ok := options.PushTargets[name]; ok {
				if pusher, ok := ctx.ResponseWriter().Naive().(http.Pusher); ok {
					var pushOpts *http.PushOptions
					if encoding != "" {
						pushOpts = &http.PushOptions{Header: r.Header}
					}

					for _, indexAsset := range indexAssets {
						if indexAsset[0] != '/' {
							// it's relative path.
							indexAsset = path.Join(r.RequestURI, indexAsset)
						}

						if err = pusher.Push(indexAsset, pushOpts); err != nil {
							break
						}
					}
				}
			}

			if regex, ok := options.PushTargetsRegexp[r.URL.Path]; ok {
				if pusher, ok := ctx.ResponseWriter().Naive().(http.Pusher); ok {
					var pushOpts *http.PushOptions
					if encoding != "" {
						pushOpts = &http.PushOptions{Header: r.Header}
					}

					prefixURL := strings.TrimSuffix(r.RequestURI, name)
					names, err := context.FindNames(fs, name)
					if err == nil {
						for _, indexAsset := range names {
							// it's an index file, do not pushed that.
							if strings.HasSuffix(prefix(indexAsset, "/"), options.IndexName) {
								continue
							}

							// match using relative path (without the first '/' slash)
							// to keep consistency between the `PushTargets` behavior
							if regex.MatchString(indexAsset) {
								// println("Regex Matched: " + indexAsset)
								if err = pusher.Push(path.Join(prefixURL, indexAsset), pushOpts); err != nil {
									break
								}
							}
						}
					}
				}
			}
		}

		// If limit is 0 then same as ServeContent.
		ctx.ServeContentWithRate(f, info.Name(), info.ModTime(), options.Attachments.Limit, options.Attachments.Burst)
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

	return func(ctx *context.Context) {
		u := ctx.Request().URL
		if p := strings.TrimPrefix(u.Path, canonicalPrefix); len(p) < len(u.Path) {
			if p == "" {
				p = "/"
			}
			u.Path = p
			h(ctx)
		} else {
			ctx.NotFound()
		}
	}
}

func toWebPath(systemPath string) string {
	// winos slash to slash
	webpath := strings.ReplaceAll(systemPath, "\\", "/")
	// double slashes to single
	webpath = strings.ReplaceAll(webpath, "//", "/")
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

func detectOrWriteContentType(ctx *context.Context, name string, content io.ReadSeeker) (string, error) {
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
func localRedirect(ctx *context.Context, newPath string) {
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

// Instead of path.Base(filepath.ToSlash(s))
// let's do something like that, it is faster
// (used to list directories on serve-time too):
func toBaseName(s string) string {
	n := len(s) - 1
	for i := n; i >= 0; i-- {
		if c := s[i]; c == '/' || c == '\\' {
			if i == n {
				// "s" ends with a slash, remove it and retry.
				return toBaseName(s[:n])
			}

			return s[i+1:] // return the rest, trimming the slash.
		}
	}

	return s
}

// IsHidden checks a file is hidden or not
func IsHidden(file os.FileInfo) bool {
	isHidden := false
	if runtime.GOOS == "windows" {
		fa := reflect.ValueOf(file.Sys()).Elem().FieldByName("FileAttributes").Uint()
		bytefa := []byte(strconv.FormatUint(fa, 2))
		if bytefa[len(bytefa)-2] == '1' {
			isHidden = true
		}
	} else {
		isHidden = file.Name()[0] == '.'
	}

	return isHidden
}

// DirList is a `DirListFunc` which renders directories and files in html, but plain, mode.
// See `DirListRich` for more.
func DirList(ctx *context.Context, dirOptions DirOptions, dirName string, dir http.File) error {
	dirs, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	ctx.ContentType(context.ContentHTMLHeaderValue)
	_, err = ctx.WriteString("<div>\n")
	if err != nil {
		return err
	}

	// show current directory
	_, err = ctx.Writef("<h2>Current Directory: %s</h2>", ctx.Request().RequestURI)
	if err != nil {
		return err
	}

	_, err = ctx.WriteString("<ul style=\"list-style: none; padding-left: 20px\">")
	if err != nil {
		return err
	}

	// link to parent directory
	_, err = ctx.WriteString("<li><span style=\"width: 150px; float: left; display: inline-block;\">drwxrwxrwx</span><a href=\"./\">../</a><li>")
	if err != nil {
		return err
	}

	for _, d := range dirs {
		if !dirOptions.ShowHidden && IsHidden(d) {
			continue
		}

		name := toBaseName(d.Name())

		u, err := url.Parse(ctx.Request().RequestURI) // clone url and remove query (#1882).
		if err != nil {
			return fmt.Errorf("name: %s: error: %w", name, err)
		}
		u.RawQuery = ""

		upath := url.URL{Path: path.Join(u.String(), name)}

		downloadAttr := ""
		if dirOptions.Attachments.Enable && !d.IsDir() {
			downloadAttr = " download" // fixes chrome Resource interpreted, other browsers will just ignore this <a> attribute.
		}

		viewName := name
		if d.IsDir() {
			viewName += "/"
		}

		// name may contain '?' or '#', which must be escaped to remain
		// part of the URL path, and not indicate the start of a query
		// string or fragment.
		_, err = ctx.Writef("<li>"+
			"<span style=\"width: 150px; float: left; display: inline-block;\">%s</span>"+
			"<a href=\"%s\"%s>%s</a>"+
			"</li>",
			d.Mode().String(), upath.String(), downloadAttr, html.EscapeString(viewName))
		if err != nil {
			return err
		}
	}
	_, err = ctx.WriteString("</ul></div>\n")
	return err
}

// DirListRichOptions the options for the `DirListRich` helper function.
type DirListRichOptions struct {
	// If not nil then this template's "dirlist" is used to render the listing page.
	Tmpl *template.Template
	// If not empty then this view file is used to render the listing page.
	// The view should be registered with `Application.RegisterView`.
	// E.g. "dirlist.html"
	TmplName string
}

// DirListRich is a `DirListFunc` which can be passed to `DirOptions.DirList` field
// to override the default file listing appearance.
// See `DirListRichTemplate` to modify the template, if necessary.
func DirListRich(opts ...DirListRichOptions) DirListFunc {
	var options DirListRichOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.TmplName == "" && options.Tmpl == nil {
		options.Tmpl = DirListRichTemplate
	}

	return func(ctx *context.Context, dirOptions DirOptions, dirName string, dir http.File) error {
		dirs, err := dir.Readdir(-1)
		if err != nil {
			return err
		}

		sortBy := ctx.URLParam("sort")
		switch sortBy {
		case "name":
			sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
		case "size":
			sort.Slice(dirs, func(i, j int) bool { return dirs[i].Size() < dirs[j].Size() })
		default:
			sort.Slice(dirs, func(i, j int) bool { return dirs[i].ModTime().After(dirs[j].ModTime()) })
		}

		pageData := listPageData{
			Title: fmt.Sprintf("List of %d files", len(dirs)),
			Files: make([]fileInfoData, 0, len(dirs)),
		}

		for _, d := range dirs {
			if !dirOptions.ShowHidden && IsHidden(d) {
				continue
			}

			name := toBaseName(d.Name())

			u, err := url.Parse(ctx.Request().RequestURI) // clone url and remove query (#1882).
			if err != nil {
				return fmt.Errorf("name: %s: error: %w", name, err)
			}
			u.RawQuery = ""

			upath := url.URL{Path: path.Join(u.String(), name)}

			viewName := name
			if d.IsDir() {
				viewName += "/"
			}

			shouldDownload := dirOptions.Attachments.Enable && !d.IsDir()
			pageData.Files = append(pageData.Files, fileInfoData{
				Info:     d,
				ModTime:  d.ModTime().UTC().Format(http.TimeFormat),
				Path:     upath.String(),
				RelPath:  path.Join(ctx.Path(), name),
				Name:     html.EscapeString(viewName),
				Download: shouldDownload,
			})
		}

		if options.TmplName != "" {
			return ctx.View(options.TmplName, pageData)
		}

		return options.Tmpl.ExecuteTemplate(ctx, "dirlist", pageData)
	}
}

type (
	listPageData struct {
		Title string // the document's title.
		Files []fileInfoData
	}

	fileInfoData struct {
		Info     os.FileInfo
		ModTime  string // format-ed time.
		Path     string // the request path.
		RelPath  string // file path without the system directory itself (we are not exposing it to the user).
		Name     string // the html-escaped name.
		Download bool   // the file should be downloaded (attachment instead of inline view).
	}
)

// FormatBytes returns a string representation of the "b" bytes.
func FormatBytes(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

// DirListRichTemplate is the html template the `DirListRich` function is using to render
// the directories and files.
var DirListRichTemplate = template.Must(template.New("dirlist").
	Funcs(template.FuncMap{
		"formatBytes": FormatBytes,
	}).Parse(`
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>{{.Title}}</title>
    <style>
        a {
            padding: 8px 8px;
            text-decoration:none;
            cursor:pointer;
            color: #10a2ff;
        }
        table {
            position: absolute;
            top: 0;
            bottom: 0;
            left: 0;
            right: 0;
            height: 100%;
            width: 100%;
            border-collapse: collapse;
            border-spacing: 0;
            empty-cells: show;
            border: 1px solid #cbcbcb;
        }
        
        table caption {
            color: #000;
            font: italic 85%/1 arial, sans-serif;
            padding: 1em 0;
            text-align: center;
        }
        
        table td,
        table th {
            border-left: 1px solid #cbcbcb;
            border-width: 0 0 0 1px;
            font-size: inherit;
            margin: 0;
            overflow: visible;
            padding: 0.5em 1em;
        }
        
        table thead {
            background-color: #10a2ff;
            color: #fff;
            text-align: left;
            vertical-align: bottom;
        }
        
        table td {
            background-color: transparent;
        }

        .table-odd td {
            background-color: #f2f2f2;
        }

        .table-bordered td {
            border-bottom: 1px solid #cbcbcb;
        }
        .table-bordered tbody > tr:last-child > td {
            border-bottom-width: 0;
        }
	</style>
</head>
<body>
    <table class="table-bordered table-odd">
        <thead>
            <tr>
                <th>#</th>
                <th>Name</th>
				<th>Size</th>
            </tr>
        </thead>
        <tbody>
            {{ range $idx, $file := .Files }}
            <tr>
                <td>{{ $idx }}</td>
                {{ if $file.Download }}
                <td><a href="{{ $file.Path }}" title="{{ $file.ModTime }}" download>{{ $file.Name }}</a></td> 
                {{ else }}
                <td><a href="{{ $file.Path }}" title="{{ $file.ModTime }}">{{ $file.Name }}</a></td>
                {{ end }}
				{{ if $file.Info.IsDir }}
				<td>Dir</td>
				{{ else }}
				<td>{{ formatBytes $file.Info.Size }}</td>
				{{ end }}
            </tr>
            {{ end }}
        </tbody>
	</table>
</body></html>
`))

// fsOpener returns the file system opener, cached one or the original based on the options Enable field.
func fsOpener(fs http.FileSystem, options DirCacheOptions) func(name string, r *http.Request) (http.File, error) {
	if !options.Enable {
		// if it's not enabled return the opener original one.
		return func(name string, _ *http.Request) (http.File, error) {
			return fs.Open(name)
		}
	}

	c, err := cache(fs, options)
	if err != nil {
		panic(err)
	}
	return c.Ropen
}

// cache returns a http.FileSystem which serves in-memory cached (compressed) files.
// Look `Verbose` function to print out information while in development status.
func cache(fs http.FileSystem, options DirCacheOptions) (*cacheFS, error) {
	start := time.Now()

	names, err := context.FindNames(fs, "/")
	if err != nil {
		return nil, err
	}

	sort.Slice(names, func(i, j int) bool {
		return strings.Count(names[j], "/") > strings.Count(names[i], "/")
	})

	dirs, err := findDirs(fs, names)
	if err != nil {
		return nil, err
	}

	files, err := cacheFiles(stdContext.Background(), fs, names,
		options.Encodings, options.CompressMinSize, options.CompressIgnore)
	if err != nil {
		return nil, err
	}

	ttc := time.Since(start)

	c := &cacheFS{dirs: dirs, files: files, algs: options.Encodings}
	go logCacheFS(c, ttc, len(names), options.Verbose)

	return c, nil
}

func logCacheFS(fs *cacheFS, ttc time.Duration, n int, level uint8) {
	if level == 0 {
		return
	}

	var (
		totalLength             int64
		totalCompressedLength   = make(map[string]int64)
		totalCompressedContents int64
	)

	for name, f := range fs.files {
		uncompressed := f.algs[""]
		totalLength += int64(len(uncompressed))

		if level == 2 {
			fmt.Printf("%s (%s)\n", name, FormatBytes(int64(len(uncompressed))))
		}

		for alg, contents := range f.algs {
			if alg == "" {
				continue
			}

			totalCompressedContents++

			if len(alg) < 7 {
				alg += strings.Repeat(" ", 7-len(alg))
			}
			totalCompressedLength[alg] += int64(len(contents))

			if level == 2 {
				fmt.Printf("%s (%s)\n", alg, FormatBytes(int64(len(contents))))
			}
		}
	}

	fmt.Printf("Time to complete the compression and caching of [%d/%d] files: %s\n", totalCompressedContents/int64(len(fs.algs)), n, ttc)
	fmt.Printf("Total size reduced from %s to:\n", FormatBytes(totalLength))
	for alg, length := range totalCompressedLength {
		// https://en.wikipedia.org/wiki/Data_compression_ratio
		reducedRatio := 1 - float64(length)/float64(totalLength)
		fmt.Printf("%s (%s) [%.2f%%]\n", alg, FormatBytes(length), reducedRatio*100)
	}
}

type cacheFS struct {
	dirs  map[string]*dir
	files fileMap
	algs  []string
}

var _ http.FileSystem = (*cacheFS)(nil)

// Open returns the http.File based on "name".
// If file, it always returns a cached file of uncompressed data.
// See `Ropen` too.
func (c *cacheFS) Open(name string) (http.File, error) {
	// we always fetch with the sep,
	// as http requests will do,
	// and the filename's info.Name() is always base
	// and without separator prefix
	// (keep note, we need that fileInfo
	// wrapper because go-bindata's Name originally
	// returns the fullname while the http.Dir returns the basename).
	if name == "" || name[0] != '/' {
		name = "/" + name
	}

	if d, ok := c.dirs[name]; ok {
		return d, nil
	}

	if f, ok := c.files[name]; ok {
		return f.Get("")
	}

	return nil, os.ErrNotExist
}

// Ropen returns the http.File based on "name".
// If file, it negotiates the content encoding,
// based on the given algorithms, and
// returns the cached file with compressed data,
// if the encoding was empty then it
// returns the cached file with its original, uncompressed data.
//
// A check of `GetEncoding(file)` is required to set
// response headers.
//
// Note: We don't require a response writer to set the headers
// because the caller of this method may stop the operation
// before file's contents are written to the client.
func (c *cacheFS) Ropen(name string, r *http.Request) (http.File, error) {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}

	if d, ok := c.dirs[name]; ok {
		return d, nil
	}

	if f, ok := c.files[name]; ok {
		encoding, _ := context.GetEncoding(r, c.algs)
		return f.Get(encoding)
	}

	return nil, os.ErrNotExist
}

// getFileEncoding returns the encoding of an http.File.
// If the "f" file was created by a `Cache` call then
// it returns the content encoding that this file was cached with.
// It returns empty string for files that
// were too small or ignored to be compressed.
//
// It also reports whether the "f" is a cached file or not.
func getFileEncoding(f http.File) (string, bool) {
	if f == nil {
		return "", false
	}

	ff, ok := f.(*file)
	if !ok {
		return "", false
	}

	return ff.alg, true
}

// type fileMap map[string] /* path */ map[string] /*compression alg or empty for original */ []byte /*contents */
type fileMap map[string]*file

func cacheFiles(ctx stdContext.Context, fs http.FileSystem, names []string, compressAlgs []string, compressMinSize int64, compressIgnore *regexp.Regexp) (fileMap, error) {
	ctx, cancel := stdContext.WithCancel(ctx)
	defer cancel()

	list := make(fileMap, len(names))
	mutex := new(sync.Mutex)

	cache := func(name string) error {
		f, err := fs.Open(name)
		if err != nil {
			return err
		}

		inf, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}

		fi := newFileInfo(path.Base(name), inf.Mode(), inf.ModTime())

		contents, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return err
		}

		algs := make(map[string][]byte, len(compressAlgs)+1)
		algs[""] = contents // original contents.

		mutex.Lock()
		list[name] = newFile(name, fi, algs)
		mutex.Unlock()
		if compressMinSize > 0 && compressMinSize > int64(len(contents)) {
			return nil
		}

		if compressIgnore != nil && compressIgnore.MatchString(name) {
			return nil
		}

		// Note:
		// We can fire a new goroutine for each compression of the same file
		// but this will have an impact on CPU cost if
		// thousands of files running 4 compressions at the same time,
		// so, unless requested keep it as it's.
		buf := new(bytes.Buffer)
		for _, alg := range compressAlgs {
			select {
			case <-ctx.Done():
				return ctx.Err() // stop all compressions if at least one file failed to.
			default:
			}

			if alg == "brotli" {
				alg = "br"
			}

			w, err := context.NewCompressWriter(buf, strings.ToLower(alg), -1)
			if err != nil {
				return err
			}
			_, err = w.Write(contents)
			w.Close()
			if err != nil {
				return err
			}

			bs := buf.Bytes()
			dest := make([]byte, len(bs))
			copy(dest, bs)
			algs[alg] = dest

			buf.Reset()
		}

		return nil
	}

	var (
		err     error
		wg      sync.WaitGroup
		errOnce sync.Once
	)

	for _, name := range names {
		wg.Add(1)

		go func(name string) {
			defer wg.Done()

			if fnErr := cache(name); fnErr != nil {
				errOnce.Do(func() {
					err = fnErr
					cancel()
				})
			}
		}(name)
	}

	wg.Wait()
	return list, err
}

type cacheStoreFile interface {
	Get(compressionAlgorithm string) (http.File, error)
}

type file struct {
	io.ReadSeeker                   // nil on cache store and filled on file Get.
	algs          map[string][]byte // non empty for store and nil for files.
	alg           string            // empty for cache store, filled with the compression algorithm of this file (useful to decompress).
	name          string
	baseName      string
	info          os.FileInfo
}

var (
	_ http.File      = (*file)(nil)
	_ cacheStoreFile = (*file)(nil)
)

func newFile(name string, fi os.FileInfo, algs map[string][]byte) *file {
	return &file{
		name:     name,
		baseName: path.Base(name),
		info:     fi,
		algs:     algs,
	}
}

func (f *file) Close() error                             { return nil }
func (f *file) Readdir(count int) ([]os.FileInfo, error) { return nil, os.ErrNotExist }
func (f *file) Stat() (os.FileInfo, error)               { return f.info, nil }

// Get returns a new http.File to be served.
// Caller should check if a specific http.File has this method as well.
func (f *file) Get(alg string) (http.File, error) {
	// The "alg" can be empty for non-compressed file contents.
	// We don't need a new structure.

	if contents, ok := f.algs[alg]; ok {
		return &file{
			name:       f.name,
			baseName:   f.baseName,
			info:       f.info,
			alg:        alg,
			ReadSeeker: bytes.NewReader(contents),
		}, nil
	}

	// When client accept compression but cached contents are not compressed,
	// e.g. file too small or ignored one.
	return f.Get("")
}

type fileInfo struct {
	baseName string
	modTime  time.Time
	isDir    bool
	mode     os.FileMode
}

var _ os.FileInfo = (*fileInfo)(nil)

func newFileInfo(baseName string, mode os.FileMode, modTime time.Time) *fileInfo {
	return &fileInfo{
		baseName: baseName,
		modTime:  modTime,
		mode:     mode,
		isDir:    mode == os.ModeDir,
	}
}

func (fi *fileInfo) Close() error       { return nil }
func (fi *fileInfo) Name() string       { return fi.baseName }
func (fi *fileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *fileInfo) ModTime() time.Time { return fi.modTime }
func (fi *fileInfo) IsDir() bool        { return fi.isDir }
func (fi *fileInfo) Size() int64        { return 0 }
func (fi *fileInfo) Sys() interface{}   { return fi }

type dir struct {
	os.FileInfo   // *fileInfo
	io.ReadSeeker // nil

	name     string // fullname, for any case.
	baseName string
	children []os.FileInfo // a slice of *fileInfo
}

var (
	_ os.FileInfo = (*dir)(nil)
	_ http.File   = (*dir)(nil)
)

func (d *dir) Close() error               { return nil }
func (d *dir) Name() string               { return d.baseName }
func (d *dir) Stat() (os.FileInfo, error) { return d.FileInfo, nil }

func (d *dir) Readdir(count int) ([]os.FileInfo, error) {
	return d.children, nil
}

func newDir(fi os.FileInfo, fullname string) *dir {
	baseName := path.Base(fullname)
	return &dir{
		FileInfo: newFileInfo(baseName, os.ModeDir, fi.ModTime()),
		name:     fullname,
		baseName: baseName,
	}
}

var _ http.File = (*dir)(nil)

// returns unorderded map of directories both reclusive and flat.
func findDirs(fs http.FileSystem, names []string) (map[string]*dir, error) {
	dirs := make(map[string]*dir)

	for _, name := range names {
		f, err := fs.Open(name)
		if err != nil {
			return nil, err
		}
		inf, err := f.Stat()
		if err != nil {
			return nil, err
		}

		dirName := path.Dir(name)
		d, ok := dirs[dirName]
		if !ok {
			fi := newFileInfo(path.Base(dirName), os.ModeDir, inf.ModTime())
			d = newDir(fi, dirName)
			dirs[dirName] = d
		}

		fi := newFileInfo(path.Base(name), inf.Mode(), inf.ModTime())

		// Add the directory file info (=this dir) to the parent one,
		// so `ShowList` can render sub-directories of this dir.
		parentName := path.Dir(dirName)
		if parent, hasParent := dirs[parentName]; hasParent {
			parent.children = append(parent.children, d)
		}

		d.children = append(d.children, fi)
	}

	return dirs, nil
}
