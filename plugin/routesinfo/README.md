## RoutesInfo plugin

This plugin collects & stores all registered  routes and gives information about them.

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
	"github.com/kataras/iris/plugin/routesinfo"
)

func main() {

	info := routesinfo.New()
	iris.Plugins().Add(info)

	iris.Get("/yourpath", func(c *iris.Context) {
		c.Write("yourpath")
	})

	iris.Post("/otherpostpath", func(c *iris.Context) {
		c.Write("other post path")
	})

	all := info.All()
	// allget := info.ByMethod("GET") -> slice
	// alllocalhost := info.ByDomain("localhost") -> slice
	// bypath:= info.ByPath("/yourpath") -> slice
	// bydomainandmethod:= info.ByDomainAndMethod("localhost","GET") -> slice
	// bymethodandpath:= info.ByMethodAndPath("GET","/yourpath") -> single (it could be slice for all domains too but it's not)

	println("The first registed route was: ", all[0].Path, "registed at: ", all[0].RegistedAt.String())
	println("All routes info:")
	for i:= range all {
		println(all[i].String())
		//outputs->
		// Domain: localhost Method: GET Path: /yourpath RegistedAt: 2016/03/27 15:27:05:029 ...
		// Domain: localhost Method: POST Path: /otherpostpath RegistedAt: 2016/03/27 15:27:05:030 ...
	}
	iris.Listen(":8080")

}


```
