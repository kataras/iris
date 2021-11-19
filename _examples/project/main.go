package main

import (
	"fmt"
	"os"

	"github.com/username/project/cmd"
)

var (
	buildRevision string
	buildTime     string
)

func main() {
	app := cmd.New(buildRevision, buildTime)
	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
