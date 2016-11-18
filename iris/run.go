package main

import (
	"os"

	"strings"

	"github.com/kataras/cli"
	"github.com/kataras/go-errors"
	"github.com/kataras/rizla/rizla"
)

func buildRunCommand() *cli.Cmd {
	return cli.Command("run", "runs and reload on source code changes, example: iris run main.go").Action(run)
}

var errInvalidManualArgs = errors.New("Invalid arguments [%s], type -h to get assistant")

func run(cli.Flags) error {
	if len(os.Args) <= 2 {
		err := errInvalidManualArgs.Format(strings.Join(os.Args, ","))
		app.Printf(err.Error()) // the return should print it too but do it for any case
		return err
	}
	programPath := os.Args[2]
	runAndWatch(programPath)
	return nil
}

func runAndWatch(programPath string) {
	/*
		project := rizla.NewProject(programPath)
		project.Name = "IRIS"
		project.AllowReloadAfter = time.Duration(3) * time.Second
		project.Out = rizla.NewPrinter(os.Stdout)
		project.Err = rizla.NewPrinter(os.Stderr)
		rizla.Add(project)

		rizla.Run()
	*/
	// or just do that:
	rizla.DefaultDisableProgramRerunOutput = true // we don't want the banner to be shown after the first run
	rizla.Run(programPath)
}
