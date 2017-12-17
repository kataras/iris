# MVC Internals

* `MakeHandler` - accepts a function which accepts any input and outputs any result, and any optional values that will be used as binders, if needed they will be converted in order to be faster at serve-time. Returns a `context/iris#Handler` and a non-nil error if passed function cannot be wrapped to a raw `context/iris#Handler`
    * Struct fields with `Struct Binding`
    * Methods with `Dynamic Binding`
* `Engine` - The "manager" of the controllers and handlers, can be grouped and an `Engine` can have any number of children.
    * `Engine#Bind` Binds values to be used inside on one or more handlers and controllers
    * `Engine#Handler` - Creates and returns a new mvc handler, which accept any input parameters (calculated by the binders) and output any result which will be sent as a response to the HTTP Client. Calls the `MakeHandler` with the Engine's `Dependencies.Values` as the binders
    * `Engine#Controller` - Creates and activates a controller based on a struct which has the `C` as an embedded , anonymous, field and defines methods to be used as routes. Can accept any optional activator listeners in order to bind any custom routes or change the bindings, called once at startup.
* The highest level feature of this package is the `Application` which contains
an `iris.Party` as its Router and an `Engine`. A new `Application` is created with `New(iris.Party)` and registers a new `Engine` for itself, `Engines` can be shared via the `Application#NewChild` or by manually creating an `&Application{ Engine: engine, Router: subRouter }`. The `Application` is being used to build complete `mvc applications through controllers`, it doesn't contain any method to convert mvc handlers to raw handlers, although its `Engine` can do that but it's not a common scenario. 

Examples can be found at: https://github.com/kataras/iris/tree/master/_examples/mvc.

## Binding

First of all, they can be binded to `func input arguments` (custom handlers) or `struct fields` (controllers). We will use the term `Input` for both of them.

```go
// consume the user here as func input argument.
func myHandler(user User) {}

type myController struct {
    // consume the user here, as struct field.
    user User 
}
```

If the input is an interface then it will check if the binding is completed this interface
and it will be binded as expected.

Two types of binders are supported:

### Dynamic Binding

`ReturnValue`, should return a single value, no pointer to, if the consumer Input (`struct field` or `func input argument`) expects `User` then it will be binded on each request, this is a dynamic binding based on the `Context`.

```go
type User struct {
    Username string
}

myBinder := func(ctx iris.Context) User {
    return User {
        Username: ctx.Params().Get("username"),
    }
}

myHandler := func(user User) {
    // ...
}
```

### Static Binding

`Static Value (Service)`, this is used to bind a value instance, like a service or a database connection.

```go
// optional but we declare interface most of the times to 
// be easier to switch from production to testing or local and visa versa.
// If input is an interface then it will check if the binding is completed this interface
// and it will be binded as expected.
type Service interface { 
    Do() string
}

type myProductionService struct {
    text string
}
func (s *myProductionService) Do() string {
    return s.text
}

myService := &myProductionService{text: "something"}

myHandler := func(service Service) {
    // ...
}
```

### Add Dependencies

#### For Handlers

MakeHandler is used to create a handler based on a function which can accept any input arguments and export any output arguments, the input arguments can be dynamic path parameters or custom [binders](#binding).

```go
h, err := MakeHandler(myHandler, reflect.ValueOf(myBinder))
```

Values passed in `Dependencies` are binded to all handlers and controllers that are expected a type of the returned value, in this case the myBinder indicates a dynamic/serve-time function which returns a User, as shown above.

```go
m := NewEngine()
m.Dependencies.Add(myBinder)

h := m.Handler(myHandler)
```

#### For Controllers

```go
app := iris.New()
m := NewEngine()
m.Dependencies.Add(myBinder)
m.Controller(app, new(myController))
// ...
```

```go
sub := app.Party("/sub")
m := NewEngine()
m.Controller(sub, &myController{service: myService})
```

```go
NewEngine().Controller(sub.Party("/subsub"), new(myController), func(b mvc.BeforeActivation) {
    b.Dependencies().Add(myService)
})
```