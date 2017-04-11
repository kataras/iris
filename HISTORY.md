# Changelog

**How to upgrade**: Open your command-line and execute this command: `go get -u gopkg.in/kataras/iris.v6`.

## Looking for free support?

	http://support.iris-go.com


## 6.1.4 -> 6.2.0 (√Νεxτ)

_Update: 12 April 2017_

```
Many of you, including myself, thought that Gerasimos is not accepting any PRs, this is wrong.

I did a long PR, which actually fixes some bugs and integrations with the godep tool, on a project that he's contributing too.

The next day I'm logged into my personal twitter and I saw a message written by him.

He, @kataras, wrote me that he was really impressed by the time I spent to actually fix a bug on an Iris sub-project.
He told me that no one did that before and he asked me if I have more time to help these days on the Iris project too.

And, Here I am! Introducing myself to the most clever community!
```

```
Hello,
my name is Esemplastic.

Iris' author, @kataras, is very busy these days on designing the new Iris' release which will contain even more prototypes and it will break any rules you knew so far.
I took a sneak preview of it, don't tell to him!


I'm the temporary maintainer of this open-source project and your new friend.
```

-  **FIX**: Upgrade the [httpcache](https://github.com/geekypanda/httpcache) vendor. As requested [here](http://support.iris-go.com/d/44-upgrade-httpcache-module).


_Update: 28 March 2017_

- **View**: Provide an easier method on the community's question about "injecting" additional data outside of the route's main handler which calls the .Render, via middleware. 
	- As discussed [above](http://support.iris-go.com/d/27-using-middleware-to-inject-properties-for-templates). 
	- Click [here](https://github.com/kataras/iris/tree/v6/_examples/intermediate/view/context-view-data) for an example.

_Update: 18 March 2017_

- **Sessions**: Enchance the community's feature request about custom encode and decode methods for the cookie value(sessionid) as requested [here](http://support.iris-go.com/d/29-mark-cookie-for-session-as-secure).

_Update: 12 March 2017_

- Enhance Custom http errors with gzip and static files handler, as requested/reported [here](http://support.iris-go.com/d/17-fallback-handler-for-non-matched-routes).
- Enhance per-party custom http errors (now it works on any wildcard path too).
- Add a third parameter on `app.OnError(...)` for custom http errors with regexp validation, see [status_test.go](https://github.com/kataras/iris/blob/v6/status_test.go) for an example.
- Add a `context.ParamIntWildcard(...)` to skip the first slash, useful for wildcarded paths' parameters.


> Prepare for nice things, tomorrow is Iris' first birthday!


_Update: 28 Feb 2017_

> Note: I want you to know that I spent more than 200 hours (16 days of ~10-15 hours per-day, do the math) for this release, two days to write these changes, please read the sections before think that you have an issue and post a new question, thanks!


Users already notified for some breaking-changes, this section will help you
to adapt the new changes to your application, it contains an overview of the new features too.

- Shutdown with `app.Shutdown(context.Context) error`, no need for any third-parties, with `EventPolicy.Interrupted` and Go's 1.8 Gracefully Shutdown feature you're ready to go!
- HTTP/2 Go 1.8 `context.Push(target string, opts *http.PushOptions) error` is supported, example can be found [here](https://github.com/kataras/iris/blob/v6/adaptors/websocket/_examples/websocket_secure/main.go)

- Router (two lines to add, new features)
- Template engines (two lines to add, same features as before, except their easier configuration)
- Basic middleware, that have been written by me, are transfared to the main repository[/middleware](https://github.com/kataras/iris/tree/v6/middleware) with a lot of improvements to the `recover middleware` (see the next)
- `func(http.ResponseWriter, r *http.Request, next http.HandlerFunc)` signature is fully compatible using `iris.ToHandler` helper wrapper func, without any need of custom boilerplate code. So all net/http middleware out there are supported, no need to re-invert the world here, search to the internet and you'll find a suitable to your case.

- Load Configuration from an external file, yaml and toml:

	- [yaml-based](http://www.yaml.org/) configuration file using the `iris.YAML` function: `app := iris.New(iris.YAML("myconfiguration.yaml"))`
	- [toml-based](https://github.com/toml-lang/toml) configuration file using the `iris.TOML` function: `app := iris.New(iris.TOML("myconfiguration.toml"))`


- Add `.Regex` middleware which does path validation using the `regexp` package, i.e `.Regex("param", "[0-9]+$")`. Useful for routers that don't support regex route path validation out-of-the-box.

- Websocket additions: `c.Context() *iris.Context`, `ws.GetConnectionsByRoom("room name") []websocket.Connection`, `c.OnLeave(func(roomName string){})`, 
```go
		// SetValue sets a key-value pair on the connection's mem store.
		c.SetValue(key string, value interface{})
		// GetValue gets a value by its key from the connection's mem store.
		c.GetValue(key string) interface{}
		// GetValueArrString gets a value as []string by its key from the connection's mem store.
		c.GetValueArrString(key string) []string
		// GetValueString gets a value as string by its key from the connection's mem store.
		c.GetValueString(key string) string
		// GetValueInt gets a value as integer by its key from the connection's mem store.
		c.GetValueInt(key string) int

``` 
[examples here](https://github.com/kataras/iris/blob/v6/adaptors/websocket/_examples). 

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
- Fix [StaticEmbedded doesn't works on root "/"](https://github.com/kataras/iris/issues/633)

Changes:

- `context.TemplateString` replaced with `app.Render(w io.Writer, name string, bind interface{}, options ...map[string]interface{}) error)` which gives you more functionality.

```go
import "bytes"
// ....
app := iris.New()
// ....

buff := &bytes.Buffer{}
app.Render(buff, "my_template.html", nil)
// buff.String() is the template parser's result, use that string to send a rich-text e-mail based on a template.
```

```go
// you can take the app(*Framework instance) via *Context.Framework() too:

app.Get("/send_mail", func(ctx *iris.Context){
	buff := &bytes.Buffer{}
	ctx.Framework().Render(buff, "my_template.html", nil)
	// ...
})

```
- `.Close() error` replaced with gracefully `.Shutdown(context.Context) error`
- Remove all the package-level functions and variables for a default `*iris.Framework, iris.Default`
- Remove `.API`, use `iris.Handle/.HandleFunc/.Get/.Post/.Put/.Delete/.Trace/.Options/.Use/.UseFunc/.UseGlobal/.Party/` instead
- Remove `.Logger`, `.Config.IsDevelopment`, `.Config.LoggerOut`, `.Config.LoggerPrefix` you can adapt a logger which will log to each log message mode by `app.Adapt(iris.DevLogger())` or adapt a new one, it's just a `func(mode iris.LogMode, message string)`.
- Remove `.Config.DisableTemplateEngines`, are disabled by-default, you have to `.Adapt` a view engine by yourself
- Remove `context.RenderTemplateSource` you should make a new template file and use the `iris.Render` to specify an `io.Writer` like `bytes.Buffer`
- Remove  `plugins`, replaced with more pluggable echosystem that I designed from zero on this release, named `Policy` [Adaptors](https://github.com/kataras/iris/tree/v6/adaptors) (all plugins have been converted, fixed and improvement, except the iriscontrol).
- `context.Log(string,...interface{})` -> `context.Log(iris.LogMode, string)`
- Remove `.Config.DisableBanner`, now it's controlled by `app.Adapt(iris.LoggerPolicy(func(mode iris.LogMode, msg string)))`
- Remove `.Config.Websocket` , replaced with the `kataras/iris/adaptors/websocket.Config` adaptor.

- https://github.com/iris-contrib/plugin      ->  https://github.com/iris-contrib/adaptors

- `import "github.com/iris-contrib/middleware/basicauth"` -> `import "gopkg.in/kataras/iris.v6/middleware/basicauth"`
- `import "github.com/iris-contrib/middleware/i18n"` -> `import "gopkg.in/kataras/iris.v6/middleware/i18n"`
- `import "github.com/iris-contrib/middleware/logger"` -> `import "gopkg.in/kataras/iris.v6/middleware/logger"`
- `import "github.com/iris-contrib/middleware/recovery"` -> `import "gopkg.in/kataras/iris.v6/middleware/recover"`


- `import "github.com/iris-contrib/plugin/typescript"` -> `import "gopkg.in/kataras/iris.v6/adaptors/typescript"`
- `import "github.com/iris-contrib/plugin/editor"` -> `import "gopkg.in/kataras/iris.v6/adaptors/typescript/editor"`
- `import "github.com/iris-contrib/plugin/cors"` -> `import "gopkg.in/kataras/iris.v6/adaptors/cors"`
- `import "github.com/iris-contrib/plugin/gorillamux"` -> `import "gopkg.in/kataras/iris.v6/adaptors/gorillamux"`
- `import github.com/iris-contrib/plugin/oauth"` -> `import "github.com/iris-contrib/adaptors/oauth"`


- `import "github.com/kataras/go-template/html"` -> `import "gopkg.in/kataras/iris.v6/adaptors/view"`
- `import "github.com/kataras/go-template/django"` -> `import "gopkg.in/kataras/iris.v6/adaptors/view"`
- `import "github.com/kataras/go-template/pug"` -> `import "gopkg.in/kataras/iris.v6/adaptors/view"`
- `import "github.com/kataras/go-template/handlebars"` -> `import "gopkg.in/kataras/iris.v6/adaptors/view"`
- `import "github.com/kataras/go-template/amber"` -> `import "gopkg.in/kataras/iris.v6/adaptors/view"`

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


- [httprouter](https://github.com/kataras/iris/tree/v6/adaptors/httprouter), the old defaulted. A router that can be adapted, it's a custom version of https://github.comjulienschmidt/httprouter which is edited to support iris' subdomains, reverse routing, custom http errors and a lot features, it should be a bit faster than the original too because of iris' Context. It uses `/mypath/:firstParameter/path/:secondParameter` and `/mypath/*wildcardParamName` .


Example:

```go
package main

import (
  "gopkg.in/kataras/iris.v6"
  "gopkg.in/kataras/iris.v6/adaptors/httprouter" // <---- NEW
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

- [gorillamux](https://github.com/kataras/iris/tree/v6/adaptors/gorillamux), a router that can be adapted, it's the https://github.com/gorilla/mux which supports subdomains, custom http errors, reverse routing, pattern matching via regex and the rest of the iris' features.


Example:

```go
package main

import (
  "gopkg.in/kataras/iris.v6"
  "gopkg.in/kataras/iris.v6/adaptors/gorillamux" // <---- NEW
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


Adaptors are located [there](https://github.com/kataras/iris/tree/v6/adaptors).

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

Each of the template engines has different options, view adaptors are located [here](https://github.com/kataras/iris/tree/v6/adaptors/view).


Example:


```go
package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/gorillamux" // <--- NEW (previous section)
	"gopkg.in/kataras/iris.v6/adaptors/view" // <--- NEW it contains all the template engines
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
import ""gopkg.in/kataras/iris.v6/adaptors/view"
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
iris.Set(OptionDisableBodyConsumptionOnUnmarshal(true))
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
app.Set(OptionDisableBodyConsumptionOnUnmarshal(true))
// same as
// app := iris.New(iris.Configuration{FireMethodNotAllowed:true, DisableBodyConsumptionOnUnmarshal:true})
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
  "gopkg.in/kataras/iris.v6"
  "gopkg.in/kataras/iris.v6/adaptors/httprouter"
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
  "gopkg.in/kataras/iris.v6"
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
	"gopkg.in/kataras/iris.v6"
  "gopkg.in/kataras/iris.v6/adaptors/gorillamux"
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
	 - Param
     - URLPath
- RouterBuilderPolicy
- RouterWrapperPolicy
- RenderPolicy
- TemplateFuncsPolicy
- SessionsPolicy


**Details** of these can be found at [policy.go](https://github.com/kataras/iris/blob/v6/policy.go).

The **Community**'s adaptors are [here](https://github.com/iris-contrib/adaptors).

**Iris' Built'n Adaptors** for these policies can be found at [/adaptors folder](https://github.com/kataras/iris/tree/v6/adaptors).

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
	"gopkg.in/kataras/iris.v6"
  "gopkg.in/kataras/iris.v6/adaptors/gorillamux"
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

- FIX: [iris run main.go](https://github.com/kataras/iris/tree/v6/iris#run) not reloading when file changes maden by some of the IDEs,
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


**[Examples folder](https://github.com/kataras/iris/tree/v6/adaptors/sessions/_examples)**



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
html and javascript sources here, you can run the full examples from [here](https://github.com/kataras/iris/tree/v6/adaptors/websocket/_examples).

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

	// the path which the websocket client should listen/registered to ->
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
		// the path which the websocket client should listen/registered to,
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

    "gopkg.in/kataras/iris.v6"
    "gopkg.in/kataras/iris.v6/adaptors/httprouter"
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
Example can be bound [here](https://github.com/kataras/iris/tree/v6/adaptors/typescript/_example).

The Cloud-based editor adaptor(old 'plugin') also fixed and improved to show debug messages to your iris' LoggerPolicy.
Example can be bound [here](https://github.com/kataras/iris/tree/v6/adaptors/typescript/editor/_example).

Their import paths also changed as the rest of the old plugins from: https://github.com/iris-contrib/plugin to https://github.com/kataras/adaptors and https://github.com/iris-contrib/adaptors
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
	"gopkg.in/kataras/iris.v6"
  "gopkg.in/kataras/iris.v6/adaptors/gorillamux"
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
	"gopkg.in/kataras/iris.v6"
  "gopkg.in/kataras/iris.v6/adaptors/httprouter"
  "gopkg.in/kataras/iris.v6/adaptors/cors"
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
