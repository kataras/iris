## RoutesInfo plugin

This plugin collects & stores all registed routes and gives information about them.

#### The RouteInfo

```go

type RouteInfo struct {
	Method     string
	Domain     string
	Path       string
	RegistedAt time.Time
}

```
## How to use

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/plugins/routesinfo"
)

func main() {

	info := routesinfo.RoutesInfo()
	iris.Plugin(info)

	iris.Get("/yourpath", func(c *iris.Context) {
		c.Write("yourpath")
	})

	iris.Post("/otherpostpath", func(c *iris.Context) {
		c.Write("other post path")
	})

	println("Your routes info: ")
	all := info.All()
	// allget := info.ByMethod("GET") -> slice
	// alllocalhost := info.ByDomain("localhost") -> slice
	// bypath:= info.ByPath("/yourpath") -> slice
	// bydomainandmethod:= info.ByDomainAndMethod("localhost","GET") -> slice
	// bymethodandpath:= info.ByMethodAndPath("GET","/yourpath") -> single (it could be slice for all domains too but it's not)

	println("The first registed route was: ", all[0].Path, "registed at: ", all[0].RegistedAt.String())

	iris.Listen(":8080")
	
}


```