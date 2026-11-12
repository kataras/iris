package context

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// ResolveFS accepts a single input argument of any type
// and tries to cast it to fs.FS.
//
// It affects the view engine's filesystem resolver.
//
// This package-level variable can be modified on initialization.
var ResolveFS = func(fsOrDir any) fs.FS {
	if fsOrDir == nil {
		return noOpFS{}
	}

	switch v := fsOrDir.(type) {
	case string:
		if v == "" {
			return noOpFS{}
		}
		return os.DirFS(v)
	case fs.FS:
		return v
	case http.FileSystem: // handles go-bindata.
		return &httpFS{v}
	default:
		panic(fmt.Errorf(`unexpected "fsOrDir" argument type of %T (string or fs.FS or embed.FS or http.FileSystem)`, v))
	}
}

type noOpFS struct{}

// ErrNoOpFS is returned by the no-operation file system's Open method.
// Use IsNoOpFS to test for the file system itself.
var ErrNoOpFS = errors.New("no file system is configured")

// Open always fails. Returning an error rather than a nil file keeps the
// idiomatic "f, err := fs.Open(name)" caller from dereferencing nil.
func (fileSystem noOpFS) Open(name string) (fs.File, error) { return nil, ErrNoOpFS }

// IsNoOpFS reports whether the given "fileSystem" is a no operation fs.
func IsNoOpFS(fileSystem fs.FS) bool {
	_, ok := fileSystem.(noOpFS)
	return ok
}

// httpFS is an http.FileSystem seen as an fs.FS that fs.WalkDir can walk.
type httpFS struct {
	fs http.FileSystem
}

// httpFSName returns the name an http.FileSystem is asked for, given the name
// of a file in an fs.FS. An fs.FS name is unrooted and its root is ".". An
// http.FileSystem gets rooted names from the file server of net/http, so that
// is the one form every implementation knows: http.Dir and go-bindata take a
// name with or without the leading slash, a file system keyed by the request
// path takes it only with one.
func httpFSName(name string) string {
	return path.Clean("/" + filepath.ToSlash(name))
}

func (f *httpFS) Open(name string) (fs.File, error) {
	return f.fs.Open(httpFSName(name))
}

// ReadDir returns the entries of the named directory, sorted by file name.
// It keeps the contract of fs.ReadDirFS: a sub-directory is one entry and is
// not read. fs.WalkDir descends into it by its own path, which is the path of
// the parent joined with the name of the entry.
func (f *httpFS) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := f.fs.Open(httpFSName(name))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	infos, err := file.Readdir(-1)
	if err != nil {
		return nil, err
	}

	entries := make([]fs.DirEntry, 0, len(infos))
	for _, info := range infos {
		// The name of an entry is its base name, which is what http.Dir
		// returns. go-bindata returns the whole path with os.Separator
		// (see ListFileNames).
		base := extractBaseName(info.Name())
		if base == "" || base == "." {
			// The directory listed itself. A walk would never leave it.
			continue
		}

		if base != info.Name() {
			info = baseNameFileInfo{FileInfo: info, name: base}
		}

		entries = append(entries, fs.FileInfoToDirEntry(info))
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	return entries, nil
}

// baseNameFileInfo is a file info under the base name of the file.
type baseNameFileInfo struct {
	fs.FileInfo
	name string
}

func (i baseNameFileInfo) Name() string { return i.name }

// ResolveHTTPFS accepts a single input argument of any type
// and tries to cast it to http.FileSystem.
//
// It affects the file server controller's `New` and `Handler` functions
// (the controller/fileserver package).
//
// This package-level variable can be modified on initialization.
var ResolveHTTPFS = func(fileSystemOrDirectory any) http.FileSystem {
	var fileSystem http.FileSystem
	switch v := fileSystemOrDirectory.(type) {
	case string:
		fileSystem = http.Dir(v)
	case http.FileSystem: // same as http.Dir
		fileSystem = v
	case embed.FS:
		direEtries, err := v.ReadDir(".")
		if err != nil {
			panic(err)
		}

		if len(direEtries) == 0 {
			panic("no directories found under the embedded file system")
		}

		subfs, err := fs.Sub(v, direEtries[0].Name())
		if err != nil {
			panic(err)
		}
		fileSystem = http.FS(subfs)
	case fs.FS:
		fileSystem = http.FS(v)
	default:
		panic(fmt.Sprintf(`unexpected "fileSystemOrDirectory" argument type of %T (string or http.FileSystem or embed.FS or fs.FS)`, v))
	}

	return fileSystem
}

// ErrInvalidFileSystemRootName is returned by ListFileNames when the given root
// name escapes the file system, either by traversal or by naming an absolute
// operating-system path.
var ErrInvalidFileSystemRootName = errors.New("invalid root name")

// ListFileNames accepts a "http.FileSystem" and a root name and returns
// the list containing its file names.
//
// The name is cleaned and must stay inside the file system. That check is worth
// keeping even though http.Dir defends itself, because the file system may be an
// embed.FS, a go-bindata bundle or any other implementation that does not.
func ListFileNames(fileSystem http.FileSystem, name string) ([]string, error) {
	name, ok := cleanFileSystemName(name)
	if !ok {
		return nil, ErrInvalidFileSystemRootName
	}

	f, err := fileSystem.Open(name) // it's the root dir.
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return []string{name}, nil
	}

	fileinfos, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)

	for _, info := range fileinfos {
		// Note:
		// go-bindata has absolute names with os.Separator,
		// http.Dir the basename.
		baseFilename := extractBaseName(info.Name())
		fullname := path.Join(name, baseFilename)
		if fullname == name { // prevent looping through itself.
			continue
		}
		rfiles, err := ListFileNames(fileSystem, fullname)
		if err != nil {
			return nil, err
		}

		files = append(files, rfiles...)
	}

	return files, nil
}

// cleanFileSystemName normalizes a file system path and reports whether it stays
// inside the file system root. A rooted "/" prefix is preserved, since
// http.FileSystem implementations expect it.
func cleanFileSystemName(name string) (string, bool) {
	name = filepath.ToSlash(strings.TrimSpace(name))
	if name == "" || name == "." {
		return "/", true
	}

	if len(name) > 1 && name[1] == ':' { // an OS-absolute path such as C:\Windows.
		return "", false
	}

	rooted := name[0] == '/'
	cleaned := path.Clean("/" + name)
	if cleaned == "/" {
		return "/", true
	}

	if !rooted {
		return cleaned[1:], true
	}

	return cleaned, true
}

// Instead of path.Base(filepath.ToSlash(s))
// let's do something like that, it is faster
// (used to list directories on serve-time too):
func extractBaseName(s string) string {
	// Trailing separators are trimmed in a loop rather than by recursing, so a
	// directory listing cannot drive one stack frame per trailing slash.
	for len(s) > 0 {
		if c := s[len(s)-1]; c == '/' || c == '\\' {
			s = s[:len(s)-1]
			continue
		}

		break
	}

	for i := len(s) - 1; i >= 0; i-- {
		if c := s[i]; c == '/' || c == '\\' {
			return s[i+1:] // return the rest, trimming the slash.
		}
	}

	return s
}
