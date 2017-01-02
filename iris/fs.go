package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/go-errors"
)

var goPath string

// returns the (last) gopath+"/src/"
func getGoPath() string {
	if goPath == "" {
		errGoPathMissing := errors.New(Name + `: $GOPATH environment is missing. Please configure your $GOPATH. Reference:
      https://github.com/golang/go/wiki/GOPATH`)

		// set the gopath
		goPath = os.Getenv("GOPATH")
		if goPath == "" {
			// we should panic here
			panic(errGoPathMissing)
		}

		if idxLSep := strings.LastIndexByte(goPath, os.PathListSeparator); idxLSep == len(goPath)-1 {
			// remove the last ';' or ':' in order to be safe to take the last correct path(if more than one )
			goPath = goPath[0 : len(goPath)-2]
		}
		if idxLSep := strings.IndexByte(goPath, os.PathListSeparator); idxLSep != -1 {
			// we have more than one user-defined gopaths
			goPath = goPath[idxLSep+1:] // take the last
		}
	}

	return join(goPath, "src")
}

const pathsep = string(os.PathSeparator)

// it just replaces / or double slashes with os.PathSeparator
// if second parameter receiver is true then it removes any ending slashes too
func fromslash(s string) string {
	if len(s) == 0 {
		return ""
	}

	s = strings.Replace(s, "//", "/", -1)
	s = strings.Replace(s, "/", pathsep, -1)

	return s
}

func removeTrailingSlash(s string) string {
	if s[len(s)-1] == os.PathSeparator {
		s = s[0 : len(s)-2]
	}
	return s
}

func toslash(paths ...string) string {
	s := join(paths...)
	s = strings.Replace(s, pathsep, "/", -1)
	return removeTrailingSlash(s)
}

// combine paths
func join(paths ...string) string {
	return removeTrailingSlash(fromslash(filepath.Join(paths...)))
}
