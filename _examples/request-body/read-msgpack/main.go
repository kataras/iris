package main

import "github.com/kataras/iris/v12"

// User example struct to bind to.
type User struct {
	Firstname string `msgpack:"firstname"`
	Lastname  string `msgpack:"lastname"`
	City      string `msgpack:"city"`
	Age       int    `msgpack:"age"`
}

// readMsgPack reads a `User` from MsgPack post body.
func readMsgPack(ctx iris.Context) {
	var u User
	err := ctx.ReadMsgPack(&u)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.Writef("Received: %#+v\n", u)
}

func main() {
	app := iris.New()
	app.Post("/", readMsgPack)

	// POST: http://localhost:8080
	//
	// To run the example, use a tool like Postman:
	// 1. Body: Binary
	// 2. Select File, select the one from "_examples/response-writer/write-rest" example.
	// The output should be:
	// Received: main.User{Firstname:"John", Lastname:"Doe", City:"Neither FBI knows!!!", Age:25}
	app.Listen(":8080")
}
