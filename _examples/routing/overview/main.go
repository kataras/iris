package main

import (
	"os"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
)

func main() {
	app := iris.New()
	// Set Logger level to "debug",
	// see your terminal and the created file.
	app.Logger().SetLevel("debug")

	// Write logs to a file too.
	f := newLogFile()
	defer f.Close()
	app.Logger().AddOutput(f)

	// Register a request logger middleware to the application.
	app.Use(logger.New())

	// GET: http://localhost:8080
	app.Get("/", info)

	// GET: http://localhost:8080/profile/anyusername
	//
	// Want to use a custom regex expression instead?
	// Easy: app.Get("/profile/{username:string regexp(^[a-zA-Z ]+$)}")
	app.Get("/profile/{username:string}", info)

	// If parameter type is missing then it's string which accepts anything,
	// i.e: /{paramname} it's exactly the same as /{paramname:string}.
	// The below is exactly the same as
	// {username:string}
	//
	// GET: http://localhost:8080/profile/anyusername/backups/any/number/of/paths/here
	app.Get("/profile/{username}/backups/{filepath:path}", info)

	// Favicon

	// GET: http://localhost:8080/favicon.ico
	app.Favicon("./public/images/favicon.ico")

	// Static assets

	// GET: http://localhost:8080/assets/css/main.css
	//	    maps to ./public/assets/css/main.css file at system location.
	app.HandleDir("/assets", iris.Dir("./public/assets"))

	/* OR

	// GET: http://localhost:8080/css/main.css
	// 		maps to ./public/assets/css/main.css file at system location.
	app.HandleDir("/css", iris.Dir("./public/assets/css"))

	// GET: http://localhost:8080/css/bootstrap.min.css
	// 		maps to ./public/assets/css/bootstrap.min.css file at system location.
	app.HandleDir("/css", iris.Dir("./public/assets/css"))

	*/

	// Grouping

	usersRoutes := app.Party("/users")
	// GET: http://localhost:8080/users/help
	usersRoutes.Get("/help", func(ctx iris.Context) {
		ctx.Writef("GET / -- fetch all users\n")
		ctx.Writef("GET /$ID -- fetch a user by id\n")
		ctx.Writef("POST / -- create new user\n")
		ctx.Writef("PUT /$ID -- update an existing user\n")
		ctx.Writef("DELETE /$ID -- delete an existing user\n")
	})

	// GET: http://localhost:8080/users
	usersRoutes.Get("/", func(ctx iris.Context) {
		ctx.Writef("get all users")
	})

	// GET: http://localhost:8080/users/42
	// **/users/42 and /users/help works after iris version 7.0.5**
	usersRoutes.Get("/{id:uint64}", func(ctx iris.Context) {
		id, _ := ctx.Params().GetUint64("id")
		ctx.Writef("get user by id: %d", id)
	})

	// POST: http://localhost:8080/users
	usersRoutes.Post("/", func(ctx iris.Context) {
		username, password := ctx.PostValue("username"), ctx.PostValue("password")
		ctx.Writef("create user for username= %s and password= %s", username, password)
	})

	// PUT: http://localhost:8080/users
	usersRoutes.Put("/{id:uint64}", func(ctx iris.Context) {
		id, _ := ctx.Params().GetUint64("id") // or .Get to get its string represatantion.
		username := ctx.PostValue("username")
		ctx.Writef("update user for id= %d and new username= %s", id, username)
	})

	// DELETE: http://localhost:8080/users/42
	usersRoutes.Delete("/{id:uint64}", func(ctx iris.Context) {
		id, _ := ctx.Params().GetUint64("id")
		ctx.Writef("delete user by id: %d", id)
	}).Describe("deletes a user")

	// Subdomains, depends on the host, you have to edit the hosts or nginx/caddy's configuration if you use them.
	//
	// See more subdomains examples at _examples/routing/subdomains folder.
	adminRoutes := app.Party("admin.")

	// GET: http://admin.localhost:8080
	adminRoutes.Get("/", info)
	// GET: http://admin.localhost:8080/settings
	adminRoutes.Get("/settings", info)

	// Wildcard/dynamic subdomain
	dynamicSubdomainRoutes := app.WildcardSubdomain()

	// GET: http://any_thing_here.localhost:8080
	dynamicSubdomainRoutes.Get("/", info)

	app.Delete("/something", func(ctx iris.Context) {
		name := ctx.URLParam("name")
		ctx.Writef(name)
	})

	app.None("/secret", privateHandler)
	app.Get("/public", execPrivateHandler)

	// GET: http://localhost:8080/
	// GET: http://localhost:8080/profile/anyusername
	// GET: http://localhost:8080/profile/anyusername/backups/any/number/of/paths/here

	// GET: http://localhost:8080/users/help
	// GET: http://localhost:8080/users
	// GET: http://localhost:8080/users/42
	// POST: http://localhost:8080/users
	// PUT: http://localhost:8080/users
	// DELETE: http://localhost:8080/users/42
	// DELETE: http://localhost:8080/something?name=iris

	// GET: http://admin.localhost:8080
	// GET: http://admin.localhost:8080/settings
	// GET: http://any_thing_here.localhost:8080
	app.Listen(":8080")
}

func privateHandler(ctx iris.Context) {
	ctx.WriteString(`This can only be executed programmatically through server's another route:
ctx.Exec(iris.MethodNone, "/secret")`)
}

func execPrivateHandler(ctx iris.Context) {
	ctx.Exec(iris.MethodNone, "/secret")
}

func info(ctx iris.Context) {
	method := ctx.Method()       // the http method requested a server's resource.
	subdomain := ctx.Subdomain() // the subdomain, if any.

	// the request path (without scheme and host).
	path := ctx.Path()
	// how to get all parameters, if we don't know
	// the names:
	paramsLen := ctx.Params().Len()

	ctx.Params().Visit(func(name string, value string) {
		ctx.Writef("%s = %s\n", name, value)
	})
	ctx.Writef("\nInfo\n\n")
	ctx.Writef("Method: %s\nSubdomain: %s\nPath: %s\nParameters length: %d", method, subdomain, path, paramsLen)
}

// get a filename based on the date, file logs works that way the most times
// but these are just a sugar.
func todayFilename() string {
	today := time.Now().Format("Jan 02 2006")
	return today + ".txt"
}

func newLogFile() *os.File {
	filename := todayFilename()
	// open an output file, this will append to the today's file if server restarted.
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	return f
}
