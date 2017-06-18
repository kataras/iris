package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
)

type Company struct {
	Name  string
	City  string
	Other string
}

func MyHandler(ctx context.Context) {
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
			"Name": "Iris-Go",
			"City": "New York",
			"Other": "Something here"
		}
	*/
	// and Content-Type to application/json
	//
	// The response should be:
	// Received: &main.Company{Name:"Iris-Go", City:"New York", Other:"Something here"}
	app.Run(iris.Addr(":8080"))
}
