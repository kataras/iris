# MVC

![](https://github.com/kataras/iris/raw/master/_examples/mvc/web_mvc_diagram.png)

Iris has **first-class support for the MVC pattern**, you'll not find
these stuff anywhere else in the Go world.

Iris supports Request data, Models, Persistence Data and Binding
with the fastest possible execution.

## Characteristics

All HTTP Methods are supported, for example if want to serve `GET`
then the controller should have a function named `Get()`,
you can define more than one method function to serve in the same Controller.

Serve custom controller's struct's methods as handlers with custom paths(even with regex parametermized path) 
via the `BeforeActivation` custom event callback, per-controller. Example:

```go
import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

func main() {
    app := iris.New()
    mvc.Configure(app.Party("/root"), myMVC)
    app.Run(iris.Addr(":8080"))
}

func myMVC(app *mvc.Application) {
    // app.Register(...)
    // app.Router.Use/UseGlobal/Done(...)
    app.Handle(new(MyController))
}

type MyController struct {}

func (m *MyController) BeforeActivation(b mvc.BeforeActivation) {
    // b.Dependencies().Add/Remove
    // b.Router().Use/UseGlobal/Done // and any standard API call you already know

    // 1-> Method
    // 2-> Path
    // 3-> The controller's function name to be parsed as handler
    // 4-> Any handlers that should run before the MyCustomHandler
    b.Handle("GET", "/something/{id:long}", "MyCustomHandler", anyMiddleware...)
}

// GET: http://localhost:8080/root
func (m *MyController) Get() string { return "Hey" }

// GET: http://localhost:8080/root/something/{id:long}
func (m *MyController) MyCustomHandler(id int64) string { return "MyCustomHandler says Hey" }
```

Persistence data inside your Controller struct (share data between requests)
by defining services to the Dependencies or have a `Singleton` controller scope.

Share the dependencies between controllers or register them on a parent MVC Application, and ability
to modify dependencies per-controller on the `BeforeActivation` optional event callback inside a Controller,
i.e `func(c *MyController) BeforeActivation(b mvc.BeforeActivation) { b.Dependencies().Add/Remove(...) }`.

Access to the `Context` as a controller's field(no manual binding is neede) i.e `Ctx iris.Context` or via a method's input argument, i.e `func(ctx iris.Context, otherArguments...)`.

Models inside your Controller struct (set-ed at the Method function and rendered by the View).
You can return models from a controller's method or set a field in the request lifecycle
and return that field to another method, in the same request lifecycle.

Flow as you used to, mvc application has its own `Router` which is a type of `iris/router.Party`, the standard iris api.
`Controllers` can be registered to any `Party`, including Subdomains, the Party's begin and done handlers work as expected.

Optional `BeginRequest(ctx)` function to perform any initialization before the method execution,
useful to call middlewares or when many methods use the same collection of data.

Optional `EndRequest(ctx)` function to perform any finalization after any method executed.

Inheritance, recursively, see for example our `mvc.SessionController`, it has the `Session *sessions.Session` and `Manager *sessions.Sessions` as embedded fields
which are filled by its `BeginRequest`, [here](https://github.com/kataras/iris/blob/master/mvc/session_controller.go).
This is just an example, you could use the `sessions.Session` as a dependency to the MVC Application, i.e
`mvcApp.Register(sessions.New(sessions.Config{Cookie: "iris_session_id"}).Start)`.

Access to the dynamic path parameters via the controller's methods' input arguments, no binding is needed.
When you use the Iris' default syntax to parse handlers from a controller, you need to suffix the methods
with the `By` word, uppercase is a new sub path. Example:

If `mvc.New(app.Party("/user")).Handle(new(user.Controller))`

- `func(*Controller) Get()` - `GET:/user`.
- `func(*Controller) Post()` - `POST:/user`.
- `func(*Controller) GetLogin()` - `GET:/user/login`
- `func(*Controller) PostLogin()` - `POST:/user/login`
- `func(*Controller) GetProfileFollowers()` - `GET:/user/profile/followers`
- `func(*Controller) PostProfileFollowers()` - `POST:/user/profile/followers`
- `func(*Controller) GetBy(id int64)` - `GET:/user/{param:long}`
- `func(*Controller) PostBy(id int64)` - `POST:/user/{param:long}`

If `mvc.New(app.Party("/profile")).Handle(new(profile.Controller))`

- `func(*Controller) GetBy(username string)` - `GET:/profile/{param:string}`

If `mvc.New(app.Party("/assets")).Handle(new(file.Controller))`

- `func(*Controller) GetByWildard(path string)` - `GET:/assets/{param:path}`

    Supported types for method functions receivers: int, int64, bool and string.

Response via output arguments, optionally, i.e

```go
func(c *ExampleController) Get() string |
                                (string, string) |
                                (string, int) |
                                int |
                                (int, string) |
                                (string, error) |
                                error |
                                (int, error) |
                                (any, bool) |
                                (customStruct, error) |
                                customStruct |
                                (customStruct, int) |
                                (customStruct, string) |
                                mvc.Result or (mvc.Result, error)
```

where [mvc.Result](https://github.com/kataras/iris/blob/master/mvc/func_result.go) is an interface which contains only that function: `Dispatch(ctx iris.Context)`.

## Using Iris MVC for code reuse

By creating components that are independent of one another, developers are able to reuse components quickly and easily in other applications. The same (or similar) view for one application can be refactored for another application with different data because the view is simply handling how the data is being displayed to the user.

If you're new to back-end web development read about the MVC architectural pattern first, a good start is that [wikipedia article](https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93controller).

## Examples

- [Hello world](hello-world/main.go) **UPDATED**
- [Session Controller](session-controller/main.go) **UPDATED**
- [Overview - Plus Repository and Service layers](overview) **UPDATED**
- [Login showcase - Plus Repository and Service layers](login) **UPDATED**
- [Singleton](singleton) **NEW**
- [Websocket Controller](websocket) **NEW**
- [Register Middleware](middleware) **NEW**

Folder structure guidelines can be found at the [_examples/#structuring](https://github.com/kataras/iris/tree/master/_examples/#structuring) section.