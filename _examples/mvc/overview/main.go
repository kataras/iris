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
	mvc.Configure(app.Party("/greet"), setup)

	// http://localhost:8080/greet?name=kataras
	app.Listen(":8080", iris.WithLogLevel("debug"))
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
      +------------+-----+     +-------------------v---v-----------------+     +----+------+
      |  greeterWithLog  |     |  NewGreetService(Env, DB) GreetService  |     |  greeter  |
      -------------+-----+     +---------------------------+-------------+     +----+------+
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
	app.Register(
		environment.DEV,         // DEV, PROD
		database.NewDB,          // sqlite, mysql
		service.NewGreetService, // greeterWithLogging, greeter
	)

	// Register Controllers.
	app.Handle(new(controller.GreetController))
}
