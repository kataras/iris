package main // #nosec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kataras/cli"
	"github.com/skratchdot/open-golang/open"
)

// we introduce a project type, because I'm (not near future) planning  dynamic inserting projects here by iris community
type project struct {
	remote string // the gopath, used to go get 'gopath',  if not exists in $GOPATH/src/'gopath'
}

// first of all: we could fetch the AIO_examples and make the projects full dynamically, this would be perfect BUT not yet,
// for now lets make it dynamic via code, we want a third-party repo to be compatible also, not only iris-contrib/examples.
var (
	commonRepo       = "github.com/iris-contrib/examples/AIO_examples/"
	relativeMainFile = "main.go"
	// the available projects/examples to build & run using this command line tool
	projects = map[string]project{
		// the project type, passed on the get command : project.gopath & project.mainfile
		"basic":  {remote: toslash(commonRepo, "basic", "backend")},
		"static": {remote: toslash(commonRepo, "static", "backend")},
		"mongo":  {remote: toslash(commonRepo, "mongo", "backend")},
	}
)

// DirectoryExists returns true if a directory(or file) exists, otherwise false
func DirectoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}

// dir returns the supposed local directory for this project
func (p project) dir() string {
	return join(getGoPath(), p.remote)
}

func (p project) mainfile() string {
	return fromslash(p.dir() + "/" + relativeMainFile)
}

func (p project) download() {
	// first, check if the repo exists locally in gopath
	if DirectoryExists(p.dir()) {
		return
	}
	app.Printf("Downloading... ")

	finish := cli.ShowIndicator(false)

	defer func() {

		finish <- true //it's autoclosed so
	}()

	// go get -u github.com/:owner/:repo
	cmd := exec.Command("go", "get", p.remote)
	cmd.Stdout = cli.Output
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error while trying to download the package: %s\n Please make sure that you're connected to the internet.\n", err.Error())
		return
	}
	for i := 0; i < len("Building"); i++ {
		app.Printf("\010\010\010") // remove the "loading" bars
	}

}

func (p project) run() {
	// in order to NOT re-write the import paths of all /backend/*.go, many things can go wrong, as they did before...
	// it's better to just let the project exists in the examples folder and user can copy
	// the source and change the import paths from there too
	// so here just let run and watch it
	mainFile := p.mainfile()
	if !DirectoryExists(mainFile) { // or file exists, same thing
		p.download()
	}
	installedDir := p.dir()
	app.Printf("Building %s\n\n", installedDir)

	// open this dir to the user, works with windows, osx, and other unix/linux & bsd*
	// even if user already had the package, help him and open it in order to locate and change its content if needed
	go open.Run(installedDir)

	// run and watch for source code changes
	runAndWatch(mainFile)
}

func buildGetCommand() *cli.Cmd {
	var availabletypes []string
	for k := range projects {
		availabletypes = append(availabletypes, "'"+k+"'")
	}
	// comma separated of projects' map key
	return cli.Command("get", "gets & runs a simple prototype-based project").
		Flag("type",
			"basic",
			// we take the os.Args in order to have access both via subcommand and when flag passed
			"downloads, installs and runs a project based on a prototype. Currently, available types are: "+strings.Join(availabletypes, ",")).
		Action(get)
}

// iris get static
// iris get -t static
// iris get = iris get basic
// and all are happy
func get(flags cli.Flags) error {
	// error and the end, not so idiomatic but needed here to make things easier

	if len(os.Args) >= 2 { // app name command name [and the package name]

		// set as the default value first
		t := flags.String("type") // this never empty

		// now check if actually user passed a package name without the -t/-type
		if len(os.Args) > 2 {
			v := os.Args[2]
			if !strings.HasPrefix(v, "-") {
				t = v // change the default given with the actual user-defined 'subcommand form'
			}
		}

		for k, p := range projects {
			if k == t {
				p.run()
				return nil
			}
		}
	}

	err := errInvalidManualArgs.Format(strings.Join(os.Args[1:], ","))
	app.Printf(err.Error())
	return err
}
