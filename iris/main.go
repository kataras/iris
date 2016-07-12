package main

import (
	"os"

	"github.com/iris-contrib/logger"
	"github.com/kataras/cli"
	"github.com/kataras/iris"
)

const (
	// Version of Iris command line tool
	Version = "0.0.8"
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

	// init the cli app
	app = cli.NewApp("iris", "Command line tool for Iris web framework", Version)
	// version command
	app.Command(cli.Command("version", "\t      prints your iris version").Action(func(cli.Flags) error { app.Printf("%s", iris.Version); return nil }))

	// create command/-/create.go
	createCmd := cli.Command("create", "create a project to a given directory").
		Flag("offline", false, "set to true to disable the packages download on each create command").
		Flag("dir", workingDir, "$GOPATH/src/$dir the directory to install the sample package").
		Flag("type", "basic", "creates a project based on the -t package. Currently, available types are 'basic' & 'static'").
		Action(create)

	// run command/-/run.go
	runAndWatchCmd := cli.Command("run", "runs and reload on source code changes, example: iris run main.go").Action(runAndWatch)

	// register the commands
	app.Command(createCmd)
	app.Command(runAndWatchCmd)

	// init the logger
	printer = logger.New(logger.DefaultConfig())
}

func main() {
	// run the application
	app.Run(func(f cli.Flags) error {
		return nil
	})
}
