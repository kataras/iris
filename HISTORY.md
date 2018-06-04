# History/Changelog <a href="HISTORY_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a><a href="HISTORY_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a><a href="HISTORY_GR.md"> <img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a>

### Looking for free and real-time support?

    https://github.com/kataras/iris/issues
    https://chat.iris-go.com

### Looking for previous versions?

    https://github.com/kataras/iris/releases

### Should I upgrade my Iris?

Developers are not forced to upgrade if they don't really need it. Upgrade whenever you feel ready.

> Iris uses the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature, so you get truly reproducible builds, as this method guards against upstream renames and deletes.

**How to upgrade**: Open your command-line and execute this command: `go get -u github.com/kataras/iris` or let the automatic updater do that for you.

# Tu, 05 June 2018 | v10.6.6

- **view/pug**: update vendor for Pug (Jade) parser and add [Iris + Pug examples](https://github.com/kataras/iris/tree/master/_examples#view) via [this commit](https://github.com/kataras/iris/commit/e0171cbed69efecba199ef547aa5e7063e18b27a), relative to [issue #1003](https://github.com/kataras/iris/issues/1003) opened by [@DjLeChuck](https://github.com/DjLeChuck)
- **middleware/logger**: new configuration field, defaults to false: `Query bool`, if true prints the full path, including the URL query as requested at [issue #1017](https://github.com/kataras/iris/issues/1017) by [@andr33z](https://github.com/andr33z). Example [here](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/main.go#L21). Implemented by [this commit](https://github.com/kataras/iris/commit/a7364876e0d1b8bd60acf94f17f6d1341b16c617)
- **cookies**: some minor but helpful additions, like `CookieOption` relative to [issue #1018](https://github.com/kataras/iris/issues/1018) asked by [@dibyendu](https://github.com/dibyendu). [Cookies examples added](https://github.com/kataras/iris/tree/master/_examples/cookies) too. Implemented by [this commit](https://github.com/kataras/iris/commit/574414a64ed3d8736c836d476e6304d915f4a511)
- **cookies**: ability to set custom cookie encoders to encode the cookie's value before sent by `context#SetCookie` and `context#SetCookieKV` and cookie decoders to decode the cookie's value when retrieving from `context#GetCookie`. That was the second and final part relative to a community's question at: [issue #1018](https://github.com/kataras/iris/issues/1018). Implemented by [this commit](https://github.com/kataras/iris/commit/f708c6098faec7c4e2232c791380cdff7a26960b)
- **fix**: [issue #1020](https://github.com/kataras/iris/issues/1020) via [this commit](https://github.com/kataras/iris/commit/3d30ccef05703246b716a14dda14d2f28294dbd2), redis database stores the int as float64, don't change that native behavior, just grab it nicely.

## Translations (2)

- [README_PT_BR.md](README_PT_BR.md) for Brazilian Portuguese language via [this PR](https://github.com/kataras/iris/pull/1008) thanks to [@gschri](https://github.com/gschri)
- [README_JPN.md](README_JPN.md) for Japanese language via [this PR](https://github.com/kataras/iris/pull/1015) thanks to [@tkhkokd](https://github.com/tkhkokd).

Thank you both for your contribution. We all looking forward for the HISTORY translations as well!!!

# Mo, 21 May 2018 | v10.6.5

First of all, special thanks to [@haritsfahreza](https://github.com/haritsfahreza) for translating the entire Iris' README page & Changelogs to the Bahasa Indonesia language via PR: [#1000](https://github.com/kataras/iris/pull/1000)!

## New Feature: `Execution Rules`

From the begin of the Iris' journey we used to use the `ctx.Next()` inside handlers in order to call the next handler in the route's registered handlers chain, otherwise the "next handler" would never be executed.

We could always "force-break" that handlers chain using the `ctx.StopExecution()` to indicate that any future `ctx.Next()` calls will do nothing.

These things will never change, they were designed in the lower possible level of the Iris' high-performant and unique router and they're working like a charm:)

We have introduced `Iris MVC Applications` two years later. Iris is the first and the only one Go web framework with a realistic point-view and feature-rich MVC architectural pattern support without sacrifices, always with speed in mind (handlers vs mvc have almost the same speed here!!!).

A bit later we introduced another two unique features, `Hero Handlers and Service/Dynamic Bindings` (see the very bottom of this HISTORY page).
You loved it, you're using it a lot, just take a look at the recent github issues the community raised about MVC and etc.

Two recent discussions/support were about calling `Done` handlers inside MVC applications, you could simply do that by implementing the optional `BaseController` as examples shown, i.e:

```go
func (c *myController) BeginRequest(ctx iris.Context) {}
func (c *myController) EndRequest(ctx iris.Context) {
    ctx.Next() // Call of any `Done` handlers.
}
```

But for some reason you found that confused. This is where the new feature comes: **The option to change the default behavior of handlers execution's rules PER PARTY**.

For example, we want to run all handlers(begin, main and done handlers) with the order you register but without the need of the `ctx.Next()` (in that case the only remained way to stop the lifecycle of an http request when next handlers are registered is to use the `ctx.StopExecution()` which, does not allow the next handler(s) to be executed even if `ctx.Next()` called in some place later on, but you're already know this, I hope :)).

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

func main() {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) { ctx.Redirect("/example") })

    m := mvc.New(app.Party("/example"))

    // IMPORTANT
    // the new feature, all options can be filled with Force:true, they are all play nice together.
    m.Router.SetExecutionRules(iris.ExecutionRules{
        // Begin:  <- from `Use[all]` to `Handle[last]` future route handlers, execute all, execute all even if `ctx.Next()` is missing.
        // Main:   <- all `Handle` future route handlers, execute all >> >>.
        Done: iris.ExecutionOptions{Force: true}, // <- from `Handle[last]` to `Done[all]` future route handlers, execute all >> >>.
    })
    m.Router.Done(doneHandler)
    // m.Router.Done(...)
    // ...
    //

    m.Handle(&exampleController{})

    app.Run(iris.Addr(":8080"))
}

func doneHandler(ctx iris.Context) {
    ctx.WriteString("\nFrom Done Handler")
}

type exampleController struct{}

func (c *exampleController) Get() string {
    return "From Main Handler"
    // Note that here we don't binding the `Context`, and we don't call its `Next()`
    // function in order to call the `doneHandler`,
    // this is done automatically for us because we changed the execution rules with the `SetExecutionRules`.
    //
    // Therefore, the final output is:
    // From Main Handler
    // From Done Handler
}
```

Example at: [_examples/mvc/middleware/without-ctx-next](_examples/mvc/middleware/without-ctx-next).

This feature can be applied to any type of application, the example is an MVC Application because many of you asked for this exactly flow the past days.

## Thank you

Thank you for your honest support once again, your posts are the heart of this framework.

Don't forget to [star](https://github.com/kataras/iris/stargazers) the Iris' github repository whenever you can and spread the world about its potentials!

Be part of this,

- complete our User Experience Report: https://goo.gl/forms/lnRbVgA6ICTkPyk02
- join to our Community live chat: https://kataras.rocket.chat/channel/iris
- connect to our [new facebook group](https://www.facebook.com/iris.framework) to get notifications about new job opportunities relatively to Iris!

Sincerely,
[Gerasimos Maropoulos](https://twitter.com/MakisMaropoulos).

# We, 09 May 2018 | v10.6.4

- [fix issue 995](https://github.com/kataras/iris/commit/62457279f41a1f157869a19ef35fb5198694fddb)
- [fix issue 996](https://github.com/kataras/iris/commit/a11bb5619ab6b007dce15da9984a78d88cd38956)

# We, 02 May 2018 | v10.6.3

**Every server should be upgraded to this version**, it contains an important, but easy, fix for the `websocket/Connection#Emit##To`.

- Websocket: fix https://github.com/kataras/iris/issues/991

# Tu, 01 May 2018 | v10.6.2

- Websocket: added OnPong to Connection via PR: https://github.com/kataras/iris/pull/988
- Websocket: `OnError` accepts a `func(error)` now instead of `func(string)`, as requested at: https://github.com/kataras/iris/issues/987

# We, 25 April 2018 | v10.6.1

- Re-implement the [BoltDB](https://github.com/coreos/bbolt) as built'n back-end storage for sessions(`sessiondb`) using the latest features: [/sessions/sessiondb/boltdb/database.go](sessions/sessiondb/boltdb/database.go), example can be found at [/_examples/sessions/database/boltdb/main.go](_examples/sessions/database/boltdb/main.go).
- Fix a minor issue on [Badger sessiondb example](_examples/sessions/database/badger/main.go). Its `sessions.Config { Expires }` field was `2 *time.Second`, it's `45 *time.Minute` now.
- Other minor improvements to the badger sessiondb.

# Su, 22 April 2018 | v10.6.0

- Fix open redirect by @wozz via PR: https://github.com/kataras/iris/pull/972.
- Fix when destroy session can't remove cookie in subdomain by @Chengyumeng via PR: https://github.com/kataras/iris/pull/964.
- Add `OnDestroy(sid string)` on sessions for registering a listener when a session is destroyed with commit: https://github.com/kataras/iris/commit/d17d7fecbe4937476d00af7fda1c138c1ac6f34d.
- Finally, sessions are in full-sync with the registered database now. That required a lot of internal code changed but **zero code change requirements by your side**. We kept only `badger` and `redis` as the back-end built'n supported sessions storages, they are enough. Made with commit: https://github.com/kataras/iris/commit/f2c3a5f0cef62099fd4d77c5ccb14f654ddbfb5c relative to many issues that you've requested it.

# Sa, 24 March 2018 | v10.5.0

### New

Add new client cache (helpers) middlewares for even faster static file servers. Read more [there](https://github.com/kataras/iris/pull/935).

### Breaking Change

Change the `Value<T>Default(<T>, error)` to `Value<T>Default(key, defaultValue) <T>`  like `ctx.PostValueIntDefault` or `ctx.Values().GetIntDefault` or `sessions/session#GetIntDefault` or `context#URLParamIntDefault`.
The proposal was made by @jefurry at https://github.com/kataras/iris/issues/937.

#### How to align your existing codebase

Just remove the second return value from these calls.

Nothing too special or hard to change here, think that in our 100+ [_examples](_examples) we had only two of them.

For example: at [_examples/mvc/basic/main.go line 100](_examples/mvc/basic/main.go#L100) the `count,_ := c.Session.GetIntDefault("count", 1)` **becomes now:** `count := c.Session.GetIntDefault("count", 1)`.

> Remember that if you can't upgrade then just don't, we dont have any security fixes in this release, but at some point you will have to upgrade for your own good, we always add new features that you will love to embrace!

# We, 14 March 2018 | v10.4.0

- fix `APIBuilder, Party#StaticWeb` and `APIBuilder, Party#StaticEmbedded` wrong strip prefix inside children parties
- keep the `iris, core/router#StaticEmbeddedHandler` and remove the `core/router/APIBuilder#StaticEmbeddedHandler`,  (note the `Handler` suffix) it's global and has nothing to do with the `Party` or the `APIBuilder`
- fix high path cleaning between `{}` (we already escape those contents at the [interpreter](core/router/macro/interpreter) level but some symbols are still removed by the higher-level api builder) , i.e `\\` from the string's macro function `regex` contents as reported at [927](https://github.com/kataras/iris/issues/927) by [commit e85b113476eeefffbc7823297cc63cd152ebddfd](https://github.com/kataras/iris/commit/e85b113476eeefffbc7823297cc63cd152ebddfd)
- sync the `golang.org/x/sys/unix` vendor

## The most important

We've made static files served up to 8 times faster using the new tool, <https://github.com/kataras/bindata> which is a fork of your beloved `go-bindata`, some unnecessary things for us were removed there and contains some additions for performance boost.

## Reqs/sec with [shuLhan/go-bindata](https://github.com/shuLhan/go-bindata) and alternatives

![go-bindata](https://github.com/kataras/bindata/raw/master/go-bindata-benchmark.png)

## Reqs/sec with [kataras/bindata](https://github.com/kataras/bindata)

![bindata](https://github.com/kataras/bindata/raw/master/bindata-benchmark.png)

A **new** function `Party#StaticEmbeddedGzip` which has the same input arguments as the `Party#StaticEmbedded` added. The difference is that the **new** `StaticEmbeddedGzip` accepts the `GzipAsset` and `GzipAssetNames` from the `bindata` (go get -u github.com/kataras/bindata/cmd/bindata).

You can still use both  `bindata` and `go-bindata` tools in the same folder, the first for embedding the rest of the static files (javascript, css, ...) and the second for embedding the templates!

A full example can be found at: [_examples/file-server/embedding-gziped-files-into-app/main.go](_examples/file-server/embedding-gziped-files-into-app/main.go).

_Happy Coding!_

# Sa, 10 March 2018 | v10.3.0

- The only one API Change is the [Application/Context/Router#RouteExists](https://godoc.org/github.com/kataras/iris/core/router#Router.RouteExists), it accepts the `Context` as its first argument instead of last now.

- Fix cors middleware via https://github.com/iris-contrib/middleware/commit/048e2be034ed172c6754448b8a54a9c55debad46, relative issue: https://github.com/kataras/iris/issues/922 (still pending for a verification).

- Add `Context#NextOr` and `Context#NextOrNotFound`

```go
// NextOr checks if chain has a next handler, if so then it executes it
// otherwise it sets a new chain assigned to this Context based on the given handler(s)
// and executes its first handler.
//
// Returns true if next handler exists and executed, otherwise false.
//
// Note that if no next handler found and handlers are missing then
// it sends a Status Not Found (404) to the client and it stops the execution.
NextOr(handlers ...Handler) bool
// NextOrNotFound checks if chain has a next handler, if so then it executes it
// otherwise it sends a Status Not Found (404) to the client and stops the execution.
//
// Returns true if next handler exists and executed, otherwise false.
NextOrNotFound() bool
```

- Add a new `Party#AllowMethods` which if called before any `Handle, Get, Post...` will clone the routes to that methods as well.

- Fix trailing slash from POST method request redirection as reported at: https://github.com/kataras/iris/issues/921 via https://github.com/kataras/iris/commit/dc589d9135295b4d080a9a91e942aacbfe5d56c5

-  Add examples for read using custom decoder per type, read using custom decoder via `iris#UnmarshalerFunc` and to complete it add an example for the `context#ReadXML`, you can find them [here](https://github.com/kataras/iris/tree/master/_examples#how-to-read-from-contextrequest-httprequest)via https://github.com/kataras/iris/commit/78cd8e5f677fe3ff2c863c5bea7d1c161bf4c31e.

- Add one more example for custom router macro functions, relative to https://github.com/kataras/iris/issues/918, you can find it [there](https://github.com/kataras/iris/blob/master/_examples/routing/dynamic-path/main.go#L144-L158), via https://github.com/kataras/iris/commit/a7690c71927cbf3aa876592fab94f04cada91b72

- Add wrappers for `Pongo`'s `AsValue()` and `AsSaveValue()` by @neenar via PR: https://github.com/kataras/iris/pull/913

- Remove unnecessary reflection usage on `context#UnmarshalBody` via https://github.com/kataras/iris/commit/4b9e41458b62035ea4933789c0a132c3ef2a90cc


# Th, 15 February 2018 | v10.2.1

Fix subdomains' `StaticEmbedded` & `StaticWeb` not found errors, as reported by [@speedwheel](https://github.com/speedwheel) via [facebook page's chat](https://facebook.com/iris.framework).

# Th, 08 February 2018 | v10.2.0

A new minor version family because it contains a **BREAKING CHANGE** and a new `Party#Reset` function.

### Party#Done behavior change & new Party#DoneGlobal introduced

As correctly pointed out by @likakuli at https://github.com/kataras/iris/issues/901, the old `Done` registered
handlers globally instead of party's and its children routes, this was not by accident because `Done` was introduced
before the `UseGlobal` idea and it didn't change for the shake of stability. Now it's time to move on, the new `Done` should be called before the routes that they care about those done handlers and the **new** `DoneGlobal` works like the old `Done`; order doesn't matter and it appends those done handlers
to the current registered routes and the future, globally (to all subdomains, parties every route in the Application).

The [routing/writing-a-middleware](_examples/routing/writing-a-middleware) examples are updated, read those to understand what's going on, although if you used iris before and you know the vocabulary we use you don't have to, the `DoneGlobal` and `Done` are clearly separated.

### Party#Reset

A new `Party#Reset()` function introduced in order to be able to clear parent's Party's begin and done handlers that are registered via `Use` and `Done` at a previous state, nothing crazy about this, it just clears the `middleware` and `doneHandlers` of the current Party instance, see `core/router#APIBuilder` for more.

### Update your codebase

Just replace all existing `.Done(` with `.DoneGlobal(` using a rich code editor (like the [VSCode](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)) which supports `find and replace all` and you're ready to Go:)

# Tu, 06 February 2018 | v10.1.0

New Features:

- Multi-Level subdomain redirect helper, you can find an example [here](https://github.com/kataras/iris/blob/master/_examples/subdomains/redirect/main.go)
- Cache middleware which makes use of the `304` status code, request fires from client to server but server respond with a status code, client is responsible to render the cached, you can find an example [here](https://github.com/kataras/iris/blob/master/_examples/cache/client-side/main.go)
- `websocket/Connection#IsJoined(roomName string)` new method to check if a user is joined to a room. An un-joined connections cannot send messages, this check is optionally.

More:

- update vendor/golang/crypto package to its latest version again, they have a lot of fixes there, as you know we're always following the dependencies for any fixes and meanful updates.
- [don't force-set content type on gzip response writer's WriteString and Writef if already there](https://github.com/kataras/iris/commit/af79aad11932f1a4fcbf7ebe28274b96675d0000)
- [new: add websocket/Connection#IsJoined](https://github.com/kataras/iris/commit/cb9e30948c8f1dd099f5168218d110765989992e)
- [fix #897](https://github.com/kataras/iris/commit/21cb572b638e82711910745cfae3c52d836f01f9)
- [add context#StatusCodeNotSuccessful variable for customize even the rfc2616-sec10](https://github.com/kataras/iris/commit/c56b7a3f04d953a264dfff15dadd2b4407d62a6f)
- [fix example comment on routing/dynamic-path/main.go#L101](https://github.com/kataras/iris/commit/0fbf1d45f7893cb1393759b7362444f3d381d182)
- [new: Cache Middleware `iris.Cache304`](https://github.com/kataras/iris/commit/1722355870174cecbc12f7beff8514b058b3b912)
- [fix comment on csrf example](https://github.com/kataras/iris/commit/a39e3d7d6cf528e51e6c7e32a884a8d9f2fadc0b)
- [un-default the Configuration.RemoteAddrHeaders](https://github.com/kataras/iris/commit/47108dc5a147a8b23de61bef86fe9327f0781396)
- [add vscode extension link and badge](https://github.com/kataras/iris/commit/6f594c0a7c641cc98bd683163fffbf5fa5fc8de6)
- [add an `app.View` example for parsing and writing templates outside of the HTTP (similar to context#View)](_examples/view/write-to)
- [new: Support multi-level subdomains redirect](https://github.com/kataras/iris/commit/12d7df113e611a75088c2a72774dab749d2c7685).

# Tu, 16 January 2018 | v10.0.2

## Security | `iris.AutoTLS`

**Every server should be upgraded to this version**, it contains fixes for the _tls-sni challenge disabled_ some days ago by letsencrypt.org which caused almost every https-enabled golang server to be unable to be functional, therefore support for the _http-01 challenge type_ added. Now the server is testing all available letsencrypt challenges.

Read more at:

- https://letsencrypt.status.io/pages/incident/55957a99e800baa4470002da/5a55777ed9a9c1024c00b241
- https://github.com/golang/crypto/commit/13931e22f9e72ea58bb73048bc752b48c6d4d4ac

# Mo, 15 January 2018 | v10.0.1

Not any serious problems were found to be resolved here but one, the first one which is important for devs that used the [cache](cache) package.

- fix a single one cache handler didn't work across multiple route handlers at the same time https://github.com/kataras/iris/pull/852, as reported at https://github.com/kataras/iris/issues/850
- merge PR https://github.com/kataras/iris/pull/862
- do not allow concurrent access to the `ExecuteWriter -> Load` when `view#Engine##Reload` was true, as requested at https://github.com/kataras/iris/issues/872
- badge for open-source projects powered by Iris, learn how to add that badge to your open-source project at [FAQ.md](FAQ.md) file
- upstream update for `golang/crypto` to apply the fix about the [tls-sni challenge disabled](https://letsencrypt.status.io/pages/incident/55957a99e800baa4470002da/5a55777ed9a9c1024c00b241) https://github.com/golang/crypto/commit/13931e22f9e72ea58bb73048bc752b48c6d4d4ac (**relative to iris.AutoTLS**)

## New Backers

1. https://opencollective.com/cetin-basoz

## New Translations

1. The Chinese README_ZH.md and HISTORY_ZH.md was translated by @Zeno-Code via https://github.com/kataras/iris/pull/858
2. New Russian README_RU.md translations by @merrydii via https://github.com/kataras/iris/pull/857
3. New Greek README_GR.md and HISTORY_GR.md translations via https://github.com/kataras/iris/commit/8c4e17c2a5433c36c148a51a945c4dc35fbe502a#diff-74b06c740d860f847e7b577ad58ddde0 and https://github.com/kataras/iris/commit/bb5a81c540b34eaf5c6c8e993f644a0e66a78fb8

## New Examples

1. [MVC - Register Middleware](_examples/mvc/middleware)

## New Articles

1. [A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
2. [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](bit.ly/2lmKaAZ)

# Mo, 01 January 2018 | v10.0.0

We must thanks [Mrs. Diana](https://www.instagram.com/merry.dii/) for our awesome new [logo](https://iris-go.com/images/icon.svg)!

You can [contact](mailto:Kovalenkodiana8@gmail.com) her for any  design-related enquiries or explore and send a direct message via [instagram](https://www.instagram.com/merry.dii/).

<p align="center">
<img width="145px" src="https://iris-go.com/images/icon.svg?v=a" />
</p>

At this version we have many internal improvements but just two major changes and one big feature, called **hero**.

> The new version adds 75 plus new commits, the PR is located [here](https://github.com/kataras/iris/pull/849) read the internal changes if you are developing a web framework based on Iris. Why 9 was skipped? Because.

## Hero

The new package [hero](hero) contains features for binding any object or function that `handlers` may use, these are called dependencies. Hero funcs can also return any type of values, these values will be dispatched to the client.

> You may saw binding before but you didn't have code editor's support, with Iris you get truly safe binding thanks to the new `hero` package. It's also fast, near to raw handlers performance because Iris calculates everything before server ran!

Below you will see some screenshots we prepared for you in order to be easier to understand:

### 1. Path Parameters - Built'n Dependencies

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-1-monokai.png)

### 2. Services - Static Dependencies

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-2-monokai.png)

### 3. Per-Request - Dynamic Dependencies

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-3-monokai.png)

`hero funcs` are very easy to understand and when you start using them **you never go back**.

Examples:

- [Basic](_examples/hero/basic/main.go)
- [Overview](_examples/hero/overview)

## MVC

You have to understand the `hero` package in order to use the `mvc`, because `mvc` uses the `hero` internally for the controller's methods you use as routes, the same rules applied to those controller's methods of yours as well.

With this version you can register **any controller's methods as routes manually**, you can **get a route based on a method name and change its `Name` (useful for reverse routing inside templates)**, you can use any **dependencies** registered from `hero.Register` or `mvc.New(iris.Party).Register` per mvc application or per-controller, **you can still use `BeginRequest` and `EndRequest`**, you can catch **`BeforeActivation(b mvc.BeforeActivation)` to add dependencies per controller and `AfterActivation(a mvc.AfterActivation)` to make any post-validations**, **singleton controllers when no dynamic dependencies are used**, **Websocket controller, as simple as a `websocket.Connection` dependency** and more...

Examples:

**If you used MVC before then read very carefully: MVC CONTAINS SOME BREAKING CHANGES BUT YOU CAN DO A LOT MORE AND EVEN FASTER THAN BEFORE**

**PLEASE READ THE EXAMPLES CAREFULLY, WE'VE MADE THEM FOR YOU**

Old examples are here as well. Compare the two different versions of each example to understand what you win if you upgrade now.

| NEW | OLD |
| -----------|-------------|
| [Hello world](_examples/mvc/hello-world/main.go) | [OLD Hello world](https://github.com/kataras/iris/blob/v8/_examples/mvc/hello-world/main.go) |
| [Session Controller](_examples/mvc/session-controller/main.go) | [OLD Session Controller](https://github.com/kataras/iris/blob/v8/_examples/mvc/session-controller/main.go) |
| [Overview - Plus Repository and Service layers](_examples/mvc/overview) | [OLD Overview - Plus Repository and Service layers](https://github.com/kataras/iris/tree/v8/_examples/mvc/overview) |
| [Login showcase - Plus Repository and Service layers](_examples/mvc/login) | [OLD Login showcase - Plus Repository and Service layers](https://github.com/kataras/iris/tree/v8/_examples/mvc/login) |
| [Singleton](_examples/mvc/singleton) |  **NEW** |
| [Websocket Controller](_examples/mvc/websocket) |  **NEW** |
| [Vue.js Todo MVC](_examples/tutorial/vuejs-todo-mvc) |  **NEW** |

## context#PostMaxMemory

Remove the old static variable `context.DefaultMaxMemory` and replace it with the configuration `WithPostMaxMemory`.

```go
// WithPostMaxMemory sets the maximum post data size
// that a client can send to the server, this differs
// from the overral request body size which can be modified
// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
//
// Defaults to 32MB or 32 << 20 if you prefer.
func WithPostMaxMemory(limit int64) Configurator
```

If you used that old static field you will have to change that single line.

Usage:

```go
import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // [...]

    app.Run(iris.Addr(":8080"), iris.WithPostMaxMemory(10 << 20))
}
```

## context#UploadFormFiles

New method to upload multiple files, should be used for common upload actions, it's just a helper function.

```go
// UploadFormFiles uploads any received file(s) from the client
// to the system physical location "destDirectory".
//
// The second optional argument "before" gives caller the chance to
// modify the *miltipart.FileHeader before saving to the disk,
// it can be used to change a file's name based on the current request,
// all FileHeader's options can be changed. You can ignore it if
// you don't need to use this capability before saving a file to the disk.
//
// Note that it doesn't check if request body streamed.
//
// Returns the copied length as int64 and
// a not nil error if at least one new file
// can't be created due to the operating system's permissions or
// http.ErrMissingFile if no file received.
//
// If you want to receive & accept files and manage them manually you can use the `context#FormFile`
// instead and create a copy function that suits your needs, the below is for generic usage.
//
// The default form's memory maximum size is 32MB, it can be changed by the
//  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
//
// See `FormFile` to a more controlled to receive a file.
func (ctx *context) UploadFormFiles(
        destDirectory string,
        before ...func(string, string),
    ) (int64, error)
```

Example can be found [here](_examples/http_request/upload-files/main.go).

## context#View

Just a minor addition, add a second optional variadic argument to the `context#View` method to accept a single value for template binding.
When you just want one value and not key-value pairs, you used to use an empty string on the `ViewData`, which is fine, especially if you preload these from a previous handler/middleware in the request handlers chain.

```go
func(ctx iris.Context) {
    ctx.ViewData("", myItem{Name: "iris" })
    ctx.View("item.html")
}
```

Same as:

```go
func(ctx iris.Context) {
    ctx.View("item.html", myItem{Name: "iris" })
}
```

```html
Item's name: {{.Name}}
```

## context#YAML

Add a new `context#YAML` function, it renders a yaml from a structured value.

```go
// YAML marshals the "v" using the yaml marshaler and renders its result to the client.
func YAML(v interface{}) (int, error)
```

## Session#GetString

`sessions/session#GetString` can now return a filled value even if the stored value is a type of integer, just like the memstore, the context's temp store, the context's path parameters and the context's url parameters.