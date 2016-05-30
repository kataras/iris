## Iris Control

### THIS IS NOT READY YET

This plugin will give you remotely ( and local ) access to your iris server's information via a web interface


### Assets
No assets here because this is go -getable folder I don't want to messup with the folder size, in order to solve this
I created a downloader manager inside this package which downloads the first time the assets and unzip them to the kataras/iris/plugin/iris-control/iris-control-assets/ .



The assets files are inside [this repository](https://github.com/iris-contrib/iris-control-assets)


## How to use

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/plugin/iriscontrol"
    "fmt"
)

func main() {

	iris.Plugins().Add(iriscontrol.Web(9090, map[string]string{
		"irisusername1": "irispassword1",
		"irisusername2": "irispassowrd2",
	}))

	iris.Get("/", func(ctx *iris.Context) {
	})

	iris.Post("/something", func(ctx *iris.Context) {
	})

	fmt.Printf("Iris is listening on :%d", 8080)
	iris.Listen(":8080")
}



```
