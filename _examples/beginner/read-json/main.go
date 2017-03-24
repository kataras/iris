package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

type Company struct {
	Name  string
	City  string
	Other string
}

func MyHandler(ctx *iris.Context) {
	c := &Company{}
	if err := ctx.ReadJSON(c); err != nil {
		ctx.Log(iris.DevMode, err.Error())
		return
	}

	ctx.Writef("Company: %#v\n", c)
}

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())

	// use postman or whatever to do a POST request
	// to the http://localhost:8080 with BODY: JSON PAYLOAD
	// and Content-Type to application/json
	app.Post("/", MyHandler)
	app.Listen(":8080")
}
