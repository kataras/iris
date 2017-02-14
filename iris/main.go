package main

import (
	"github.com/kataras/cli"
	"gopkg.in/kataras/iris.v6"
)

var (
	// Name the name of the cmd tool
	Name = "Iris Command Line Tool"
	app  *cli.App
)

func init() {
	// init the cli app
	app = cli.NewApp("iris", "Command line tool for Iris web framework", iris.Version)
	// version command
	app.Command(cli.Command("version", "\t      prints your iris version").
		Action(func(cli.Flags) error { app.Printf("%s", app.Version); return nil }))
	// run command/-/run.go

	// register the commands
	app.Command(buildGetCommand())
	app.Command(buildRunCommand())

}

func main() {
	// run the application
	app.Run(func(cli.Flags) error {
		return nil
	})
}
