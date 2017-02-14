//Package main rizla builds, runs and monitors your Go Applications with ease.
//
//   rizla main.go
//   rizla C:/myprojects/project1/main.go C:/myprojects/project2/main.go C:/myprojects/project3/main.go
//   rizla -walk main.go [if -walk then rizla uses the stdlib's filepath.Walk method instead of file system's signals]
//
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/kataras/rizla/rizla"
)

const (
	// Version of rizla command line tool
	Version = "0.0.6"
	// Name of rizla
	Name = "Rizla"
	// Description of rizla
	Description = "Rizla builds, runs and monitors your Go Applications with ease."
)

var helpTmpl = fmt.Sprintf(`NAME:
   %s - %s

USAGE:
   rizla main.go
   rizla C:/myprojects/project1/main.go C:/myprojects/project2/main.go C:/myprojects/project3/main.go
   rizla -walk main.go [if -walk then rizla uses the stdlib's filepath.Walk method instead of file system's signals]
VERSION:
   %s
   `, Name, Description, Version)

func main() {
	argsLen := len(os.Args)

	if argsLen <= 1 {
		help(-1)
	} else if isArgHelp(os.Args[1]) {
		help(0)
	}

	args := os.Args[1:]
	programFiles := make([]string, 0)
	var fsWatcher rizla.Watcher

	for i, a := range args {
		// The first argument must be the method type of the file system's watcher.
		// if -w,-walk,walk then
		//   asks to use the stdlib's filepath.walk method instead of the operating system's signal.
		//   It's only usage is when the user's IDE overrides the os' signals.
		// otherwise
		//   use the fsnotify's operating system's file system's signals.
		if i == 0 {
			fsWatcher = rizla.WatcherFromFlag(a)
		}

		// it's main.go or any go main program
		if strings.HasSuffix(a, ".go") {
			programFiles = append(programFiles, a)
			continue
		}
	}

	// no program files given
	if len(programFiles) == 0 {
		color.Red("Error: Please provide a *.go file.\n")
		help(-1)
		return
	}

	// check if given program files exist
	for _, a := range programFiles {
		// the argument is not the first  given is *.go but doesn't exists on user's disk
		if p, _ := filepath.Abs(a); !fileExists(p) {
			color.Red("Error: File " + p + " does not exists.\n")
			help(-1)
			return
		}
	}

	rizla.RunWith(fsWatcher, programFiles...)
}

func help(code int) {
	os.Stdout.WriteString(helpTmpl)
	os.Exit(code)
}

func isArgHelp(s string) bool {
	return s == "help" || s == "-h" || s == "-help"
}

func fileExists(f string) bool {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return false
	}
	return true
}
