package main

import (
	"app/controller"
	"app/database"
	"app/environment"
	"app/service"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	app.Get("/ping", pong).Describe("healthcheck")

	mvc.Configure(app.Party("/greet"), setup)

	// http://localhost:8080/greet?name=kataras
	addr := ":" + environment.Getenv("PORT", "8080")
	app.Listen(addr)
}

func pong(ctx iris.Context) {
	ctx.WriteString("pong")
}

/*
                                         +-------------------+
                                         |  Env (DEV, PROD)  |
                                         +---------+---------+
                                         |         |         |
                                         |         |         |
                                         |         |         |
                                    DEV  |         |         |  PROD
-------------------+---------------------+         |         +----------------------+-------------------
                   |                               |                                |
                   |                               |                                |
               +---+-----+        +----------------v------------------+        +----+----+
               | sqlite  |        |         NewDB(Env) DB             |        |  mysql  |
               +---+-----+        +----------------+---+--------------+        +----+----+
                   |                               |   |                            |
                   |                               |   |                            |
                   |                               |   |                            |
    +--------------+-----+     +-------------------v---v-----------------+     +----+------+
    | greeterWithLogging |     |  NewGreetService(Env, DB) GreetService  |     |  greeter  |
    +--------------+-----+     +---------------------------+-------------+     +----+------+
                   |                                       |                        |
                   |                                       |                        |
                   |           +-----------------------------------------+          |
                   |           |  GreetController          |             |          |
                   |           |                           |             |          |
                   |           |  - Service GreetService <--             |          |
                   |           |                                         |          |
                   |           +-------------------+---------------------+          |
                   |                               |                                |
                   |                               |                                |
                   |                               |                                |
                   |                   +-----------+-----------+                    |
                   |                   |      HTTP Request     |                    |
                   |                   +-----------------------+                    |
                   |                   |  /greet?name=kataras  |                    |
                   |                   +-----------+-----------+                    |
                   |                               |                                |
+------------------+--------+         +------------+------------+           +-------+------------------+
|  model.Response (JSON)    |         |  Response (JSON, error) |           |  Bad Request             |
+---------------------------+         +-------------------------+           +--------------------------+
|  {                        |                                               |  mysql: not implemented  |
|    "msg": "Hello kataras" |                                               +--------------------------+
|  }                        |
+---------------------------+
*/
func setup(app *mvc.Application) {
	// Register Dependencies.
	// Tip: A dependency can depend on other dependencies too.
	env := environment.ReadEnv("ENVIRONMENT", environment.DEV)
	app.Register(
		env,                     // DEV, PROD
		database.NewDB,          // sqlite, mysql
		service.NewGreetService, // greeterWithLogging, greeter
	)

	// Register Controllers.
	app.Handle(new(controller.GreetController))
}
