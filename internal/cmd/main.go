package main

import (
	"os"

	"github.com/kataras/iris/internal/cmd/gen/website/examples"
)

func main() {
	// just for testing, the cli will be coded when I finish at least with this one command.
	_, err := examples.WriteExamplesTo(os.Stdout) // doesn't work yet.
	if err != nil {
		println(err.Error())
	}
}
