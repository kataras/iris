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
	var c Company

	if err := ctx.ReadJSON(&c); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.WriteString(err.Error())
		return
	}

	ctx.Writef("Received: %#+v\n", c)
}

// simple json stuff, read more at https://golang.org/pkg/encoding/json
type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// MyHandler2 reads a collection of Person from JSON post body.
func MyHandler2(ctx iris.Context) {
	var persons []Person
	err := ctx.ReadJSON(&persons)

	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.WriteString(err.Error())
		return
	}

	ctx.Writef("Received: %#+v\n", persons)
}

func main() {
	app := iris.New()

	app.Post("/", MyHandler)
	app.Post("/slice", MyHandler2)

	// use Postman or whatever to do a POST request
	// to the http://localhost:8080 with RAW BODY:
	/*
		{
			"Name": "iris-Go",
			"City": "New York",
			"Other": "Something here"
		}
	*/
	// and Content-Type to application/json (optionally but good practise)
	//
	// The response should be:
	// Received: main.Company{Name:"iris-Go", City:"New York", Other:"Something here"}
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed), iris.WithOptimizations)
}
