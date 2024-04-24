package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

func main() {
	auth := basicauth.Load("users.yml", basicauth.BCRYPT)
	/* Same as:
	opts := basicauth.Options{
		Realm: basicauth.DefaultRealm,
		Allow: basicauth.AllowUsersFile("users.yml", basicauth.BCRYPT),
	}

	auth := basicauth.New(opts)
	*/

	app := iris.New()
	app.Use(auth)
	app.Get("/", index)
	// kataras:kataras_pass
	// makis:makis_pass
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	user := ctx.User()
	ctx.JSON(user)
}
