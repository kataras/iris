# Quick start

The following guide is just a simple example of usage of some of the **Iris MVC** features. You are not limited to that data structure or code flow. 

Create a folder, let's assume its path is `app`. The structure should look like that:

```
│   main.go
│	go.mod
│	go.sum
└───environment
│      environment.go
└───model
│      request.go
│      response.go
└───database
│       database.go
│       mysql.go
│       sqlite.go
└───service
│       greet_service.go
└───controller
        greet_controller.go
```

Navigate to that `app` folder and execute the following command:

```sh
$ go init app
$ go get github.com/kataras/iris/v12@main
#								 	or @latest for the latest official release.
```

## Environment

Let's start by defining the available environments that our web-application can behave on.

We'll just work on two available environments, the "development" and the "production", as they define the two most common scenarios. The `ReadEnv` will read from the `Env` type of a system's environment variable (see `main.go` in the end of the page).

Create a `environment/environment.go` file and put the following contents:

```go
package environment

import (
    "os"
    "strings"
)

const (
	PROD Env = "production"
	DEV  Env = "development"
)

type Env string

func (e Env) String() string {
    return string(e)
}

func ReadEnv(key string, def Env) Env {
    v := Getenv(key, def.String())
    if v == "" {
        return def
    }

    env := Env(strings.ToLower(v))
    switch env {
    case PROD, DEV: // allowed.
    default:
        panic("unexpected environment " + v)
    }

    return env
}

func Getenv(key string, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }

    return def
}
```

## Database

We will use two database management systems, the `MySQL` and the `SQLite`. The first one for "production" use and the other for "development".

Create a `database/database.go` file and copy-paste the following:

```go
package database

import "app/environment"

type DB interface {
	Exec(q string) error
}

func NewDB(env environment.Env) DB {
	switch env {
	case environment.PROD:
		return &mysql{}
	case environment.DEV:
		return &sqlite{}
	default:
		panic("unknown environment")
	}
}
```

Let's simulate our MySQL and SQLite `DB` instances. Create a `database/mysql.go` file which looks like the following one:

```go
package database

import "fmt"

type mysql struct{}

func (db *mysql) Exec(q string) error {
	return fmt.Errorf("mysql: not implemented <%s>", q)
}
```

And a `database/sqlite.go` file.

```go
package database

type sqlite struct{}

func (db *sqlite) Exec(q string) error { return nil }
```

The `DB` depends on the `Environment.

> A practical and operational database example, including Docker images, can be found at the following guide: https://github.com/kataras/iris/tree/main/_examples/database/mysql

## Service

We'll need a service that will communicate with a database instance in behalf of our Controller(s).

In our case we will only need a single service, the Greet Service.

For the sake of the example, let's use two implementations of a greet service based on the `Environment`. The `GreetService` interface contains a single method of `Say(input string) (output string, err error)`. Create a `./service/greet_service.go` file and write the following code:

```go
package service

import (
    "fmt"

    "app/database"
    "app/environment"
)

// GreetService example service.
type GreetService interface {
	Say(input string) (string, error)
}

// NewGreetService returns a service backed with a "db" based on "env".
func NewGreetService(env environment.Env, db database.DB) GreetService {
	service := &greeter{db: db, prefix: "Hello"}

	switch env {
	case environment.PROD:
		return service
	case environment.DEV:
		return &greeterWithLogging{service}
	default:
		panic("unknown environment")
	}
}

type greeter struct {
	prefix string
	db     database.DB
}

func (s *greeter) Say(input string) (string, error) {
	if err := s.db.Exec("simulate a query..."); err != nil {
		return "", err
	}

	result := s.prefix + " " + input
	return result, nil
}

type greeterWithLogging struct {
	*greeter
}

func (s *greeterWithLogging) Say(input string) (string, error) {
	result, err := s.greeter.Say(input)
	fmt.Printf("result: %s\nerror: %v\n", result, err)
	return result, err
}

```

The `greeter` will be used on "production" and the `greeterWithLogging` on "development". The `GreetService` depends on the `Environment` and the `DB`.

## Models

Continue by creating our HTTP request and response models.

Create a `model/request.go` file and copy-paste the following code:

```go
package model

type Request struct {
    Name string `url:"name"`
}
```

Same for the `model/response.go` file.

```go
package model

type Response struct {
    Message string `json:"msg"`
}
```

The server will accept a URL Query Parameter of `name` (e.g. `/greet?name=kataras`) and will reply back with a JSON message.

## Controller

MVC Controllers are responsible for controlling the flow of the application execution. When you make a request (means request a page) to MVC Application, a controller is responsible for returning the response to that request.

We will only need the `GreetController` for our mini web-application. Create a file at `controller/greet_controller.go` which looks like that:

```go
package controller

import (
	"app/model"
	"app/service"
)

type GreetController struct {
	Service service.GreetService
	// Ctx iris.Context
}

func (c *GreetController) Get(req model.Request) (model.Response, error) {
	message, err := c.Service.Say(req.Name)
	if err != nil {
		return model.Response{}, err
	}

	return model.Response{Message: message}, nil
}
```

The `GreetController` depends on the `GreetService`. It serves the `GET: /greet` index path through its `Get` method. The `Get` method accepts a `model.Request` which contains a single field name of `Name` which will be extracted from the `URL Query Parameter 'name'` (because it's a `GET` requst and its `url:"name"` struct field).

## Wrap up

```sh
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
```

Now it's the time to wrap all the above into our `main.go` file. Copy-paste the following code:

```go
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
	app.Get("/ping", pong).Describe("healthcheck")

	mvc.Configure(app.Party("/greet"), setup)

	// http://localhost:8080/greet?name=kataras
	app.Listen(":8080", iris.WithLogLevel("debug"))
}

func pong(ctx iris.Context) {
	ctx.WriteString("pong")
}

func setup(app *mvc.Application) {
	// Register Dependencies.
	app.Register(
		environment.DEV,         // DEV, PROD
		database.NewDB,          // sqlite, mysql
		service.NewGreetService, // greeterWithLogging, greeter
	)

	// Register Controllers.
	app.Handle(new(controller.GreetController))
}
```

The `mvc.Application.Register` method registers one more dependencies, dependencies can depend on previously registered dependencies too. Thats the reason we pass, first, the `Environment(DEV)`, then the `NewDB` which depends on that `Environment`, following by the `NewGreetService` function which depends on both the `Environment(DEV)` and the `DB`.

The `mvc.Application.Handle` registers a new controller, which depends on the `GreetService`, for the targeted sub-router of `Party("/greet")`.

## Run

Install [Go](https://golang.org/dl) and run the application with:

```sh
go run main.go
```

<details><summary>Docker</summary>

Download the [Dockerfile](https://raw.githubusercontent.com/kataras/iris/9b93c0dbb491dcedf49c91e89ca13bab884d116f/_examples/mvc/overview/Dockerfile) and [docker-compose.yml](https://raw.githubusercontent.com/kataras/iris/9b93c0dbb491dcedf49c91e89ca13bab884d116f/_examples/mvc/overview/docker-compose.yml) files to the `app` folder.

Install [Docker](https://www.docker.com/) and execute the following command:

```sh
$ docker-compose up
```
</details>

Visit http://localhost:8080?name=kataras.

Optionally, replace the `main.go`'s `app.Register(environment.DEV` with `environment.PROD`, restart the application and refresh. You will see that a new database (`sqlite`) and another service of (`greeterWithLogging`) will be binded to the `GreetController`. With **a single change** you achieve to completety change the result.
