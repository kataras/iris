package main

import (
	"github.com/kataras/iris"
)

type Company struct {
	Name  string
	City  string
	Other string
}

func MyHandler(ctx iris.Context) {
	c := &Company{}
	if err := ctx.ReadJSON(c); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.WriteString(err.Error())
		return
	}

	ctx.Writef("Received: %#v\n", c)
}

func main() {
	app := iris.New()

	app.Post("/", MyHandler)

	// use Postman or whatever to do a POST request
	// to the http://localhost:8080 with RAW BODY:
	/*
		{
			"Name": "iris-Go",
			"City": "New York",
			"Other": "Something here"
		}
	*/
	// and Content-Type to application/json
	//
	// The response should be:
	// Received: &main.Company{Name:"iris-Go", City:"New York", Other:"Something here"}
	app.Run(iris.Addr(":8080"))
}
