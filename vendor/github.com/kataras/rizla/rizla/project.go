package rizla

import (
	"os"
	"path/filepath"

	"strings"
	"time"
)

const minimumAllowReloadAfter = time.Duration(3) * time.Second

// DefaultDisableProgramRerunOutput a long name but, it disables the output of the program's 'messages' after the first successfully run for each of the projects
// the project iteral can be override this value.
// set to true to disable the program's output when reloads
var DefaultDisableProgramRerunOutput = false

// MatcherFunc returns whether the file should be watched for the reload
type MatcherFunc func(string) bool

// DefaultGoMatcher is the default Matcher for the Project iteral
func DefaultGoMatcher(fullname string) bool {
	return (filepath.Ext(fullname) == goExt) ||
		(!isWindows && strings.Contains(fullname, goExt))
}

// DefaultWatcher is the default Watcher for the Project iteral
// allows all subdirs except .git, node_modules and vendor
func DefaultWatcher(abs string) bool {
	base := filepath.Base(abs)
	// by-default ignore .git folder, node_modules, vendor and any hidden files.
	return !(base == ".git" || base == "node_modules" || base == "vendor" || base == ".")
}

// DefaultOnReload fired when file has changed and reload going to happens
func DefaultOnReload(p *Project) func(string) {
	return func(string) {
		fromproject := ""
		if p.Name != "" {
			fromproject = "From project '" + p.Name + "': "
		}
		p.Out.Infof("\n%sA change has been detected, reloading now...", fromproject)
	}
}

// DefaultOnReloaded fired when reload has been finished
func DefaultOnReloaded(p *Project) func(string) {
	return func(string) {
		p.Out.Successf("ready!\n")
	}
}

// Project the struct which contains the necessary fields to watch and reload(rerun) a go project
type Project struct {
	// optional Name for the project
	Name string
	// MainFile is the absolute path of the go project's main file source.
	MainFile string
	Args     []string
	// The Output destination (sent by rizla and your program)
	Out *Printer
	// The Err Output destination (sent on rizla errors and your program's errors)
	Err *Printer
	// Watcher accepts subdirectories by the watcher
	// executes before the watcher starts,
	// if return true, then this (absolute) subdirectory is watched by watcher
	// the default accepts all subdirectories but ignores the ".git", "node_modules" and "vendor"
	Watcher MatcherFunc
	Matcher MatcherFunc
	// AllowReloadAfter skip reload on file changes that made too fast from the last reload
	// minimum allowed duration is 3 seconds.
	AllowReloadAfter time.Duration
	// OnReload fires when when file has been changed and rizla is going to reload the project
	// the parameter is the changed file name
	OnReload func(string)
	// OnReloaded fires when rizla finish with the reload
	// the parameter is the changed file name
	OnReloaded func(string)
	// DisableRuntimeDir set to true to disable adding subdirectories into the watcher, when a folder created at runtime
	// set to true to disable the program's output when reloads
	// defaults to false
	DisableRuntimeDir bool
	// DisableProgramRerunOutput a long name but, it disables the output of the program's 'messages' after the first successfully run
	// defaults to false
	DisableProgramRerunOutput bool

	dir string
	// proc the system Process of a running instance (if any)
	proc *os.Process
	// when the last change was made
	lastChange time.Time
	// i%2 ==0 if windows, then the reload is allowed
	i int
}

// NewProject returns a simple project iteral which doesn't needs argument parameters
// and has the default file matcher ( which is valid if you want to reload only on .Go files).
//
// You can change all of its fields before the .Run function.
func NewProject(mainfile string) *Project {
	if mainfile == "" {
		mainfile = "main.go"
	}
	mainfile, _ = filepath.Abs(mainfile)

	dir := filepath.Dir(mainfile)

	p := &Project{
		MainFile:                  mainfile,
		Out:                       NewPrinter(os.Stdout),
		Err:                       NewPrinter(os.Stderr),
		Watcher:                   DefaultWatcher,
		Matcher:                   DefaultGoMatcher,
		AllowReloadAfter:          minimumAllowReloadAfter,
		DisableProgramRerunOutput: DefaultDisableProgramRerunOutput,
		dir:        dir,
		lastChange: time.Now(),
	}

	p.OnReload = DefaultOnReload(p)
	p.OnReloaded = DefaultOnReloaded(p)
	return p
}
