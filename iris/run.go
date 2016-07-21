package main

import (
	"os"

	"strings"

	"github.com/kataras/cli"
	"github.com/kataras/rizla/rizla"
)

func runAndWatch(flags cli.Flags) error {
	if len(os.Args) <= 2 {
		printer.Dangerf("Invalid arguments [%s], type -h to get assistant", strings.Join(os.Args, ","))
		os.Exit(-1)
	}
	programPath := os.Args[2]

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

	return nil
}
