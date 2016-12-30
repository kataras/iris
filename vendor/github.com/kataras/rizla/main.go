//Package main rizla builds, runs and monitors your Go Applications with ease.
//
//   rizla main.go
//   rizla C:/myprojects/project1/main.go C:/myprojects/project2/main.go C:/myprojects/project3/main.go
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
	for _, a := range args {
		if !strings.HasSuffix(a, ".go") {
			color.Red("Error: Please provide files with '.go' extension.\n")
			help(-1)
		} else if p, _ := filepath.Abs(a); !fileExists(p) {
			color.Red("Error: File " + p + " does not exists.\n")
			help(-1)
		}
	}

	for _, a := range args {
		p := rizla.NewProject(a)
		rizla.Add(p)
	}
	rizla.Run()
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
