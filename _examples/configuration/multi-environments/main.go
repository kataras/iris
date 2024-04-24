package main

import (
	"fmt"
	"os"

	"github.com/kataras/my-iris-app/cmd"
)

func main() {
	app := cmd.New()
	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
