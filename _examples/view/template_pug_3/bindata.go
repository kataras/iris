// Code generated for package main by go-bindata DO NOT EDIT. (@generated)
// sources:
// templates/index.pug
// templates/layout.pug
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _templatesIndexPug = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4a\xad\x28\x49\xcd\x4b\x29\x56\xc8\x49\xac\xcc\x2f\x2d\xd1\x2b\x28\x4d\xe7\xe5\xe2\xe5\x4a\xca\xc9\x4f\xce\x56\x28\xc9\x2c\xc9\x49\xe5\xe5\x52\x80\x30\x14\x1c\x8b\x4a\x32\x93\x73\x52\x15\x42\x20\xc2\x30\x55\xc9\xf9\x79\x25\xa9\x79\x25\x20\x75\x19\x86\x0a\xbe\x95\x30\x75\x80\x00\x00\x00\xff\xff\xa6\xfd\x18\x8c\x5a\x00\x00\x00")

func templatesIndexPugBytes() ([]byte, error) {
	return bindataRead(
		_templatesIndexPug,
		"templates/index.pug",
	)
}

func templatesIndexPug() (*asset, error) {
	bytes, err := templatesIndexPugBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/index.pug", size: 90, mode: os.FileMode(438), modTime: time.Unix(1581790962, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesLayoutPug = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x4a\xc9\x4f\x2e\xa9\x2c\x48\x55\xc8\x28\xc9\xcd\xe1\xe5\x82\x90\x0a\x0a\x19\xa9\x89\x29\x20\x5a\x41\x21\x29\x27\x3f\x39\x5b\xa1\x24\xb3\x24\x27\x15\x22\xa0\x00\xe1\x28\xb8\xa4\xa6\x25\x96\xe6\x94\x20\xa4\x92\xf2\x53\x2a\x91\xf5\x24\xe7\xe7\x95\xa4\xe6\x95\x00\x02\x00\x00\xff\xff\x5f\xa5\x93\xf9\x61\x00\x00\x00")

func templatesLayoutPugBytes() ([]byte, error) {
	return bindataRead(
		_templatesLayoutPug,
		"templates/layout.pug",
	)
}

func templatesLayoutPug() (*asset, error) {
	bytes, err := templatesLayoutPugBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/layout.pug", size: 97, mode: os.FileMode(438), modTime: time.Unix(1581790962, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"templates/index.pug":  templatesIndexPug,
	"templates/layout.pug": templatesLayoutPug,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"templates": &bintree{nil, map[string]*bintree{
		"index.pug":  &bintree{templatesIndexPug, map[string]*bintree{}},
		"layout.pug": &bintree{templatesLayoutPug, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
