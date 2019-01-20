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

# Fr, 11 January 2019 | v11.1.1

Happy new year! This is a minor release, contains mostly bug fixes.

Strange that we don't have major features in this release, right? Don't worry, I am not out of ideas (at least not yet!).
I have some features in-mind but lately I do not have the time to humanize those ideas for you due to my new position in [Netdata Inc.](https://github.com/netdata/netdata), so be patient and [stay-tuned](https://github.com/kataras/iris/stargazers). Read the current changelog below:

- session/redis: fix unused service config var. IdleTimeout witch was replaced by default values. [#1140](https://github.com/kataras/iris/pull/1140) ([@d7561985](https://github.com/d7561985))

- fix [#1141](https://github.com/kataras/iris/issues/1141) and [#1142](https://github.com/kataras/iris/issues/1142). [2bd7a8e88777766d1f4cac7562feec304112d2b1](https://github.com/kataras/iris/commit/2bd7a8e88777766d1f4cac7562feec304112d2b1) (@kataras)

- fix cache corruption due to recorder reuse. [#1146](https://github.com/kataras/iris/pull/1146) ([@Slamper](https://github.com/Slamper))

- add `StatusTooEarly`, compatible with: https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/425#Browser_compatibility. [31b2913447aa9e41e16a3eb33eb0019427e15cea](https://github.com/kataras/iris/commit/31b2913447aa9e41e16a3eb33eb0019427e15cea) (@kataras)

- fix [#1164](https://github.com/kataras/iris/issues/1164). [701e8e46c20395f87fa34bf9fabd145074c7b78c](https://github.com/kataras/iris/commit/701e8e46c20395f87fa34bf9fabd145074c7b78c) (@kataras)

- `context#ReadForm` can skip unkown fields by `IsErrPath(err)`, fixes: [#1157](https://github.com/kataras/iris/issues/1157). [1607bb5113568af6a34142f23bfa44903205b314](https://github.com/kataras/iris/commit/1607bb5113568af6a34142f23bfa44903205b314) (@kataras)


Doc updates:

- fix grammar and misspell. [5069e9afd8700d20dfd04cdc008efd671b5d0b40](https://github.com/kataras/iris/commit/5069e9afd8700d20dfd04cdc008efd671b5d0b40) (@kataras)

- fix link for httpexpect in README. [#1148](https://github.com/kataras/iris/pull/1148) ([@drenel18](https://github.com/drenel18))

- translate _examples/README.md into Chinese. [#1156](https://github.com/kataras/iris/pull/1156) ([@fduxiao](https://github.com/fduxiao))

- add https://github.com/snowlyg/IrisApiProject to starter kits (Chinese). [ea12533871253afc34e40e36ba658b51955ea82d](https://github.com/kataras/iris/commit/ea12533871253afc34e40e36ba658b51955ea82d)

- add https://github.com/yz124/superstar to starter kits (Chinese). [0e734ff8445f07482c28881347c1e564dc5aab9c](https://github.com/kataras/iris/commit/0e734ff8445f07482c28881347c1e564dc5aab9c)

# Su, 18 November 2018 | v11.1.0

PR: https://github.com/kataras/iris/pull/1130

This release contains a new feature for versioning your Iris APIs. The initial motivation and feature request came by https://github.com/kataras/iris/issues/1129.

The [versioning](https://github.com/kataras/iris/tree/master/versioning) package provides [semver](https://semver.org/) versioning for your APIs. It implements all the suggestions written at [api-guidelines](https://github.com/byrondover/api-guidelines/blob/master/Guidelines.md#versioning) and more.


The version comparison is done by the [go-version](https://github.com/hashicorp/go-version) package. It supports matching over patterns like `">= 1.0, < 3"` and etc.

## Features

- per route version matching, a normal iris handler with "switch" cases via Map for version => handler
- per group versioned routes and deprecation API
- version matching like ">= 1.0, < 2.0" or just "2.0.1" and etc.
- version not found handler (can be customized by simply adding the versioning.NotFound: customNotMatchVersionHandler on the Map)
- version is retrieved from the "Accept" and "Accept-Version" headers (can be customized via middleware)
- respond with "X-API-Version" header, if version found.
- deprecation options with customizable "X-API-Warn", "X-API-Deprecation-Date", "X-API-Deprecation-Info" headers via `Deprecated` wrapper.

## Get version

Current request version is retrieved by `versioning.GetVersion(ctx)`.

By default the `GetVersion` will try to read from:
- `Accept` header, i.e `Accept: "application/json; version=1.0"`
- `Accept-Version` header, i.e `Accept-Version: "1.0"`

You can also set a custom version for a handler via a middleware by using the context's store values.
For example:
```go
func(ctx iris.Context) {
    ctx.Values().Set(versioning.Key, ctx.URLParamDefault("version", "1.0"))
    ctx.Next()
}
```

## Match version to handler

The `versioning.NewMatcher(versioning.Map) iris.Handler` creates a single handler which decides what handler need to be executed based on the requested version.

```go
app := iris.New()

// middleware for all versions.
myMiddleware := func(ctx iris.Context) {
    // [...]
    ctx.Next()
}

myCustomNotVersionFound := func(ctx iris.Context) {
    ctx.StatusCode(404)
    ctx.Writef("%s version not found", versioning.GetVersion(ctx))
}

userAPI := app.Party("/api/user")
userAPI.Get("/", myMiddleware, versioning.NewMatcher(versioning.Map{
    "1.0":               sendHandler(v10Response),
    ">= 2, < 3":         sendHandler(v2Response),
    versioning.NotFound: myCustomNotVersionFound,
}))
```

### Deprecation

Using the `versioning.Deprecated(handler iris.Handler, options versioning.DeprecationOptions) iris.Handler` function you can mark a specific handler version as deprecated.


```go
v10Handler := versioning.Deprecated(sendHandler(v10Response), versioning.DeprecationOptions{
    // if empty defaults to: "WARNING! You are using a deprecated version of this API."
    WarnMessage string 
    DeprecationDate time.Time
    DeprecationInfo string
})

userAPI.Get("/", versioning.NewMatcher(versioning.Map{
    "1.0": v10Handler,
    // [...]
}))
```

This will make the handler to send these headers to the client:

- `"X-API-Warn": options.WarnMessage`
- `"X-API-Deprecation-Date": context.FormatTime(ctx, options.DeprecationDate))`
- `"X-API-Deprecation-Info": options.DeprecationInfo`

> versioning.DefaultDeprecationOptions can be passed instead if you don't care about Date and Info.

## Grouping routes by version

Grouping routes by version is possible as well.

Using the `versioning.NewGroup(version string) *versioning.Group` function you can create a group to register your versioned routes.
The `versioning.RegisterGroups(r iris.Party, versionNotFoundHandler iris.Handler, groups ...*versioning.Group)` must be called in the end in order to register the routes to a specific `Party`.

```go
app := iris.New()

userAPI := app.Party("/api/user")
// [... static serving, middlewares and etc goes here].

userAPIV10 := versioning.NewGroup("1.0")
userAPIV10.Get("/", sendHandler(v10Response))

userAPIV2 := versioning.NewGroup(">= 2, < 3")
userAPIV2.Get("/", sendHandler(v2Response))
userAPIV2.Post("/", sendHandler(v2Response))
userAPIV2.Put("/other", sendHandler(v2Response))

versioning.RegisterGroups(userAPI, versioning.NotFoundHandler, userAPIV10, userAPIV2)
```

> A middleware can be registered to the actual `iris.Party` only, using the methods we learnt above, i.e by using the `versioning.Match` in order to detect what code/handler you want to be executed when "x" or no version is requested.

### Deprecation for Group

Just call the `Deprecated(versioning.DeprecationOptions)` on the group you want to notify your API consumers that this specific version is deprecated.

```go
userAPIV10 := versioning.NewGroup("1.0").Deprecated(versioning.DefaultDeprecationOptions)
```

## Compare version manually from inside your handlers

```go
// reports if the "version" is matching to the "is".
// the "is" can be a constraint like ">= 1, < 3".
If(version string, is string) bool
```

```go
// same as `If` but expects a Context to read the requested version.
Match(ctx iris.Context, expectedVersion string) bool
```

```go
app.Get("/api/user", func(ctx iris.Context) {
    if versioning.Match(ctx, ">= 2.2.3") {
        // [logic for >= 2.2.3 version of your handler goes here]
        return
    }
})
```

Example can be found [here](_examples/versioning/main.go).

# Fr, 09 November 2018 | v11.0.4

Add `Configuration.DisablePathCorrectionRedirection` - `iris.WithoutPathCorrectionRedirection` to support
direct handler execution of the matching route without the last `'/'` instead of sending a redirect response when `DisablePathCorrection` is set to false(default behavior).

Usage:

For example, CORS needs the allow origin headers in redirect response as well,
however is not possible from the router to know what headers a route's handler will send to the client.
So the best option we have is to just execute the handler itself instead of sending a redirect response.
Add the `app.Run(..., iris.WithoutPathCorrectionRedirection)` on the server side if you wish
to directly fire the handler instead of redirection (which is the default behavior)
on request paths like `"$yourdomain/v1/mailer/"` when `"/v1/mailer"` route handler is registered.

Example Code:

```go
package main

import "github.com/kataras/iris"


func main() {
    app := iris.New()

    crs := func(ctx iris.Context) {
        ctx.Header("Access-Control-Allow-Origin", "*")
        ctx.Header("Access-Control-Allow-Credentials", "true")
        ctx.Header("Access-Control-Allow-Headers",
            "Access-Control-Allow-Origin,Content-Type")
        ctx.Next()
    }

    v1 := app.Party("/api/v1", crs).AllowMethods(iris.MethodOptions)
    {
        v1.Post("/mailer", func(ctx iris.Context) {
            var any iris.Map
            err := ctx.ReadJSON(&any)
            if err != nil {
                ctx.WriteString(err.Error())
                ctx.StatusCode(iris.StatusBadRequest)
                return
            }
            ctx.Application().Logger().Infof("received %#+v", any)
        })
    }

    //                        HERE:
    app.Run(iris.Addr(":80"), iris.WithoutPathCorrectionRedirection)
}
```

# Tu, 06 November 2018 | v11.0.3

- add "part" html view engine's tmpl function: [15bb55d](https://github.com/kataras/iris/commit/15bb55d85eac378bbe0c98c10ffea938cc05fe4d)

- update pug engine's vendor: [c20bc3b](https://github.com/kataras/iris/commit/c20bc3bceef158ef99931e609123fa0aca2a918c)

# Tu, 30 October 2018 | v11.0.2

Fix [memstore](core/memstore/memstore.go) overflows when build 32 bit app, reported and fixed by [@bouroo](https://github.com/bouroo) at: https://github.com/kataras/iris/issues/1118

# Su, 28 October 2018 | v11.0.1

- Update benchmarks: https://github.com/kataras/iris/commit/d1b47b1ec65ae77a2ca7485e510386f4a5456ac4
- Add link for third-party source benchmarks: https://github.com/kataras/iris/commit/64e80a7ee5c23ed938ddc8b68d181a25420c7653
- Add optionally custom low-level websocket message data prefix as requested at: https://github.com/kataras/iris/issues/1113 by [@jjhesk](https://github.com/jjhesk). Example:

```go
app := iris.New()

// [...]
wsServer := websocket.New(websocket.Config{
    // [...]
    EvtMessagePrefix: []byte("my-custom-prefix:"),
})

// [...]

// serve the javascript built'n client-side library,
// see websockets.html script tags, this path is used.
app.Any("/iris-ws.js", func(ctx iris.Context) {
    ctx.Write(wsServer.ClientSource)
})

// [...]
```

# Su, 21 October 2018 | v11.0.0

For the craziest of us, click [here](https://github.com/kataras/iris/compare/v10.7.0...v11) 🔥 to find out the commits and the code changes since our previous release.

## Breaking changes

- Remove the "Configurator" `WithoutVersionChecker` and the configuration field `DisableVersionChecker`
- `:int` parameter type **can accept negative numbers now**.
- `app.Macros().String/Int/Uint64/Path...RegisterFunc` should be replaced to: `app.Macros().Get("string" or "int" or "uint64" or "path" when "path" is the ":path" parameter type).RegisterFunc`, because you can now add custom macros and parameter types as well, see [here](_examples/routing/macros).
- `RegisterFunc("min", func(paramValue string) bool {...})` should be replaced to `RegisterFunc("min", func(paramValue <T>) bool {...})`, the `paramValue` argument is now stored in the exact type the macro's type evaluator inits it, i.e `uint64` or `int` and so on, therefore you don't have to convert the parameter value each time (this should make your handlers with macro functions activated even faster now) 
- The `Context#ReadForm` will no longer return an error if it has no value to read from the request, we let those checks to the caller and validators as requested at: https://github.com/kataras/iris/issues/1095 by [@haritsfahreza](https://github.com/haritsfahreza)

## Routing

I wrote a [new router implementation](https://github.com/kataras/muxie#philosophy) for our Iris internal(low-level) routing mechanism, it is good to know that this was the second time we have updated the router internals without a single breaking change after the v6, thanks to the very well-written and designed-first code we have for the high-level path syntax component called [macro interpreter](macro/interpreter).

The new router supports things like **closest wildcard resolution**.

> If the name doesn't sound good to you it is because I named that feature myself, I don't know any other framework or router that supports a thing like that so be gentle:)

Previously you couldn't register routes like: `/{myparam:path}` and `/static` and `/{myparam:string}` and `/{myparam:string}/static` and `/static/{myparam:string}` all in one path prefix without a "decision handler". And generally if you had a wildcard it was possible to add (a single) static part and (a single) named parameter but not without performance cost and limits, why only one? (one is better than nothing: look the Iris' alternatives) We struggle to overcome our own selves, now you **can definitely do it without a bit of performance cost**, and surely we hand't imagine the wildcard to **catch all if nothing else found** without huge routing performance cost, the wildcard(`:path`) meant ONLY: "accept one or more path segments and put them into the declared parameter" so if you had register a dynamic single-path-segment named parameter like `:string, :int, :uint, :alphabetical...` in between those path segments it wouldn't work. The **closest wildcard resolution** offers you the opportunity to design your APIs even better via custom handlers and error handlers like `404 not found` to path prefixes for your API's groups, now you can do it without any custom code for path resolution inside a "decision handler" or a middleware.

Code worths 1000 words, now it is possible to define your routes like this without any issues:

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/context"
)

func main() {
    app := iris.New()

    // matches everyhing if nothing else found,
    // so you can use it for custom 404 root-level/main pages!
    app.Get("/{p:path}", func(ctx context.Context) {
        path := ctx.Params().Get("p")
        // gives the path without the first "/".
        ctx.Writef("Site Custom 404 Error Message\nPage of: '%s' not found", path)
    })

    app.Get("/", indexHandler)

    // request: http://localhost:8080/profile
    // response: "Profile Index"
    app.Get("/profile", func(ctx context.Context) {
        ctx.Writef("Profile Index")
    })

    // request: http://localhost:8080/profile/kataras
    // response: "Profile of username: 'kataras'"
    app.Get("/profile/{username}", func(ctx context.Context) {
        username := ctx.Params().Get("username")
        ctx.Writef("Profile of username: '%s'", username)
    })

    // request: http://localhost:8080/profile/settings
    // response: "Profile personal settings"
    app.Get("/profile/settings", func(ctx context.Context) {
        ctx.Writef("Profile personal settings")
    })

    // request: http://localhost:8080/profile/settings/security
    // response: "Profile personal security settings"
    app.Get("/profile/settings/security", func(ctx context.Context) {
        ctx.Writef("Profile personal security settings")
    })

    // matches everyhing /profile/*somethng_here*
    // if no other route matches the path semgnet after the
    // /profile or /profile/
    //
    // So, you can use it for custom 404 profile pages
    // side-by-side to your root wildcard without issues!
    // For example:
    // request: http://localhost:8080/profile/kataras/what
    // response:
    // Profile Page Custom 404 Error Message
    // Profile Page of: 'kataras/what' was unable to be found
    app.Get("/profile/{p:path}", func(ctx context.Context) {
        path := ctx.Params().Get("p")
        ctx.Writef("Profile Page Custom 404 Error Message\nProfile Page of: '%s' not found", path)
    })

    app.Run(iris.Addr(":8080"))
}

func indexHandler(ctx context.Context) {
    ctx.HTML("This is the <strong>index page</strong>")
}

``` 

The `github.com/kataras/iris/core/router.AllMethods` is now a variable that can be altered by end-developers, so things like `app.Any` can register to custom methods as well, as requested at: https://github.com/kataras/iris/issues/1102. For example, import that package and do `router.AllMethods = append(router.AllMethods, "LINK")` in your `main` or `init` function.

The old `github.com/kataras/iris/core/router/macro` package was moved to `guthub.com/kataras/iris/macro` to allow end-developers to add custom parameter types and macros, it supports all go standard types by default as you will see below.

- `:int` parameter type as an alias to the old `:int` which can accept any numeric path segment now, both negative and positive numbers
- Add `:int8` parameter type and `ctx.Params().GetInt8`
- Add `:int16` parameter type and `ctx.Params().GetInt16`
- Add `:int32` parameter type and `ctx.Params().GetInt32`
- Add `:int64` parameter type and `ctx.Params().GetInt64`
- Add `:uint` parameter type and `ctx.Params().GetUint`
- Add `:uint8` parameter type and `ctx.Params().GetUint8`
- Add `:uint16` parameter type and `ctx.Params().GetUint16`
- Add `:uint32` parameter type and `ctx.Params().GetUint32`
- Add `:uint64` parameter type and `ctx.Params().GetUint64`
- Add alias `:bool` for the `:boolean` parameter type

Here is the full list of the built'n parameter types that we support now, including their validations/path segment rules.

| Param Type | Go Type | Validation | Retrieve Helper |
| -----------------|------|-------------|------|
| `:string` | string | the default if param type is missing, anything (single path segment) | `Params().Get` |
| `:int` | int | -9223372036854775808 to 9223372036854775807 (x64) or -2147483648 to 2147483647 (x32), depends on the host arch | `Params().GetInt` |
| `:int8` | int8 | -128 to 127 | `Params().GetInt8` |
| `:int16` | int16 | -32768 to 32767 | `Params().GetInt16` |
| `:int32` | int32 | -2147483648 to 2147483647 | `Params().GetInt32` |
| `:int64` | int64 | -9223372036854775808 to 9223372036854775807 | `Params().GetInt64` |
| `:uint` | uint | 0 to 18446744073709551615 (x64) or 0 to 4294967295 (x32), depends on the host arch | `Params().GetUint` |
| `:uint8` | uint8 | 0 to 255 | `Params().GetUint8` |
| `:uint16` | uint16 | 0 to 65535 | `Params().GetUint16` |
| `:uint32` | uint32 | 0 to 4294967295 | `Params().GetUint32` |
| `:uint64` | uint64 | 0 to 18446744073709551615 | `Params().GetUint64` |
| `:bool` | bool | "1" or "t" or "T" or "TRUE" or "true" or "True" or "0" or "f" or "F" or "FALSE" or "false" or "False" | `Params().GetBool` |
| `:alphabetical` | string | lowercase or uppercase letters | `Params().Get` |
| `:file` | string | lowercase or uppercase letters, numbers, underscore (_), dash (-), point (.) and no spaces or other special characters that are not valid for filenames | `Params().Get` |
| `:path` | string | anything, can be separated by slashes (path segments) but should be the last part of the route path | `Params().Get` | 

**Usage**:

```go
app.Get("/users/{id:uint64}", func(ctx iris.Context){
    id, _ := ctx.Params().GetUint64("id")
    // [...]
})
```

| Built'n Func | Param Types |
| -----------|---------------|
| `regexp`(expr string) | :string |
| `prefix`(prefix string) | :string |
| `suffix`(suffix string) | :string |
| `contains`(s string) | :string |
| `min`(minValue int or int8 or int16 or int32 or int64 or uint8 or uint16 or uint32 or uint64  or float32 or float64) | :string(char length), :int, :int8, :int16, :int32, :int64, :uint, :uint8, :uint16, :uint32, :uint64  |
| `max`(maxValue int or int8 or int16 or int32 or int64 or uint8 or uint16 or uint32 or uint64 or float32 or float64) | :string(char length), :int, :int8, :int16, :int32, :int64, :uint, :uint8, :uint16, :uint32, :uint64 |
| `range`(minValue, maxValue int or int8 or int16 or int32 or int64 or uint8 or uint16 or uint32 or uint64 or float32 or float64) | :int, :int8, :int16, :int32, :int64, :uint, :uint8, :uint16, :uint32, :uint64 |

**Usage**:

```go
app.Get("/profile/{name:alphabetical max(255)}", func(ctx iris.Context){
    name := ctx.Params().Get("name")
    // len(name) <=255 otherwise this route will fire 404 Not Found
    // and this handler will not be executed at all.
})
```

## Vendoring

- Rename the vendor `sessions/sessiondb/vendor/...bbolt` from `coreos/bbolt` to `etcd-io/bbolt` and update to v1.3.1, based on [that](https://github.com/etcd-io/bbolt/releases/tag/v1.3.1-etcd.7)
- Update the vendor `sessions/sessiondb/vendor/...badger` to v1.5.3

I believe it is soon to adapt the new [go modules](https://github.com/golang/go/wiki/Modules#table-of-contents) inside Iris, the new `go mod` command may change until go 1.12, it is still an experimental feature.
The [vendor](https://github.com/kataras/iris/tree/master/vendor) folder will be kept until the majority of Go developers get acquainted with the new `go modules`.  The `go.mod` and `go.sum` files will come at `iris v12` (or `go 1.12`), we could do that on this version as well but I don't want to have half-things, versioning should be passed on import path as well and that is a large breaking change to go with it right now, so it will probably have a new path such as `github.com/kataras/iris/v12` based on a `git tag` like every Iris release (we are lucky here because we used semantic versioning from day zero). No folder re-structure inside the root git repository to split versions will ever happen, so backwards-compatibility for older go versions(before go 1.9.3) and iris versions will be not enabled by-default although it's easy for anyone to grab any version from older [releases](https://github.com/kataras/iris/releases) or branch and target that.

# Sat, 11 August 2018 | v10.7.0

I am overjoyed to announce stage 1 of the the Iris Web framework **10.7 stable release is now available**.

Version 10.7.0 is part of the official [releases](https://github.com/kataras/iris/releases).

This release does not contain any breaking changes to existing Iris-based projects built on older versions of Iris. Iris developers can upgrade with absolute safety.

Read below the changes and the improvements to the framework's internals. We also have more examples for beginners in our community.

## New Examples

- [Iris + WebAssemply = 💓](_examples/webassembly/basic/main.go) **compatible only for projects built with go11.beta and above**
- [Server-Sent Events](_examples/http_responsewriter/sse/main.go)
- [Struct Validation on context.ReadJSON](_examples/http_request/read-json-struct-validation/main.go)
- [Extract referrer from "referer" header or URL query parameter](_examples/http_request/extract-referer/main.go)
- [Hero Sessions](_examples/hero/sessions)
- [Yet another dependency injection example with hero](_examples/hero/smart-contract/main.go)
- [Writing an API for the Apache Kafka](_examples/tutorial/api-for-apache-kafka)

> Also, all "sessions" examples have been customized to include the `AllowReclaim: true` option.

## kataras/iris/websocket

- Change connection list from a customized slice to `sync.Map` with: [this](https://github.com/kataras/iris/commit/5f16704f45bedd767527eadf411cf9bc0f8edaee) and [that commit](https://github.com/kataras/iris/commit/16b30e8eed1406c61abc01282120870bd9fa31d8)
- Minify and add the `iris-ws.js` to the famous https://cdnjs.com via [this PR](https://github.com/kataras/iris/pull/1053) made by [Dibyendu Das](https://github.com/dibyendu)

## kataras/iris/core/router

- Add `json` field tags and new functions such as `ChangeMethod`, `SetStatusOffline` and `RestoreStatus` to the `Route` structure, these type of changes to the routes at runtime have effect after the manual call of the `Router/Application.RefreshRouter()` (not recommended but useful for custom Iris web server's remote control panels)
- Add `GetRoutesReadOnly` function to the `APIBuilder` structure

## kataras/iris/context

- Add `GetReferrer`, `GetContentTypeRequested` and `URLParamInt32Default` functions
- Insert `Trace`, `Tmpl` and `MainHandlerName` functions to the `RouteReadOnly` interface
- Add `OnConnectionClose` function listener to fire a callback when the underline tcp connection is closed, extremely useful for SSE or other loop-forever implementations inside a handler -- and `OnClose` which is the same as `OnConnectionClose(myFunc)` and `defer myFunc()` [*](https://github.com/kataras/iris/commit/6898c2f755a0e22aa42e3b1799e29c857777a6f9)

This release contains minor grammar and typo fixes and more meaningful [godoc](https://godoc.org/github.com/kataras/iris) code comments too. 

## Industry

I am glad to announce that Iris has been chosen as the main development kit for eight medium-to-large sized companies and a new very promising India-based startup. I want to thank you once again for the unwavering support and trust you have shown me, especially this year, despite the past unfair rumours and defamation that we suffered by the merciless competition.

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
- fix high path cleaning between `{}` (we already escape those contents at the [interpreter](macro/interpreter) level but some symbols are still removed by the higher-level api builder) , i.e `\\` from the string's macro function `regex` contents as reported at [927](https://github.com/kataras/iris/issues/927) by [commit e85b113476eeefffbc7823297cc63cd152ebddfd](https://github.com/kataras/iris/commit/e85b113476eeefffbc7823297cc63cd152ebddfd)
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