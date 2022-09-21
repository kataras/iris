package context

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

// ResolveFS accepts a single input argument of any type
// and tries to cast it to http.FileSystem.
//
// It affects the view engine filesystem resolver
// and the Application's API Builder's `HandleDir` method.
//
// This package-level variable can be modified on initialization.
var ResolveFS = func(fsOrDir interface{}) http.FileSystem {
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
