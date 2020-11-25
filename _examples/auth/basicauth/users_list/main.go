package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

// User is just an example structure of a user,
// it MUST contain a Username and Password exported fields
// or complete the basicauth.User interface.
type User struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

var users = []User{
	{"admin", "admin", []string{"admin"}},
	{"kataras", "kataras_pass", []string{"manager", "author"}},
	{"george", "george_pass", []string{"member"}},
	{"john", "john_pass", []string{}},
}

func main() {
	opts := basicauth.Options{
		Realm: basicauth.DefaultRealm,
		// Defaults to 0, no expiration.
		// Prompt for new credentials on a client's request
		// made after 10 minutes the user has logged in:
		MaxAge: 10 * time.Minute,
		// Clear any expired users from the memory every one hour,
		// note that the user's expiration time will be
		// reseted on the next valid request (when Allow passed).
		GC: basicauth.GC{
			Every: 2 * time.Hour,
		},
		// The users can be a slice of custom users structure
		// or a map[string]string (username:password)
		// or []map[string]interface{} with username and passwords required fields,
		// read the godocs for more.
		Allow: basicauth.AllowUsers(users),
	}

	auth := basicauth.New(opts)
	// OR: basicauth.Default(users)

	app := iris.New()
	app.Use(auth)
	app.Get("/", index)
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	user, _ := ctx.User().GetRaw()
	ctx.JSON(user)
}
