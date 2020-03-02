<!-- # History/Changelog <a href="HISTORY_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a><a href="HISTORY_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a><a href="HISTORY_GR.md"> <img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> -->

# Changelog

### Looking for free and real-time support?

    https://github.com/kataras/iris/issues
    https://chat.iris-go.com

### Looking for previous versions?

    https://github.com/kataras/iris/releases

### Want to be hired?

    https://facebook.com/iris.framework

### Should I upgrade my Iris?

Developers are not forced to upgrade if they don't really need it. Upgrade whenever you feel ready.

**How to upgrade**: Open your command-line and execute this command: `go get github.com/kataras/iris/v12@latest`.

# Next

This release introduces new features and some breaking changes inside the `mvc` and `hero` packages.
The codebase for dependency injection has been simplified a lot (fewer LOCs and easier to read and follow up).

Before this release the `iris.Context` was the only one dependency has been automatically binded to the controller's fields or handler's inputs, now  the standard `"context"` package's `Context` is also automatically binded and all structs that are not mapping to a registered dependency are now automatically resolved to `payload` XML, YAML, Query, Form and JSON dependencies based on the request's `Content-Type` header (defaults to JSON if client didn't specified a content-type).

The new release contains a fresh new and awesome feature....**a function dependency can accept previous registered dependencies and update or return a new value of any type**.

The new implementation is **faster** on both design and serve-time.

The most common scenario from a route to handle is to:
- accept one or more path parameters and request data, a payload
- send back a response, a payload (JSON, XML,...)

The new Iris Dependency Injection feature is about **[33.2% faster](_benchmarks/_internal/README.md#dependency-injection)** than its predecessor on the above case. This drops down even more the performance cost between native handlers and dynamic handlers with dependencies. This reason itself brings us, with safety and performance-wise, to the new `Party.HandleFunc(method, relativePath string, handlersFn ...interface{}) *Route` and `Party.RegisterDependency` method.

Look how clean your codebase can be when using Iris':

```go
package main

import "github.com/kataras/iris/v12"

type (
    testInput struct {
        Email string `json:"email"`
    }

    testOutput struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    }
)

func handler(id int, in testInput) testOutput {
    return testOutput{
        ID:   id,
        Name: in.Email,
    }
}

func main() {
    app := iris.New()
    app.HandleFunc(iris.MethodPost, "/{id:int}", handler)
    app.Listen(":5000", iris.WithOptimizations)
}
```

Your eyes don't lie you. You read well, no `ctx.ReadJSON(&v)` and `ctx.JSON(send)` neither `error` handling are presented. It is a huge relief but don't worry you can still control everything if you ever need, even errors from dependencies. Any error may occur from request-scoped dependencies or your own handler is dispatched through `Party.GetContainer().GetErrorHandler` which defaults to the `hero.DefaultErrorHandler` which sends a `400 Bad Request` response with the error's text as its body contents, you can change it through `Party#OnErrorFunc`. If you want to handle `testInput` otherwise then just add a `Party.RegisterDependency(func(ctx iris.Context) testInput {...})` and you are ready to go. Here is a quick list of the new Party's methods:

```go
// GetContainer returns the DI Container of this Party.
// Use it to manually convert functions or structs(controllers) to a Handler.
//
// See `OnErrorFunc`, `RegisterDependency`, `UseFunc`, `DoneFunc` and `HandleFunc` too.
GetContainer() *hero.Container
```
```go
// OnErrorFunc adds an error handler for this Party's DI Hero Container and its handlers (or controllers).
// The "errorHandler" handles any error may occurred and returned
// during dependencies injection of the Party's hero handlers or from the handlers themselves.
//
// Same as:
// GetContainer().GetErrorHandler = func(ctx iris.Context) hero.ErrorHandler { return errorHandler }
//
// See `RegisterDependency`, `UseFunc`, `DoneFunc` and `HandleFunc` too.
OnErrorFunc(errorHandler func(context.Context, error))
```
```go
// RegisterDependency adds a dependency.
// The value can be a single struct value or a function.
// Follow the rules:
// * <T> {structValue}
// * func(accepts <T>)                                 returns <D> or (<D>, error)
// * func(accepts iris.Context)                        returns <D> or (<D>, error)
// * func(accepts1 iris.Context, accepts2 *hero.Input) returns <D> or (<D>, error)
//
// A Dependency can accept a previous registered dependency and return a new one or the same updated.
// * func(accepts1 <D>, accepts2 <T>)                  returns <E> or (<E>, error) or error
// * func(acceptsPathParameter1 string, id uint64)     returns <T> or (<T>, error)
//
// Usage:
//
// - RegisterDependency(loggerService{prefix: "dev"})
// - RegisterDependency(func(ctx iris.Context) User {...})
// - RegisterDependency(func(User) OtherResponse {...})
//
// See `OnErrorFunc`, `UseFunc`, `DoneFunc` and `HandleFunc` too.
RegisterDependency(dependency interface{})
```
```go
// UseFunc same as "Use" but it accepts dynamic functions as its "handlersFn" input.
// See `OnErrorFunc`, `RegisterDependency`, `DoneFunc` and `HandleFunc` for more.
UseFunc(handlersFn ...interface{})
// DoneFunc same as "Done" but it accepts dynamic functions as its "handlersFn" input.
// See `OnErrorFunc`, `RegisterDependency`, `UseFunc` and `HandleFunc` for more.
DoneFunc(handlersFn ...interface{})
```
```go
// HandleFunc same as `HandleFunc` but it accepts one or more "handlersFn" functions which each one of them
// can accept any input arguments that match with the Party's registered Container's `Dependencies` and
// any output result; like custom structs <T>, string, []byte, int, error,
// a combination of the above, hero.Result(hero.View | hero.Response) and more.
//
// It's common from a hero handler to not even need to accept a `Context`, for that reason,
// the "handlersFn" will call `ctx.Next()` automatically when not called manually.
// To stop the execution and not continue to the next "handlersFn"
// the end-developer should output an error and return `iris.ErrStopExecution`.
//
// See `OnErrorFunc`, `RegisterDependency`, `UseFunc` and `DoneFunc` too.
HandleFunc(method, relativePath string, handlersFn ...interface{}) *Route
```

Other Improvements:

- A result of <T> can implement the new `hero.PreflightResult` interface which contains a single method of `Preflight(iris.Context) error`. If this method exists on a custom struct value which is returned from a handler then it will fire that `Preflight` first and if not errored then it will cotninue by sending the struct value as JSON(by-default) response body.

- `ctx.JSON, JSONP, XML`: if `iris.WithOptimizations` is NOT passed on `app.Run/Listen` then the indentation defaults to `"  "` (two spaces) otherwise it is empty or the provided value.

- Hero Handlers (and `app.HandleFunc`) do not have to require `iris.Context` just to call `ctx.Next()` anymore, this is done automatically now. 

New Context Methods:

- `context.Defer(Handler)` works like `Party.Done` but for the request life-cycle.
- `context.ReflectValue() []reflect.Value` stores and returns the `[]reflect.ValueOf(context)`
- `context.Controller() reflect.Value` returns the current MVC Controller value (when fired from inside a controller's method).

Breaking Changes:

- `var mvc.AutoBinding` removed as the default behavior now resolves such dependencies automatically (see [[FEATURE REQUEST] MVC serving gRPC-compatible controller](https://github.com/kataras/iris/issues/1449))
- `mvc#Application.SortByNumMethods()` removed as the default behavior now binds the "thinnest"  empty `interface{}` automatically (see [MVC: service injecting fails](https://github.com/kataras/iris/issues/1343))
- `mvc#BeforeActivation.Dependencies().Add` should be replaced with `mvc#BeforeActivation.Dependencies().Register` instead.

# Su, 16 February 2020 | v12.1.8

New Features:

-  [[FEATURE REQUEST] MVC serving gRPC-compatible controller](https://github.com/kataras/iris/issues/1449)

Fixes:

- [App can't find embedded pug template files by go-bindata](https://github.com/kataras/iris/issues/1450)

New Examples:

- [_examples/mvc/grpc-compatible](_examples/mvc/grpc-compatible)

# Mo, 10 February 2020 | v12.1.7

Implement **new** `SetRegisterRule(iris.RouteOverride, RouteSkip, RouteError)` to resolve: https://github.com/kataras/iris/issues/1448

New Examples:

- [_examples/Docker](_examples/Docker)
- [_examples/routing/route-register-rule](_examples/routing/route-register-rule)

# We, 05 February 2020 | v12.1.6

Fixes:

- [jet.View - urlpath error](https://github.com/kataras/iris/issues/1438)
- [Context.ServeFile send 'application/wasm' with a wrong extra field](https://github.com/kataras/iris/issues/1440)

# Su, 02 February 2020 | v12.1.5

Various improvements and linting.

# Su, 29 December 2019 | v12.1.4

Minor fix on serving [embedded files](https://github.com/kataras/iris/wiki/File-server).

# We, 25 December 2019 | v12.1.3

Fix [[BUG] [iris.Default] RegisterView](https://github.com/kataras/iris/issues/1410)

# Th, 19 December 2019 | v12.1.2

Fix [[BUG]Session works incorrectly when meets the multi-level TLDs](https://github.com/kataras/iris/issues/1407).

# Mo, 16 December 2019 | v12.1.1

Add [Context.FindClosest(n int) []string](https://github.com/kataras/iris/blob/master/_examples/routing/not-found-suggests/main.go#L22)

```go
app := iris.New()
app.OnErrorCode(iris.StatusNotFound, notFound)
```

```go
func notFound(ctx iris.Context) {
    suggestPaths := ctx.FindClosest(3)
    if len(suggestPaths) == 0 {
        ctx.WriteString("404 not found")
        return
    }

    ctx.HTML("Did you mean?<ul>")
    for _, s := range suggestPaths {
        ctx.HTML(`<li><a href="%s">%s</a></li>`, s, s)
    }
    ctx.HTML("</ul>")
}
```

![](https://iris-go.com/images/iris-not-found-suggests.png)

# Fr, 13 December 2019 | v12.1.0

## Breaking Changes

Minor as many of you don't even use them but, indeed, they need to be covered here.

- Old i18n middleware(iris/middleware/i18n) was replaced by the [i18n](i18n) sub-package which lives as field at your application: `app.I18n.Load(globPathPattern string, languages ...string)` (see below)
- Community-driven i18n middleware(iris-contrib/middleware/go-i18n) has a `NewLoader` function which returns a loader which can be passed at `app.I18n.Reset(loader i18n.Loader, languages ...string)` to change the locales parser
- The Configuration's `TranslateFunctionContextKey` was replaced by `LocaleContextKey` which Context store's value (if i18n is used) returns the current Locale which contains the translate function, the language code, the language tag and the index position of it
- The `context.Translate` method was replaced by `context.Tr` as a shortcut for the new `context.GetLocale().GetMessage(format, args...)` method and it matches the view's function `{{tr format args}}` too
- If you used [Iris Django](https://github.com/kataras/iris/tree/master/_examples/view/template_django_0) view engine with `import _ github.com/flosch/pongo2-addons` you **must change** the import path to `_ github.com/iris-contrib/pongo2-addons` or add a [go mod replace](https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive) to your `go.mod` file, e.g. `replace github.com/flosch/pongo2-addons => github.com/iris-contrib/pongo2-addons v0.0.1`.

## Fixes

All known issues.

1. [#1395](https://github.com/kataras/iris/issues/1395) 
2. [#1369](https://github.com/kataras/iris/issues/1369)
3. [#1399](https://github.com/kataras/iris/issues/1399) with PR [#1400](https://github.com/kataras/iris/pull/1400)
4. [#1401](https://github.com/kataras/iris/issues/1401) 
5. [#1406](https://github.com/kataras/iris/issues/1406)
6. [neffos/#20](https://github.com/kataras/neffos/issues/20)
7. [pio/#5](https://github.com/kataras/pio/issues/5)

## New Features

### Internationalization and localization

Support for i18n is now a **builtin feature** and is being respected across your entire application, per say [sitemap](https://github.com/kataras/iris/wiki/Sitemap) and [views](https://github.com/kataras/iris/blob/master/_examples/i18n/main.go#L50).

Refer to the wiki section: https://github.com/kataras/iris/wiki/Sitemap for details.

### Sitemaps

Iris generates and serves one or more [sitemap.xml](https://www.sitemaps.org/protocol.html) for your static routes.

Navigate through: https://github.com/kataras/iris/wiki/Sitemap for more.

## New Examples

2. [_examples/i18n](_examples/i18n)
1. [_examples/sitemap](_examples/sitemap)
3. [_examples/desktop-app/blink](_examples/desktop-app/blink)
4. [_examples/desktop-app/lorca](_examples/desktop-app/lorca)
5. [_examples/desktop-app/webview](_examples/desktop-app/webview)

# Sa, 26 October 2019 | v12.0.0

- Add version suffix of the **import path**, learn why and see what people voted at [issue #1370](https://github.com/kataras/iris/issues/1370)

![](https://iris-go.com/images/vote-v12-version-suffix_26_oct_2019.png)

- All errors are now compatible with go1.13 `errors.Is`, `errors.As` and `fmt.Errorf` and a new `core/errgroup` package created
- Fix [#1383](https://github.com/kataras/iris/issues/1383)
- Report whether system couldn't find the directory of view templates
- Remove the `Party#GetReport` method, keep `Party#GetReporter` which is an `error` and an `errgroup.Group`.
- Remove the router's deprecated methods such as StaticWeb and StaticEmbedded_XXX
- The `Context#CheckIfModifiedSince` now returns an `context.ErrPreconditionFailed` type of error when client conditions are not met. Usage: `if errors.Is(err, context.ErrPreconditionFailed) { ... }`
- Add `SourceFileName` and `SourceLineNumber` to the `Route`, reports the exact position of its registration inside your project's source code.
- Fix a bug about the MVC package route binding, see [PR #1364](https://github.com/kataras/iris/pull/1364)
- Add `mvc/Application#SortByNumMethods` as requested at [#1343](https://github.com/kataras/iris/issues/1343#issuecomment-524868164)
- Add status code `103 Early Hints`
- Fix performance of session.UpdateExpiration on 200 thousands+ keys with new radix as reported at [issue #1328](https://github.com/kataras/iris/issues/1328)
- New redis session database configuration field: `Driver: redis.Redigo()` or `redis.Radix()`, see [updated examples](_examples/sessions/database/redis/)
- Add Clusters support for redis:radix session database (`Driver: redis:Radix()`) as requested at [issue #1339](https://github.com/kataras/iris/issues/1339)
- Create Iranian [README_FA](README_FA.md) translation with [PR #1360](https://github.com/kataras/iris/pull/1360) 
- Create Korean [README_KO](README_KO.md) translation with [PR #1356](https://github.com/kataras/iris/pull/1356)
- Create Spanish [README_ES](README_ES.md) and [HISTORY_ES](HISTORY_ES.md) translations with [PR #1344](https://github.com/kataras/iris/pull/1344).

The iris-contrib/middleare and examples are updated to use the new `github.com/kataras/iris/v12` import path.

# Fr, 16 August 2019 | v11.2.8

- Set `Cookie.SameSite` to `Lax` when subdomains sessions share is enabled[*](https://github.com/kataras/iris/commit/6bbdd3db9139f9038641ce6f00f7b4bab6e62550)
- Add and update all [experimental handlers](https://github.com/kataras/iris/tree/master/_examples/experimental-handlers) 
- New `XMLMap` function which wraps a `map[string]interface{}` and converts it to a valid xml content to render through `Context.XML` method
- Add new `ProblemOptions.XML` and `RenderXML` fields to render the `Problem` as XML(application/problem+xml) instead of JSON("application/problem+json) and enrich the `Negotiate` to easily accept the `application/problem+xml` mime.

Commit log: https://github.com/kataras/iris/compare/v11.2.7...v11.2.8

# Th, 15 August 2019 | v11.2.7

This minor version contains improvements on the Problem Details for HTTP APIs implemented on [v11.2.5](#mo-12-august-2019--v1125).

- Fix https://github.com/kataras/iris/issues/1335#issuecomment-521319721
- Add `ProblemOptions` with `RetryAfter` as requested at: https://github.com/kataras/iris/issues/1335#issuecomment-521330994.
- Add `iris.JSON` alias for `context#JSON` options type.

[Example](https://github.com/kataras/iris/blob/45d7c6fedb5adaef22b9730592255f7bb375e809/_examples/routing/http-errors/main.go#L85) and [wikis](https://github.com/kataras/iris/wiki/Routing-error-handlers#the-problem-type) updated. 

References:

- https://tools.ietf.org/html/rfc7231#section-7.1.3
- https://tools.ietf.org/html/rfc7807

Commit log: https://github.com/kataras/iris/compare/v11.2.6...v11.2.7

# We, 14 August 2019 | v11.2.6

Allow [handle more than one route with the same paths and parameter types but different macro validation functions](https://github.com/kataras/iris/issues/1058#issuecomment-521110639).

```go
app.Get("/{alias:string regexp(^[a-z0-9]{1,10}\\.xml$)}", PanoXML)
app.Get("/{alias:string regexp(^[a-z0-9]{1,10}$)}", Tour)
```

Commit log: https://github.com/kataras/iris/compare/v11.2.5...v11.2.6

# Mo, 12 August 2019 | v11.2.5

- [New Feature: Problem Details for HTTP APIs](https://github.com/kataras/iris/pull/1336)
- [Add Context.AbsoluteURI](https://github.com/kataras/iris/pull/1336/files#diff-15cce7299aae8810bcab9b0bf9a2fdb1R2368)

Commit log: https://github.com/kataras/iris/compare/v11.2.4...v11.2.5

# Fr, 09 August 2019 | v11.2.4

- Fixes [iris.Jet: no view engine found for '.jet' or '.html'](https://github.com/kataras/iris/issues/1327)
- Fixes [ctx.ViewData not work with JetEngine](https://github.com/kataras/iris/issues/1330)
- **New Feature**: [HTTP Method Override](https://github.com/kataras/iris/issues/1325)
- Fixes [Poor performance of session.UpdateExpiration on 200 thousands+ keys with new radix lib](https://github.com/kataras/iris/issues/1328) by introducing the `sessions.Config.Driver` configuration field which defaults to `Redigo()` but can be set to `Radix()` too, future additions are welcomed.

Commit log: https://github.com/kataras/iris/compare/v11.2.3...v11.2.4

# Tu, 30 July 2019 | v11.2.3

- [New Feature: Handle different parameter types in the same path](https://github.com/kataras/iris/issues/1315)
- [New Feature: Content Negotiation](https://github.com/kataras/iris/issues/1319)
- [Context.ReadYAML](https://github.com/kataras/iris/tree/master/_examples/http_request/read-yaml)
- Fixes https://github.com/kataras/neffos/issues/1#issuecomment-515698536

# We, 24 July 2019 | v11.2.2

Sessions as middleware:

```go
import "github.com/kataras/iris/v12/sessions"
// [...]

app := iris.New()
sess := sessions.New(sessions.Config{...})

app.Get("/path", func(ctx iris.Context){
    session := sessions.Get(ctx)
    // [work with session...]
})
```

- Add `Session.Len() int` to return the total number of stored values/entries.
- Make `Context.HTML` and `Context.Text` to accept an optional, variadic, `args ...interface{}` input arg(s) too.

## v11.1.1

- https://github.com/kataras/iris/issues/1298
- https://github.com/kataras/iris/issues/1207

# Tu, 23 July 2019 | v11.2.0

Read about the new release at: https://www.facebook.com/iris.framework/posts/3276606095684693
