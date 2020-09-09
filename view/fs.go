package view

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"sort"
)

// walk recursively in "fs" descends "root" path, calling "walkFn".
func walk(fs http.FileSystem, root string, walkFn filepath.WalkFunc) error {
	names, err := assetNames(fs, root)
	if err != nil {
		return fmt.Errorf("%s: %w", root, err)
	}

	for _, name := range names {
		fullpath := path.Join(root, name)
		f, err := fs.Open(fullpath)
		if err != nil {
			return fmt.Errorf("%s: %w", fullpath, err)
		}
		stat, err := f.Stat()
		err = walkFn(fullpath, stat, err)
		if err != nil {
			if err != filepath.SkipDir {
				return fmt.Errorf("%s: %w", fullpath, err)
			}

			continue
		}

		if stat.IsDir() {
			if err := walk(fs, fullpath, walkFn); err != nil {
				return fmt.Errorf("%s: %w", fullpath, err)
			}
		}
	}

	return nil
}

// assetNames returns the first-level directories and file, sorted, names.
func assetNames(fs http.FileSystem, name string) ([]string, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}

	if f == nil {
		return nil, nil
	}

	infos, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(infos))
	for _, info := range infos {
		// note: go-bindata fs returns full names whether
		// the http.Dir returns the base part, so
		// we only work with their base names.
		name := filepath.ToSlash(info.Name())
		name = path.Base(name)

		names = append(names, name)
	}

	sort.Strings(names)
	return names, nil
}

func asset(fs http.FileSystem, name string) ([]byte, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadAll(f)
	f.Close()
	return contents, err
}

func getFS(fsOrDir interface{}) (fs http.FileSystem) {
	if fsOrDir == nil {
		return noOpFS{}
	}

	switch v := fsOrDir.(type) {
	case string:
		if v == "" {
			fs = noOpFS{}
		} else {
			fs = httpDirWrapper{http.Dir(v)}
		}
	case http.FileSystem:
		fs = v
	default:
		panic(fmt.Errorf(`unexpected "fsOrDir" argument type of %T (string or http.FileSystem)`, v))
	}

	return
}

type noOpFS struct{}

func (fs noOpFS) Open(name string) (http.File, error) { return nil, nil }

func isNoOpFS(fs http.FileSystem) bool {
	_, ok := fs.(noOpFS)
	return ok
}

// fixes: "invalid character in file path"
// on amber engine (it uses the virtual fs directly
// and it uses filepath instead of the path package...).
type httpDirWrapper struct {
	http.Dir
}

func (fs httpDirWrapper) Open(name string) (http.File, error) {
	return fs.Dir.Open(filepath.ToSlash(name))
}
