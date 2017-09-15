# FAQ

### Looking for free support?

    http://support.iris-go.com
    https://kataras.rocket.chat/channel/iris

### Looking for previous versions?

    https://github.com/kataras/iris#-version


### Should I upgrade my Iris?

Developers are not forced to upgrade if they don't really need it. Upgrade whenever you feel ready.

> Iris uses the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature, so you get truly reproducible builds, as this method guards against upstream renames and deletes.

**How to upgrade**: Open your command-line and execute this command: `go get -u github.com/kataras/iris`.

# Fr, 15 September 2017 | v8.4.2

## MVC

Support more than one dynamic method function receivers.

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    app.Controller("/user", new(UserController))
    app.Run(iris.Addr("localhost:8080"))
}

type UserController struct { iris.Controller }

// Maps to GET /user
// Request example: http://localhost:8080/user
// as usual.
func (c *UserController) Get() {
    c.Text = "hello from /user"
}

// Maps to GET /user/{paramfirst:long}
// Request example: http://localhost:8080/user/42
// as usual.
func (c *UserController) GetBy(userID int64) {
    c.Ctx.Writef("hello user with id: %d", userID)
}

// NEW:
// Maps to GET /user/{paramfirst:long}/business/{paramsecond:long}
// Request example: http://localhost:8080/user/42/business/93
func (c *UserController) GetByBusinessBy(userID int64, businessID int64) {
    c.Ctx.Writef("fetch a business id: %d that user with id: %d owns, may make your db query faster",
    businessID, userID)
}
```

# Th, 07 September 2017 | v8.4.1

## Routing

Add a macro type for booleans: `app.Get("/mypath/{paramName:boolean}", myHandler)`.

```sh
+------------------------+
| {param:boolean}        |
+------------------------+
bool type
only "1" or "t" or "T" or "TRUE" or "true" or "True"
or "0" or "f" or "F" or "FALSE" or "false" or "False"
```

Add `context.Params().GetBool(paramName string) (bool, error)` respectfully.

```go
app := iris.New()
app.Get("/mypath/{has:boolean}", func(ctx iris.Context) { // <--
    // boolean first return value
    // error as second return value
    //
    // error will be always nil here because
    // we use the {has:boolean} so router
    // makes sure that the parameter is a boolean
    // otherwise it will return a 404 not found http error code
    // skipping the call of this handler.
    has, _ := ctx.Params().GetBool("has") // <--
    if has {
        ctx.HTML("<strong>it's true</strong>")
    }else {
        ctx.HTML("<strong>it's false</string>")
    }
})
// [...]
```

## MVC

Support for boolean method receivers, i.e `GetBy(bool), PostBy(bool)...`.


```go
app := iris.New()

app.Controller("/equality", new(Controller))
```

```go
type Controller struct {
    iris.Controller
}

// handles the "/equality" path.
func (c *Controller) Get() {

}

// registers and handles the path: "/equality/{param:boolean}".
func (c *Controller) GetBy(is bool) { // <--
    // [...]
}
```

> Supported types for method functions receivers are: int, int64, bool and string.

# Su, 27 August 2017 | v8.4.0

## Miscellaneous

- Update `vendor blackfriday` package to its latest version, 2.0.0
- Update [documentation](https://godoc.org/github.com/kataras/iris) for go 1.9
- Update [_examples](_examples) folder for go 1.9
- Update examples inside https://github.com/iris-contrib/middleware for go 1.9
- Update https://github.com/kataras/iris-contrib/examples for go 1.9
- Update https://iris-go.com/v8/recipe for go 1.9

## Router

Add a new macro type for path parameters, `long`, it's the go type `int64`.

```go
app.Get("/user/{id:long}", func(ctx context.Context) {
	userID, _ := ctx.Params().GetInt64("id")
})
```

## MVC

The ability to pre-calculate, register and map different (relative) paths inside a single controller
with zero performance cost.

Meaning that after a `go get -u github.com/kataras/iris` you will be able to use things like these:

If `app.Controller("/user", new(user.Controller))`

- `func(*Controller) Get()` - `GET:/user` , as usual.
- `func(*Controller) Post()` - `POST:/user`, as usual.
- `func(*Controller) GetLogin()` - `GET:/user/login`
- `func(*Controller) PostLogin()` - `POST:/user/login`
- `func(*Controller) GetProfileFollowers()` - `GET:/user/profile/followers`
- `func(*Controller) PostProfileFollowers()` - `POST:/user/profile/followers`
- `func(*Controller) GetBy(id int64)` - `GET:/user/{param:long}`
- `func(*Controller) PostBy(id int64)` - `POST:/user/{param:long}`

If `app.Controller("/profile", new(profile.Controller))`

- `func(*Controller) GetBy(username string)` - `GET:/profile/{param:string}`

If `app.Controller("/assets", new(file.Controller))`

- `func(*Controller) GetByWildard(path string)` - `GET:/assets/{param:path}`


**Example** can be found at: [_examples/mvc/login/user/controller.go](_examples/mvc/login/user/controller.go).

## Pretty [awesome](https://github.com/kataras/iris/stargazers), right?

# We, 23 August 2017 | v8.3.4

Give read access to the current request context's route, a feature that many of you asked a lot.

```go
func(ctx context.Context) {
	_ = ctx.GetCurrentRoute().Name()
	//					.Method() returns string, same as ctx.Method().
	//					.Subdomain() returns string, the registered subdomain.
	//					.Path() returns string, the registered path.
	//					.IsOnline() returns boolean.
}
```  

```go
type MyController struct {
	mvc.Controller
}

func (c *MyController) Get(){
	_ = c.Route().Name() // same as `c.Ctx.GetCurrentRoute().Name()`.
	// [...]
}
```

**Updated: 24 August 2017**

This evening, on the next version 8.3.5:

Able to pre-calculate, register and map different (relative) paths inside a single controller
with zero performance cost.

Meaning that in the future you will be able to use something like these:

If `app.Controller("/user", new(user.Controller))`

- `func(c *Controller) Get()` - `GET:/user` , as usual.
- `func(c *Controller) Post()` - `POST:/user`, as usual.
- `func(c *Controller) GetLogin()` - `GET:/user/login`
- `func(c *Controller) PostLogin()` - `POST:/user/login`
- `func(c *Controller) GetProfileFollowers()` - `GET:/user/profile/followers`
- `func(c *Controller) PostProfileFollowers()` - `POST:/user/profile/followers`
- `func(c *Controller) GetBy()` - `GET:/user/{param}`
- `func(c *Controller) GetByName(name string)` - `GET:/user/{name}`
- `func(c *Controller) PostByName(name string)` - `POST:/user/{name}`
- `func(c *Controller) GetByID(id int64 || int)` - `GET:/user/{id:int}`
- `func(c *Controller) PostByID(id int64 || int)` - `POST:/user/{id:int}`

Watch and stay tuned my friends.

# We, 23 August 2017 | v8.3.3

Better debug messages when using MVC.

Add support for recursively binding and **custom controllers embedded to other custom controller**, that's the new feature. That simply means that Iris users are able to use "shared" controllers everywhere; when binding, using models, get/set persistence data, adding middleware, intercept request flow.

This will help web authors to split the logic at different controllers. Those controllers can be also used as "standalone" to serve a page somewhere else in the application as well.

My personal advice to you is to always organize and split your code nicely and wisely in order to avoid using such as an advanced MVC feature, at least any time soon.

I'm aware that this is not always an easy task to do, therefore is here if you ever need it :)

A ridiculous simple example of this feature can be found at the [mvc/controller_test.go](https://github.com/kataras/iris/blob/master/mvc/controller_test.go#L424) file.


# Tu, 22 August 2017 | v8.3.2

### MVC

When one or more values of handler type (`func(ctx context.Context)`) are passed
right to the controller initialization then they will be recognised and act as middleware(s)
that ran even before the controller activation, there is no reason to load
the whole controller if the main handler or its `BeginRequest` are not "allowed" to be executed.

Example Code

```go
func checkLogin(ctx context.Context) {
	if !myCustomAuthMethodPassed {
		// [set a status or redirect, you know what to do]
		ctx.StatusCode(iris.StatusForbidden)
		return
	}

	// [continue to the next handler, at this example is our controller itself]
	ctx.Next()
}

// [...]
app.Controller(new(ProfileController), checkLogin)
// [...]
```

Usage of these kind of MVC features could be found at the [mvc/controller_test.go](https://github.com/kataras/iris/blob/master/mvc/controller_test.go#L174) file.

### Other minor enhancements

- fix issue [#726](https://github.com/kataras/iris/issues/726)[*](https://github.com/kataras/iris/commit/5e435fc54fe3dbf95308327c2180d1b444ef7e0d)
- fix redis sessiondb expiration[*](https://github.com/kataras/iris/commit/85cfc91544c981e87e09c5aa86bad4b85d0b96d3)
- update recursively when new version is available[*](https://github.com/kataras/iris/commit/cd3c223536c6a33653a7fcf1f0648123f2b968fd)
- some minor session enhancements[*](https://github.com/kataras/iris/commit/2830f3b50ee9c526ac792c3ce1ec1c08c24ea024)


# Sa, 19 August 2017 | v8.3.1

First of all I want to thank you for the 100% green feedback you gratefully sent me you about
my latest article `Go vs .NET Core in terms of HTTP performance`, published at [medium's hackernoon.com](https://hackernoon.com/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8) and [dev.to](https://dev.to/kataras/go-vsnet-core-in-terms-of-http-performance). I really appreciate it💓

No API Changes.

However two more methods added to the `Controller`.

- `RelPath() string`, returns the relative path based on the controller's name and the request path.
- `RelTmpl() string`, returns the relative template directory based on the controller's name.

These are useful when dealing with big `controllers`, they help you to keep align with any
future changes inside your application. 

Let's refactor our [ProfileController](_examples/mvc/controller-with-model-and-view/main.go) enhancemed by these two new functions.

```go
func (pc *ProfileController) tmpl(relativeTmplPath string) {
	// the relative template files directory of this controller.
	views := pc.RelTmpl()
	pc.Tmpl = views + relativeTmplPath
}

func (pc *ProfileController) match(relativeRequestPath string) bool {
	// the relative request path of this controller.
	path := pc.RelPath()
	return path == relativeRequestPath
}

func (pc *ProfileController) Get() {
	// requested: "/profile"
	// so relative path is "/" because of the ProfileController.
	if pc.match("/") {

		// views/profile/index.html
		pc.tmpl("index.html")
		return
	}

	// requested: "/profile/browse"
	// so relative path is "/browse".
	if pc.match("/browse") {
		pc.Path = "/profile"
		return
	}

	// requested: "/profile/me"
	// so the relative path is "/me"
	if pc.match("/me") {
		
		// views/profile/me.html
		pc.tmpl("me.html")
		return
	}

	// requested: "/profile/$ID"
	// so the relative path is "/$ID"
	id, _ := pc.Params.GetInt64("id")

	user, found := pc.DB.GetUserByID(id)
	if !found {
		pc.Status = iris.StatusNotFound

		// views/profile/notfound.html
		pc.tmpl("notfound.html")
		pc.Data["ID"] = id
		return
	}

	// views/profile/profile.html
	pc.tmpl("profile.html")
	pc.User = user
}
```

Want to learn more about these functions? Go to the [mvc/controller_test.go](mvc/controller_test.go) file and scroll to the bottom!

# Fr, 18 August 2017 | v8.3.0

Good news for devs that are used to write their web apps using the `MVC` architecture pattern.

Implement a whole new `mvc` package with additional support for models and easy binding.

@kataras started to develop that feature by version 8.2.5, back then it didn't seem
to be a large feature and maybe a game-changer, so it lived inside the `kataras/iris/core/router/controller.go` file.
However with this version, so many things are implemented for the MVC and we needed a new whole package,
this new package is the `kataras/iris/mvc`, but if you used go 1.9 to build then you don't have to do any refactor, you could use the `iris.Controller` type alias.

People who used the mvc from its baby steps(v8.2.5) the only syntactic change you'll have to do is to rename the `router.Controller` to `mvc.Controller`:

Before: 
```go
import "github.com/kataras/iris/core/router"
type MyController struct {
    router.Controller
}
```
Now:
```go
import "github.com/kataras/iris/mvc"
type MyController struct {
    mvc.Controller
    // if you build with go1.9 you can omit the import of mvc package
    // and just use `iris.Controller` instead.
}
```

### MVC (Model View Controller)

![](_examples/mvc/web_mvc_diagram.png)

From version 8.3 and after Iris has **first-class support for the MVC pattern**, you'll not find
these stuff anywhere else in the Go world.


Example Code


```go
package main

import (
	"sync"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	// when we have a path separated by spaces
	// then the Controller is registered to all of them one by one.
	//
	// myDB is binded to the controller's `*DB` field: use only structs and pointers.
	app.Controller("/profile /profile/browse /profile/{id:int} /profile/me",
		new(ProfileController), myDB) // IMPORTANT

	app.Run(iris.Addr(":8080"))
}

// UserModel our example model which will render on the template.
type UserModel struct {
	ID       int64
	Username string
}

// DB is our example database.
type DB struct {
	usersTable map[int64]UserModel
	mu         sync.RWMutex
}

// GetUserByID imaginary database lookup based on user id.
func (db *DB) GetUserByID(id int64) (u UserModel, found bool) {
	db.mu.RLock()
	u, found = db.usersTable[id]
	db.mu.RUnlock()
	return
}

var myDB = &DB{
	usersTable: map[int64]UserModel{
		1:  {1, "kataras"},
		2:  {2, "makis"},
		42: {42, "jdoe"},
	},
}

// ProfileController our example user controller which controls
// the paths of "/profile" "/profile/{id:int}" and "/profile/me".
type ProfileController struct {
	mvc.Controller // IMPORTANT

	User UserModel `iris:"model"`
	// we will bind it but you can also tag it with`iris:"persistence"`
	// and init the controller with manual &PorifleController{DB: myDB}.
	DB *DB
}

// Get method handles all "GET" HTTP Method requests of the controller's paths.
func (pc *ProfileController) Get() { // IMPORTANT
	path := pc.Path

	// requested: /profile path
	if path == "/profile" {
		pc.Tmpl = "profile/index.html"
		return
	}
	// requested: /profile/browse
	// this exists only to proof the concept of changing the path:
	// it will result to a redirection.
	if path == "/profile/browse" {
		pc.Path = "/profile"
		return
	}

	// requested: /profile/me path
	if path == "/profile/me" {
		pc.Tmpl = "profile/me.html"
		return
	}

	// requested: /profile/$ID
	id, _ := pc.Params.GetInt64("id")

	user, found := pc.DB.GetUserByID(id)
	if !found {
		pc.Status = iris.StatusNotFound
		pc.Tmpl = "profile/notfound.html"
		pc.Data["ID"] = id
		return
	}

	pc.Tmpl = "profile/profile.html"
	pc.User = user
}


/*
func (pc *ProfileController) Post() {}
func (pc *ProfileController) Put() {}
func (pc *ProfileController) Delete() {}
func (pc *ProfileController) Connect() {}
func (pc *ProfileController) Head() {}
func (pc *ProfileController) Patch() {}
func (pc *ProfileController) Options() {}
func (pc *ProfileController) Trace() {}
*/

/*
func (pc *ProfileController) All() {}
//        OR
func (pc *ProfileController) Any() {}
*/
```

Iris web framework supports Request data, Models, Persistence Data and Binding
with the fastest possible execution.

**Characteristics**

All HTTP Methods are supported, for example if want to serve `GET`
then the controller should have a function named `Get()`,
you can define more than one method function to serve in the same Controller struct.

Persistence data inside your Controller struct (share data between requests)
via `iris:"persistence"` tag right to the field or Bind using `app.Controller("/" , new(myController), theBindValue)`.

Models inside your Controller struct (set-ed at the Method function and rendered by the View)
via `iris:"model"` tag right to the field, i.e ```User UserModel `iris:"model" name:"user"` ``` view will recognise it as `{{.user}}`.
If `name` tag is missing then it takes the field's name, in this case the `"User"`.

Access to the request path and its parameters via the `Path and Params` fields.

Access to the template file that should be rendered via the `Tmpl` field.

Access to the template data that should be rendered inside
the template file via `Data` field.

Access to the template layout via the `Layout` field.

Access to the low-level `context.Context` via the `Ctx` field.

Get the relative request path by using the controller's name via `RelPath()`.

Get the relative template path directory by using the controller's name via `RelTmpl()`.

Flow as you used to, `Controllers` can be registered to any `Party`,
including Subdomains, the Party's begin and done handlers work as expected.

Optional `BeginRequest(ctx)` function to perform any initialization before the method execution,
useful to call middlewares or when many methods use the same collection of data.

Optional `EndRequest(ctx)` function to perform any finalization after any method executed.

Inheritance, recursively, see for example our `mvc.SessionController`, it has the `mvc.Controller` as an embedded field
and it adds its logic to its `BeginRequest`, [here](https://github.com/kataras/iris/blob/master/mvc/session_controller.go). 

Read access to the current route  via the `Route` field.

**Using Iris MVC for code reuse** 

By creating components that are independent of one another, developers are able to reuse components quickly and easily in other applications. The same (or similar) view for one application can be refactored for another application with different data because the view is simply handling how the data is being displayed to the user.

If you're new to back-end web development read about the MVC architectural pattern first, a good start is that [wikipedia article](https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93controller).


Follow the examples below,

- [Hello world](_examples/mvc/hello-world/main.go)
- [Session Controller](_examples/mvc/session-controller/main.go)
- [A simple but featured Controller with model and views](_examples/mvc/controller-with-model-and-view).

### Bugs

Fix [#723](https://github.com/kataras/iris/issues/723) reported by @speedwheel.


# Mo, 14 August 2017 | v8.2.6

Able to call done/end handlers inside a `Controller`, via optional `EndRequest(ctx context.Context)` function inside the controller struct.

```go
// it's called after t.Get()/Post()/Put()/Delete()/Connect()/Head()/Patch()/Options()/Trace().
func (t *testControllerEndRequestFunc) EndRequest(ctx context.Context) {
    // 2.
    // [your code goes here...]
}

// will handle "GET" request HTTP method only.
func (t *testControllerEndRequestFunc) Get() {
    // 1.
    // [your code goes here...]
}
```

Look at the [v8.2.5 changelog](#su-13-august-2017--v825) to learn more about the new Iris Controllers feature.

# Su, 13 August 2017 | v8.2.5

Good news for devs that are used to write their web apps using the `MVC-style` app architecture.

Yesterday I wrote a [tutorial](tutorial/mvc-from-scratch) on how you can transform your raw `Handlers` to `Controllers` using the existing tools only ([Iris is the most modular web framework out there](https://medium.com/@corebreaker/iris-web-cd684b4685c7), we all have no doubt about this).

Today, I did implement the `Controller` idea as **built'n feature inside Iris**.
Our `Controller` supports many things among them are:

- all HTTP Methods are supported, for example if want to serve `GET` then the controller should have a function named `Get()`, you can define more than one method function to serve in the same Controller struct
- `persistence` data inside your Controller struct (share data between requests) via **`iris:"persistence"`** tag right to the field
- optional `BeginRequest(ctx)` function to perform any initialization before the methods, useful to call middlewares or when many methods use the same collection of data
- optional `EndRequest(ctx)` function to perform any finalization after the methods executed
- access to the request path parameters via the `Params` field
- access to the template file that should be rendered via the `Tmpl` field
- access to the template data that should be rendered inside the template file via `Data` field
- access to the template layout via the `Layout` field
- access to the low-level `context.Context` via the `Ctx` field
- flow as you used to, `Controllers` can be registered to any `Party`, including Subdomains, the Party's begin and done handlers work as expected. 

It's very easy to get started, the only function you need to call instead of `app.Get/Post/Put/Delete/Connect/Head/Patch/Options/Trace` is the `app.Controller`.

Example Code:

```go
// file: main.go

package main

import (
    "github.com/kataras/iris"

    "controllers"
)

func main() {
    app := iris.New()
    app.RegisterView(iris.HTML("./views", ".html"))

    app.Controller("/", new(controllers.Index))

    // http://localhost:8080/
    app.Run(iris.Addr(":8080"))
}

```

```go
// file: controllers/index.go

package controllers

import (
    "github.com/kataras/iris/core/router"
)

// Index is our index example controller.
type Index struct {
    mvc.Controller
    // if you're using go1.9: 
    // you can omit the /core/router import statement
    // and just use the `iris.Controller` instead.
}

// will handle GET method on http://localhost:8080/
func (c *Index) Get() {
    c.Tmpl = "index.html"
    c.Data["title"] = "Index page"
    c.Data["message"] = "Hello world!"
}

// will handle POST method on http://localhost:8080/
func (c *Index) Post() {}

```

> Tip: declare a func(c *Index) All() {} or Any() to register all HTTP Methods.

A full example can be found at the [_examples/mvc](_examples/mvc) folder.


# Sa, 12 August 2017 | v8.2.4

No API Changes.

Fix https://github.com/kataras/iris/issues/717, users are welcomed to follow the thread for any questions or reports about Gzip and Static Files Handlers **only**.

# Th, 10 August 2017 | v8.2.3

No API Changes.

Fix https://github.com/kataras/iris/issues/714

Continue to v8.2.2 for more...

# Th, 10 August 2017 | v8.2.2

No API Changes.

- Implement [Google reCAPTCHA](middleware/recaptcha) middleware, example [here](_examples/miscellaneous/recaptcha/main.go)
- Fix [kataras/golog](https://github.com/kataras/golog) prints with colors on windows server 2012 while it shouldn't because its command line tool does not support 256bit colors
- Improve the updater by a custom self-updated back-end version checker, can be disabled by:

```go
app.Run(iris.Addr(":8080"), iris.WithoutVersionChecker)
```
Or
```go
app.Configure(iris.WithoutVersionChecker)
```
Or 
```go
app.Configure(iris.WithConfiguration(iris.Configuration{DisableVersionChecker:true}))
```

# Tu, 08 August 2017 | v8.2.1

No API Changes. Great news for the unique iris sessions library, once again.

**NEW**: [LevelDB-based](https://github.com/google/leveldb) session database implemented, example [here](_examples/sessions/database/leveldb/main.go).

[Redis-based sessiondb](sessions/sessiondb/redis) has no longer the `MaxAgeSeconds` config field,
this is passed automatically by the session manager, now.

All [sessions databases](sessions/sessiondb) have an `Async(bool)` function, if turned on
then all synchronization between the memory store and the back-end database will happen
inside different go routines. By-default async is false but it's recommended to turn it on, it will make sessions to be stored faster, at most.

All reported issues have been fixed, the API is simplified by `v8.2.0` so everyone can
create and use any back-end storage for application's sessions persistence.

# Mo, 07 August 2017 | v8.2.0

No Common-API Changes.

Good news for [iris sessions back-end databases](_examples/sessions) users.

<details>
<summary>Info for session database authors</summary>
Session Database API Changed to:

```go
type Database interface {
	Load(sid string) RemoteStore
	Sync(p SyncPayload)
}

// SyncPayload reports the state of the session inside a database sync action.
type SyncPayload struct {
	SessionID string

	Action Action
	// on insert it contains the new key and the value
	// on update it contains the existing key and the new value
	// on delete it contains the key (the value is nil)
	// on clear it contains nothing (empty key, value is nil)
	// on destroy it contains nothing (empty key, value is nil)
	Value memstore.Entry
	// Store contains the whole memory store, this store
	// contains the current, updated from memory calls,
	// session data (keys and values). This way
	// the database has access to the whole session's data
	// every time.
	Store RemoteStore
}


// RemoteStore is a helper which is a wrapper
// for the store, it can be used as the session "table" which will be
// saved to the session database.
type RemoteStore struct {
	// Values contains the whole memory store, this store
	// contains the current, updated from memory calls,
	// session data (keys and values). This way
	// the database has access to the whole session's data
	// every time.
	Values memstore.Store
	// on insert it contains the expiration datetime
	// on update it contains the new expiration datetime(if updated or the old one)
	// on delete it will be zero
	// on clear it will be zero
	// on destroy it will be zero
	Lifetime LifeTime
}
```

Read more at [sessions/database.go](sessions/database.go), view how three built'n session databases are being implemented [here](sessions/sessiondb).
</details> 

All sessions databases are updated and they performant even faster than before.

- **NEW** raw file-based session database implemented, example [here](_examples/sessions/database/file)
- **NEW** [boltdb-based](https://github.com/boltdb/bolt) session database implemented, example [here](_examples/sessions/database/boltdb) (recommended as it's safer and faster)
- [redis sessiondb](_examples/sessions/database/redis) updated to the latest api

Under the cover, session database works entirely differently than before but nothing changed from the user's perspective, so upgrade with `go get -u github.com/kataras/iris` and sleep well.

# Tu, 01 August 2017 | v8.1.3

- Add `Option` function to the `html view engine`: https://github.com/kataras/iris/issues/694
- Fix sessions backend databases restore expiration: https://github.com/kataras/iris/issues/692 by @corebreaker
- Add `PartyFunc`, same as `Party` but receives a function with the sub router as its argument instead [GO1.9 Users-ONLY]

# Mo, 31 July 2017 | v8.1.2

Add a `ConfigureHost` function as an alternative way to customize the hosts via `host.Configurator`.
The first way was to pass `host.Configurator` as optional arguments on `iris.Runner`s built'n functions (`iris#Server, iris#Listener, iris#Addr, iris#TLS, iris#AutoTLS`), example of this can be found [there](https://github.com/kataras/iris/blob/master/_examples/http-listening/notify-on-shutdown).

Example Code:

```go
package main

import (
	stdContext "context"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/host"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.HTML("<h1>Hello, try to refresh the page after ~10 secs</h1>")
	})

    app.ConfigureHost(configureHost) // or pass "configureHost" as `app.Addr` argument, same result.

	app.Logger().Info("Wait 10 seconds and check your terminal again")
	// simulate a shutdown action here...
	go func() {
		<-time.After(10 * time.Second)
		timeout := 5 * time.Second
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
		defer cancel()
		// close all hosts, this will notify the callback we had register
		// inside the `configureHost` func.
		app.Shutdown(ctx)
	}()

	// http://localhost:8080
	// wait 10 seconds and check your terminal.
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

func configureHost(su *host.Supervisor) {
	// here we have full access to the host that will be created
	// inside the `app.Run` or `app.NewHost` function .
	//
	// we're registering a shutdown "event" callback here:
	su.RegisterOnShutdown(func() {
		println("server is closed")
	})
	// su.RegisterOnError
	// su.RegisterOnServe
}
```

# Su, 30 July 2017

Greetings my friends, nothing special today, no version number yet.

We just improve the, external, Iris Logging library and the `Columns` config field from `middleware/logger` defaults to `false` now. Upgrade with `go get -u github.com/kataras/iris` and have fun!

# Sa, 29 July 2017 | v8.1.1

No breaking changes, just an addition to make your life easier.

This feature has been implemented after @corebreaker 's request, posted at: https://github.com/kataras/iris/issues/688. He was also tried to fix that by a [PR](https://github.com/kataras/iris/pull/689), we thanks him but the problem with that PR was the duplication and the separation of concepts, however we thanks him for pushing for a solution. The current feature's implementation gives a permant solution to host supervisor access issues.

Optional host configurators added to all common serve and listen functions.

Below you'll find how to gain access to the host, **the second way is the new feature.**

### Hosts

Access to all hosts that serve your application can be provided by
the `Application#Hosts` field, after the `Run` method.

But the most common scenario is that you may need access to the host before the `Run` method,
there are two ways of gain access to the host supervisor, read below.

First way is to use the `app.NewHost` to create a new host
and use one of its `Serve` or `Listen` functions
to start the application via the `iris#Raw` Runner.
Note that this way needs an extra import of the `net/http` package.

Example Code:

```go
h := app.NewHost(&http.Server{Addr:":8080"})
h.RegisterOnShutdown(func(){
    println("server was closed!")
})

app.Run(iris.Raw(h.ListenAndServe))
```

Second, and probably easier way is to use the `host.Configurator`.

Note that this method requires an extra import statement of
"github.com/kataras/iris/core/host" when using go < 1.9,
if you're targeting on go1.9 then you can use the `iris#Supervisor`
and omit the extra host import.

All common `Runners` we saw earlier (`iris#Addr, iris#Listener, iris#Server, iris#TLS, iris#AutoTLS`)
accept a variadic argument of `host.Configurator`, there are just `func(*host.Supervisor)`.
Therefore the `Application` gives you the rights to modify the auto-created host supervisor through these.


Example Code:

```go
package main

import (
    stdContext "context"
    "time"

    "github.com/kataras/iris"
    "github.com/kataras/iris/context"
    "github.com/kataras/iris/core/host"
)

func main() {
    app := iris.New()

    app.Get("/", func(ctx context.Context) {
        ctx.HTML("<h1>Hello, try to refresh the page after ~10 secs</h1>")
    })

    app.Logger().Info("Wait 10 seconds and check your terminal again")
    // simulate a shutdown action here...
    go func() {
        <-time.After(10 * time.Second)
        timeout := 5 * time.Second
        ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
        defer cancel()
        // close all hosts, this will notify the callback we had register
        // inside the `configureHost` func.
        app.Shutdown(ctx)
    }()

    // start the server as usual, the only difference is that
    // we're adding a second (optional) function
    // to configure the just-created host supervisor.
    //
    // http://localhost:8080
    // wait 10 seconds and check your terminal.
    app.Run(iris.Addr(":8080", configureHost), iris.WithoutServerError(iris.ErrServerClosed))

}

func configureHost(su *host.Supervisor) {
    // here we have full access to the host that will be created
    // inside the `Run` function.
    //
    // we register a shutdown "event" callback
    su.RegisterOnShutdown(func() {
        println("server is closed")
    })
    // su.RegisterOnError
    // su.RegisterOnServe
}
```

Read more about listening and gracefully shutdown by navigating to: https://github.com/kataras/iris/tree/master/_examples/#http-listening

# We, 26 July 2017 | v8.1.0

The `app.Logger() *logrus.Logger` was replaced with a custom implementation [[golog](https://github.com/kataras/golog)], it's compatible with the [logrus](https://github.com/sirupsen/logrus) package and other open-source golang loggers as well, because of that: https://github.com/kataras/iris/issues/680#issuecomment-316184570. 

The API didn't change much except these:

-  the new implementation does not recognise `Fatal` and `Panic` because, actually, iris never panics
- the old `app.Logger().Out = io.Writer` should be written as `app.Logger().SetOutput(io.Writer)`

The new implementation, [golog](https://github.com/kataras/golog) is featured, **[three times faster than logrus](https://github.com/kataras/golog/tree/master/_benchmarks)**
and it completes every common usage.

### Integration

I understand that many of you may use logrus outside of Iris too. To integrate an external `logrus` logger just 
`Install` it-- all print operations will be handled by the provided `logrus instance`.

```go
import (
    "github.com/kataras/iris"
    "github.com/sirupsen/logrus"
)

package main(){
    app := iris.New()
    app.Logger().Install(logrus.StandardLogger()) // the package-level logrus instance
    // [...]
}
```

For more information about our new logger please navigate to: https://github.com/kataras/golog -  contributions are welcomed as well!

# Sa, 23 July 2017 | v8.0.7

Fix [It's true that with UseGlobal the "/path1.txt" route call the middleware but cause the prepend, the order is inversed](https://github.com/kataras/iris/issues/683#issuecomment-317229068)

# Sa, 22 July 2017 | v8.0.5 & v8.0.6

No API Changes.

### Performance

Add an experimental [Configuration#EnableOptimizations](https://github.com/kataras/iris/blob/master/configuration.go#L170) option.

```go
type Configuration {
    // [...]

    // EnableOptimization when this field is true
    // then the application tries to optimize for the best performance where is possible.
    //
    // Defaults to false.
    EnableOptimizations bool `yaml:"EnableOptimizations" toml:"EnableOptimizations"`

    // [...]
}
```

Usage:

```go
app.Run(iris.Addr(":8080"), iris.WithOptimizations)
```

### Django view engine

@corebreaker pushed a [PR](https://github.com/kataras/iris/pull/682) to solve the [Problem for {%extends%} in Django Engine with embedded files](https://github.com/kataras/iris/issues/681).

### Logger

Remove the `vendor/github.com/sirupsen/logrus` folder, as a temporary solution for the https://github.com/kataras/iris/issues/680#issuecomment-316196126.

#### Future versions

The logrus will be replaced with a custom implementation, because of that: https://github.com/kataras/iris/issues/680#issuecomment-316184570. 

As far as we know, @kataras is working on this new implementation, see [here](https://github.com/kataras/iris/issues/680#issuecomment-316544906), 
which will be compatible with the logrus package and other open-source golang loggers as well.


# Mo, 17 July 2017 | v8.0.4

No API changes.

### HTTP Errors

Fix a rare behavior: error handlers are not executed correctly
when a before-handler by-passes the order of execution, relative to the [previous feature](https://github.com/kataras/iris/blob/master/HISTORY.md#su-16-july-2017--v803). 

### Request Logger

Add `Configuration#MessageContextKey`. Example can be found at [_examples/http_request/request-logger/main.go](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/main.go#L48).

# Su, 16 July 2017 | v8.0.3

No API changes.

Relative issues: 

- https://github.com/kataras/iris/issues/674
- https://github.com/kataras/iris/issues/675
- https://github.com/kataras/iris/issues/676

### HTTP Errors

Able to register a chain of Handlers (and middleware with `ctx.Next()` support like routes) for a specific error code, read more at [issues/674](https://github.com/kataras/iris/issues/674). Usage example can be found at [_examples/http_request/request-logger/main.go](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/main.go#L41).


New function to register a Handler or a chain of Handlers for all official http error codes, by calling the new `app.OnAnyErrorCode(func(ctx context.Context){})`, read more at [issues/675](https://github.com/kataras/iris/issues/675). Usage example can be found at [_examples/http_request/request-logger/main.go](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/main.go#L42).

### Request Logger

Add `Configuration#LogFunc` and `Configuration#Columns` fields, read more at [issues/676](https://github.com/kataras/iris/issues/676). Example can be found at [_examples/http_request/request-logger/request-logger-file/main.go](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/request-logger-file/main.go).


Have fun and don't forget to [star](https://github.com/kataras/iris/stargazers) the github repository, it gives me power to continue publishing my work!

# Sa, 15 July 2017 | v8.0.2

Okay my friends, this is a good time to upgrade, I did implement a feature that you were asking many times at the past.

Iris' router can now handle root-level wildcard paths `app.Get("/{paramName:path})`.

In case you're wondering: no it does not conflict with other static or dynamic routes, meaning that you can code something like this:

```go
// it isn't conflicts with the rest of the static routes or dynamic routes with a path prefix.
app.Get("/{pathParamName:path}", myHandler) 
```

Or even like this:

```go
package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	// this works as expected now,
	// will handle all GET requests
	// except:
	// /                     -> because of app.Get("/", ...)
	// /other/anything/here  -> because of app.Get("/other/{paramother:path}", ...)
	// /other2/anything/here -> because of app.Get("/other2/{paramothersecond:path}", ...)
	// /other2/static        -> because of app.Get("/other2/static", ...)
	//
	// It isn't conflicts with the rest of the routes, without routing performance cost!
	//
	// i.e /something/here/that/cannot/be/found/by/other/registered/routes/order/not/matters
	app.Get("/{p:path}", h)

	// this will handle only GET /
	app.Get("/", staticPath)

	// this will handle all GET requests starting with "/other/"
	//
	// i.e /other/more/than/one/path/parts
	app.Get("/other/{paramother:path}", other)

	// this will handle all GET requests starting with "/other2/"
	// except /other2/static (because of the next static route)
	//
	// i.e /other2/more/than/one/path/parts
	app.Get("/other2/{paramothersecond:path}", other2)

	// this will handle only GET /other2/static
	app.Get("/other2/static", staticPath)

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

func h(ctx context.Context) {
	param := ctx.Params().Get("p")
	ctx.WriteString(param)
}

func other(ctx context.Context) {
	param := ctx.Params().Get("paramother")
	ctx.Writef("from other: %s", param)
}

func other2(ctx context.Context) {
	param := ctx.Params().Get("paramothersecond")
	ctx.Writef("from other2: %s", param)
}

func staticPath(ctx context.Context) {
	ctx.Writef("from the static path: %s", ctx.Path())
}
``` 

If you find any bugs with this change please send me a [chat message](https://kataras.rocket.chat/channel/iris) in order to investigate it, I'm totally free at weekends.

Have fun and don't forget to [star](https://github.com/kataras/iris/stargazers) the github repository, it gives me power to continue publishing my work!

# Th, 13 July 2017 | v8.0.1

Nothing tremendous at this minor version.

We've just added a configuration field in order to ignore errors received by the `Run` function, see below.

[Configuration#IgnoreServerErrors](https://github.com/kataras/iris/blob/master/configuration.go#L255)
```go
type Configuration struct {
    // [...]

    // IgnoreServerErrors will cause to ignore the matched "errors"
    // from the main application's `Run` function.
    // This is a slice of string, not a slice of error
    // users can register these errors using yaml or toml configuration file
    // like the rest of the configuration fields.
    //
    // See `WithoutServerError(...)` function too.
    //
    // Defaults to an empty slice.
    IgnoreServerErrors []string `yaml:"IgnoreServerErrors" toml:"IgnoreServerErrors"`

    // [...]
}
```
[Configuration#WithoutServerError](https://github.com/kataras/iris/blob/master/configuration.go#L106)
```go
// WithoutServerError will cause to ignore the matched "errors"
// from the main application's `Run` function.
//
// Usage:
// err := app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
// will return `nil` if the server's error was `http/iris#ErrServerClosed`.
//
// See `Configuration#IgnoreServerErrors []string` too.
WithoutServerError(errors ...error) Configurator
```

By default no error is being ignored, of course.

Example code:
[_examples/http-listening/listen-addr/omit-server-errors](https://github.com/kataras/iris/tree/master/_examples/http-listening/listen-addr/omit-server-errors)
```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/context"
)

func main() {
    app := iris.New()

    app.Get("/", func(ctx context.Context) {
    	ctx.HTML("<h1>Hello World!/</h1>")
    })

    err := app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
    if err != nil {
        // do something
    }
    // same as:
    // err := app.Run(iris.Addr(":8080"))
    // if err != nil && (err != iris.ErrServerClosed || err.Error() != iris.ErrServerClosed.Error()) {
    //     [...]
    // }
}
```

At first we didn't want to implement something like that because it's ridiculous easy to do it manually but a second thought came to us,
that many applications are based on configuration, therefore it would be nice to have something to ignore errors
by simply string values that can be passed to the application's configuration via `toml` or `yaml` files too.

This feature has been implemented after a request of ignoring the `iris/http#ErrServerClosed` from the `Run` function: 
https://github.com/kataras/iris/issues/668

# Mo, 10 July 2017 | v8.0.0

## 📈 One and a half years with Iris and You...

Despite the deflamations, the clickbait articles, the removed posts of mine at reddit/r/golang, the unexpected and inadequate ban from the gophers slack room by @dlsniper alone the previous week without any reason or inform, Iris is still here and will be.

- 7070 github stars
- 749 github forks
- 1m total views at its documentation
- ~800$ at donations (there're a lot for a golang open-source project, thanks to you)
- ~550 reported bugs fixed
- ~30 community feature requests have been implemented

## 🔥 Reborn

As you may have heard I have huge responsibilities on my new position at Dubai nowadays, therefore I don't have the needed time to work on this project anymore.

After a month of negotiations and searching I succeed to find a decent software engineer to continue my work on the open source community.

The leadership of this, open-source, repository was transferred to [hiveminded](https://github.com/hiveminded), the author of iris-based [get-ion/ion](https://github.com/get-ion/ion), he actually did an excellent job on the framework, he kept the code as minimal as possible and at the same time added more features, examples and middleware(s).

These types of projects need heart and sacrifices to continue offer the best developer experience like a paid software, please do support him as you did with me!

## 📰 Changelog

> app. = `app := iris.New();` **app.**

> ctx. = `func(ctx context.Context) {` **ctx.** `}`

### Docker

Docker and kubernetes integration showcase, see the [iris-contrib/cloud-native-go](https://github.com/iris-contrib/cloud-native-go) repository as an example.

### Logger

* Logger which was an `io.Writer` was replaced with the pluggable `logrus`.
    * which you still attach an `io.Writer` with `app.Logger().Out = an io.Writer`.
    * iris as always logs only critical errors, you can disable them with `app.Logger().Level = iris.NoLog`
    * the request logger outputs the incoming requests as INFO level.

### Sessions

Remove `ctx.Session()` and `app.AttachSessionManager`, devs should import and use the `sessions` package as standalone, it's totally optional, devs can use any other session manager too. [Examples here](sessions#table-of-contents).

### Websockets

The `github.com/kataras/iris/websocket` package does not handle the endpoint and client side automatically anymore. Example code:

```go
func setupWebsocket(app *iris.Application) {
    // create our echo websocket server
    ws := websocket.New(websocket.Config{
    	ReadBufferSize:  1024,
    	WriteBufferSize: 1024,
    })
    ws.OnConnection(handleConnection)
    // serve the javascript built'n client-side library,
    // see weboskcets.html script tags, this path is used.
    app.Any("/iris-ws.js", func(ctx context.Context) {
    	ctx.Write(websocket.ClientSource)
    })

    // register the server on an endpoint.
    // see the inline javascript code in the websockets.html, this endpoint is used to connect to the server.
    app.Get("/echo", ws.Handler())
}
```

> More examples [here](websocket#table-of-contents)

### View

Rename `app.AttachView(...)` to `app.RegisterView(...)`.

Users can omit the import of `github.com/kataras/iris/view` and use the `github.com/kataras/iris` package to
refer to the view engines, i.e: `app.RegisterView(iris.HTML("./templates", ".html"))` is the same as `import "github.com/kataras/iris/view" [...] app.RegisterView(view.HTML("./templates" ,".html"))`.

> Examples [here](_examples/#view)

### Security

At previous versions, when you called `ctx.Remoteaddr()` Iris could parse and return the client's IP from the "X-Real-IP", "X-Forwarded-For" headers. This was a security leak as you can imagine, because the user can modify them. So we've disabled these headers by-default and add an option to add/remove request headers that are responsible to parse and return the client's real IP.

```go
// WithRemoteAddrHeader enables or adds a new or existing request header name
// that can be used to validate the client's real IP.
//
// Existing values are:
// "X-Real-Ip":             false,
// "X-Forwarded-For":       false,
// "CF-Connecting-IP": false
//
// Look `context.RemoteAddr()` for more.
WithRemoteAddrHeader(headerName string) Configurator // enables a header.
WithoutRemoteAddrHeader(headerName string) Configurator // disables a header.
```
For example, if you want to enable the "CF-Connecting-IP" header (cloudflare) 
you have to add the `WithRemoteAddrHeader` option to the `app.Run` function, at the end of your program.

```go
app.Run(iris.Addr(":8080"), iris.WithRemoteAddrHeader("CF-Connecting-IP"))
// This header name will be checked when ctx.RemoteAddr() called and if exists
// it will return the client's IP, otherwise it will return the default *http.Request's `RemoteAddr` field.
```

### Miscellaneous

Fix [typescript tools](typescript).

[_examples](_examples/) folder has been ordered by feature and usage:
    - contains tests on some examples
    - new examples added, one of them shows how the `reuseport` feature on UNIX and BSD systems can be used to listen for incoming connections, [see here](_examples/#http-listening)


Replace supervisor's tasks with events, like `RegisterOnShutdown`, `RegisterOnError`, `RegisterOnServe` and fix the (unharmful) race condition when output the banner to the console. Global notifier for interrupt signals which can be disabled via `app.Run([...], iris.WithoutInterruptHandler)`, look [graceful-shutdown](_examples/http-listening/graceful-shutdown/main.go) example for more.


More handlers are ported to Iris (they can be used as they are without `iris.FromStd`), these handlers can be found at [iris-contrib/middleware](https://github.com/iris-contrib/middleware). Feel free to put your own there.


| Middleware | Description | Example |
| -----------|--------|-------------|
| [jwt](https://github.com/iris-contrib/middleware/tree/master/jwt) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it. | [iris-contrib/middleware/jwt/_example](https://github.com/iris-contrib/middleware/tree/master/jwt/_example) |
| [cors](https://github.com/iris-contrib/middleware/tree/master/cors) | HTTP Access Control. | [iris-contrib/middleware/cors/_example](https://github.com/iris-contrib/middleware/tree/master/cors/_example) |
| [secure](https://github.com/iris-contrib/middleware/tree/master/secure) | Middleware that implements a few quick security wins. | [iris-contrib/middleware/secure/_example](https://github.com/iris-contrib/middleware/tree/master/secure/_example/main.go) |
| [tollbooth](https://github.com/iris-contrib/middleware/tree/master/tollboothic) | Generic middleware to rate-limit HTTP requests. | [iris-contrib/middleware/tollbooth/_examples/limit-handler](https://github.com/iris-contrib/middleware/tree/master/tollbooth/_examples/limit-handler) |
| [cloudwatch](https://github.com/iris-contrib/middleware/tree/master/cloudwatch) |  AWS cloudwatch metrics middleware. |[iris-contrib/middleware/cloudwatch/_example](https://github.com/iris-contrib/middleware/tree/master/cloudwatch/_example) |
| [new relic](https://github.com/iris-contrib/middleware/tree/master/newrelic) | Official [New Relic Go Agent](https://github.com/newrelic/go-agent). | [iris-contrib/middleware/newrelic/_example](https://github.com/iris-contrib/middleware/tree/master/newrelic/_example) |
| [prometheus](https://github.com/iris-contrib/middleware/tree/master/prometheus)| Easily create metrics endpoint for the [prometheus](http://prometheus.io) instrumentation tool | [iris-contrib/middleware/prometheus/_example](https://github.com/iris-contrib/middleware/tree/master/prometheus/_example) |


v7.x is deprecated because it sold as it is and it is not part of the public, stable `gopkg.in` iris versions. Developers/users of this library should upgrade their apps to v8.x, the refactor process will cost nothing for most of you, as the most common API remains as it was. The changelog history from that are being presented below.


# Th, 15 June 2017 | v7.2.0

### About our new home page
    https://iris-go.com

Thanks to [Santosh Anand](https://github.com/santoshanand) the https://iris-go.com has been upgraded and it's really awesome!

[Santosh](https://github.com/santoshanand) is a freelancer, he has a great knowledge of nodejs and express js, Android, iOS, React Native, Vue.js etc, if you need a developer to find or create a solution for your problem or task, please contact with him.


The amount of the next two or three donations you'll send they will be immediately transferred to his own account balance, so be generous please!

### Cache

Declare the `iris.Cache alias` to the new, improved and most-suited for common usage, `cache.Handler function`.

`iris.Cache` be used as middleware in the chain now, example [here](_examples/intermediate/cache-markdown/main.go). However [you can still use the cache as a wrapper](cache/cache_test.go) by importing the `github.com/kataras/iris/cache` package. 


### File server

- **Fix** [that](https://github.com/iris-contrib/community-board/issues/12).

- `app.StaticHandler(requestPath string, systemPath string, showList bool, gzip bool)` -> `app.StaticHandler(systemPath,showList bool, gzip bool)`

- **New** feature for Single Page Applications, `app.SPA(assetHandler context.Handler)` implemented.

- **New** `app.StaticEmbeddedHandler(vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string)` added in order to be able to pass that on `app.SPA(app.StaticEmbeddedHandler("./public", Asset, AssetNames))`.

- **Fix** `app.StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string)`.

Examples: 
- [Embedding Files Into Executable App](_examples/file-server/embedding-files-into-app)
- [Single Page Application](_examples/file-server/single-page-application)
- [Embedding Single Page Application](_examples/file-server/embedding-single-page-application)

> [app.StaticWeb](_examples/file-server/basic/main.go) doesn't works for root request path "/"  anymore, use the new `app.SPA` instead.   

### WWW subdomain entry

- [Example](_examples/subdomains/www/main.go) added to copy all application's routes, including parties, to the `www.mydomain.com`


### Wrapping the Router

- [Example](_examples/routing/custom-wrapper/main.go) added to show you how you can use the `app.WrapRouter` 
to implement a similar to `app.SPA` functionality, don't panic, it's easier than it sounds.


### Testing

- `httptest.New(app *iris.Application, t *testing.T)` -> `httptest.New(t *testing.T, app *iris.Application)`.

- **New** `httptest.NewLocalListener() net.Listener` added.
- **New** `httptest.NewLocalTLSListener(tcpListener net.Listener) net.Listener` added.

Useful for testing tls-enabled servers: 

Proxies are trying to understand local addresses in order to allow `InsecureSkipVerify`.

-  `host.ProxyHandler(target *url.URL) *httputil.ReverseProxy`.
-  `host.NewProxy(hostAddr string, target *url.URL) *Supervisor`.
        
    Tests [here](core/host/proxy_test.go).

# Tu, 13 June 2017 | v7.1.1

Fix [that](https://github.com/iris-contrib/community-board/issues/11).

# Mo, 12 June 2017 | v7.1.0

Fix [that](https://github.com/iris-contrib/community-board/issues/10).


# Su, 11 June 2017 | v7.0.5

Iris now supports static paths and dynamic paths for the same path prefix with zero performance cost:

`app.Get("/profile/{id:int}", handler)` and `app.Get("/profile/create", createHandler)` are not in conflict anymore.


The rest of the special Iris' routing features, including static & wildcard subdomains are still work like a charm.

> This was one of the most popular community's feature requests. Click [here](https://github.com/kataras/iris/blob/master/_examples/beginner/routing/overview/main.go) to see a trivial example.

# Sa, 10 June 2017 | v7.0.4

- Simplify and add a test for the [basicauth middleware](https://github.com/kataras/iris/tree/master/middleware/basicauth), no need to be
stored inside the Context anymore, developers can get the validated user(username and password) via `context.Request().BasicAuth()`. `basicauth.Config.ContextKey` was removed, just remove that field from your configuration, it's useless now. 

# Sa, 10 June 2017 | v7.0.3

- New `context.Session().PeekFlash("key")` added, unlike `GetFlash` this will return the flash value but keep the message valid for the next requests too.
- Complete the [httptest example](https://github.com/iris-contrib/examples/tree/master/httptest).
- Fix the (marked as deprecated) `ListenLETSENCRYPT` function.
- Upgrade the [iris-contrib/middleware](https://github.com/iris-contrib/middleware) including JWT, CORS and Secure handlers.
- Add [OAuth2 example](https://github.com/iris-contrib/examples/tree/master/oauth2) -- showcases the third-party package [goth](https://github.com/markbates/goth) integration with Iris.

### Community

 - Add github integration on https://kataras.rocket.chat/channel/iris , so users can login with their github accounts instead of creating new for the chat only.

# Th, 08 June 2017 | v7.0.2

- Able to set **immutable** data on sessions and context's storage. Aligned to fix an issue on slices and maps as reported [here](https://github.com/iris-contrib/community-board/issues/5).

# We, 07 June 2017 | v7.0.1

- Proof of concept of an internal release generator, navigate [here](https://github.com/iris-contrib/community-board/issues/2) to read more. 
- Remove tray icon "feature", click [here](https://github.com/iris-contrib/community-board/issues/1) to learn why.

# Sa, 03 June 2017 

After 2+ months of hard work and collaborations, Iris [version 7](https://github.com/kataras/iris) was published earlier today.

If you're new to Iris you don't have to read all these, just navigate to the [updated examples](https://github.com/kataras/iris/tree/master/_examples) and you should be fine:)

Note that this section will not
cover the internal changes, the difference is so big that anybody can see them with a glimpse, even the code structure itself.


## Changes from [v6](https://github.com/kataras/iris/tree/v6)

The whole framework was re-written from zero but I tried to keep the most common public API that iris developers use.

Vendoring /w update 

The previous vendor action for v6 was done by-hand, now I'm using the [go dep](https://github.com/golang/dep) tool, I had to do
some small steps:

- remove files like testdata to reduce the folder size
- rollback some of the "golang/x/net/ipv4" and "ipv6" source files because they are downloaded to their latest versions
by go dep, but they had lines with the `typealias` feature, which is not ready by current golang version (it will be on August)
- fix "cannot use internal package" at golang/x/net/ipv4 and ipv6 packages
	- rename the interal folder to was-internal, everywhere and fix its references.
- fix "main redeclared in this block"
	- remove all examples folders.
- remove main.go files on jsondiff lib, used by gavv/httpexpect, produces errors on `test -v ./...` while jd and jp folders are not used at all.

The go dep tool does what is says, as expected, don't be afraid of it now.
I am totally recommending this tool for package authors, even if it's in its alpha state.
I remember when Iris was in its alpha state and it had 4k stars on its first weeks/or month and that helped me a lot to fix reported bugs by users and make the framework even better, so give love to go dep from today!

General

- Several enhancements for the typescript transpiler, view engine, websocket server and sessions manager
- All `Listen` methods replaced with a single `Run` method, see [here](https://github.com/kataras/iris/tree/master/_examples/beginner/listening)
- Configuration, easier to modify the defaults, see [here](https://github.com/kataras/iris/tree/master/_examples/beginner/cofiguration)
- `HandlerFunc` removed, just `Handler` of `func(context.Context)` where context.Context derives from `import "github.com/kataras/iris/context"` (**NEW**: this import path is optional, use `iris.Context` if you've installed Go 1.9)
    - Simplify API, i.e: instead of `Handle,HandleFunc,Use,UseFunc,Done,DoneFunc,UseGlobal,UseGlobalFunc` use `Handle,Use,Done,UseGlobal`.
- Response time decreased even more (9-35%, depends on the application)
- The `Adaptors` idea replaced with a more structural design pattern, but you have to apply these changes: 
    - `app.Adapt(view.HTML/Pug/Amber/Django/Handlebars...)` -> `app.AttachView(view.HTML/Pug/Amber/Django/Handlebars...)` 
    - `app.Adapt(sessions.New(...))` -> `app.AttachSessionManager(sessions.New(...))`
    - `app.Adapt(iris.LoggerPolicy(...))` -> `app.AttachLogger(io.Writer)`
    - `app.Adapt(iris.RenderPolicy(...))` -> removed and replaced with the ability to replace the whole context with a custom one or override some methods of it, see below.

Routing
- Remove of multiple routers, now we have the fresh Iris router which is based on top of the julien's [httprouter](https://github.com/julienschmidt/httprouter).
    > Update 11 June 2017: As of 7.0.5 this is changed, read [here](https://github.com/kataras/iris/blob/master/HISTORY.md#su-11-june-2017--v705).
- Subdomains routing algorithm has been improved.
- Iris router is using a custom interpreter with parser and path evaluator to achieve the best expressiveness, with zero performance loss, you ever seen so far, i.e: 
    - `app.Get("/", "/users/{userid:int min(1)}", handler)`,
        - `{username:string}` or just `{username}`
        - `{asset:path}`,
        - `{firstname:alphabetical}`,
        - `{requestfile:file}` ,
        - `{mylowercaseParam regexp([a-z]+)}`.
        - The previous syntax of `:param` and `*param` still working as expected. Previous rules for paths confliction remain as they were.
            - Also, path parameter names should be only alphabetical now, numbers and symbols are not allowed (for your own good, I have seen a lot the last year...).

Click [here](https://github.com/kataras/iris/tree/master/_examples/beginner/routing) for details.
> It was my first attempt/experience on the interpreters field, so be good with it :)

Context
- `iris.Context pointer` replaced with `context.Context interface` as we already mention
    - in order to be able to use a custom context and/or catch lifetime like `BeginRequest` and `EndRequest` from context itself, see below
- `context.JSON, context.JSONP, context.XML, context.Markdown, context.HTML` work faster
- `context.Render("filename.ext", bindingViewData{}, options) ` -> `context.View("filename.ext")`
    - `View` renders only templates, it will not try to search if you have a restful renderer adapted, because, now, you can do it via method overriding using a custom Context.
    - Able to set `context.ViewData` and `context.ViewLayout` via middleware when executing a template.
- `context.SetStatusCode(statusCode)` -> `context.StatusCode(statusCode)`
    - which is equivalent with the old `EmitError` too:
        - if status code >=400 given can automatically fire a custom http error handler if response wasn't written already.
    - `context.StatusCode()` -> `context.GetStatusCode()`
    - `app.OnError` -> `app.OnErrorCode`
    - Errors per party are removed by-default, you can just use one global error handler with logic like "if path starts with 'prefix' fire this error handler, else...". 
- Easy way to change Iris' default `Context` with a custom one, see [here](https://github.com/kataras/iris/tree/master/_examples/intermediate/custom-context)
- `context.ResponseWriter().SetBeforeFlush(...)` works for Flush and HTTP/2 Push, respectfully
- Several improvements under the `Request transactions` 
- Remember that you had to set a status code on each of the render-relative methods? Now it's not required, it just renders
with the status code that user gave with `context.StatusCode` or with `200 OK`, i.e:
    -`context.JSON(iris.StatusOK, myJSON{})` -> `context.JSON(myJSON{})`.
    - Each one of the context's render methods has optional per-call settings,
    - **the new API is even more easier to read, understand and use.**

Server
- Able to set custom underline *http.Server(s) with new Host (aka Server Supervisor) feature 
    - `Done` and `Err` channels to catch shutdown or any errors on custom hosts,
    - Schedule custom tasks(with cancelation) when server is running, see [here](https://github.com/kataras/iris/tree/master/_examples/intermediate/graceful-shutdown)
- Interrupt handler task for gracefully shutdown (when `CTRL/CMD+C`) are enabled by-default, you can disable its via configuration: `app.Run(iris.Addr(":8080"), iris.WithoutInterruptHandler)`

Future plans
- Future Go1.9's [ServeTLS](https://go-review.googlesource.com/c/38114/2/src/net/http/server.go) is ready when 1.9 released
- Future Go1.9's typealias feature is ready when 1.9 released, i.e `context.Context` -> `iris.Context` just one import path instead of todays' two.