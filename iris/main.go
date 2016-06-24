package main

import (
	"os"

	_ "syscall"

	"strings"

	"github.com/kataras/cli"
	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/logger"
)

const (
	Version = "0.0.7"
)

var (
	app        *cli.App
	printer    *logger.Logger
	workingDir string
)

func init() {

	// set the current working dir
	if d, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		workingDir = d
	}

	// defaultInstallDir is the default directory which the create will copy and run the package when finish downloading
	// it's just the last path part of the workingDir
	defaultInstallDir := workingDir[strings.LastIndexByte(workingDir, os.PathSeparator)+1:]

	// init the cli app
	app = cli.NewApp("iris", "Command line tool for Iris web framework", Version)
	// version command
	app.Command(cli.Command("version", "\t      prints your iris version").Action(func(cli.Flags) error { app.Printf("%s", iris.Version); return nil }))

	// create command/-/create.go
	createCmd := cli.Command("create", "create a project to a given directory").
		Flag("offline", false, "set to true to disable the packages download on each create command").
		Flag("dir", defaultInstallDir, "$GOPATH/src/$dir the directory to install the sample package").
		Flag("type", "basic", "creates a project based on the -t package. Currently, available types are 'basic' & 'static'").
		Action(create)

	// run command/-/run.go
	runAndWatchCmd := cli.Command("run", "runs and reload on source code changes, example: iris run main.go").Action(runAndWatch)

	// register the commands
	app.Command(createCmd)
	app.Command(runAndWatchCmd)

	// init the logger
	printer = logger.New(config.DefaultLogger())
}

func main() {
	// run the application
	app.Run(func(f cli.Flags) error {
		return nil
	})
}
