package router

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"io"
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

// DirListFunc is the function signature for customizing directory and file listing.
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

// DirOptions contains the optional settings that
// `FileServer` and `Party#HandleDir` can use to serve files and assets.
type DirOptions struct {
	// Defaults to "/index.html", if request path is ending with **/*/$IndexName
	// then it redirects to **/*(/) which another handler is handling it,
	// that another handler, called index handler, is auto-registered by the framework
	// if end developer does not managed to handle it by hand.
	IndexName string
	// PushTargets optionally absolute filenames (map's value) to be served without any
	// additional client's requests (HTTP/2 Push)
	// when a specific path (map's key) is requested and
	// it's not a directory (it's an `IndexFile`).
	PushTargets map[string][]string
	// When files should served under compression.
	Compress bool

	// List the files inside the current requested directory if `IndexName` not found.
	ShowList bool
	// If `ShowList` is true then this function will be used instead
	// of the default one to show the list of files of a current requested directory(dir).
	// See `DirListRich` package-level function too.
	DirList DirListFunc

	// Files downloaded and saved locally.
	Attachments Attachments

	// When embedded.
	Asset      func(name string) ([]byte, error)      // we need this to make it compatible os.File.
	AssetInfo  func(name string) (os.FileInfo, error) // we need this for range support on embedded files.
	AssetNames func() []string                        // called once.

	// Optional validator that loops through each requested resource.
	AssetValidator func(ctx *context.Context, name string) bool
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

	if !options.Attachments.Enable {
		// make sure rate limiting is not used when attachments are not.
		options.Attachments.Limit = 0
		options.Attachments.Burst = 0
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
	if name != "/" {
		// http://localhost:8080/app2/app2app3/dirs/
		// = http://localhost:8080/app2/app2app3/dirs
		name = strings.TrimSuffix(name, "/")
	}

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
	baseName      string
	modTimeUnix   int64
	list          []os.FileInfo
	*bytes.Reader // never used, will always be nil.
}

var _ http.File = (*embeddedDir)(nil)

func (f *embeddedDir) Close() error               { return nil }
func (f *embeddedDir) Name() string               { return f.baseName }
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
					baseName:    path.Base(dirName),
					modTimeUnix: time.Now().Unix(),
				}
				dirNames[dirName] = d
			}

			info, err := assetInfo(name)
			if err != nil {
				panic(fmt.Sprintf("FileServer: report as bug: file info: %s not found in: %s", name, dirName))
			}

			// Add the directory file info (=this dir) to the parent one,
			// so `ShowList` can render sub-directories of this dir.
			if parent, hasParent := dirNames[path.Dir(dirName)]; hasParent {
				parent.list = append(parent.list, d)
			}

			f := &embeddedBaseFileInfo{path.Base(name), info}
			d.list = append(d.list, f)
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

	plainStatusCode := func(ctx *context.Context, statusCode int) {
		if writer, ok := ctx.ResponseWriter().(*context.CompressResponseWriter); ok {
			writer.Disabled = true
		}
		ctx.StatusCode(statusCode)
	}

	dirList := options.DirList
	if dirList == nil {
		dirList = func(ctx *context.Context, dirOptions DirOptions, dirName string, dir http.File) error {
			dirs, err := dir.Readdir(-1)
			if err != nil {
				return err
			}

			sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

			ctx.ContentType(context.ContentHTMLHeaderValue)
			_, err = ctx.WriteString("<pre>\n")
			if err != nil {
				return err
			}

			for _, d := range dirs {
				name := d.Name()

				upath := ""
				if !strings.HasSuffix(ctx.Path(), "/") && dirName != "" {
					upath = "./" + path.Base(dirName) + "/" + name
				} else {
					upath = "./" + name
				}

				url := url.URL{Path: upath}

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
				_, err = ctx.Writef("<a href=\"%s\"%s>%s</a>\n", url.String(), downloadAttr, html.EscapeString(viewName))
				if err != nil {
					return err
				}
			}
			_, err = ctx.WriteString("</pre>\n")
			return err
		}
	}

	h := func(ctx *context.Context) {
		name := prefix(ctx.Request().URL.Path, "/")
		ctx.Request().URL.Path = name

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

		indexFound := false
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
					indexFound = true
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

		// if not index file and attachments should be force-sent:
		if !indexFound && options.Attachments.Enable {
			destName := info.Name()
			// diposition := "attachment"
			if nameFunc := options.Attachments.NameFunc; nameFunc != nil {
				destName = nameFunc(destName)
			}

			ctx.ResponseWriter().Header().Set(context.ContentDispositionHeaderKey, "attachment;filename="+destName)
		}

		ctx.Compress(options.Compress)

		if indexFound && len(options.PushTargets) > 0 && !options.Attachments.Enable {
			if indexAssets, ok := options.PushTargets[name]; ok {
				if pusher, ok := ctx.ResponseWriter().(http.Pusher); ok {
					for _, indexAsset := range indexAssets {
						pusher.Push(indexAsset, nil)
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
			name := d.Name()

			upath := ""
			if !strings.HasSuffix(ctx.Path(), "/") && dirName != "" {
				upath = "./" + path.Base(dirName) + "/" + name
			} else {
				upath = "./" + name
			}

			url := url.URL{Path: upath}

			viewName := name
			if d.IsDir() {
				viewName += "/"
			}

			shouldDownload := dirOptions.Attachments.Enable && !d.IsDir()
			pageData.Files = append(pageData.Files, fileInfoData{
				Info:     d,
				ModTime:  d.ModTime().UTC().Format(http.TimeFormat),
				Path:     url.String(),
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

// DirListRichTemplate is the html template the `DirListRich` function is using to render
// the directories and files.
var DirListRichTemplate = template.Must(template.New("dirlist").
	Funcs(template.FuncMap{
		"formatBytes": func(b int64) string {
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
		},
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
