# History

**How to upgrade**: remove your `$GOPATH/src/github.com/kataras` folder, open your command-line and execute this command: `go get -u github.com/kataras/iris/iris`.

## 6.1.4 -> 6.2.0

> Note: I want you to know that I spent more than 200 hours (16 days of ~10-15 hours per-day, do the math) for this release, two days to write these changes, please read the sections before think that you have an issue and post a new question, thanks!


Users already notified for some breaking-changes, this section will help you
to adapt the new changes to your application, it contains an overview of the new features too.

- Router (two lines to add, new features)
- Template engines (two lines to add, same features as before, except their easier configuration)
- Basic middleware, that have been written by me, are transfared to the main repository[/middleware](https://github.com/kataras/iris/tree/master/middleware) with a lot of improvements to the `recover middleware` (see the next)
- `func(http.ResponseWriter, r *http.Request, next http.HandlerFunc)` signature is fully compatible using `iris.ToHandler` helper wrapper func, without any need of custom boilerplate code. So all net/http middleware out there are supported, no need to re-invert the world here, search to the internet and you'll find a suitable to your case.
- Developers can use a `yaml` files for the configuration using the `iris.YAML` function: `app := iris.New(iris.YAML("myconfiguration.yaml"))`

Fixes:

- Websocket improvements and fix errors when using custom golang client
- Sessions performance improvements
- Fix cors by using `rs/cors` and add a new adaptor to be able to wrap the entire router
- Fix and improve oauth/oauth2 plugin(now adaptor)
- Improve and fix recover middleware
- Fix typescript compiler and hot-reloader plugin(now adaptor)
- Fix and improve the cloud-editor `alm/alm-tools` plugin(now adaptor)
- Fix gorillamux serve static files (custom routers are supported with a workaround, not a complete solution as they are now)
- Fix `iris run main.go` app reload while user saved the file from gogland

Changes:

- Remove all the package-level functions and variables for a default `*iris.Framework, iris.Default`
- Remove `.API`, use `iris.Handle/.HandleFunc/.Get/.Post/.Put/.Delete/.Trace/.Options/.Use/.UseFunc/.UseGlobal/.Party/` instead
- Remove `.Logger`, `.Config.IsDevelopment`, `.Config.LoggerOut`, `.Config.LoggerPrefix` you can adapt a logger which will log to each log message mode by `app.Adapt(iris.DevLogger())` or adapt a new one, it's just a `func(mode iris.LogMode, message string)`.
- Remove `.Config.DisableTemplateEngines`, are disabled by-default, you have to `.Adapt` a view engine by yourself
- Remove `context.RenderTemplateSource` you should make a new template file and use the `iris.Render` to specify an `io.Writer` like `bytes.Buffer`
- Remove  `plugins`, replaced with more pluggable echosystem that I designed from zero on this release, named `Policy` [Adaptors](https://github.com/kataras/iris/tree/master/adaptors) (all plugins have been converted, fixed and improvement, except the iriscontrol).
- `context.Log(string,...interface{})` -> `context.Log(iris.LogMode, string)`
- Remove `.Config.Websocket` , replaced with the `kataras/iris/adaptors/websocket.Config` adaptor.

- https://github.com/iris-contrib/plugin      ->  https://github.com/iris-contrib/adaptors

- `import "github.com/iris-contrib/middleware/basicauth"` -> `import "github.com/kataras/iris/middleware/basicauth"`
- `import "github.com/iris-contrib/middleware/i18n"` -> `import "github.com/kataras/iris/middleware/i18n"`
- `import "github.com/iris-contrib/middleware/logger"` -> `import "github.com/kataras/iris/middleware/logger"`
- `import "github.com/iris-contrib/middleware/recovery"` -> `import "github.com/kataras/iris/middleware/recover"`


- `import "github.com/iris-contrib/plugin/typescript"` -> `import "github.com/kataras/iris/adaptors/typescript"`
- `import "github.com/iris-contrib/plugin/editor"` -> `import "github.com/kataras/iris/adaptors/typescript/editor"`
- `import "github.com/iris-contrib/plugin/cors"` -> `import "github.com/kataras/iris/adaptors/cors"`
- `import "github.com/iris-contrib/plugin/gorillamux"` -> `import "github.com/kataras/iris/adaptors/gorillamux"`
- `import github.com/iris-contrib/plugin/oauth"` -> `import "github.com/iris-contrib/adaptors/oauth"`


- `import "github.com/kataras/go-template/html"` -> `import "github.com/kataras/iris/adaptors/view"`
- `import "github.com/kataras/go-template/django"` -> `import "github.com/kataras/iris/adaptors/view"`
- `import "github.com/kataras/go-template/pug"` -> `import "github.com/kataras/iris/adaptors/view"`
- `import "github.com/kataras/go-template/handlebars"` -> `import "github.com/kataras/iris/adaptors/view"`
- `import "github.com/kataras/go-template/amber"` -> `import "github.com/kataras/iris/adaptors/view"`

**Read more below** for the lines you have to change. Package-level removal is critical, you will have build-time errors. Router(less) is MUST, otherwise your app will fatal with a detailed error message.

> If I missed something please [chat](https://kataras.rocket.chat/channel/iris).


### Router(less)

**Iris server does not contain a default router anymore**, yes your eyes are ok.

This decision came up because of your requests of using other routers than the iris' defaulted.
At the past I gave you many workarounds, but they are just workarounds, not a complete solution.

**Don't worry:**

- you have to add only two lines, one is the `import path` and another is the `.Adapt`, after the `iris.New()`, so it can be tolerated.
- you are able to use all iris' features as you used before, **the API for routing has not been changed**.

Two routers available to use, today:


- [httprouter](https://github.com/kataras/iris/tree/master/adaptors/httprouter), the old defaulted. A router that can be adapted, it's a custom version of https://github.comjulienschmidt/httprouter which is edited to support iris' subdomains, reverse routing, custom http errors and a lot features, it should be a bit faster than the original too because of iris' Context. It uses `/mypath/:firstParameter/path/:secondParameter` and `/mypath/*wildcardParamName` .


Example:

```go
package main

import (
  "github.com/kataras/iris"
  "github.com/kataras/iris/adaptors/httprouter" // <---- NEW
)

func main() {
  app := iris.New()
  app.Adapt(iris.DevLogger())
  app.Adapt(httprouter.New()) // <---- NEW


  app.OnError(iris.StatusNotFound, func(ctx *iris.Context){
    ctx.HTML(iris.StatusNotFound, "<h1> custom http error page </h1>")
  })


  app.Get("/healthcheck", h)

  gamesMiddleware := func(ctx *iris.Context) {
    println(ctx.Method() + ": " + ctx.Path())
    ctx.Next()
  }

  games:= app.Party("/games", gamesMiddleware)
  { // braces are optional of course, it's just a style of code
	 games.Get("/:gameID/clans", h)
	 games.Get("/:gameID/clans/clan/:publicID", h)
	 games.Get("/:gameID/clans/search", h)

	 games.Put("/:gameID/players/:publicID", h)
	 games.Put("/:gameID/clans/clan/:publicID", h)

	 games.Post("/:gameID/clans", h)
	 games.Post("/:gameID/players", h)
	 games.Post("/:gameID/clans/:publicID/leave", h)
	 games.Post("/:gameID/clans/:clanPublicID/memberships/application", h)
	 games.Post("/:gameID/clans/:clanPublicID/memberships/application/:action", h)
	 games.Post("/:gameID/clans/:clanPublicID/memberships/invitation", h)
	 games.Post("/:gameID/clans/:clanPublicID/memberships/invitation/:action", h)
	 games.Post("/:gameID/clans/:clanPublicID/memberships/delete", h)
	 games.Post("/:gameID/clans/:clanPublicID/memberships/promote", h)
	 games.Post("/:gameID/clans/:clanPublicID/memberships/demote", h)
  }

  app.Get("/anything/*anythingparameter", func(ctx *iris.Context){
    s := ctx.Param("anythingparameter")
    ctx.Writef("The path after /anything is: %s",s)
  })

  app.Listen(":80")

	/*
		gameID  = 1
		publicID = 2
		clanPublicID = 22
		action = 3

		GET
		http://localhost/healthcheck
		http://localhost/games/1/clans
		http://localhost/games/1/clans/clan/2
		http://localhost/games/1/clans/search

		PUT
		http://localhost/games/1/players/2
		http://localhost/games/1/clans/clan/2

		POST
		http://localhost/games/1/clans
		http://localhost/games/1/players
		http://localhost/games/1/clans/2/leave
		http://localhost/games/1/clans/22/memberships/application -> 494
		http://localhost/games/1/clans/22/memberships/application/3- > 404
		http://localhost/games/1/clans/22/memberships/invitation
		http://localhost/games/1/clans/22/memberships/invitation/3
		http://localhost/games/1/clans/2/memberships/delete
		http://localhost/games/1/clans/22/memberships/promote
		http://localhost/games/1/clans/22/memberships/demote

	*/
}

func h(ctx *iris.Context) {
	ctx.HTML(iris.StatusOK, "<h1>Path<h1/>"+ctx.Path())
}

```

- [gorillamux](https://github.com/kataras/iris/tree/master/adaptors/gorillamux), a router that can be adapted, it's the https://github.com/gorilla/mux which supports subdomains, custom http errors, reverse routing, pattern matching via regex and the rest of the iris' features.


Example:

```go
package main

import (
  "github.com/kataras/iris"
  "github.com/kataras/iris/adaptors/gorillamux" // <---- NEW
)

func main() {
  app := iris.New()
  app.Adapt(iris.DevLogger())
  app.Adapt(gorillamux.New()) // <---- NEW


  app.OnError(iris.StatusNotFound, func(ctx *iris.Context){
    ctx.HTML(iris.StatusNotFound, "<h1> custom http error page </h1>")
  })


  app.Get("/healthcheck", h)

  gamesMiddleware := func(ctx *iris.Context) {
    println(ctx.Method() + ": " + ctx.Path())
    ctx.Next()
  }

  games:= app.Party("/games", gamesMiddleware)
  { // braces are optional of course, it's just a style of code
	 games.Get("/{gameID:[0-9]+}/clans", h)
	 games.Get("/{gameID:[0-9]+}/clans/clan/{publicID:[0-9]+}", h)
	 games.Get("/{gameID:[0-9]+}/clans/search", h)

	 games.Put("/{gameID:[0-9]+}/players/{publicID:[0-9]+}", h)
	 games.Put("/{gameID:[0-9]+}/clans/clan/{publicID:[0-9]+}", h)

	 games.Post("/{gameID:[0-9]+}/clans", h)
	 games.Post("/{gameID:[0-9]+}/players", h)
	 games.Post("/{gameID:[0-9]+}/clans/{publicID:[0-9]+}/leave", h)
	 games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/application", h)
	 games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/application/:action", h)
	 games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/invitation", h)
	 games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/invitation/:action", h)
	 games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/delete", h)
	 games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/promote", h)
	 games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/demote", h)
  }

  app.Get("/anything/{anythingparameter:.*}", func(ctx *iris.Context){
    s := ctx.Param("anythingparameter")
    ctx.Writef("The path after /anything is: %s",s)
  })

  app.Listen(":80")

	/*
		gameID  = 1
		publicID = 2
		clanPublicID = 22
		action = 3

		GET
		http://localhost/healthcheck
		http://localhost/games/1/clans
		http://localhost/games/1/clans/clan/2
		http://localhost/games/1/clans/search

		PUT
		http://localhost/games/1/players/2
		http://localhost/games/1/clans/clan/2

		POST
		http://localhost/games/1/clans
		http://localhost/games/1/players
		http://localhost/games/1/clans/2/leave
		http://localhost/games/1/clans/22/memberships/application -> 494
		http://localhost/games/1/clans/22/memberships/application/3- > 404
		http://localhost/games/1/clans/22/memberships/invitation
		http://localhost/games/1/clans/22/memberships/invitation/3
		http://localhost/games/1/clans/2/memberships/delete
		http://localhost/games/1/clans/22/memberships/promote
		http://localhost/games/1/clans/22/memberships/demote

	*/
}

func h(ctx *iris.Context) {
	ctx.HTML(iris.StatusOK, "<h1>Path<h1/>"+ctx.Path())
}

```

**No changes whatever router you use**, only the `path` is changed(otherwise it doesn't make sense to support more than one router).
At the `gorillamux`'s path example we get pattern matching using regexp, at the other hand `httprouter` doesn't provides path validations
but it provides parameter and wildcard parameters too, it's also a lot faster than gorillamux.

Original Gorilla Mux made my life easier when I had to adapt the reverse routing and subdomains features, it has got these features by its own too, so it was easy.

Original Httprouter doesn't supports subdomains, multiple paths on different methods, reverse routing, custom http errors, I had to implement
all of them by myself and after adapt them using the policies, it was a bit painful but this is my job. Result: It runs blazy-fast!


As we said, all iris' features works as before even if you are able to adapt any custom router. Template funcs that were relative-closed to reverse router, like `{{ url }} and {{ urlpath }}`, works as before too, no change for your app's side need.


> I would love to see more routers (as more as they can provide different `path declaration` features) from the community, create an adaptor for an iris' router and I will share your repository to the rest of the users!


Adaptors are located [there](https://github.com/kataras/iris/tree/master/adaptors).

### View engine (5 template engine adaptors)

At the past, If no template engine was used then iris selected the [html standard](https://github.com/kataras/go-template/tree/master/html).

**Now, iris doesn't defaults any template engine** (also the `.Config.DisableTemplateEngines` has been removed, it has no use anymore).

So, again you have to do two changes, the `import path` and the `.Adapt`.

**Template files are no need to change, the template engines does the same exactly things as before**


All of these **five template engines** have common features with common API, like Layout, Template Funcs, Party-specific layout, partial rendering and more.
   - **the standard html**, based on [go-template/html](https://github.com/kataras/go-template/tree/master/html), its template parser is the [html/template](https://golang.org/pkg/html/template/).

   - **django**, based on [go-template/django](https://github.com/kataras/go-template/tree/master/django), its template parser is the [pongo2](https://github.com/flosch/pongo2)

   - **pug**, based on [go-template/pug](https://github.com/kataras/go-template/tree/master/pug), its template parser is the [jade](https://github.com/Joker/jade)

   - **handlebars**, based on [go-template/handlebars](https://github.com/kataras/go-template/tree/master/handlebars), its template parser is the [raymond](https://github.com/aymerick/raymond)

   - **amber**, based on [go-template/amber](https://github.com/kataras/go-template/tree/master/amber), its template parser is the [amber](https://github.com/eknkc/amber).

Each of the template engines has different options, view adaptors are located [here](https://github.com/kataras/iris/tree/master/adaptors/view).


Example:


```go
package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/adaptors/gorillamux" // <--- NEW (previous section)
	"github.com/kataras/iris/adaptors/view" // <--- NEW it contains all the template engines
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())
	app.Adapt(gorillamux.New()) // <--- NEW (previous section)

  // - standard html  | view.HTML(...)
  // - django         | view.Django(...)
  // - pug(jade)      | view.Pug(...)
  // - handlebars     | view.Handlebars(...)
  // - amber          | view.Amber(...)
	app.Adapt(view.HTML("./templates", ".html").Reload(true)) // <---- NEW (set .Reload to true when you're in dev mode.)

  // default template funcs:
  //
  // - {{ url "mynamedroute" "pathParameter_ifneeded"} }
  // - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
  // - {{ render "header.html" }}
  // - {{ render_r "header.html" }} // partial relative path to current page
  // - {{ yield }}
  // - {{ current }}
  //
  // to adapt custom funcs, use:
  app.Adapt(iris.TemplateFuncsPolicy{"myfunc": func(s string) string {
        return "hi "+s
  }}) // usage inside template: {{ hi "kataras"}}

	app.Get("/hi", func(ctx *iris.Context) {
		ctx.MustRender(
			"hi.html",                // the file name of the template relative to the './templates'
			iris.Map{"Name": "Iris"}, // the .Name inside the ./templates/hi.html
			iris.Map{"gzip": false},  // enable gzip for big files
		)

	})

  // http://127.0.0.1:8080/hi
	app.Listen(":8080")
}


```

`.UseTemplate` have been removed and replaced with the `.Adapt` which is using `iris.RenderPolicy` and `iris.TemplateFuncsPolicy`
to adapt the behavior of the custom template engines.

**BEFORE**
```go
import "github.com/kataras/go-template/django"
// ...
app := iris.New()
app.UseTemplate(django.New()).Directory("./templates", ".html")/*.Binary(...)*/)
```

**AFTER**
```go
import "github.com/kataras/iris/adaptors/view"
// ...
app := iris.New()
app.Adapt(view.Django("./templates",".htmll")/*.Binary(...)*/)
```

The rest remains the same. Don't forget the real changes were `only import path and .Adapt(imported)`, at general when you see an 'adaptor' these two declarations should happen to your code.



### Package-level functions and variables for `iris.Default` have been removed.

The form of variable use for an Iris *Framework remains as it was:
```go
app := iris.New()
app.$FUNCTION/$VARIABLE
```


> When I refer to `iris.$FUNCTION/$VARIABLE` it means `iris.Handle/.HandleFunc/.Get/.Post/.Put/.Delete/.Trace/.Options/.Use/.UseFunc/.UseGlobal/.Party/.Set/.Config`
and the rest of the package-level functions referred to the `iris.Default` variable.

**BEFORE**

```go
iris.Config.FireMethodNotAllowed = true
iris.Set(OptionDisableBanner(true))
```

```go
iris.Get("/", func(ctx *iris.Context){

})

iris.ListenLETSENCRYPT(":8080")
```

**AFTER**


```go
app := iris.New()
app.Config.FireMethodNotAllowed = true
// or iris.Default.Config.FireMethodNotAllowed = true and so on
app.Set(OptionDisableBanner(true))
// same as
// app := iris.New(iris.Configuration{FireMethodNotAllowed:true, DisableBanner:true})
```

```go
app := iris.New()
app.Get("/", func(ctx *iris.Context){

})

app.ListenLETSENCRYPT(":8080")
```

For those who had splitted the application in different packages they could do just that `iris.$FUNCTION/$VARIABLE` without the need
of import a singleton package which would initialize a new `App := iris.New()`.

`Iris.Default` remains, so you can refer to that if you don't want to initialize a new `App := iris.New()` by your own.

**BEFORE**

```go
package controllers
import "github.com/kataras/iris"
func init(){
   iris.Get("/", func(ctx *iris.Context){

   })
}
```

```go
package main

import (
  "github.com/kataras/iris"
   _ "github.com/mypackage/controllers"
)

func main(){
	iris.Listen(":8080")
}
```


**AFTER**

```go
package controllers

import (
  "github.com/kataras/iris"
  "github.com/kataras/iris/adaptors/httprouter"
)

func init(){
   iris.Default.Adapt(httprouter.New())
   iris.Default.Get("/", func(ctx *iris.Context){

   })
}
```

```go
package main

import (
  "github.com/kataras/iris"
   _ "github.com/mypackage/controllers"
)

func main(){
 iris.Default.Listen(":8080")
}
```

You got the point, let's continue to the next conversion.

### Remove the slow .API | iris.API(...) / app := iris.New(); app.API(...)


The deprecated `.API` has been removed entirely, it should be removed after v5(look on the v5 history tag).

At first I created that func in order to give newcovers a chance to be able to quick start a new `controller-like`
with one function, but that function was using generics at runtime and it was very slow compared to the
`iris.Handle/.HandleFunc/.Get/.Post/.Put/.Delete/.Trace/.Options/.Use/.UseFunc/.UseGlobal/.Party`.

Also some users they used only `.API`, they didn't bother to 'learn' about the standard rest api functions
and their power(including per-route middleware, cors, recover and so on). So we had many unrelational questions about the `.API` func.


**BEFORE**

```go
package main

import (
	"github.com/kataras/iris"
)

type UserAPI struct {
	*iris.Context
}

// GET /users
func (u UserAPI) Get() {
	u.Writef("Get from /users")
	// u.JSON(iris.StatusOK,myDb.AllUsers())
}

// GET /users/:param1 which its value passed to the id argument
func (u UserAPI) GetBy(id string) { // id equals to u.Param("param1")
	u.Writef("Get from /users/%s", id)
	// u.JSON(iris.StatusOK, myDb.GetUserById(id))

}

// POST /users
func (u UserAPI) Post() {
	name := u.FormValue("name")
	// myDb.InsertUser(...)
	println(string(name))
	println("Post from /users")
}

// PUT /users/:param1
func (u UserAPI) PutBy(id string) {
	name := u.FormValue("name") // you can still use the whole Context's features!
	// myDb.UpdateUser(...)
	println(string(name))
	println("Put from /users/" + id)
}

// DELETE /users/:param1
func (u UserAPI) DeleteBy(id string) {
	// myDb.DeleteUser(id)
	println("Delete from /" + id)
}

func main() {

	iris.API("/users", UserAPI{})
	iris.Listen(":8080")
}

```


**AFTER**

```go
package main

import (
	"github.com/kataras/iris"
  "github.com/kataras/iris/adaptors/gorillamux"
)

func  GetAllUsersHandler(ctx *iris.Context) {
	ctx.Writef("Get from /users")
	// ctx.JSON(iris.StatusOK,myDb.AllUsers())
}

func GetUserByIdHandler(ctx *iris.Context) {
	ctx.Writef("Get from /users/%s",
	  ctx.Param("id")) 	// or id, err := ctx.ParamInt("id")
	// ctx.JSON(iris.StatusOK, myDb.GetUserById(id))
}

func InsertUserHandler(ctx *iris.Context){
	name := ctx.FormValue("name")
	// myDb.InsertUser(...)
	println(string(name))
	println("Post from /users")
}

func UpdateUserHandler(ctx *iris.Context) {
	name := ctx.FormValue("name")
	// myDb.UpdateUser(...)
	println(string(name))
	println("Put from /users/" + ctx.Param("id"))
}

func  DeleteUserById(id string) {
	// myDb.DeleteUser(id)
	println("Delete from /" + ctx.param("id"))
}

func main() {
	app := iris.New()
  app.Adapt(gorillamux.New())

	// create a new router targeted for "/users" path prefix
	// you can learn more about Parties on the examples and book too
	// they can share middleware, template layout and more.
	userRoutes := app.Party("users")

	// GET  http://localhost:8080/users/ and /users
	userRoutes.Get("/", GetAllUsersHandler)

	// GET  http://localhost:8080/users/:id
	userRoutes.Get("/:id", GetUserByIdHandler)
	// POST  http://localhost:8080/users
	userRoutes.Post("/", InsertUserHandler)

	// PUT  http://localhost:8080/users/:id
	userRoutes.Put("/:id", UpdateUserHandler)

	// DELETE http://localhost:8080/users/:id
	userRoutes.Delete("/:id", DeleteUserById)

  app.Listen(":8080")
}

```

### Old Plugins and the new `.Adapt` Policies

A lot of changes to old -so-called Plugins and many features have been adopted to this new ecosystem.


First of all plugins renamed to `policies with adaptors which, adaptors, adapts the policies to the framework`
(it is not just a simple rename of the word, it's a new concept).


Policies are declared inside Framework, they are implemented outside of the Framework and they are adapted to Framework by a user call.

Policy adaptors are just like a plugins but they have to implement a specific action/behavior to a specific policy type(or more than one at the time).

The old plugins are fired 'when something happens do that' (ex: PreBuild,PostBuild,PreListen and so on) this behavior is the new `EventPolicy`
which has **4 main flow events** with their callbacks been wrapped, so you can use more than EventPolicy (most of the policies works this way).

```go
type (
	// EventListener is the signature for type of func(*Framework),
	// which is used to register events inside an EventPolicy.
	//
	// Keep note that, inside the policy this is a wrapper
	// in order to register more than one listener without the need of slice.
	EventListener func(*Framework)

	// EventPolicy contains the available Framework's flow event callbacks.
	// Available events:
	// - Boot
	// - Build
	// - Interrupted
	// - Recovery
	EventPolicy struct {
		// Boot with a listener type of EventListener.
		//   Fires when '.Boot' is called (by .Serve functions or manually),
		//   before the Build of the components and the Listen,
		//   after VHost and VSCheme configuration has been setted.
		Boot EventListener
		// Before Listen, after Boot
		Build EventListener
		// Interrupted with a listener type of EventListener.
		//   Fires after the terminal is interrupted manually by Ctrl/Cmd + C
		//   which should be used to release external resources.
		// Iris will close and os.Exit at the end of custom interrupted events.
		// If you want to prevent the default behavior just block on the custom Interrupted event.
		Interrupted EventListener
		// Recovery with a listener type of func(*Framework,error).
		//   Fires when an unexpected error(panic) is happening at runtime,
		//   while the server's net.Listener accepting requests
		//   or when a '.Must' call contains a filled error.
		//   Used to release external resources and '.Close' the server.
		//   Only one type of this callback is allowed.
		//
		//   If not empty then the Framework will skip its internal
		//   server's '.Close' and panic to its '.Logger' and execute that callback instaed.
		//   Differences from Interrupted:
		//    1. Fires on unexpected errors
		//    2. Only one listener is allowed.
		Recovery func(*Framework, error)
	}
)
```

**A quick overview on how they can be adapted** to an iris *Framework (iris.New()'s result).
Let's adapt `EventPolicy`:

```go
app := iris.New()

evts := iris.EventPolicy{
  // we ommit the *Framework's variable name because we have already the 'app'
  // if we were on different file with no access to the 'app' then the varialbe name will be useful.
  Boot: func(*Framework){
      app.Log("Here you can change any field and configuration for iris before being used
        also you can adapt more policies that should be used to the next step which is the Build and Listen,
        only the app.Config.VHost and  app.Config.VScheme have been setted here, but you can change them too\n")
  },
  Build: func(*Framework){
    app.Log("Here all configuration and all app' fields and features  have been builded, here you are ready to call
      anything (you shouldn't change fields and configuration here)\n")
  },
}
// Adapt the EventPolicy 'evts' to the Framework
app.Adapt(evts)

// let's register one more
app.Adapt(iris.EventPolicy{
  Boot: func(*Framework){
      app.Log("the second log message from .Boot!\n")
}})

// you can also adapt multiple and different(or same) types of policies in the same call
// using: app.Adapt(iris.EventPolicy{...}, iris.LoggerPolicy(...), iris.RouterWrapperPolicy(...))

// starts the server, executes the Boot -> Build...
app.Adapt(httprouter.New()) // read below for this line
app.Listen(":8080")
```



This pattern allows us to be very pluggable and add features that the *Framework itself doesn't knows,
it knows only the main policies which implement but their features are our(as users) business.


We have 8 policies, so far, and some of them have 'subpolicies' (the RouterReversionPolicy for example).

- LoggerPolicy
- EventPolicy
     - Boot
     - Build
     - Interrupted
     - Recover
- RouterReversionPolicy
     - StaticPath
     - WildcardPath
     - URLPath
     - RouteContextLinker
- RouterBuilderPolicy
- RouterWrapperPolicy
- RenderPolicy
- TemplateFuncsPolicy
- SessionsPolicy


**Details** of these can be found at [policy.go](https://github.com/kataras/iris/blob/master/policy.go).

The **Community**'s adaptors are [here](https://github.com/iris-contrib/adaptors).

**Iris' Built'n Adaptors** for these policies can be found at [/adaptors folder](https://github.com/kataras/iris/tree/master/adaptors).

The folder contains:

- cors, a cors (router) wrapper based on `rs/cors`.
It's a `RouterWrapperPolicy`

- gorillamux, a router that can be adapted, it's the `gorilla/mux` which supports subdomains, custom http errors, reverse routing, pattern matching.
It's a compination of`EventPolicy`, `RouterReversionPolicy with StaticPath, WildcardPath, URLPath, RouteContextLinker` and the `RouterBuilderPolicy`.

- httprouter, a router that can be adapted, it's a custom version of `julienschmidt/httprouter` which is edited to support iris' subdomains, reverse routing, custom http errors and a lot features, it should be a bit faster than the original too.
It's a compination of`EventPolicy`, `RouterReversionPolicy with StaticPath, WildcardPath, URLPath, RouteContextLinker` and the `RouterBuilderPolicy`.


- typescript and cloud editor, contains the typescript compiler with hot reload feature and a typescript cloud editor ([alm-tools](https://github.com/alm-tools/alm)), it's an `EventPolicy`

- view, contains 5 template engines based on the `kataras/go-template`.
All of these have common features with common API, like Layout, Template Funcs, Party-specific layout, partial rendering and more.
It's a `RenderPolicy` with a compinaton of `EventPolicy` and use of `TemplateFuncsPolicy`.
   - the standard html
   - pug(jade)
   - django(pongo2)
   - handlebars
   - amber.





#### Note
Go v1.8 introduced a new plugin system with `.so` files, users should not be confused with old iris' plugins and new adaptors.
It is not ready for all operating systems(yet) when it will be ready, Iris will take leverage of this Golang's feature.


### http.Handler and third-party middleware

We were compatible before this version but if a third-party middleware had the form of:
`func(http.ResponseWriter, *http.Request, http.HandlerFunc)`you were responsible of make a wrapper
which would return an `iris.Handler/HandlerFunc`.

Now you're able to pass an `func(http.ResponseWriter, *http.Request, http.HandlerFunc)` third-party net/http middleware(Chain-of-responsibility pattern)  using the `iris.ToHandler` wrapper func without any other custom boilerplate.

Example:

```go
package main

import (
	"github.com/kataras/iris"
  "github.com/kataras/adaptors/gorillamux"
	"github.com/rs/cors"
)

// myCors returns a new cors middleware
// with the provided options.
myCors := func(opts cors.Options) iris.HandlerFunc {
  handlerWithNext := cors.New(opts).ServeHTTP
	return iris.ToHandler(handlerWithNext)
}

func main(){
   app := iris.New()
   app.Adapt(httprouter.New())

   app.Post("/user", myCors(cors.Options{}), func(ctx *iris.Context){
     // ....
   })

   app.Listen(":8080")
}

```

-  Irrelative info but this is the best place to put it: `iris/app.AcquireCtx/.ReleaseCtx` replaced to: `app.Context.Acquire/.Release/.Run`.



### iris cmd

- FIX: [iris run main.go](https://github.com/kataras/iris/tree/master/iris#run) not reloading when file changes maden by some of the IDEs,
because they do override the operating system's fs signals. The majority of
editors worked before but I couldn't let some developers without support.


### Sessions


Sessions manager is also an Adaptor now, `iris.SessionsPolicy`.
So far we used the `kataras/go-sessions`, you could always use other session manager ofcourse but you would lose the `context.Session()`
and its returning value, the `iris.Session` now.

`SessionsPolicy` gives the developers the opportunity to adapt any,
compatible with a particular simple interface(Start and Destroy methods), third-party sessions managers.

- The API for sessions inside context is the same, no  matter what session manager you wanna to adapt.
- The API for sessions inside context didn't changed, it's the same as you knew it.

- Iris, of course, has built'n `SessionsPolicy` adaptor(the kataras/go-sessions: edited to remove fasthttp dependencies).
    - Sessions manager works even faster now and a bug fixed for some browsers.

- Functions like, adding a database or store(i.e: `UseDatabase`) depends on the session manager of your choice,
Iris doesn't requires these things
to adapt a package as a session manager. So `iris.UseDatabase` has been removed and depends on the `mySessions.UseDatabase` you 'll see below.

- `iris.DestroySessionByID and iris.DestroyAllSessions` have been also removed, depends on the session manager of your choice, `mySessions.DestroyByID and mySessions.DestroyAll`  should do the job now.


> Don't worry about forgetting to adapt any feature that you use inside Iris, Iris will print you a how-to-fix message at iris.DevMode log level.

**[Example](https://github.com/kataras/iris/tree/6.2/adaptors/sessions/_example) code:**

```go
package main

import (
	"time"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/sessions"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger()) // enable all (error) logs
	app.Adapt(httprouter.New()) // select the httprouter as the servemux

	mySessions := sessions.New(sessions.Config{
		// Cookie string, the session's client cookie name, for example: "mysessionid"
		//
		// Defaults to "irissessionid"
		Cookie: "mysessionid",
		// base64 urlencoding,
		// if you have strange name cookie name enable this
		DecodeCookie: false,
		// it's time.Duration, from the time cookie is created, how long it can be alive?
		// 0 means no expire.
		// -1 means expire when browser closes
		// or set a value, like 2 hours:
		Expires: time.Hour * 2,
		// the length of the sessionid's cookie's value
		CookieLength: 32,
		// if you want to invalid cookies on different subdomains
		// of the same host, then enable it
		DisableSubdomainPersistence: false,
	})

	// OPTIONALLY:
	// import "gopkg.in/kataras/iris.v6/adaptors/sessions/sessiondb/redis"
	// or import "github.com/kataras/go-sessions/sessiondb/$any_available_community_database"
	// mySessions.UseDatabase(redis.New(...))

	app.Adapt(mySessions) // Adapt the session manager we just created.

	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})
	app.Get("/set", func(ctx *iris.Context) {

		//set session values
		ctx.Session().Set("name", "iris")

		//test if setted here
		ctx.Writef("All ok session setted to: %s", ctx.Session().GetString("name"))
	})

	app.Get("/get", func(ctx *iris.Context) {
		// get a specific key, as string, if no found returns just an empty string
		name := ctx.Session().GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
	})

	app.Get("/delete", func(ctx *iris.Context) {
		// delete a specific key
		ctx.Session().Delete("name")
	})

	app.Get("/clear", func(ctx *iris.Context) {
		// removes all entries
		ctx.Session().Clear()
	})

	app.Get("/destroy", func(ctx *iris.Context) {

		//destroy, removes the entire session and cookie
		ctx.SessionDestroy()
		msg := "You have to refresh the page to completely remove the session (browsers works this way, it's not iris-specific.)"

		ctx.Writef(msg)
		ctx.Log(iris.DevMode, msg)
	}) // Note about destroy:
	//
	// You can destroy a session outside of a handler too, using the:
	// mySessions.DestroyByID
	// mySessions.DestroyAll

	app.Listen(":8080")
}

```

### Websockets

There are many internal improvements to the websocket server, it
operates slighty faster to.


Websocket is an Adaptor too and you can edit more configuration fields than before.
No Write and Read timeout by default, you have to set the fields if you want to enable timeout.

Below you'll see the before and the after, keep note that the static and templates didn't changed, so I am not putting the whole
html and javascript sources here, you can run the full examples from [here](https://github.com/kataras/iris/tree/6.2/adaptors/websocket/_examples).

**BEFORE:***

```go

package main

import (
	"fmt" // optional

	"github.com/kataras/iris"
)

type clientPage struct {
	Title string
	Host  string
}

func main() {
	iris.StaticWeb("/js", "./static/js")

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Render("client.html", clientPage{"Client Page", ctx.Host()})
	})

	// the path which the websocket client should listen/registed to ->
	iris.Config.Websocket.Endpoint = "/my_endpoint"
	// by-default all origins are accepted, you can change this behavior by setting:
	// iris.Config.Websocket.CheckOrigin

	var myChatRoom = "room1"
	iris.Websocket.OnConnection(func(c iris.WebsocketConnection) {
		// Request returns the (upgraded) *http.Request of this connection
		// avoid using it, you normally don't need it,
		// websocket has everything you need to authenticate the user BUT if it's necessary
		// then  you use it to receive user information, for example: from headers.

		// httpRequest := c.Request()
		// fmt.Printf("Headers for the connection with ID: %s\n\n", c.ID())
		// for k, v := range httpRequest.Header {
		// fmt.Printf("%s = '%s'\n", k, strings.Join(v, ", "))
		// }

		// join to a room (optional)
		c.Join(myChatRoom)

		c.On("chat", func(message string) {
			if message == "leave" {
				c.Leave(myChatRoom)
				c.To(myChatRoom).Emit("chat", "Client with ID: "+c.ID()+" left from the room and cannot send or receive message to/from this room.")
				c.Emit("chat", "You have left from the room: "+myChatRoom+" you cannot send or receive any messages from others inside that room.")
				return
			}
			// to all except this connection ->
			// c.To(iris.Broadcast).Emit("chat", "Message from: "+c.ID()+"-> "+message)
			// to all connected clients: c.To(iris.All)

			// to the client itself ->
			//c.Emit("chat", "Message from myself: "+message)

			//send the message to the whole room,
			//all connections are inside this room will receive this message
			c.To(myChatRoom).Emit("chat", "From: "+c.ID()+": "+message)
		})

		// or create a new leave event
		// c.On("leave", func() {
		// 	c.Leave(myChatRoom)
		// })

		c.OnDisconnect(func() {
			fmt.Printf("Connection with ID: %s has been disconnected!\n", c.ID())

		})
	})

	iris.Listen(":8080")
}



```


**AFTER**
```go
package main

import (
	"fmt" // optional

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/view"
	"gopkg.in/kataras/iris.v6/adaptors/websocket"
)

type clientPage struct {
	Title string
	Host  string
}

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger())                  // enable all (error) logs
	app.Adapt(httprouter.New())                  // select the httprouter as the servemux
	app.Adapt(view.HTML("./templates", ".html")) // select the html engine to serve templates

	ws := websocket.New(websocket.Config{
		// the path which the websocket client should listen/registed to,
		Endpoint: "/my_endpoint",
		// the client-side javascript static file path
		// which will be served by Iris.
		// default is /iris-ws.js
		// if you change that you have to change the bottom of templates/client.html
		// script tag:
		ClientSourcePath: "/iris-ws.js",
		//
		// Set the timeouts, 0 means no timeout
		// websocket has more configuration, go to ../../config.go for more:
		// WriteTimeout: 0,
		// ReadTimeout:  0,
		// by-default all origins are accepted, you can change this behavior by setting:
		// CheckOrigin: (r *http.Request ) bool {},
		//
		//
		// IDGenerator used to create (and later on, set)
		// an ID for each incoming websocket connections (clients).
		// The request is an argument which you can use to generate the ID (from headers for example).
		// If empty then the ID is generated by DefaultIDGenerator: randomString(64):
		// IDGenerator func(ctx *iris.Context) string {},
	})

	app.Adapt(ws) // adapt the websocket server, you can adapt more than one with different Endpoint

	app.StaticWeb("/js", "./static/js") // serve our custom javascript code

	app.Get("/", func(ctx *iris.Context) {
		ctx.Render("client.html", clientPage{"Client Page", ctx.Host()})
	})

	var myChatRoom = "room1"

	ws.OnConnection(func(c websocket.Connection) {
		// Context returns the (upgraded) *iris.Context of this connection
		// avoid using it, you normally don't need it,
		// websocket has everything you need to authenticate the user BUT if it's necessary
		// then  you use it to receive user information, for example: from headers.

		// ctx := c.Context()

		// join to a room (optional)
		c.Join(myChatRoom)

		c.On("chat", func(message string) {
			if message == "leave" {
				c.Leave(myChatRoom)
				c.To(myChatRoom).Emit("chat", "Client with ID: "+c.ID()+" left from the room and cannot send or receive message to/from this room.")
				c.Emit("chat", "You have left from the room: "+myChatRoom+" you cannot send or receive any messages from others inside that room.")
				return
			}
			// to all except this connection ->
			// c.To(websocket.Broadcast).Emit("chat", "Message from: "+c.ID()+"-> "+message)
			// to all connected clients: c.To(websocket.All)

			// to the client itself ->
			//c.Emit("chat", "Message from myself: "+message)

			//send the message to the whole room,
			//all connections are inside this room will receive this message
			c.To(myChatRoom).Emit("chat", "From: "+c.ID()+": "+message)
		})

		// or create a new leave event
		// c.On("leave", func() {
		// 	c.Leave(myChatRoom)
		// })

		c.OnDisconnect(func() {
			fmt.Printf("Connection with ID: %s has been disconnected!\n", c.ID())
		})
	})

	app.Listen(":8080")
}

```




If the iris' websocket feature does not cover your app's needs, you can simply use any other
library for websockets that you used to use, like the Golang's compatible to `socket.io`, simple example:

```go
package main

import (
     "log"

    "github.com/kataras/iris"
    "github.com/kataras/iris/adaptors/httprouter"
    "github.com/googollee/go-socket.io"
)

func main() {
    app := iris.New()
    app.Adapt(httprouter.New())
    server, err := socketio.NewServer(nil)
    if err != nil {
        log.Fatal(err)
    }
    server.On("connection", func(so socketio.Socket) {
        log.Println("on connection")
        so.Join("chat")
        so.On("chat message", func(msg string) {
            log.Println("emit:", so.Emit("chat message", msg))
            so.BroadcastTo("chat", "chat message", msg)
        })
        so.On("disconnection", func() {
            log.Println("on disconnect")
        })
    })
    server.On("error", func(so socketio.Socket, err error) {
        log.Println("error:", err)
    })

    app.Any("/socket.io", iris.ToHandler(server))

    app.Listen(":5000")
}
```

### Typescript compiler and cloud-based editor

The Typescript compiler adaptor(old 'plugin') has been fixed (it had an issue on new typescript versions).
Example can be bound [here](https://github.com/kataras/iris/tree/master/adaptors/typescript/_example).

The Cloud-based editor adaptor(old 'plugin') also fixed and improved to show debug messages to your iris' LoggerPolicy.
Example can be bound [here](https://github.com/kataras/iris/tree/master/adaptors/typescript/editor/_example).

Their import paths also changed as the rest of the old plugins from: https://github.com/iris-contrib/plugin to https://github.com/kataras/adaptors.
I had them on iris-contrib because I thought that community would help but it didn't, no problem, they are at the same codebase now
which making things easier to debug for me.


### Oauth/OAuth2
Fix the oauth/oauth2 adaptor (old 'plugin') .
Example can be found [here](https://github.com/iris-contrib/adaptors/tree/master/oauth/_example).


### CORS Middleware and the new Wrapper

Lets speak about history of cors middleware, almost all the issues users reported to the iris-contrib/middleware repository
were relative to the CORS middleware, some users done it work some others don't... it was strange. Keep note that this was one of the two middleware that I didn't
wrote by myself, it was a PR by a member who wrote that middleware and after didn't answer on users' issues.

Forget about it I removed it entirely and replaced with the `rs/cors`: we now use the https://github.com/rs/cors in two forms:

First, you can use the original middlare that you can install by `go get -u github.com/rs/cors`
(You had already see its example on the net/http handlers and iris.ToHandler section)

Can be registered globally or per-route but the `MethodsAllowed option doesn't works`.

Example:

```go
package main

import (
	"github.com/kataras/iris"
  "github.com/kataras/adaptors/gorillamux"
	"github.com/rs/cors"
)

func main(){
   app := iris.New()
   app.Adapt(httprouter.New()) // see below for that
   corsMiddleware := iris.ToHandler(cors.Default().ServeHTTP)
   app.Post("/user", corsMiddleware, func(ctx *iris.Context){
     // ....
   })

   app.Listen(":8080")
}
```

Secondly, probably the one which you will choose to use, is the `cors` Router Wrapper Adaptor.
It's already installed when you install iris because it's located at `kataras/iris/adaptors/cors`.

This will wrap the entirely router so the whole of your app will be passing by the rules you setted up on its `cors.Options`.

Again, it's functionality comes from the well-tested `rs/cors`, all known Options are working as expected.

Example:

```go
package main

import (
	"github.com/kataras/iris"
  "github.com/kataras/adaptors/httprouter"
  "github.com/kataras/adaptors/cors"
)

func main(){
   app := iris.New()
   app.Adapt(httprouter.New()) // see below for that
   app.Adapt(cors.New(cors.Options{})) // or cors.Default()

   app.Post("/user", func(ctx *iris.Context){
     // ....
   })

   app.Listen(":8080")
}

```

### FAQ
You know that you can always share your opinion and ask anything iris-relative with the rest of us, [here](https://kataras.rocket.chat/channel/iris).


## 6.1.3 -> 6.1.4

- FIX: [iris run main.go](https://github.com/kataras/iris/tree/master/iris#run) not reloading when file changes maden by some of the IDEs,
because they do override the operating system's fs signals. The majority of
editors worked before but I couldn't let some developers without support.

- IMPROVEMENT: Now you're able to pass an `func(http.ResponseWriter, *http.Request, http.HandlerFunc)` third-party net/http middleware(Chain-of-responsibility pattern)  using the `iris.ToHandler` wrapper func without any other custom boilerplate.

- IMPROVEMENT: [Sessions manager](https://github.com/kataras/go-sessions) works even faster now.
     * Change: `context.Session().GetAll()` returns an empty map instead of nil when session has no values.


## 6.1.2 -> 6.1.3

- Added a configuration field `iris.Config.DisableBodyConsumptionOnUnmarshal`

```go
// DisableBodyConsumptionOnUnmarshal manages the reading behavior of the context's body readers/binders.
// If setted to true then it
// disables the body consumption by the `context.UnmarshalBody/ReadJSON/ReadXML`.
//
// By-default io.ReadAll` is used to read the body from the `context.Request.Body which is an `io.ReadCloser`,
// if this field setted to true then a new buffer will be created to read from and the request body.
// The body will not be changed and existing data before the context.UnmarshalBody/ReadJSON/ReadXML will be not consumed.
DisableBodyConsumptionOnUnmarshal bool
```

If that option is setted to true then you can read more than one times from the same `context.Request.Body`.
Defaults to false because the majority of developers expecting request body to be empty after unmarshal.


## 6.1.1 -> 6.1.2

Better internalization and localization support, with ability to change the cookie's key and context's keys.

- Real example: https://github.com/iris-contrib/examples/tree/master/middleware_internationalization_i18n

**read the comments:**
```go
package main

import (
	"github.com/iris-contrib/middleware/i18n"
	"github.com/kataras/iris"
)

func main() {

	iris.Use(i18n.New(i18n.Config{
		Default:      "en-US",
		URLParameter: "lang",
		Languages: map[string]string{
			"en-US": "./locales/locale_en-US.ini",
			"el-GR": "./locales/locale_el-GR.ini",
			"zh-CN": "./locales/locale_zh-CN.ini"}}))

	iris.Get("/", func(ctx *iris.Context) {

		// it tries to find the language by:
		// ctx.Get("language") , that should be setted on other middleware before the i18n middleware*
		// if that was empty then
		// it tries to find from the URLParameter setted on the configuration
		// if not found then
		// it tries to find the language by the "lang" cookie
		// if didn't found then it it set to the Default setted on the configuration

		// hi is the key, 'kataras' is the %s on the .ini file
		// the second parameter is optional

		// hi := ctx.Translate("hi", "kataras")
		// or:
		hi := i18n.Translate(ctx, "hi", "kataras")

		language := ctx.Get(iris.TranslateLanguageContextKey) // language is the language key, example 'en-US'

		// The first succeed language found saved at the cookie with name ("language"),
		//  you can change that by changing the value of the:  iris.TranslateLanguageContextKey
		ctx.Writef("From the language %s translated output: %s", language, hi)
	})

	// go to http://localhost:8080/?lang=el-GR
	// or http://localhost:8080
	// or http://localhost:8080/?lang=zh-CN
	iris.Listen(":8080")

}

```

## 6.1.0 -> 6.1.1

- **NEW FEATURE**: `Offline routes`.

- Discussion: https://github.com/kataras/iris/issues/585
- Test: https://github.com/kataras/iris/blob/master/http_test.go#L735
- Example 1: https://github.com/iris-contrib/examples/tree/master/route_state
- Example 2, SPA: https://github.com/iris-contrib/examples/tree/master/spa_2_using_offline_routing

**What?**


- Give priority to an API path inside a Static route

```go
package main

import (
	"github.com/kataras/iris"
)

func main() {

	usersAPI := iris.None("/api/users/:userid", func(ctx *iris.Context) {
		ctx.Writef("user with id: %s", ctx.Param("userid"))
	})("api.users.id")

	iris.StaticWeb("/", "./www", usersAPI)

	//
	// START THE SERVER
	//
	iris.Listen("localhost:8080")
}


```

- Play with(very advanced usage, used by big companies): enable(online) or disable(offline) routes at runtime with one line of code.

```go
package main

import (
	"github.com/kataras/iris"
)

func main() {

	// You can find the Route by iris.Lookup("theRouteName")
	// you can set a route name as: myRoute := iris.Get("/mypath", handler)("theRouteName")
	// that will set a name to the route and returns its iris.Route instance for further usage.
	api := iris.None("/api/users/:userid", func(ctx *iris.Context) {
		userid := ctx.Param("userid")
		ctx.Writef("user with id: %s", userid)
	})("users.api")

	// change the "users.api" state from offline to online and online to offline
	iris.Get("/change", func(ctx *iris.Context) {
		if api.IsOnline() {
			// set to offline
			iris.SetRouteOffline(api)
		} else {
			// set to online if it was not online(so it was offline)
			iris.SetRouteOnline(api, iris.MethodGet)
		}
	})

	iris.Get("/execute", func(ctx *iris.Context) {
		// change the path in order to be catcable from the ExecuteRoute
		// ctx.Request.URL.Path = "/api/users/42"
		// ctx.ExecRoute(iris.Route)
		// or:
		ctx.ExecRouteAgainst(api, "/api/users/42")
	})

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from index /")
	})

	//
	// START THE SERVER
	//
	// STEPS:
	// 1. navigate to http://localhost:8080/api/users/42
	// you should get 404 error
	// 2. now, navigate to http://localhost:8080/change
	// you should see a blank page
	// 3. now, navigate to http://localhost:8080/api/users/42
	// you should see the page working, NO 404 error
	// go back to the http://localhost:8080/change
	// you should get 404 error again
	// You just dynamically changed the state of a route with 3 lines of code!
	// you can do the same with group of routes and subdomains :)
	iris.Listen(":8080")
}

```

- New built'n Middleware:  `iris.Prioritize(route)` in order to give priority to a route inside other handler (used internally on StaticWeb's builder)

```go
usersAPI := iris.None("/api/users/:userid", func(ctx *iris.Context) {
	ctx.Writef("user with id: %s", ctx.Param("userid"))
})("api.users.id") // we need to call empty ("") in order to get its iris.Route instance
// or ("the name of the route")
// which later on can be found with iris.Lookup("the name of the route")

static := iris.StaticHandler("/", "./www", false, false)
// manually give a priority to the usersAPI, if not found then continue to the static handler
iris.Get("/*file", iris.Prioritize(usersAPI), static)

iris.Get("/*file", static)

iris.Listen(":8080")

```

## 6.0.9 -> 6.1.0

- Fix a not found error when serving static files through custom subdomain, this should work again: `iris.Party("mysubdomain.").StaticWeb("/", "./static")`

- Add SPA Example (separate REST API from the index page): https://github.com/iris-contrib/examples/tree/master/spa_1_using_subdomain

## 6.0.8 -> 6.0.9

- Add `PostInterrupt` plugin, useful for customization of the **os.Interrupt** singal, before that Iris closed the server automatically.

```go
iris.Plugins.PostInterrupt(func(s *Framework){
  // when os.Interrupt signal is fired the body of this function will be fired,
  // you're responsible for closing the server with s.Close()

  // if that event is not registered then the framework
  // will close the server for you.



  /* Do  any custom cleanup and finally call the s.Close()
     remember you have the iris.Plugins.PreClose(func(s *Framework)) event too
     so you can split your logic in two logically places.
  */

})

```

- Example: https://github.com/iris-contrib/examples/tree/master/os_interrupt

## 6.0.7 -> 6.0.8

- Add `iris.UseTemplateFunc(functionName string, function interface{})`. You could always set custom template funcs by using each of [template engine's](https://github.com/kataras/go-template) configuration but this function will help newcomers to start creating their custom template funcs.

Example:

-  https://github.com/iris-contrib/examples/tree/master/template_engines/template_funcmap

## 6.0.6 -> 6.0.7

- `iris.Config.DisablePathEscape` -> renamed to `iris.Config.EnablePathEscape`, which defaults to false. Path escape is turned off by-default now,
if you're waiting for unescaped path parameters, then just enable it by putting: `iris.Config.EnablePathEscape = true` anywhere in your code OR
use the `context.ParamDecoded` instead of the context.Param when you want to escape a single path parameter.

- Example for `iris.UsePreRender` https://github.com/iris-contrib/examples/tree/master/template_engines/template_prerender

## 6.0.5 -> 6.0.6

http.Request access from WebsocketConnection.

Example:

- https://github.com/iris-contrib/examples/blob/master/websocket/main.go#L34

Relative commits to kataras/go-websocket:
- https://github.com/kataras/go-websocket/commit/550fc8b32eb13b3b4a4bfeb227ef1a896c8f8698

- https://github.com/kataras/go-websocket/commit/62c2d989d8b5e9126cdbf451c0e41e2e2b0b31b8

## 6.0.4 -> 6.0.5

- Add `iris.DestroySessionByID(string)` and `iris.DestroyAllSessions()` functions as requested by a community member in the [chat](https://kataras.rocket.chat/channel/iris)

```go
// DestroySessionByID removes the session entry
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
//
// It's safe to use it even if you are not sure if a session with that id exists.
DestroySessionByID(string)
// DestroyAllSessions removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
DestroyAllSessions()
```

## 6.0.3 -> 6.0.4

- Add a simple `context.StreamWriter` to fill the v5's StreamWriter, it's a `io.Writer` instead of `bufio.Writer` and returns false when stop otherwise true. Take a look at the silly book examples [here](https://docs.iris-go.com/streaming).



## 6.0.2 -> 6.0.3

- Give the users an easy to way to set a limit to the body size comes from the client, globally or per-route (useful when you want to disable/enable limit on certain clients).

```go
// ...
const maxBodySize =  1 << 20
// ...

api := iris.New()

api.Use(iris.LimitRequestBodySize(maxBodySize))
// or do it manually under certain situations,
// inside the route's handler:
// ctx.SetMaxRequestBodySize(maxBodySize)

// routes after
```

## 6.0.1 -> 6.0.2

- Fix subdomains (silly fix by checking the Request.Host vs Request.URL.Host) and add a more realistic test, as reported [here](https://github.com/kataras/iris/issues/574).


## 6.0.0 -> 6.0.1

We had(for 2 days) one ResponseWriter which has special and unique features, but it slowed the execution a little bit, so I had to think more about it, I want to keep iris as the fastest http/2 web framework, well-designed and also to be usable and very easy for new programmers, performance vs design is tough decision. I choose performance most of the times but golang gives us the way to have a good design with that too.

I had to convert that ResponseWriter to a 'big but simple golang' interface and split the behavior into two parts, one will be the default and fast response writer, and the other will be the most useful response writer + transactions = iris.ResponseRecorder (no other framework or library have these features as far as I know). At the same time I had to provide an easy one-call way to wrap the basic response writer to a response recorder and set it to the context or to the whole app.

> Of course I give the green light to other authors to copy these response writers as I already did with the whole source code and I'm happy to see my code exists into other famous web frameworks even when they don't notice my name anywhere :)

-  **response_writer.go**: is the response writer as you knew it with iris' bonus like the `StatusCode() int` which returns the http status code (useful for middleware which needs to know the previous status code), `WriteHeader` which doesn't let you write the status code more than once and so on.

- **response_recorder.go**: is the response writer used by `Transactions` but you can use it by calling the `context.Record/Redorder/IsRecording()`. It lets you `ResetBody` , `ResetHeaders and cookies`, set the `status code` at any time (before or after its Write method) and more.


Transform the responseWriter to a ResponseRecorder is ridiculous easily, depending on yours preferences select one of these methods:

- context call (lifetime only inside route's handlers/middleware): `context.Record();` which will convert the context.ResponseWriter to a ResponseRecorder. All previous methods works as before but if you want to `ResetBody/Reset/ResetHeaders/SetBody/SetBodyString` you will have to use the `w := context.Recorder()` or just cast the context.ResponseWriter to a pointer of iris.ResponseRecorder.

- middleware (global, per party, per route...): `iris.UseGlobal(iris.Recorder)`/`app := iris.New(); app.UseGlobal(iris.Recorder)` or `iris.Get("/mypath", iris.Recorder, myPathHandler)`



## v5/fasthttp -> 6.0.0

As I [promised to the community](https://github.com/kataras/iris/issues/565) and a lot others, HTTP/2 support on Iris is happening!

I tried to minimize the side affects.

If you don't find something you used to use come here and check that conversional list:

- `iris.ToHandlerFunc` -> `iris.ToHandler`.

- `context.Response.BodyWriter() io.Writer` -> `context.ResponseWriter` is a http.ResponseWriter(and io.Writer) now.

- `context.Request/Response.Body()` -> `body,err := io.ReadAll(context.Request.Body)` or for response's previous body(useful on middleware) `ctx.Record() /* middleware here */ body:= ctx.Recorder().Body()`.

- `context.RequestCtx` removed and replaced by `context.ResponseWriter (*iris.ResponseWriter -> http.ResponseWriter)` and `context.Request (*http.Request)`.

- `context.Write(string, ...string)` -> `context.Writef(string, ...string)` | Write now has this form: Write([]byte) (int,error). All other write methods didn't changed.

- `context.GetFlash/SetFlash` -> `context.Session().GetFlash/GetFlashString/SetFlash/DeleteFlash/ClearFlashes/Flashes/HasFlash`.

- `context.FormValueString(string)` -> `context.FormValue(string)`.
- `context.PathString()` -> `context.Path()`.
- `context.HostString()` -> `context.Host()`.

- `iris.Config.DisablePathEscape` -> `iris.Config.EnablePathEscape`, defaults to false. Now we have two methods to get a decoded parameter.

- `context.Param/ParamDecoded` without need of this to be true, if it's true then the path parameters are query-decoded and .ParamDecoded returns the uri-decoded result.

- `context.RequestIP` -> `context.Request.RemoteAddr` but I recommend use the previous context's function: `context.RemoteAddr()` which will search for the client's IP in detail.

- All net/http middleware/handlers are **COMPATIBLE WITH IRIS NOW**, read more there](https://github.com/iris-contrib/middleware/blob/master/README.md#can-i-use-standard-nethttp-handler-with-iris).


**Static methods changes**

- `iris.StaticServe/StaticContent/StaticEmbedded/Favicon stay as they were before this version.`.

-	`iris.StaticHandler(string, int, bool, bool, []string) HandlerFunc` -> `iris.StaticHandler(reqPath string, systemPath string, showList bool, enableGzip bool) HandlerFunc`.


- `iris.StaticWeb(string, string, int) RouteNameFunc` -> `iris.StaticWeb(routePath string, systemPath string) RouteNameFunc`.
- `iris.Static` -> removed and joined to the new iris.StaticHandler
- `iris.StaticFS` -> removed and joined into the new `iris.StaticWeb`.



**More on Transictions vol 4**:

- Add support for custom `transactions scopes`, two scopes already implemented: `iris.TransientTransactionScope(default) and iris.RequestTransactionScope `

- `ctx.BeginTransaction(pipe func(*iris.TransactionScope))` -> `ctx.BeginTransaction(pipe func(*iris.Transaction))`

- [from](https://github.com/iris-contrib/examples/blob/5.0.0/transactions/main.go) -> [to](https://github.com/iris-contrib/examples/blob/master/transactions/main.go). Further research `context_test.go:TestTransactions` and https://www.codeproject.com/Articles/690136/All-About-TransactionScope (for .NET C#, I got the idea from there, it's a unique(golang web) feature so please read this before use transactions inside iris)


[Examples](https://github.com/iris-contrib/examples/tree/master), [middleware](https://github.com/iris-contrib/middleware/tree/master) & [plugins](https://github.com/iris-contrib/plugin) were been refactored for this new (net/http2 compatible) release.


## 5.1.1 -> 5.1.3
- **More on Transactions vol 3**: Recovery from any (unexpected error) panics inside `context.BeginTransaction` without loud, continue the execution as expected. Next version will have a little cleanup if I see that the transactions code is going very large or hard to understand the flow*

## 5.1.1 -> 5.1.2

- **More on Transactions vol 2**: Added **iris.UseTransaction** and **iris.DoneTransaction** to register transactions as you register middleware(handlers). new named type **iris.TransactionFunc**, shortcut of `func(scope *iris.TransactionScope)`, that gives you a function which you can convert a transaction to a normal handler/middleware using its `.ToMiddleware()`, for more see the `test code inside context_test.go:TestTransactionsMiddleware`.

## 5.1.0 -> 5.1.1
Two hours after the previous update,

- **More on Transactions**: By-default transaction's lifetime is 'per-call/transient' meaning that each transaction has its own scope on the context, rollbacks when `scope.Complete(notNilAndNotEmptyError)` and the rest of transactions in chain are executed as expected, from now and on you have the ability to `skip the rest of the next transactions on first failure` by simply call `scope.RequestScoped(true)`.

Note: `RequestTransactionScope` renamed to ,simply, `TransactionScope`.

## 5.0.4 -> 5.1.0

- **NEW (UNIQUE?) FEATURE**: Request-scoped transactions inside handler's context. Proof-of-concept example [here](https://github.com/iris-contrib/examples/tree/master/transactions).

## 5.0.3 -> 5.0.4


The use of `iris.BodyDecoder` as a custom decoder that you can implement to a type in order to be used as the decoder/binder for the request body and override the json.Unmarshal(`context.ReadJSON`) or xml.Unmarshal(`context.ReadXML`) was very useful and gave you some kind of **per-type-binder** extensibility.




**NEW** `context.UnmarshalBody`: **Per-service-binder**. Side by side with the `iris.BodyDecoder`. We now have a second way to pass a custom `Unmarshaler` to override the `json.Unmarshal` and `xml.Unmarshal`.

 If the object doesn't implements the `iris.BodyDecoder` but you still want to implement your own algorithm to parse []byte as an 'object' instead of the iris' defaults.

 ```go
 type Unmarshaler interface {
 	Unmarshal(data []byte, v interface{}) error
 }

 ```
`context.ReadJSON & context.ReadXML` have been also refactored to work with this interface and the new `context.DeodeBody` function, look:

```go
// ReadJSON reads JSON from request's body
// and binds it to a value of any json-valid type
func (ctx *Context) ReadJSON(jsonObject interface{}) error {
	return ctx.UnmarshalBody(jsonObject, UnmarshalerFunc(json.Unmarshal))
}

// ReadXML reads XML from request's body
// and binds it to a value of any xml-valid type
func (ctx *Context) ReadXML(xmlObject interface{}) error {
	return ctx.UnmarshalBody(xmlObject, UnmarshalerFunc(xml.Unmarshal))
}

```

Both  `encoding/json` and `encoding/xml` standard packages have valid `Unmarshal function` so they can be used as `iris.Unmarshaller` (with the help of `iris.UnmarshallerFunc` which just converts the signature to the `iris.Unmarshaller` interface). You only have to implement one function and it will work with any 'object' passed to the `UnmarshalBody` even if the object doesn't implements the `iris.BodyDecoder`.


## 5.0.2 -> 5.0.3

- Fix `https relative redirect paths`, a very old issue, which I just saw, peaceful, again :)

## 5.0.1 -> 5.0.2

- [geekypanda/httpcache](https://github.com/geekypanda/httpcache) has been re-written,
 by me, got rid of the mutex locks and use individual statcks instead,
 gain even more performance boost

- `InvalidateCache` has been removed,
	it wasn't working well for big apps, let cache work with
	its automation, is better.

- Add tests for the `iris.Cache`

## v3 -> [v4](https://github.com/kataras/iris/tree/4.0.0) (fasthttp-based) long term support

- **NEW FEATURE**: `CacheService` simple, cache service for your app's static body content(can work as external service if you are doing horizontal scaling, the `Cache` is just a `Handler` :) )

Cache any content, templates, static files, even the error handlers, anything.

> Bombardier: 5 million requests and 100k clients per second to this markdown  static content(look below) with cache(3 seconds) can be served up to ~x12 times faster. Imagine what happens with bigger content like full page and templates!


**OUTLINE**
```go

// Cache is just a wrapper for a route's handler which you want to enable body caching
// Usage: iris.Get("/", iris.Cache(func(ctx *iris.Context){
//    ctx.WriteString("Hello, world!") // or a template or anything else
// }, time.Duration(10*time.Second))) // duration of expiration
// if <=time.Second then it tries to find it though request header's "cache-control" maxage value
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.Cache instead of iris.Cache
Cache(bodyHandler HandlerFunc, expiration time.Duration) HandlerFunc

// InvalidateCache clears the cache body for a specific context's url path(cache unique key)
//
// Note that it depends on a station instance's cache service.
// Do not try to call it from default' station if you use the form of app := iris.New(),
// use the app.InvalidateCache instead of iris.InvalidateCache
InvalidateCache(ctx *Context)


```

**OVERVIEW**

```go
iris.Get("/hi", iris.Cache(func(c *iris.Context) {
	c.WriteString("Hi this is a big content, do not try cache on small content it will not make any significant difference!")
}, time.Duration(10)*time.Second))

```

[EXAMPLE](https://github.com/iris-contrib/examples/tree/master/cache_body):



```go
package main

import (
	"github.com/kataras/iris"
	"time"
)

var testMarkdownContents = `## Hello Markdown from Iris

This is an example of Markdown with Iris



Features
--------

All features of Sundown are supported, including:

*   **Compatibility**. The Markdown v1.0.3 test suite passes with
    the --tidy option.  Without --tidy, the differences are
    mostly in whitespace and entity escaping, where blackfriday is
    more consistent and cleaner.

*   **Common extensions**, including table support, fenced code
    blocks, autolinks, strikethroughs, non-strict emphasis, etc.

*   **Safety**. Blackfriday is paranoid when parsing, making it safe
    to feed untrusted user input without fear of bad things
    happening. The test suite stress tests this and there are no
    known inputs that make it crash.  If you find one, please let me
    know and send me the input that does it.

    NOTE: "safety" in this context means *runtime safety only*. In order to
    protect yourself against JavaScript injection in untrusted content, see
    [this example](https://github.com/russross/blackfriday#sanitize-untrusted-content).

*   **Fast processing**. It is fast enough to render on-demand in
    most web applications without having to cache the output.

*   **Thread safety**. You can run multiple parsers in different
    goroutines without ill effect. There is no dependence on global
    shared state.

*   **Minimal dependencies**. Blackfriday only depends on standard
    library packages in Go. The source code is pretty
    self-contained, so it is easy to add to any project, including
    Google App Engine projects.

*   **Standards compliant**. Output successfully validates using the
    W3C validation tool for HTML 4.01 and XHTML 1.0 Transitional.

	[this is a link](https://github.com/kataras/iris) `

func main() {
	// if this is not setted then iris set this duration to the lowest expiration entry from the cache + 5 seconds
	// recommentation is to left as it's or
	// iris.Config.CacheGCDuration = time.Duration(5) * time.Minute

	bodyHandler := func(ctx *iris.Context) {
		ctx.Markdown(iris.StatusOK, testMarkdownContents)
	}

	expiration := time.Duration(5 * time.Second)

	iris.Get("/", iris.Cache(bodyHandler, expiration))

	// if expiration is <=time.Second then the cache tries to set the expiration from the "cache-control" maxage header's value(in seconds)
	// // if this header doesn't founds then the default is 5 minutes
	iris.Get("/cache_control", iris.Cache(func(ctx *iris.Context) {
		ctx.HTML(iris.StatusOK, "<h1>Hello!</h1>")
	}, -1))

	iris.Listen(":8080")
}

```

- **IMPROVE**: [Iris command line tool](https://github.com/kataras/iris/tree/master/iris) introduces a **new** `get` command (replacement for the old `create`)


**The get command** downloads, installs and runs a project based on a `prototype`, such as `basic`, `static` and `mongo` .

> These projects are located [online](https://github.com/iris-contrib/examples/tree/master/AIO_examples)


```sh
iris get basic
```

Downloads the  [basic](https://github.com/iris-contrib/examples/tree/master/AIO_examples/basic) sample protoype project to the `$GOPATH/src/github.com/iris-contrib/examples` directory(the iris cmd will open this folder to you, automatically) builds, runs and watch for source code changes (hot-reload)

[![Iris get command preview](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iriscmd.gif)](https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/iriscmd.gif)


- **CHANGE**: The `Path parameters` are now **immutable**. Now you don't have to copy a `path parameter` before passing to another function which maybe modifies it, this has a side-affect of `context.GetString("key") = context.Param("key")`  so you have to be careful to not override a path parameter via other custom (per-context) user value.


- **NEW**: `iris.StaticEmbedded`/`app := iris.New(); app.StaticEmbedded` - Embed static assets into your executable with [go-bindata](https://github.com/jteeuwen/go-bindata) and serve them.

> Note: This was already buitl'n feature for templates using `iris.UseTemplate(html.New()).Directory("./templates",".html").Binary(Asset,AssetNames)`, after v4.6.1 you can do that for other static files too, with the `StaticEmbedded` function

**outline**
```go

// StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
// First parameter is the request path, the path which the files in the vdir(second parameter) will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets"
// Third parameter is the Asset function
// Forth parameter is the AssetNames function
//
// For more take a look at the
// example: https://github.com/iris-contrib/examples/tree/master/static_files_embedded
StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) RouteNameFunc

```

**example**

You can view and run it from [here](https://github.com/iris-contrib/examples/tree/master/static_files_embedded) *

```go
package main

// First of all, execute: $ go get https://github.com/jteeuwen/go-bindata
// Secondly, execute the command: cd $GOPATH/src/github.com/iris-contrib/examples/static_files_embedded && go-bindata ./assets/...

import (
	"github.com/kataras/iris"
)

func main() {

	// executing this go-bindata command creates a source file named 'bindata.go' which
	// gives you the Asset and AssetNames funcs which we will pass into .StaticAssets
	// for more viist: https://github.com/jteeuwen/go-bindata
	// Iris gives you a way to integrade these functions to your web app

	// For the reason that you may use go-bindata to embed more than your assets, you should pass the 'virtual directory path', for example here is the : "./assets"
	// and the request path, which these files will be served to, you can set as "/assets" or "/static" which resulting on http://localhost:8080/static/*anyfile.*extension
	iris.StaticEmbedded("/static", "./assets", Asset, AssetNames)


	// that's all
	// this will serve the ./assets (embedded) files to the /static request path for example the favicon.ico will be served as :
	// http://localhost:8080/static/favicon.ico
	// Methods: GET and HEAD



	iris.Get("/", func(ctx *iris.Context) {
		ctx.HTML(iris.StatusOK, "<b> Hi from index</b>")
	})

	iris.Listen(":8080")
}

// Navigate to:
// http://localhost:8080/static/favicon.ico
// http://localhost:8080/static/js/jquery-2.1.1.js
// http://localhost:8080/static/css/bootstrap.min.css

// Now, these files are stored inside into your executable program, no need to keep it in the same location with your assets folder.


```


- **FIX**: httptest flags caused by httpexpect which used to help you with tests inside **old** func `iris.Tester` as reported [here]( https://github.com/kataras/iris/issues/337#issuecomment-253429976)

- **NEW**: `iris.ResetDefault()` func which resets the default iris instance which is the station for the most part of the public/package API

- **NEW**: package `httptest` with configuration which can be passed per 'tester' instead of iris instance( this is very helpful for testers)

- **CHANGED**: All tests are now converted for 'white-box' testing, means that tests now have package named: `iris_test` instead of `iris` in the same main directory.

- **CHANGED**: `iris.Tester` moved to `httptest.New` which lives inside the new `/kataras/iris/httptest` package, so:


**old**
```go
import (
	"github.com/kataras/iris"
	"testing"
)

func MyTest(t *testing.T) {
	iris.Get("/mypath", func(ctx *iris.Context){
		ctx.Write("my body")
	})
	// with configs: iris.Config.Tester.ExplicitURL/Debug = true
	e:= iris.Tester(t)
	e.GET("/mypath").Expect().Status(iris.StatusOK).Body().Equal("my body")
}
```
**used that instead/new**
```go
import (
	"github.com/kataras/iris/httptest"
	"github.com/kataras/iris"
	"testing"
)

func MyTest(t *testing.T) {
	// make sure that you reset your default station if you don't use the form of app := iris.New()
	iris.ResetDefault()

	iris.Get("/mypath", func(ctx *iris.Context){
		ctx.Write("my body")
	})

	e:= httptest.New(iris.Default, t)
	// with configs: e:= httptest.New(iris.Default, t, httptest.ExplicitURL(true), httptest.Debug(true))
	e.GET("/mypath").Expect().Status(iris.StatusOK).Body().Equal("my body")
}
```


Finally, some plugins container's additions:

- **NEW**: `iris.Plugins.Len()` func which returns the length of the current activated plugins in the default station

- **NEW**: `iris.Plugins.Fired("event") int` func which returns how much times and from how many plugins a particular event type is fired, event types are: `"prelookup", "prebuild", "prelisten", "postlisten", "preclose", "predownload"`

- **NEW**: `iris.Plugins.PreLookupFired() bool` func which returns true if `PreLookup` fired at least one time

- **NEW**: `iris.Plugins.PreBuildFired() bool` func which returns true if `PreBuild` fired at least one time

- **NEW**: `iris.Plugins.PreListenFired() bool` func which returns true if `PreListen/PreListenParallel` fired at least one time

- **NEW**: `iris.Plugins.PostListenFired() bool` func which returns true if `PostListen` fired at least one time

- **NEW**: `iris.Plugins.PreCloseFired() bool` func which returns true if `PreClose` fired at least one time

- **NEW**: `iris.Plugins.PreDownloadFired() bool` func which returns true if `PreDownload` fired at least one time


- **Feature request**: I never though that it will be easier for users to catch 405 instead of simple 404, I though that will make your life harder, but it's requested by the Community [here](https://github.com/kataras/iris/issues/469), so I did my duty. Enable firing Status Method Not Allowed (405) with a simple configuration field: `iris.Config.FireMethodNotAllowed=true` or `iris.Set(iris.OptionFireMethodNotAllowed(true))` or `app := iris.New(iris.Configuration{FireMethodNotAllowed:true})`. A trivial, test example can be shown here:

```go
func TestMuxFireMethodNotAllowed(t *testing.T) {

	iris.Config.FireMethodNotAllowed = true // enable catching 405 errors

	h := func(ctx *iris.Context) {
		ctx.Write("%s", ctx.MethodString())
	}

	iris.OnError(iris.StatusMethodNotAllowed, func(ctx *iris.Context) {
		ctx.Write("Hello from my custom 405 page")
	})

	iris.Get("/mypath", h)
	iris.Put("/mypath", h)

	e := iris.Tester(t)

	e.GET("/mypath").Expect().Status(iris.StatusOK).Body().Equal("GET")
	e.PUT("/mypath").Expect().Status(iris.StatusOK).Body().Equal("PUT")
	// this should fail with 405 and catch by the custom http error

	e.POST("/mypath").Expect().Status(iris.StatusMethodNotAllowed).Body().Equal("Hello from my custom 405 page")
	iris.Close()
}
```


- **NEW**: `PreBuild` plugin type, raises before `.Build`. Used by third-party plugins to register any runtime routes or make any changes to the iris main configuration, example of this usage is the [OAuth/OAuth2 Plugin](https://github.com/iris-contrib/plugin/tree/master/oauth).

- **FIX**: The [OAuth example](https://github.com/iris-contrib/examples/tree/master/plugin_oauth_oauth2).


- **NEW**: Websocket configuration fields:
	- `Error func(ctx *Context, status int, reason string)`. Manually catch  any handshake errors. Default calls the `ctx.EmitError(status)` with a stored error message in the `WsError` key(`ctx.Set("WsError", reason)`), as before.
	- `CheckOrigin func(ctx *Context)`. Manually allow or dissalow client's websocket access, ex: via header **Origin**. Default allow all origins(CORS-like) as before.
	- `Headers bool`. Allow websocket handler to copy request's headers on the handshake. Default is true
	 With these in-mind the `WebsocketConfiguration` seems like this now :

```go
type WebsocketConfiguration struct {
	// WriteTimeout time allowed to write a message to the connection.
	// Default value is 15 * time.Second
	WriteTimeout time.Duration
	// PongTimeout allowed to read the next pong message from the connection
	// Default value is 60 * time.Second
	PongTimeout time.Duration
	// PingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
	// Default value is (PongTimeout * 9) / 10
	PingPeriod time.Duration
	// MaxMessageSize max message size allowed from connection
	// Default value is 1024
	MaxMessageSize int64
	// BinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
	// see https://github.com/kataras/iris/issues/387#issuecomment-243006022 for more
	// defaults to false
	BinaryMessages bool
	// Endpoint is the path which the websocket server will listen for clients/connections
	// Default value is empty string, if you don't set it the Websocket server is disabled.
	Endpoint string
	// ReadBufferSize is the buffer size for the underline reader
	ReadBufferSize int
	// WriteBufferSize is the buffer size for the underline writer
	WriteBufferSize int
	// Headers  if true then the client's headers are copy to the websocket connection
	//
	// Default is true
	Headers bool
	// Error specifies the function for generating HTTP error responses.
	//
	// The default behavior is to store the reason in the context (ctx.Set(reason)) and fire any custom error (ctx.EmitError(status))
	Error func(ctx *Context, status int, reason string)
	// CheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, the host in the Origin header must not be set or
	// must match the host of the request.
	//
	// The default behavior is to allow all origins
	// you can change this behavior by setting the iris.Config.Websocket.CheckOrigin = iris.WebsocketCheckSameOrigin
	CheckOrigin func(ctx *Context) bool
}

```

- **REMOVE**: `github.com/kataras/iris/context/context.go` , this is no needed anymore. Its only usage was inside `sessions` and `websockets`, a month ago I did improvements to the sessions as a standalone package, the IContext interface is not being used there. With the today's changes, the iris-contrib/websocket doesn't needs the IContext interface too, so the whole folder `./context` is useless and removed now. Users developers don't have any side-affects from this change.


[Examples](https://github.com/iris-contrib/examples), [Book](https://github.com/iris-contrib/gitbook) are up-to-date, just new configuration fields.



- **FIX**: Previous CORS fix wasn't enough and produces error before server's startup[*](https://github.com/kataras/iris/issues/461) if many paths were trying to auto-register an `.OPTIONS` route, now this is fixed in combined with some improvements on the [cors middleware](https://github.com/iris-contrib/middleware/tree/master/cors) too.


- **NEW**: `BodyDecoder` gives the ability to set a custom decoder **per passed object** when `context.ReadJSON` and `context.ReadXML`

```go
// BodyDecoder is an interface which any struct can implement in order to customize the decode action
// from ReadJSON and ReadXML
//
// Trivial example of this could be:
// type User struct { Username string }
//
// func (u *User) Decode(data []byte) error {
//	  return json.Unmarshal(data, u)
// }
//
// the 'context.ReadJSON/ReadXML(&User{})' will call the User's
// Decode option to decode the request body
//
// Note: This is totally optionally, the default decoders
// for ReadJSON is the encoding/json and for ReadXML is the encoding/xml
type BodyDecoder interface {
	Decode(data []byte) error
}

```

> for a usage example go to https://github.com/kataras/iris/blob/master/context_test.go#L262


- **small fix**: websocket server is nil when more than the default websocket station tries to be registered before `OnConnection` called[*](https://github.com/kataras/iris/issues/460)


- **FIX**: CORS not worked for all http methods
- **FIX**: Unexpected Party root's route slash  when `DisablePathCorrection` is false(default), as reported [here](https://github.com/kataras/iris/issues/453)
- **small fix**: DisablePathEscape not affects the uri string
- **small fix**: when Path Correction on POST redirect to the GET instead of POST


- **NEW**: Template PreRenders, as requested [here](https://github.com/kataras/iris/issues/412).

```go
// ...
iris.UsePreRender(func(ctx *iris.Context, filename string, binding interface{}, options ...map[string]interface{}) bool {
	// put the 'Error' binding here, for the shake of the test
	if b, isMap := binding.(map[string]interface{}); isMap {
		b["Error"] = "error!"
	}
	// true= continue to the next PreRender
  // false= do not continue to the next PreRender
  // * but the actual Render will be called at any case *
  return true
})

iris.Get("/", func(ctx *Context) {
  	ctx.Render("hi.html", map[string]interface{}{"Username": "anybody"})
    // hi.html: <h1>HI {{.Username}}. Error: {{.Error}}</h1>
})

// ...
```



**NOTE**: For normal users this update offers nothing, read that only if you run Iris behind a proxy or balancer like `nginx` or you need to serve using a custom `net.Listener`.

This update implements the [support of using native servers and net.Listener instead of Iris' defined](https://github.com/kataras/iris/issues/438).

### Breaking changes
- `iris.Config.Profile` field removed and the whole pprof transfered to the [iris-contrib/middleware/pprof](https://github.com/iris-contrib/middleware/tree/master/pprof).
- `iris.ListenTLSAuto` renamed to `iris.ListenLETSENCRYPT`
- `iris.Default.Handler` is `iris.Router` which is the Handler you can use to setup a custom router or bind Iris' router, after `iris.Build` call, to an external custom server.
- `iris.ServerConfiguration`, `iris.ListenVirtual`, `iris.AddServer`, `iris.Go` & `iris.TesterConfiguration.ListeningAddr` removed, read below the reason and their alternatives

### New features

- Boost Performance on server's startup
- **NEW**: `iris.Reserve()` re-starts the server if `iris.Close()` called previously.
- **NEW**: `iris.Config.VHost` and `iris.Config.VScheme` replaces the previous `ListenVirtual`, `iris.TesterConfiguration.ListeningAddr`, `iris.ServerConfiguration.VListeningAddr`, `iris.ServerConfiguration.VScheme`.
- **NEW**: `iris.Build` it's called automatically on Listen functions or Serve function. **CALL IT, MANUALLY, ONLY** WHEN YOU WANT TO BE ABLE TO GET THE IRIS ROUTER(`iris.Router`) AND PASS THAT, HANDLER, TO ANOTHER EXTERNAL FASTHTTP SERVER.
- **NEW**: `iris.Serve(net.Listener)`. Starts the server using a custom net.Listener, look below for example link
- **NEW**: now that iris supports custom net.Listener bind, I had to provide to you some net.Listeners too, such as `iris.TCP4`, `iris.UNIX`, `iris.TLS` , `iris.LETSENCRPYPT` & `iris.CERT` , all of these are optionals because you can just use the `iris.Listen`, `iris.ListenUNIX`, `iris.ListenTLS` & `iris.ListenLETSENCRYPT`, but if you want, for example, to pass your own `tls.Config` then you will have to create a custom net.Listener and pass that to the `iris.Serve(net.Listener)`.

With these in mind, developers are now able to fill their advanced needs without use the `iris.AddServer, ServerConfiguration and V fields`, so it's easier to:

- use any external (fasthttp compatible) server or router. Examples: [server](https://github.com/iris-contrib/tree/master/custom_fasthtthttp_server) and [router]((https://github.com/iris-contrib/tree/master/custom_fasthtthttp_router)
- bind any `net.Listener` which will be used to run the Iris' HTTP server, as requested [here](https://github.com/kataras/iris/issues/395). Example [here](https://github.com/iris-contrib/tree/master/custom_net_listener)
- setup virtual host and scheme, useful when you run Iris behind `nginx` (etc) and want template function `{{url }}` and subdomains to work as you expected. Usage:

```go
iris.Config.VHost = "mydomain.com"
iris.Config.VScheme = "https://"

iris.Listen(":8080")

// this will run on localhost:8080 but templates, subdomains and all that will act like https://mydomain.com,
// before this update you used the iris.AddServer and iris.Go and pass some strange fields into

```

Last, for testers:


Who used the `iris.ListenVirtual(...).Handler`:
If closed server, then `iris.Build()` and `iris.Router`, otherwise just `iris.Router`.


To test subdomains or a custom domain just set the `iris.Config.VHost` and `iris.Config.VScheme` fields, instead of the old `subdomain_test_handler := iris.AddServer(iris.ServerConfiguration{VListeningAddr:"...", Virtual: true, VScheme:false}).Handler`. Usage [here](https://github.com/kataras/blob/master/http_test.go).



**Finally**, I have to notify you that [examples](https://github.com/iris-contrib/examples), [plugins](https://github.com/iris-contrib/plugin), [middleware](https://github.com/iris-contrib/middleware) and [book](https://github.com/iris-contrib/gitbook) have been updated.


- Align with the latest version of [go-websocket](https://github.com/kataras/go-websocket), remove vendoring for compression on [go-fs](https://github.com/kataras/go-fs) which produced errors on sqllite and gorm(mysql and mongo worked fine before too) as reported [here](https://github.com/kataras/go-fs/issues/1).


- **External FIX**: [template syntax error causes a "template doesn't exist"](https://github.com/kataras/iris/issues/415)


- **ADDED**: You are now able to use a raw fasthttp handler as the router instead of the default Iris' one. Example [here](https://github.com/iris-contrib/examples/blob/master/custom_fasthttp_router/main.go). But remember that I'm always recommending to use the Iris' default which supports subdomains, group of routes(parties), auto path correction and many other built'n features. This exists for specific users who told me that they need a feature like that inside Iris, we have no performance cost at all so that's ok to exists.


- **CHANGE**: Updater (See 4.2.4 and 4.2.3) runs in its own goroutine now, unless the `iris.Config.CheckForUpdatesSync` is true.
- **ADDED**: To align with fasthttp server's configuration, iris has these new Server Configuration's fields, which allows you to set a type of rate limit:
```go
// Maximum number of concurrent client connections allowed per IP.
//
// By default unlimited number of concurrent connections
// may be established to the server from a single IP address.
MaxConnsPerIP int

// Maximum number of requests served per connection.
//
// The server closes connection after the last request.
// 'Connection: close' header is added to the last response.
//
// By default unlimited number of requests may be served per connection.
MaxRequestsPerConn int

// Usage: iris.ListenTo{iris.OptionServerListeningAddr(":8080"), iris.OptionServerMaxConnsPerIP(300)}
//    or: iris.ListenTo(iris.ServerConfiguration{ListeningAddr: ":8080", MaxConnsPerIP: 300, MaxRequestsPerConn:100})
// for an optional second server with a different port you can always use:
//        iris.AddServer(iris.ServerConfiguration{ListeningAddr: ":9090", MaxConnsPerIP: 300, MaxRequestsPerConn:100})
```

- **ADDED**: `iris.CheckForUpdates(force bool)` which can run the updater(look 4.2.4) at runtime too, updater is tested and worked at dev machine.


- **NEW Experimental feature**: Updater with a `CheckForUpdates` [configuration](https://github.com/kataras/iris/blob/master/configuration.go) field, as requested [here](https://github.com/kataras/iris/issues/401)
```go
// CheckForUpdates will try to search for newer version of Iris based on the https://github.com/kataras/iris/releases
// If a newer version found then the app will ask the he dev/user if want to update the 'x' version
// if 'y' is pressed then the updater will try to install the latest version
// the updater, will notify the dev/user that the update is finished and should restart the App manually.
// Notes:
// 1. Experimental feature
// 2. If setted to true, the app will have a little startup delay
// 3. If you as developer edited the $GOPATH/src/github/kataras or any other Iris' Go dependencies at the past
//    then the update process will fail.
//
// Usage: iris.Set(iris.OptionCheckForUpdates(true)) or
//        iris.Config.CheckForUpdates = true or
//        app := iris.New(iris.OptionCheckForUpdates(true))
// Default is false
CheckForUpdates bool
```

- [Add IsAjax() convenience method](https://github.com/kataras/iris/issues/423)


- Fix [sessiondb issue 416](https://github.com/kataras/iris/issues/416)


- **CHANGE**: No front-end changes if you used the default response engines before. Response Engines to Serializers, `iris.ResponseEngine` `serializer.Serializer`, comes from `kataras/go-serializer` which is installed automatically when you upgrade iris with `-u` flag.

    - the repo "github.com/iris-contrib/response" is a clone of "github.com/kataras/go-serializer", to keep compatibility state. examples and gitbook updated to work with the last.

    - `iris.UseResponse(iris.ResponseEngine, ...string)func (string)` was used to register custom response engines, now you use: `iris.UseSerializer(key string, s serializer.Serializer)`.

    - `iris.ResponseString` same defintion but differnet name now: `iris.SerializeToString`

[Serializer examples](https://github.com/iris-contrib/examples/tree/master/serialize_engines) and [Book section](https://kataras.gitbooks.io/iris/content/serialize-engines.html) updated.


- **ADDED**: `iris.TemplateSourceString(src string, binding interface{}) string` this will parse the src raw contents to the template engine and return the string result & `context.RenderTemplateSource(status int, src string, binding interface{}, options ...map[string]interface{}) error` this will parse the src raw contents to the template engine and render the result to the client, as requseted [here](https://github.com/kataras/iris/issues/409).


This version has 'breaking' changes if you were, directly, passing custom configuration to a custom iris instance before.
As the TODO2 I had to think and implement a way to make configuration even easier and more simple to use.

With last changes in place, Iris is using new, cross-framework, and more stable packages made by me(so don't worry things are working and will as you expect) to render [templates](https://github.com/kataras/go-template), manage [sessions](https://github.com/kataras/go-sesions) and [websockets](https://github.com/kataras/go-websocket). So the `/kataras/iris/config` is no longer need to be there, we don't have core packages inside iris which need these configuration to other package-folder than the main anymore(in order to avoid the import-cycle), new file `/kataras/iris/configuration.go` is created for the configuration, which lives inside the main package, means that now:

- **if you want to pass directly configuration to a new custom iris instance, you don't have to import the github.com/kataras/iris/config package**

Naming changes:

- `config.Iris` -> `iris.Configuration`, which is the parent/main configuration. Added: `TimeFormat` and `Other` (pass any dynamic custom, other options there)
- `config.Sessions` -> `iris.SessionsConfiguration`
- `config.Websocket` -> `iris.WebscoketConfiguration`
- `config.Server` -> `iris.ServerConfiguration`
- `config.Tester` -> `iris.TesterConfiguration`

All these changes wasn't made only to remove the `./config` folder but to make easier for you to pass the exact configuration field/option you need to edit at the top of the default configuration, without need to pass the whole Configuration object. **Attention**: old way, pass `iris.Configuration` directly, is still valid object to pass to the  `iris.New`, so don't be afraid for breaking change, the only thing you will need to edit is the names of the configuration you saw on the previous paragraph.

**Configuration Declaration**:

instead of old, but still valid to pass to the `iris.New`:
- `iris.New(iris.Configuration{Charset: "UTF-8", Sessions: iris.SessionsConfiguration{Cookie: "cookienameid"}})`
now you can just write this:
- `iris.New(iris.OptionCharset("UTF-8"), iris.OptionSessionsCookie("cookienameid"))`

`.New` **by configuration**
```go
import "github.com/kataras/iris"
//...
myConfig := iris.Configuration{Charset: "UTF-8", IsDevelopment:true, Sessions: iris.SessionsConfiguration{Cookie:"mycookie"}, Websocket: iris.WebsocketConfiguration{Endpoint: "/my_endpoint"}}
iris.New(myConfig)
```

`.New` **by options**

```go
import "github.com/kataras/iris"
//...
iris.New(iris.OptionCharset("UTF-8"), iris.OptionIsDevelopment(true), iris.OptionSessionsCookie("mycookie"), iris.OptionWebsocketEndpoint("/my_endpoint"))

// if you want to set configuration after the .New use the .Set:
iris.Set(iris.OptionDisableBanner(true))
```

**List** of all available options:
```go
// OptionDisablePathCorrection corrects and redirects the requested path to the registered path
// for example, if /home/ path is requested but no handler for this Route found,
// then the Router checks if /home handler exists, if yes,
// (permant)redirects the client to the correct path /home
//
// Default is false
OptionDisablePathCorrection(val bool)

// OptionDisablePathEscape when is false then its escapes the path, the named parameters (if any).
OptionDisablePathEscape(val bool)

// OptionDisableBanner outputs the iris banner at startup
//
// Default is false
OptionDisableBanner(val bool)

// OptionLoggerOut is the destination for output
//
// Default is os.Stdout
OptionLoggerOut(val io.Writer)

// OptionLoggerPreffix is the logger's prefix to write at beginning of each line
//
// Default is [IRIS]
OptionLoggerPreffix(val string)

// OptionProfilePath a the route path, set it to enable http pprof tool
// Default is empty, if you set it to a $path, these routes will handled:
OptionProfilePath(val string)

// OptionDisableTemplateEngines set to true to disable loading the default template engine (html/template) and disallow the use of iris.UseEngine
// Default is false
OptionDisableTemplateEngines(val bool)

// OptionIsDevelopment iris will act like a developer, for example
// If true then re-builds the templates on each request
// Default is false
OptionIsDevelopment(val bool)

// OptionTimeFormat time format for any kind of datetime parsing
OptionTimeFormat(val string)

// OptionCharset character encoding for various rendering
// used for templates and the rest of the responses
// Default is "UTF-8"
OptionCharset(val string)

// OptionGzip enables gzip compression on your Render actions, this includes any type of render, templates and pure/raw content
// If you don't want to enable it globally, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true})
// Default is false
OptionGzip(val bool)

// OptionOther are the custom, dynamic options, can be empty
// this fill used only by you to set any app's options you want
// for each of an Iris instance
OptionOther(val ...options.Options) //map[string]interface{}, options is github.com/kataras/go-options

// OptionSessionsCookie string, the session's client cookie name, for example: "qsessionid"
OptionSessionsCookie(val string)

// OptionSessionsDecodeCookie set it to true to decode the cookie key with base64 URLEncoding
// Defaults to false
OptionSessionsDecodeCookie(val bool)

// OptionSessionsExpires the duration of which the cookie must expires (created_time.Add(Expires)).
// If you want to delete the cookie when the browser closes, set it to -1 but in this case, the server side's session duration is up to GcDuration
//
// Default infinitive/unlimited life duration(0)
OptionSessionsExpires(val time.Duration)

// OptionSessionsCookieLength the length of the sessionid's cookie's value, let it to 0 if you don't want to change it
// Defaults to 32
OptionSessionsCookieLength(val int)

// OptionSessionsGcDuration every how much duration(GcDuration) the memory should be clear for unused cookies (GcDuration)
// for example: time.Duration(2)*time.Hour. it will check every 2 hours if cookie hasn't be used for 2 hours,
// deletes it from backend memory until the user comes back, then the session continue to work as it was
//
// Default 2 hours
OptionSessionsGcDuration(val time.Duration)

// OptionSessionsDisableSubdomainPersistence set it to true in order dissallow your q subdomains to have access to the session cookie
// defaults to false
OptionSessionsDisableSubdomainPersistence(val bool)

// OptionWebsocketWriteTimeout time allowed to write a message to the connection.
// Default value is 15 * time.Second
OptionWebsocketWriteTimeout(val time.Duration)

// OptionWebsocketPongTimeout allowed to read the next pong message from the connection
// Default value is 60 * time.Second
OptionWebsocketPongTimeout(val time.Duration)

// OptionWebsocketPingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
// Default value is (PongTimeout * 9) / 10
OptionWebsocketPingPeriod(val time.Duration)

// OptionWebsocketMaxMessageSize max message size allowed from connection
// Default value is 1024
OptionWebsocketMaxMessageSize(val int64)

// OptionWebsocketBinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
// see https://github.com/kataras/iris/issues/387#issuecomment-243006022 for more
// defaults to false
OptionWebsocketBinaryMessages(val bool)

// OptionWebsocketEndpoint is the path which the websocket server will listen for clients/connections
// Default value is empty string, if you don't set it the Websocket server is disabled.
OptionWebsocketEndpoint(val string)

// OptionWebsocketReadBufferSize is the buffer size for the underline reader
OptionWebsocketReadBufferSize(val int)

// OptionWebsocketWriteBufferSize is the buffer size for the underline writer
OptionWebsocketWriteBufferSize(val int)

// OptionTesterListeningAddr is the virtual server's listening addr (host)
// Default is "iris-go.com:1993"
OptionTesterListeningAddr(val string)

// OptionTesterExplicitURL If true then the url (should) be prepended manually, useful when want to test subdomains
// Default is false
OptionTesterExplicitURL(val bool)

// OptionTesterDebug if true then debug messages from the httpexpect will be shown when a test runs
// Default is false
OptionTesterDebug(val bool)


```

Now, some of you maybe use more than one server inside their iris instance/app, so you used the `iris.AddServer(config.Server{})`, which now becomes `iris.AddServer(iris.ServerConfiguration{})`, ServerConfiguration has also (optional) options to pass there and to `iris.ListenTo(OptionServerListeningAddr("mydomain.com"))`:


```go
// examples:
iris.AddServer(iris.OptionServerCertFile("file.cert"),iris.OptionServerKeyFile("file.key"))
iris.ListenTo(iris.OptionServerReadBufferSize(42000))

// or, old way but still valid:
iris.AddServer(iris.ServerConfiguration{ListeningAddr: "mydomain.com", CertFile: "file.cert", KeyFile: "file.key"})
iris.ListenTo(iris.ServerConfiguration{ReadBufferSize:42000, ListeningAddr: "mydomain.com"})
```

**List** of all Server's options:

```go
OptionServerListeningAddr(val string)

OptionServerCertFile(val string)

OptionServerKeyFile(val string)

// AutoTLS enable to get certifications from the Letsencrypt
// when this configuration field is true, the CertFile & KeyFile are empty, no need to provide a key.
//
// example: https://github.com/iris-contrib/examples/blob/master/letsencyrpt/main.go
OptionServerAutoTLS(val bool)

// Mode this is for unix only
OptionServerMode(val os.FileMode)
// OptionServerMaxRequestBodySize Maximum request body size.
//
// The server rejects requests with bodies exceeding this limit.
//
// By default request body size is 8MB.
OptionServerMaxRequestBodySize(val int)

// Per-connection buffer size for requests' reading.
// This also limits the maximum header size.
//
// Increase this buffer if your clients send multi-KB RequestURIs
// and/or multi-KB headers (for example, BIG cookies).
//
// Default buffer size is used if not set.
OptionServerReadBufferSize(val int)

// Per-connection buffer size for responses' writing.
//
// Default buffer size is used if not set.
OptionServerWriteBufferSize(val int)

// Maximum duration for reading the full request (including body).
//
// This also limits the maximum duration for idle keep-alive
// connections.
//
// By default request read timeout is unlimited.
OptionServerReadTimeout(val time.Duration)

// Maximum duration for writing the full response (including body).
//
// By default response write timeout is unlimited.
OptionServerWriteTimeout(val time.Duration)

// RedirectTo, defaults to empty, set it in order to override the station's handler and redirect all requests to this address which is of form(HOST:PORT or :PORT)
//
// NOTE: the http status is 'StatusMovedPermanently', means one-time-redirect(the browser remembers the new addr and goes to the new address without need to request something from this server
// which means that if you want to change this address you have to clear your browser's cache in order this to be able to change to the new addr.
//
// example: https://github.com/iris-contrib/examples/tree/master/multiserver_listening2
OptionServerRedirectTo(val string)

// OptionServerVirtual If this server is not really listens to a real host, it mostly used in order to achieve testing without system modifications
OptionServerVirtual(val bool)

// OptionServerVListeningAddr, can be used for both virtual = true or false,
// if it's setted to not empty, then the server's Host() will return this addr instead of the ListeningAddr.
// server's Host() is used inside global template helper funcs
// set it when you are sure you know what it does.
//
// Default is empty ""
OptionServerVListeningAddr(val string)

// OptionServerVScheme if setted to not empty value then all template's helper funcs prepends that as the url scheme instead of the real scheme
// server's .Scheme returns VScheme if  not empty && differs from real scheme
//
// Default is empty ""
OptionServerVScheme(val string)

// OptionServerName the server's name, defaults to "iris".
// You're free to change it, but I will trust you to don't, this is the only setting whose somebody, like me, can see if iris web framework is used
OptionServerName(val string)

```

View all configuration fields and options by navigating to the [kataras/iris/configuration.go source file](https://github.com/kataras/iris/blob/master/configuration.go)

[Book](https://kataras.gitbooks.io/iris/content/configuration.html) & [Examples](https://github.com/iris-contrib/examples) are updated (website docs will be updated soon).


- **CHANGED**: Use of the standard `log.Logger` instead of the `iris-contrib/logger`(colorful logger), these changes are reflects some middleware, examples and plugins, I updated all of them, so don't worry.

So, [iris-contrib/middleware/logger](https://github.com/iris-contrib/middleware/tree/master/logger) will now NO need to pass other Logger instead, instead of: `iris.Use(logger.New(iris.Logger))` use -> `iris.Use(logger.New())` which will use the iris/instance's Logger.

- **ADDED**: `context.Framework()` which returns your Iris instance (typeof `*iris.Framework`), useful for the future(Iris will give you, soon, the ability to pass custom options inside an iris instance).


- Align with [go-sessions](https://github.com/kataras/go-sessions), no front-end changes, however I think that the best time to make an upgrade to your local Iris is right now.


- Remove unused Plugin's custom callbacks, if you still need them in your project use this instead: https://github.com/kataras/go-events


Zero front-end changes. No real improvements, developers can ignore this update.

- Replace the iris sessions with a new cross-framework package, [go-sessions](https://github.com/kataras/go-sessions). Same front-end API, sessions examples are compatible, configuration of `kataras/iris/config/sessions.go` is compatible. `kataras/context.SessionStore` is now `kataras/go-sessions.Session` (normally you, as user, never used it before, because of automatically session getting by `context.Session()`)

- `GzipWriter` is taken, now, from the `kataras/go-fs` package which has improvements versus the previous implementation.



Zero front-end changes. No real improvements, developers can ignore this update.

- Replace the template engines with a new cross-framework package, [go-template](https://github.com/kataras/go-websocket). Same front-end API, examples and iris-contrib/template are compatible.


Zero front-end changes. No real improvements, developers can ignore this update.

- Replace the main and underline websocket implementation with [go-websocket](https://github.com/kataras/go-websocket). Note that we still need the [ris-contrib/websocket](https://github.com/iris-contrib/websocket) package.
- Replace the use of iris-contrib/errors with [go-errors](https://github.com/kataras/go-errors), which has more features

- **NEW FEATURE**: Optionally `OnError` foreach Party (by prefix, use it with your own risk), example [here](https://github.com/iris-contrib/examples/blob/master/httperrors/main.go#L37)
- **NEW**: `iris.Config.Sessions.CookieLength`, You're able to customize the length of each sessionid's cookie's value. Default (and previous' implementation) is 32.
- **FIX**: Websocket panic on non-websocket connection[*](https://github.com/kataras/iris/issues/367)
- **FIX**: Multi websocket servers client-side source route panic[*](https://github.com/kataras/iris/issues/365)
- Better gzip response managment


- **Feature request has been implemented**: Add layout support for Pug/Jade, example [here](https://github.com/iris-contrib/examples/tree/master/template_engines/template_pug_2).
- **Feature request has been implemented**: Forcefully closing a Websocket connection, `WebsocketConnection.Disconnect() error`.

- **FIX**: WebsocketConnection.Leave() will hang websocket server if .Leave was called manually when the websocket connection has been closed.
- **FIX**: StaticWeb not serving index.html correctly, align the func with the rest of Static funcs also, [example](https://github.com/iris-contrib/examples/tree/master/static_web) added.

Notes: if you compare it with previous releases (13+ versions before v3 stable), the v4 stable release was fast, now we had only 6 versions before stable, that was happened because many of bugs have been already fixed and we hadn't new bug reports and secondly, and most important for me, some third-party features are implemented mostly by third-party packages via other developers!


- **NEW FEATURE**: Letsencrypt.org integration[*](https://github.com/kataras/iris/issues/220)
   - example [here](https://github.com/iris-contrib/examples/blob/master/letsencrypt/main.go)
- **FIX**: (ListenUNIX adds :80 to filename)[https://github.com/kataras/iris/issues/321]
- **FIX**: (Go-Bindata + ctx.Render)[https://github.com/kataras/iris/issues/315]
- **FIX** (auto-gzip doesn't really compress data in latest code)[https://github.com/kataras/iris/issues/312]




**The important** , is that the [book](https://kataras.gitbooks.io/iris/content/) is finally updated!

If you're **willing to donate** click [here](DONATIONS.md)!


- `iris.Config.Gzip`, enables gzip compression on your Render actions, this includes any type of render, templates and pure/raw content. If you don't want to enable it globally, you could just use the third parameter on context.Render("myfileOrResponse", structBinding{}, iris.RenderOptions{"gzip": true}). It defaults to false


-  **Added** `config.Server.Name` as requested


**Fix**
- https://github.com/kataras/iris/issues/301

**Sessions changes **

- `iris.Config.Sessions.Expires` it was time.Time, changed to time.Duration, which defaults to 0, means unlimited session life duration, if you change it then the correct date is setted on client's cookie but also server destroys the session automatically when the duration passed, this is better approach, see [here](https://github.com/kataras/iris/issues/301)

**New**

A **Response Engine** gives you the freedom to create/change the render/response writer for

- `context.JSON`
- `context.JSONP`
- `context.XML`
- `context.Text`
- `context.Markdown`
- `context.Data`
- `context.Render("my_custom_type",mystructOrData{}, iris.RenderOptions{"gzip":false,"charset":"UTF-8"})`
- `context.MarkdownString`
- `iris.ResponseString(...)`


**Fix**
- https://github.com/kataras/iris/issues/294
- https://github.com/kataras/iris/issues/303


**Small changes**

- `iris.Config.Charset`, before alpha.3 was `iris.Config.Rest.Charset` & `iris.Config.Render.Template.Charset`, but you can override it at runtime by passinth a map `iris.RenderOptions` on the `context.Render` call .
- `iris.Config.IsDevelopment`, before alpha.1 was `iris.Config.Render.Template.IsDevelopment`


**Websockets changes**

No need to import the `github.com/kataras/iris/websocket` to use the `Connection` iteral, the websocket moved inside `kataras/iris` , now all exported variables' names have the prefix of `Websocket`, so the old `websocket.Connection` is now `iris.WebsocketConnection`.


Generally, no other changes on the 'frontend API', for response engines examples and how you can register your own to add more features on existing response engines or replace them, look [here](https://github.com/iris-contrib/response).

**BAD SIDE**: E-Book is still pointing on the v3 release, but will be updated soon.


**Sessions were re-written **

- Developers can use more than one 'session database', at the same time, to store the sessions
- Easy to develop a custom session database (only two functions are required (Load & Update)), [learn more](https://github.com/iris-contrib/sessiondb/blob/master/redis/database.go)
- Session databases are located [here](https://github.com/iris-contrib/sessiondb), contributions are welcome
- The only frontend deleted 'thing' is the: **config.Sessions.Provider**
- No need to register a database, the sessions works out-of-the-box
- No frontend/API changes except the `context.Session().Set/Delete/Clear`, they doesn't return errors anymore, btw they (errors) were always nil :)
- Examples (master branch) were updated.

```sh
$ go get github.com/kataras/go-sessions/sessiondb/$DATABASE
```

```go
db := $DATABASE.New(configurationHere{})
iris.UseSessionDB(db)
```


## 3.0.0 -> 4.0.0-alpha.1

[logger](https://github.com/iris-contrib/logger), [rest](https://github.com/iris-contrib/rest) and all [template engines](https://github.com/iris-contrib/template) **moved** to the [iris-contrib](https://github.com/iris-contrib).

- `config.Logger` -> `iris.Logger.Config`
- `config.Render/config.Render.Rest/config.Render.Template` -> **Removed**
- `config.Render.Rest` -> `rest.Config`
- `config.Render.Template` -> `$TEMPLATE_ENGINE.Config` except Directory,Extensions, Assets, AssetNames,
- `config.Render.Template.Directory` -> `iris.UseTemplate($TEMPLAET_ENGINE.New()).Directory("./templates", ".html")`
- `config.Render.Template.Assets` -> `iris.UseTemplate($TEMPLAET_ENGINE.New()).Directory("./templates",".html").Binary(assetFn func(name string) ([]byte, error), namesFn func() []string)`

- `context.ExecuteTemplate` -> **Removed**, you can use the `context.Response.BodyWriter()` to get its writer and execute html/template engine manually, but this is useless because we support the best support for template engines among all other (golang) web frameworks
- **Added** `config.Server.ReadBufferSize & config.Server.WriteBufferSize` which can be passed as configuration fields inside `iris.ListenTo(config.Server{...})`, which does the same job as `iris.Listen`
- **Added** `iris.UseTemplate($TEMPLAET_ENGINE.New()).Directory("./templates", ".html")` to register a template engine, now iris supports multi template engines, each template engine has its own file extension, no big changes on context.Render except the last parameter:
- `context.Render(filename string, binding interface{}, layout string{})` -> `context.Render(filename string, binding interface{}, options ...map[string]interface{})  | context.Render("myfile.html", myPage{}, iris.Map{"gzip":true,"layout":"layouts/MyLayout.html"}) |`

E-book and examples are not yet updated, no big changes.


## 3.0.0-rc.4 -> 3.0.0-pre.release

- `context.PostFormValue` -> `context.FormValueString`, old func stays until the next revision
- `context.PostFormMulti` -> `context.FormValues` , old func stays until the next revision

- Added `context.VisitAllCookies(func(key,value string))` to visit all your cookies (because `context.Request.Header.VisitAllCookie` has a bug(I can't fix/pr it because the author is away atm))
- Added `context.GetFlashes` to get all available flash messages for a particular request
- Fix flash message removed after the first `GetFlash` call in the same request

**NEW FEATURE**: Built'n support for multi listening servers per iris station, secondary and virtual servers with one-line using the `iris.AddServer` & `iris.Go` to start all servers.

- `iris.SecondaryListen` -> `iris.AddServer`, old func stays until the next revision
- Added `iris.Servers` with this field you can manage your servers very easy
- Added `iris.AddServer/iris.ListenTo/iris.Go`, but funcs like `Listen/ListenTLS/ListenUNIX` will stay forever
- Added `config.Server.Virtual(bool), config.Server.RedirectTo(string) and config.Server.MaxRequestBodySize(int64)`
- Added `iris.Available (channel bool)`
- `iris.HTTPServer` -> `iris.Servers.Main()` to get the main server, which is always the last registered server (if more than one used), old field removed
- `iris.Config.MaxRequestBodySize` -> `config.Server.MaxRequestBodySize`, old field removed

**NEW FEATURE**: Build'n support for your API's end-to-end tests

- Added `tester := iris.Tester(*testing.T)` , look inside: [http_test.go](https://github.com/kataras/iris/blob/master/http_test.go) & [./context_test.go](https://github.com/kataras/iris/blob/master/context_test.go) for `Tester` usage, you can also look inside the [httpexpect's repo](https://github.com/gavv/httpexpect/blob/master/example/iris_test.go) for extended examples with Iris.



## 3.0.0-rc.3 -> 3.0.0-rc.4

**NEW FEATURE**: **Handlebars** template engine support with all Iris' view engine's functions/helpers support, as requested [here](https://github.com/kataras/iris/issues/239):
- `iris.Config.Render.Template.Layout = "layouts/layout.html"`
- `config.NoLayout`
- **dynamic** optional layout on `context.Render`
- **Party specific** layout
- `iris.Config.Render.Template.Handlebars.Helpers["myhelper"] = func()...`
- `{{ yield }} `
- `{{ render }}`
- `{{ url "myroute" myparams}}`
- `{{ urlpath "myroute" myparams}}`

For a complete example please, click [here](https://github.com/iris-contrib/examples/tree/master/templates_handlebars).

**NEW:** Iris **can listen to more than one server per station** now, as requested [here](https://github.com/kataras/iris/issues/235).
For example you can have https with SSL/TLS and one more server http which navigates to the secure location.
Take a look [here](https://github.com/kataras/iris/issues/235#issuecomment-229399829) for an example of this.


**FIXES**
- Fix  `sessions destroy`
- Fix  `sessions persistence on subdomains` (as RFC2109 commands but you can disable it with `iris.Config.Sessions.DisableSubdomainPersistence = true`)


**IMPROVEMENTS**
- Improvements on `iris run` && `iris create`, note that the underline code for hot-reloading moved to [rizla](https://github.com/kataras/rizla).



## 3.0.0-rc.2 -> 3.0.0-rc.3

**Breaking changes**
- Move middleware & their configs to the  [iris-contrib/middleware](https://github.com/iris-contrib/middleware) repository
- Move all plugins & their configs to the [iris-contrib/plugin](https://github.com/iris-contrib/plugin) repository
- Move the graceful package to the [iris-contrib/graceful](https://github.com/iris-contrib/graceful) repository
- Move the mail package & its configs to the [iris-contrib/mail](https://github.com/iris-contrib/mail) repository

Note 1: iris.Config.Mail doesn't not logger exists, use ` mail.Config` from the `iris-contrib/mail`, and ` service:= mail.New(configs); service.Send(....)`.

Note 2: basicauth middleware's context key changed from `context.GetString("auth")` to ` context.GetString("user")`.

Underline changes, libraries used by iris' base code:
- Move the errors package to the [iris-contrib/errors](https://github.com/iris-contrib/errors) repository
- Move the tests package to the [iris-contrib/tests](https://github.com/iris-contrib/tests) repository (Yes, you should make PRs now with no fear about breaking the Iris).

**NEW**:
- OAuth, OAuth2 support via plugin (facebook,gplus,twitter and 25 more), gitbook section [here](https://kataras.gitbooks.io/iris/content/plugin-oauth.html), plugin [example](https://github.com/iris-contrib/examples/blob/master/plugin_oauth_oauth2/main.go), low-level package example [here](https://github.com/iris-contrib/examples/tree/master/oauth_oauth2) (no performance differences, it's just a working version of [goth](https://github.com/markbates/goth) which is converted to work with Iris)

- JSON Web Tokens support via [this middleware](https://github.com/iris-contrib/middleware/tree/master/jwt), book section [here](https://kataras.gitbooks.io/iris/content/jwt.html), as requested [here](https://github.com/kataras/iris/issues/187).

**Fixes**:
- [Iris run fails when not running from ./](https://github.com/kataras/iris/issues/215)
- [Fix or disable colors in iris run](https://github.com/kataras/iris/issues/217).


Improvements to the `iris run` **command**, as requested [here](https://github.com/kataras/iris/issues/192).

[Book](https://kataras.gitbooks.io/iris/content/) and [examples](https://github.com/iris-contrib/examples) are **updated** also.

## 3.0.0-rc.1 -> 3.0.0-rc.2

New:
- ` iris.MustUse/MustUseFunc`  - registers middleware for all route parties, all subdomains and all routes.
- iris control plugin re-written, added real time browser request logger
- `websocket.OnError` - Add OnError to be able to catch internal errors from the connection
- [command line tool](https://github.com/kataras/iris/tree/master/iris) - `iris run main.go` runs, watch and reload on source code changes. As requested [here](https://github.com/kataras/iris/issues/192)

Fixes: https://github.com/kataras/iris/issues/184 , https://github.com/kataras/iris/issues/175 .

## 3.0.0-beta.3, 3.0.0-beta.4 -> 3.0.0-rc.1

This version took me many days because the whole framework's underline code is rewritten after many many many 'yoga'. Iris is not so small anymore, so I (tried) to organized it a little better. Note that, today, you can just go to [iris.go](https://github.com/kataras/iris/tree/master/iris.go) and [context.go](https://github.com/kataras/iris/tree/master/context/context.go) and look what functions you can use. You had some 'bugs' to subdomains, mail service, basic authentication and logger, these are fixed also, see below...

All [examples](https://github.com/iris-contrib/examples) are updated, and I tested them one by one.


Many underline changes but the public API didn't changed much, of course this is relative to the way you use this framework, because that:

- Configuration changes: **0**

- iris.Iris pointer -> **iris.Framework** pointer

- iris.DefaultIris -> **iris.Default**
- iris.Config() -> **iris.Config** is field now
- iris.Websocket() -> **iris.Websocket** is field now
- iris.Logger() -> **iris.Logger** is field now
- iris.Plugins() -> **iris.Plugins** is field now
- iris.Server() -> **iris.HTTPServer** is field now
- iris.Rest() -> **REMOVED**

- iris.Mail() -> **REMOVED**
- iris.Mail().Send() -> **iris.SendMail()**
- iris.Templates() -> **REMOVED**
- iris.Templates().RenderString() -> **iris.TemplateString()**

- iris.StaticHandlerFunc -> **iris.StaticHandler**
- iris.URIOf() -> **iris.URL()**
- iris.PathOf() -> **iris.Path()**

- context.RenderString() returned string,error -> **context.TemplateString() returns only string, which is empty on any parse error**
- context.WriteHTML() -> **context.HTML()**
- context.HTML() -> **context.RenderWithStatus()**

Entirely new

-  -> **iris.ListenUNIX(addr string, socket os.Mode)**
-  -> **context.MustRender, same as Render but send response 500 and logs the error on parse error**
-  -> **context.Log(format string, a...interface{})**
-  -> **context.PostFormMulti(name string) []string**
-  -> **iris.Lookups() []Route**
-  -> **iris.Lookup(routeName string) Route**
-  -> **iris.Plugins.On(eventName string, ...func())** and fire all by **iris.Plugins.Call(eventName)**

- iris.Wildcard() **REMOVED**, subdomains and dynamic(wildcard) subdomains can only be registered with **iris.Party("mysubdomain.") && iris.Party("*.")**


Semantic change for static subdomains

**1**

**BEFORE** :
```go
apiSubdomain := iris.Party("api.mydomain.com")
{
//....
}
iris.Listen("mydomain.com:80")
```


**NOW** just subdomain part, no need to duplicate ourselves:
```go
apiSubdomain := iris.Party("api.")
{
//....
}
iris.Listen("mydomain.com:80")
```
**2**

Before you couldn't set dynamic subdomains and normal subdomains at the same iris station, now you can.
**NOW, this is possible**

```go
/* admin.mydomain.com,  and for other subdomains the Party(*.) */

admin := iris.Party("admin.")
{
	// admin.mydomain.com
	admin.Get("/", func(c *iris.Context) {
		c.Write("INDEX FROM admin.mydomain.com")
	})
	// admin.mydomain.com/hey
	admin.Get("/hey", func(c *iris.Context) {
		c.Write("HEY FROM admin.mydomain.com/hey")
	})
	// admin.mydomain.com/hey2
	admin.Get("/hey2", func(c *iris.Context) {
		c.Write("HEY SECOND FROM admin.mydomain.com/hey")
	})
}

// other.mydomain.com, otadsadsadsa.mydomain.com  and so on....
dynamicSubdomains := iris.Party("*.")
{
	dynamicSubdomains.Get("/", dynamicSubdomainHandler)

	dynamicSubdomains.Get("/something", dynamicSubdomainHandler)

	dynamicSubdomains.Get("/something/:param1", dynamicSubdomainHandlerWithParam)
}
```

Minor change for listen


**BEFORE you could just listen to a port**
```go
iris.Listen("8080")
```
**NOW you have set a HOSTNAME:PORT**
```go
iris.Listen(":8080")
```

Relative issues/features:  https://github.com/kataras/iris/issues/166 , https://github.com/kataras/iris/issues/176, https://github.com/kataras/iris/issues/183,  https://github.com/kataras/iris/issues/184


**Plugins**

PreHandle and PostHandle are removed, no need to use them anymore you can take routes by **iris.Lookups()**, but add support for custom event listeners by **iris.Plugins.On("event",func(){})** and fire all callbacks by **iris.Plugins.Call("event")** .

**FOR TESTERS**

**BEFORE** :
```go
api := iris.New()
//...

api.PreListen(config.Server{ListeningAddr: ""})

e := httpexpect.WithConfig(httpexpect.Config{
	Reporter: httpexpect.NewAssertReporter(t),
	Client:   fasthttpexpect.NewBinder(api.ServeRequest),
})

```

**NOW**:

```go
api := iris.New()
//...

e := httpexpect.WithConfig(httpexpect.Config{
	Reporter: httpexpect.NewAssertReporter(t),
	Client:   fasthttpexpect.NewBinder(api.NoListen().Handler),
})


```

## 3.0.0-beta.2 -> 3.0.0-beta.3

- Complete the Jade Template Engine support, {{ render }} and {{ url }} done also.

- Fix Mail().Send

- Iriscontrol plugin: Replace login using session to basic authentication

And other not-too-important fixes

## 3.0.0-beta -> 3.0.0-beta.2

- NEW: Wildcard(dynamic) subdomains, read [here](https://kataras.gitbooks.io/iris/content/subdomains.html)

- NEW: Implement feature request [#165](https://github.com/kataras/iris/issues/165). Routes can now be selected by `a custom name`, and this allows us to use the {{ url "custom-name" "any" "named" "parameters"}}``: For HTML & Amber engines, example [here](https://github.com/iris-contrib/examples/tree/master/templates_9). For PongoEngine, example [here](https://github.com/iris-contrib/examples/tree/master/templates_10_pongo)

- Remove the [x/net/context](https://godoc.org/golang.org/x/net/context), it has been useless after v2.


## 3.0.0-alpha.beta -> 3.0.0-beta


- New iris.API for easy API declaration, read more [here](https://kataras.gitbooks.io/iris/content/using-handlerapi.html), example [there](https://github.com/iris-contrib/examples/tree/master/api_handler_2).


- Add [example](https://github.com/iris-contrib/examples/tree/master/middleware_basicauth_2) and fix the Basic Authentication middleware

## 3.0.0-alpha.6 -> 3.0.0-alpha.beta

- [Implement feature request to add Globals on the pongo2](https://github.com/kataras/iris/issues/145)

- [Implement feature request for static Favicon ](https://github.com/kataras/iris/issues/141)

- Implement a unique easy only-websocket support:



```go
OnConnection(func(c websocket.Connection){})
```

websocket.Connection
```go

// Receive from the client
On("anyCustomEvent", func(message string) {})
On("anyCustomEvent", func(message int){})
On("anyCustomEvent", func(message bool){})
On("anyCustomEvent", func(message anyCustomType){})
On("anyCustomEvent", func(){})

// Receive a native websocket message from the client
// compatible without need of import the iris-ws.js to the .html
OnMessage(func(message []byte){})

// Send to the client
Emit("anyCustomEvent", string)
Emit("anyCustomEvent", int)
Emit("anyCustomEvent", bool)
Emit("anyCustomEvent", anyCustomType)

// Send via native websocket way, compatible without need of import the iris-ws.js to the .html
EmitMessage([]byte("anyMessage"))

// Send to specific client(s)
To("otherConnectionId").Emit/EmitMessage...
To("anyCustomRoom").Emit/EmitMessage...

// Send to all opened connections/clients
To(websocket.All).Emit/EmitMessage...

// Send to all opened connections/clients EXCEPT this client(c)
To(websocket.NotMe).Emit/EmitMessage...

// Rooms, group of connections/clients
Join("anyCustomRoom")
Leave("anyCustomRoom")


// Fired when the connection is closed
OnDisconnect(func(){})

```

- [Example](https://github.com/iris-contrib/examples/tree/master/websocket)
- [E-book section](https://kataras.gitbooks.io/iris/content/package-websocket.html)


We have some base-config's changed, these configs which are defaulted to true renamed to 'Disable+$oldName'
```go

		// DisablePathCorrection corrects and redirects the requested path to the registered path
		// for example, if /home/ path is requested but no handler for this Route found,
		// then the Router checks if /home handler exists, if yes,
		// (permant)redirects the client to the correct path /home
		//
		// Default is false
		DisablePathCorrection bool

		// DisablePathEscape when is false then its escapes the path, the named parameters (if any).
		// Change to true it if you want something like this https://github.com/kataras/iris/issues/135 to work
		//
		// When do you need to Disable(true) it:
		// accepts parameters with slash '/'
		// Request: http://localhost:8080/details/Project%2FDelta
		// ctx.Param("project") returns the raw named parameter: Project%2FDelta
		// which you can escape it manually with net/url:
		// projectName, _ := url.QueryUnescape(c.Param("project").
		// Look here: https://github.com/kataras/iris/issues/135 for more
		//
		// Default is false
		DisablePathEscape bool

		// DisableLog turn it to true if you want to disable logger,
		// Iris prints/logs ONLY errors, so be careful when you enable it
		DisableLog bool

		// DisableBanner outputs the iris banner at startup
		//
		// Default is false
		DisableBanner bool

```


## 3.0.0-alpha.5 -> 3.0.0-alpha.6

Changes:
	- config/iris.Config().Render.Template.HTMLTemplate.Funcs typeof `[]template.FuncMap` -> `template.FuncMap`


Added:
	- iris.AmberEngine [Amber](https://github.com/eknkc/amber). [View an example](https://github.com/iris-contrib/examples/tree/master/templates_7_html_amber)
	- iris.JadeEngine [Jade](https://github.com/Joker/jade). [View an example](https://github.com/iris-contrib/examples/tree/master/templates_6_html_jade)

Book section [Render/Templates updated](https://kataras.gitbooks.io/iris/content/render_templates.html)



## 3.0.0-alpha.4 -> 3.0.0-alpha.5

- [NoLayout support for particular templates](https://github.com/kataras/iris/issues/130#issuecomment-219754335)
- [Raw Markdown Template Engine](https://kataras.gitbooks.io/iris/content/render_templates.html)
- [Markdown to HTML](https://kataras.gitbooks.io/iris/content/render_rest.html) > `context.Markdown(statusCode int, markdown string)` , `context.MarkdownString(markdown string) (htmlReturn string)`
- [Simplify the plugin registration](https://github.com/kataras/iris/issues/126#issuecomment-219622481)

## 3.0.0-alpha.3 -> 3.0.0-alpha.4

Community suggestions implemented:

- [Request: Rendering html template to string](https://github.com/kataras/iris/issues/130)
	> New RenderString(name string, binding interface{}, layout ...string) added to the Context & the Iris' station (iris.Templates().RenderString)
- [Minify Templates](https://github.com/kataras/iris/issues/129)
	> New config field for minify, defaulted to true: iris.Config().Render.Template.Minify  = true
	> 3.0.0-alpha5+ this has been removed because the minify package has bugs, one of them is this: https://github.com/tdewolff/minify/issues/35.



Bugfixes and enhancements:

- [Static not allowing configuration of `IndexNames`](https://github.com/kataras/iris/issues/128)
- [Processing access error](https://github.com/kataras/iris/issues/125)
- [Invalid header](https://github.com/kataras/iris/issues/123)

## 3.0.0-alpha.2 -> 3.0.0-alpha.3

The only change here is a panic-fix on form bindings. Now **no need to make([]string,0)** before form binding, new example:

```go
 //./main.go

package main

import (
	"fmt"

	"github.com/kataras/iris"
)

type Visitor struct {
	Username string
	Mail     string
	Data     []string `form:"mydata"`
}

func main() {

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Render("form.html", nil)
	})

	iris.Post("/form_action", func(ctx *iris.Context) {
		visitor := Visitor{}
		err := ctx.ReadForm(&visitor)
		if err != nil {
			fmt.Println("Error when reading form: " + err.Error())
		}
		fmt.Printf("\n Visitor: %v", visitor)
	})

	fmt.Println("Server is running at :8080")
	iris.Listen(":8080")
}

```

```html

<!-- ./templates/form.html -->
<!DOCTYPE html>
<head>
<meta charset="utf-8">
</head>
<body>
<form action="/form_action" method="post">
<input type="text" name="Username" />
<br/>
<input type="text" name="Mail" /><br/>
<select multiple="multiple" name="mydata">
<option value='one'>One</option>
<option value='two'>Two</option>
<option value='three'>Three</option>
<option value='four'>Four</option>
</select>
<hr/>
<input type="submit" value="Send data" />

</form>
</body>
</html>

```



## 3.0.0-alpha.1 -> 3.0.0-alpha.2

*The e-book was updated, take a closer look [here](https://www.gitbook.com/book/kataras/iris/details)*


**Breaking changes**

**First**. Configuration owns a package now `github.com/kataras/iris/config` . I took this decision after a lot of thought and I ensure you that this is the best
architecture to easy:

- change the configs without need to re-write all of their fields.
	```go
	irisConfig := config.Iris { Profile: true, PathCorrection: false }
	api := iris.New(irisConfig)
	```

- easy to remember: `iris` type takes config.Iris, sessions takes config.Sessions`, `iris.Config().Render` is `config.Render`, `iris.Config().Render.Template` is `config.Template`, `Logger` takes `config.Logger` and so on...

- easy to find what features are exists and what you can change: just navigate to the config folder and open the type you want to learn about, for example `/iris.go` Iris' type configuration is on `/config/iris.go`

- default setted fields which you can use. They are already setted by iris, so don't worry too much, but if you ever need them you can find their default configs by this pattern: for example `config.Template` has `config.DefaultTemplate()`, `config.Rest` has `config.DefaultRest()`, `config.Typescript()` has `config.DefaultTypescript()`, note that only `config.Iris` has `config.Default()`. I wrote that all structs even the plugins have their default configs now, to make it easier for you, so you can do this without set a config by yourself: `iris.Config().Render.Template.Engine = config.PongoEngine` or `iris.Config().Render.Template.Pongo.Extensions = []string{".xhtml", ".html"}`.



**Second**. Template & rest package moved to the `render`, so

		*  a new config field named `render` of type `config.Render` which nests the `config.Template` & `config.Rest`
		-  `iris.Config().Templates` -> `iris.Config().Render.Template` of type `config.Template`
		- `iris.Config().Rest` -> `iris.Config().Render.Rest` of type `config.Rest`

**Third, sessions**.



Configuration instead of parameters. Before `sessions.New("memory","sessionid",time.Duration(42) * time.Minute)` -> Now:  `sessions.New(config.DefaultSessions())` of type `config.Sessions`

- Before this change the cookie's life was the same as the manager's Gc duration. Now added an Expires option for the cookie's life time which defaults to infinitive, as you (correctly) suggests me in the chat community.-

- Default Cookie's expiration date: from 42 minutes -> to  `infinitive/forever`
- Manager's Gc duration: from 42 minutes -> to '2 hours'
- Redis store's MaxAgeSeconds: from 42 minutes -> to '1 year`


**Four**. Typescript, Editor & IrisControl plugins now accept a config.Typescript/ config.Editor/ config.IrisControl as parameter

Bugfixes

- [can't open /xxx/ path when PathCorrection = false ](https://github.com/kataras/iris/issues/120)
- [Invalid content on links on debug page when custom ProfilePath is set](https://github.com/kataras/iris/issues/118)
- [Example with custom config not working ](https://github.com/kataras/iris/issues/115)
- [Debug Profiler writing escaped HTML?](https://github.com/kataras/iris/issues/107)
- [CORS middleware doesn't work](https://github.com/kataras/iris/issues/108)



## 2.3.2 -> 3.0.0-alpha.1

**Changed**
- `&render.Config` -> `&iris.RestConfig` . All related to the html/template are removed from there.
- `ctx.Render("index",...)` -> `ctx.Render("index.html",...)` or any extension you have defined in iris.Config().Templates.Extensions
- `iris.Config().Render.Layout = "layouts/layout"` -> `iris.Config().Templates.Layout = "layouts/layout.html"`
- `License BSD-3 Clause Open source` -> `MIT License`
**Added**

- Switch template engines via `IrisConfig`. Currently, HTMLTemplate is 'html/template'. Pongo is 'flosch/pongo2`. Refer to the Book, which is updated too, [read here](https://kataras.gitbooks.io/iris/content/render.html).
