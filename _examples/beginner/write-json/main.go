package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

// User bind struct
type User struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	City      string `json:"city"`
	Age       int    `json:"age"`
}

func main() {
	app := iris.New()

	app.Post("/decode", func(ctx context.Context) {
		var user User
		ctx.ReadJSON(&user)

		ctx.Writef("%s %s is %d years old and comes from %s!", user.Firstname, user.Lastname, user.Age, user.City)
	})

	app.Get("/encode", func(ctx context.Context) {
		peter := User{
			Firstname: "John",
			Lastname:  "Doe",
			City:      "Neither FBI knows!!!",
			Age:       25,
		}

		ctx.StatusCode(iris.StatusOK)
		ctx.JSON(peter)
	})

	app.Run(iris.Addr(":8080"))
}
