## Package information

Iris control is a plugin which gives you a little control over your Iris station.


```go
iris.Plugins.Add(iriscontrol.New(PORT, AUTHENTICATED_USERS))
```

```go
package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/plugin/iriscontrol"
)

func main() {

	iris.Plugins.Add(iriscontrol.New(9090, map[string]string{
		"1":             "1",
		"irisusername2": "irispassowrd2",
	}))

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Write("Root path from  server")
	})

	iris.Get("/something", func(ctx *iris.Context) {
		ctx.Write("Something path from server")
	})

  // Iris control will listen on mydomain.com:9090
	iris.Listen("mydomain.com:8080")
}
```

[![Iris control show case](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iriscontrolplugin.gif](https://github.com/iris-contrib/examples/tree/master/plugin_iriscontrol)
