package context

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ResolveFS accepts a single input argument of any type
// and tries to cast it to fs.FS.
//
// It affects the view engine's filesystem resolver.
//
// This package-level variable can be modified on initialization.
var ResolveFS = func(fsOrDir interface{}) fs.FS {
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

func (fileSystem noOpFS) Open(name string) (fs.File, error) { return nil, nil }

// IsNoOpFS reports whether the given "fileSystem" is a no operation fs.
func IsNoOpFS(fileSystem fs.FS) bool {
	_, ok := fileSystem.(noOpFS)
	return ok
}

type httpFS struct {
	fs http.FileSystem
}

func (f *httpFS) Open(name string) (fs.File, error) {
	if name == "." {
		name = "/"
	}

	return f.fs.Open(filepath.ToSlash(name))
}

func (f *httpFS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = filepath.ToSlash(name)
	if name == "." {
		name = "/"
	}

	file, err := f.fs.Open(name)
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
		if info.IsDir() { // http file's does not return the whole tree, so read it.
			sub, err := f.ReadDir(info.Name())
			if err != nil {
				return nil, err
			}

			entries = append(entries, sub...)
			continue
		}

		entry := fs.FileInfoToDirEntry(info)
		entries = append(entries, entry)
	}

	return entries, nil
}

// ResolveHTTPFS accepts a single input argument of any type
// and tries to cast it to http.FileSystem.
//
// It affects the Application's API Builder's `HandleDir` method.
//
// This package-level variable can be modified on initialization.
var ResolveHTTPFS = func(fsOrDir interface{}) http.FileSystem {
	var fileSystem http.FileSystem
	switch v := fsOrDir.(type) {
	case string:
		fileSystem = http.Dir(v)
	case http.FileSystem:
		fileSystem = v
	case embed.FS:
		direEtries, err := v.ReadDir(".")
		if err != nil {
			panic(err)
		}

		if len(direEtries) == 0 {
			panic("HandleDir: no directories found under the embedded file system")
		}

		subfs, err := fs.Sub(v, direEtries[0].Name())
		if err != nil {
			panic(err)
		}
		fileSystem = http.FS(subfs)
	case fs.FS:
		fileSystem = http.FS(v)
	default:
		panic(fmt.Sprintf(`unexpected "fsOrDir" argument type of %T (string or http.FileSystem or embed.FS or fs.FS)`, v))
	}

	return fileSystem
}

// FindNames accepts a "http.FileSystem" and a root name and returns
// the list containing its file names.
func FindNames(fileSystem http.FileSystem, name string) ([]string, error) {
	if strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid root name")
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
		baseFilename := toBaseName(info.Name())
		fullname := path.Join(name, baseFilename)
		if fullname == name { // prevent looping through itself.
			continue
		}
		rfiles, err := FindNames(fileSystem, fullname)
		if err != nil {
			return nil, err
		}

		files = append(files, rfiles...)
	}

	return files, nil
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
