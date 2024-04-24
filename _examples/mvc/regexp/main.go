// Package main shows how to match "/xxx.json" in MVC handler.
package main

/*
There is no MVC naming pattern for such these things,you can imagine the limitations of that.
Instead you can use the `BeforeActivation` on your controller to add more advanced routing features
(https://github.com/kataras/iris/tree/main/_examples/routing).

You can also create your own macro,
i.e: /{file:json} or macro function of a specific parameter type i.e: (/{file:string json()}).
Read the routing examples and you will gain a deeper view, there are all covered.
*/

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()

	mvcApp := mvc.New(app.Party("/module"))
	mvcApp.Handle(new(myController))

	// http://localhost:8080/module/xxx.json (OK)
	// http://localhost:8080/module/xxx.xml  (Not Found)
	app.Listen(":8080")
}

type myController struct{}

func (m *myController) BeforeActivation(b mvc.BeforeActivation) {
	// b.Dependencies().Register
	// b.Router().Use/UseGlobal/Done // and any standard API call you already know

	// 1-> Method
	// 2-> Path
	// 3-> The controller's function name to be parsed as handler
	// 4-> Any handlers that should run before the HandleJSON

	// "^[a-zA-Z0-9_.-]+.json$)" to validate file-name pattern and json
	// or just:  ".json$" to validate suffix.

	b.Handle("GET", "/{file:string regexp(^[a-zA-Z0-9_.-]+.json$))}", "HandleJSON" /*optionalMiddleware*/)
}

func (m *myController) HandleJSON(file string) string {
	return "custom serving of json: " + file
}
