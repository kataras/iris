# MVC Internals

* `MakeHandler` - accepts a function which accepts any input and outputs any result, and any optional values that will be used as binders, if needed they will be converted in order to be faster at serve-time. Returns a `context/iris#Handler` and a non-nil error if passed function cannot be wrapped to a raw `context/iris#Handler`
* `Engine` - The "manager" of the controllers and handlers, can be grouped and an `Engine` can have any number of children.
    * `Engine#Bind` Binds values to be used inside on one or more handlers and controllers
    * `Engine#Handler` - Creates and returns a new mvc handler, which accept any input parameters (calculated by the binders) and output any result which will be sent as a response to the HTTP Client. Calls the `MakeHandler` with the Engine's `Input` values as the binders
    * `Engine#Controller` - Creates and activates a controller based on a struct which has the `C` as an embedded , anonymous, field and defines methods to be used as routes. Can accept any optional activator listeners in order to bind any custom routes or change the bindings, called once at startup
* `C`
    * Struct fields with `Struct Binding`
    * Methods with `Dynamic Binding`


Examples can be found at: https://github.com/kataras/iris/tree/master/_examples/mvc.

## Binding

First of all, they can be binded to `func input arguments` (custom handlers) or `struct fields` (controllers). We will use the term `Input` for both of them.

```go
// consume the user here as func input argument.
func myHandler(user User) {}

type myController struct {
    C

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

`Value (Service)`, this is used to bind a value instance, like a service or a database connection.

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

### Bind

#### For Handlers

MakeHandler is used to create a handler based on a function which can accept any input arguments and export any output arguments, the input arguments can be dynamic path parameters or custom [binders](#binding).

```go
h, err := MakeHandler(myHandler, reflect.ValueOf(myBinder))
```

Values passed in `Bind` are binded to all handlers and controllers that are expected a type of the returned value, in this case the myBinder indicates a dynamic/serve-time function which returns a User, as shown above.

```go
m := New().Bind(myBinder)

h := m.Handler(myHandler)
```

#### For Controllers

```go
app := iris.New()
New().Bind(myBinder).Controller(app, new(myController))
// ...
```

```go
sub := app.Party("/sub")
New().Controller(sub, &myController{service: myService})
```

```go
New().Controller(sub.Party("/subsub"), new(myController), func(ca *ControllerActivator) {
    ca.Bind(myService)
})
```