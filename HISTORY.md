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

**How to upgrade**: Open your command-line and execute this command: `go get github.com/kataras/iris/v12@latest` and `go mod tidy -compat=1.21`.


# Next

Changes apply to `main` branch.

- New `Application/Party.MiddlewareExists(handlerNameOrHandler)` method added, example:

```go
package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"
)

func main() {
	app := iris.New()

	app.UseRouter(errors.RecoveryHandler)

	if app.MiddlewareExists(errors.RecoveryHandler) { // <- HERE.
		fmt.Println("errors.RecoveryHandler exists")
	}
	// OR:
	// if app.MiddlewareExists("iris.errors.recover") {
	// 	fmt.Println("Iris.Errors.Recovery exists")
	// }

	app.Get("/", indexHandler)

	app.Listen(":8080")
}

func indexHandler(ctx iris.Context) {
	panic("an error here")
	ctx.HTML("<h1>Index</h1>")
}

```
- New `x/errors.Intercept(func(ctx iris.Context, req *CreateRequest, resp *CreateResponse) error{ ... })` package-level function.

```go
func main() {
    app := iris.New()

    // Create a new service and pass it to the handlers.
    service := new(myService)

    app.Post("/", errors.Intercept(responseHandler), errors.CreateHandler(service.Create))

    // [...]
}

func responseHandler(ctx iris.Context, req *CreateRequest, resp *CreateResponse) error {
    fmt.Printf("intercept: request got: %+v\nresponse sent: %#+v\n", req, resp)
    return nil
}
```

- Rename `x/errors/ContextValidator.ValidateContext(iris.Context) error` to `x/errors/RequestHandler.HandleRequest(iris.Context) error`.

# Thu, 18 Jan 2024 | v12.2.10

- Simplify the `/core/host` subpackage and remove its `DeferFlow` and `RestoreFlow` methods. These methods are replaced with: `Supervisor.Configure(host.NonBlocking())` before `Serve` and ` Supervisor.Wait(context.Context) error` after `Serve`.
- Fix internal `trimHandlerName` and other minor stuff.
- New `iris.NonBlocking()` configuration option to run the server without blocking the main routine, `Application.Wait(context.Context) error` method can be used to block and wait for the server to be up and running. Example:

```go
func main() {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) {
        ctx.Writef("Hello, %s!", "World")
    })

    app.Listen(":8080", iris.NonBlocking(), iris.WithoutServerError(iris.ErrServerClosed))

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := app.Wait(ctx); err != nil {
        log.Fatal(err)
    }

    // [Server is up and running now, you may continue with other functions below].
}
```

- Add `x/mathx.RoundToInteger` math helper function.

# Wed, 10 Jan 2024 | v12.2.9

- Add `x/errors.RecoveryHandler` package-level function.
- Add `x/errors.Validation` package-level function to add one or more validations for the request payload before a service call of the below methods.
- Add `x/errors.Handler`, `CreateHandler`, `NoContentHandler`, `NoContentOrNotModifiedHandler` and `ListHandler` ready-to-use handlers for service method calls to Iris Handler.
- Add `x/errors.List` package-level function to support `ListObjects(ctx context.Context, opts pagination.ListOptions, f Filter) ([]Object, int64, error)` type of service calls.
- Simplify how validation errors on `/x/errors` package works. A new `x/errors/validation` sub-package added to make your life easier (using the powerful Generics feature).
- Add `x/errors.OK`, `Create`, `NoContent` and `NoContentOrNotModified` package-level generic functions as custom service method caller helpers. Example can be found [here](_examples/routing/http-wire-errors/service/main.go).
- Add `x/errors.ReadPayload`, `ReadQuery`, `ReadPaginationOptions`, `Handle`, `HandleCreate`, `HandleCreateResponse`, `HandleUpdate` and `HandleDelete` package-level functions as helpers for common actions.
- Add `x/jsonx.GetSimpleDateRange(date, jsonx.WeekRange, time.Monday, time.Sunday)` which returns all dates between the given range and start/end weekday values for WeekRange.
- Add `x/timex.GetMonthDays` and `x/timex.GetMonthEnd` functions.
- Add `iris.CookieDomain` and `iris.CookieOverride` cookie options to handle [#2309](https://github.com/kataras/iris/issues/2309).
- New `x/errors.ErrorCodeName.MapErrorFunc`, `MapErrors`, `Wrap` methods and `x/errors.HandleError` package-level function.

# Sun, 05 Nov 2023 | v12.2.8

- A new way to customize the handler's parameter among with the `hero` and `mvc` packages. New `iris.NewContextWrapper` and
 `iris.NewContextPool` methods were added to wrap a handler (`.Handler`, `.Handlers`, `.HandlerReturnError`, `HandlerReturnDuration`, `Filter` and `FallbackViewFunc` methods) and use a custom context instead of the iris.Context directly. Example at: https://github.com/kataras/iris/tree/main/_examples/routing/custom-context.

- The `cache` sub-package has an update, 4 years after:

    - Add support for custom storage on `cache` package, through the `Handler#Store` method.
    - Add support for custom expiration duration on `cache` package, trough the `Handler#MaxAge` method.
    - Improve the overral performance of the `cache` package.
    - The `cache.Handler` input and output arguments remain as it is.
    - The `cache.Cache` input argument changed from `time.Duration` to `func(iris.Context) time.Duration`.

# Mon, 25 Sep 2023 | v12.2.7

Minor bug fixes and support of multiple `block` and `define` directives in multiple layouts and templates in the `Blocks` view engine.

# Mon, 21 Aug 2023 | v12.2.5

- Add optional `Singleton() bool` method to controllers to mark them as singleton, will panic with a specific error if a controller expects dynamic dependencies. This behavior is idendical to the app-driven `app.EnsureStaticBindings()`.

- Non-zero fields of a controller that are marked as ignored, with `ignore:"true"` field tag, they are not included in the dependencies at all now.

- Re-add error log on context rich write (e.g. JSON) failures when the application is running under debug mode (with `app.Logger().SetLevel("debug")`) and there is no a registered context error handler at place.

- `master` branch finally renamed to `main`. Don't worry GitHub will still navigate any `master` request to `main` automatically. Examples, Documentation and other Pages are refactored too.

# Sat, 12 Aug 2023 | v12.2.4

- Add new `iris.WithDynamicHandler` option (`EnableDynamicHandler` setting) to work with `iris.Application.RefreshRouter` method. It allows to change the entire router while your server is up and running. Handles [issue #2167](https://github.com/kataras/iris/issues/2167). Example at [_examples/routing/route-state/main.go](_examples/routing/route-state/main.go).

> We jumped v12.2.2 and v12.2.3.

# Mon, 17 July 2023 | v12.2.1

- Add `mvc.Application.EnableStructDependents()` method to handle [#2158](https://github.com/kataras/iris/issues/2158).

- Fix [iris-premium#17](https://github.com/kataras/iris-premium/issues/17).

- Replace [russross/blackfriday](github.com/russross/blackfriday/v2) with [gomarkdown](https://github.com/gomarkdown/markdown) as requested at [#2098](https://github.com/kataras/iris/issues/2098).

- Add `mvc.IgnoreEmbedded` option to handle [#2103](https://github.com/kataras/iris/issues/2103). Example Code:

```go
func configure(m *mvc.Application) {
	m.Router.Use(cacheHandler)
	m.Handle(&exampleController{
		timeFormat: "Mon, Jan 02 2006 15:04:05",
	}, mvc.IgnoreEmbedded /* BaseController.GetDoSomething will not be parsed at all */)
}

type BaseController struct {
	Ctx iris.Context
}

func (c *BaseController) GetDoSomething(i interface{}) error {
	return nil
}

type exampleController struct {
	BaseController

	timeFormat string
}

func (c *exampleController) Get() string {
	now := time.Now().Format(c.timeFormat)
	return "last time executed without cache: " + now
}
```

- Add `LoadKV` method on `Iris.Application.I18N` instance. It should be used when no locale files are available. It loads locales via pure Go Map (or database decoded values).

- Remove [ace](https://github.com/eknkc/amber) template parser support, as it was discontinued by its author more than five years ago.

# Sa, 11 March 2023 | v12.2.0

This release introduces new features and some breaking changes.
The codebase for Dependency Injection, Internationalization and localization and more have been simplified a lot (fewer LOCs and easier to read and follow up).

## 24 Dec 2022

All new features have been tested in production and seem to work fine. Fixed all reported and reproducible bugs. The `v12.2.0-beta7` is the latest beta release of v12.2.0. Expect the final public and stable release of v12.2.0 shortly after February 2023.

## Fixes and Improvements
 
- Add `iris.TrimParamFilePart` to handle cases like [#2024](https://github.com/kataras/iris/issues/2024) and improve the [_examples/routing/dynamic-path/main.go](_examples/routing/dynamic-path/main.go#L356) example to include that case as well.

- **Breaking-change**: HTML template functions `yield`, `part`, `partial`, `partial_r` and `render` now accept (and require for some cases) a second argument of the binding data context too. Convert: `{{ yield }}` to `{{ yield . }}`, `{{ render "templates/mytemplate.html" }}` to `{{ render "templates/mytemplate.html" . }}`, `{{ partial "partials/mypartial.html" }}` to `{{ partial "partials/mypartial.html" . }}` and so on.

- Add new `URLParamSeparator` to the configuration. Defaults to "," but can be set to an empty string to disable splitting query values on `Context.URLParamSlice` method.

- [PR #1992](https://github.com/kataras/iris/pull/1992): Added support for third party packages on [httptest](https://github.com/kataras/iris/tree/main/httptest). An example using 3rd-party module named [Ginkgo](github.com/onsi/ginkgo) can be found [here](https://github.com/kataras/iris/blob/main/_examples/testing/ginkgotest). 

- Add `Context.Render` method for compatibility.

- Support of embedded [locale files](https://github.com/kataras/iris/blob/main/_examples/i18n/template-embedded/main.go) using standard `embed.FS` with the new `LoadFS` function.
- Support of direct embedded view engines (`HTML, Blocks, Django, Handlebars, Pug, Jet` and `Ace`) with `embed.FS` or `fs.FS` (in addition to `string` and `http.FileSystem` types).
- Add support for `embed.FS` and `fs.FS` on `app.HandleDir`.

- Add `iris.Patches()` package-level function to customize Iris Request Context REST (and more to come) behavior.
- Minor fixes.

- Enable setting a custom "go-redis" client through `SetClient` go redis driver method or `Client` struct field on sessions/database/redis driver as requested at [chat](https://chat.iris-go.com).
- Ignore `"csrf.token"` form data key when missing on `ctx.ReadForm` by default as requested at [#1941](https://github.com/kataras/iris/issues/1941).

- Fix [CVE-2020-5398](https://github.com/advisories/GHSA-8wx2-9q48-vm9r).

- New `{x:weekday}` path parameter type, example code:

```go
// 0 to 7 (leading zeros don't matter) or "Sunday" to "Monday" or "sunday" to "monday".
// http://localhost:8080/schedule/monday or http://localhost:8080/schedule/Monday or
// http://localhost:8080/schedule/1 or http://localhost:8080/schedule/0001.
app.Get("/schedule/{day:weekday}", func(ctx iris.Context) {
    day, _ := ctx.Params().GetWeekday("day")
    ctx.Writef("Weekday requested was: %v\n", day)
})
```

- Make the `Context.JSON` method customizable by modifying the `context.WriteJSON` package-level function.
- Add new `iris.NewGuide` which helps you build a simple and nice JSON API with services as dependencies and better design pattern.
- Make `Context.Domain()` customizable by letting developers to modify the `Context.GetDomain` package-level function.
- Remove Request Context-based Transaction feature as its usage can be replaced with just the Iris Context (as of go1.7+) and better [project](_examples/project) structure.
- Fix [#1882](https://github.com/kataras/iris/issues/1882)
- Fix [#1877](https://github.com/kataras/iris/issues/1877)
- Fix [#1876](https://github.com/kataras/iris/issues/1876)

- New `date` dynamic path parameter type. E.g. `/blog/{param:date}` matches to `"/blog/2022/04/21"`.

- Add `iris.AllowQuerySemicolons` and `iris.WithoutServerError(iris.ErrURLQuerySemicolon)` to handle golang.org/issue/25192 as reported at: https://github.com/kataras/iris/issues/1875. 
- Add new `Application.SetContextErrorHandler` to globally customize the default behavior (status code 500 without body) on `JSON`, `JSONP`, `Protobuf`, `MsgPack`, `XML`, `YAML` and `Markdown` method call write errors instead of catching the error on each handler.
- Add new [x/pagination](x/pagination/pagination.go) sub-package which supports generics code (go 1.18+).
- Add new [middleware/modrevision](middleware/modrevision) middleware (example at [_examples/project/api/router.go]_examples/project/api/router.go).
- Add `iris.BuildRevision` and `iris.BuildTime` to embrace the new go's 1.18 debug build information.

- ~Add `Context.SetJSONOptions` to customize on a higher level the JSON options on `Context.JSON` calls.~ update: remains as it's, per JSON call.
- Add new [auth](auth) sub-package which helps on any user type auth using JWT (access & refresh tokens) and a cookie (optional).

- Add `Party.EnsureStaticBindings` which, if called, the MVC binder panics if a struct's input binding depends on the HTTP request data instead of a static dependency. This is useful to make sure your API crafted through `Party.PartyConfigure` depends only on struct values you already defined at `Party.RegisterDependency` == will never use reflection at serve-time (maximum performance).

- Add a new [x/sqlx](/x/sqlx/) sub-package ([example](_examples/database/sqlx/main.go)).

- Add a new [x/reflex](/x/reflex) sub-package. 

- Add `Context.ReadMultipartRelated` as requested at: [issues/#1787](https://github.com/kataras/iris/issues/1787).

- Add `Container.DependencyMatcher` and `Dependency.Match` to implement the feature requested at [issues/#1842](https://github.com/kataras/iris/issues/1842).

- Register [CORS middleware](middleware/cors) to the Application by default when `iris.Default()` is used instead of `iris.New()`.

- Add [x/jsonx: DayTime](/x/jsonx/day_time.go) for JSON marshal and unmarshal of "15:04:05" (hour, minute, second).

- Fix a bug of `WithoutBodyConsumptionOnUnmarshal` configurator and a minor dependency injection issue caused by the previous alpha version between 20 and 26 February of 2022.

- New basic [cors middleware](middleware/cors).
- New `httptest.NewServer` helper.
- New [x/errors](x/errors) sub-package, helps with HTTP Wire Errors. Example can be found [here](_examples/routing/http-wire-errors/main.go).

- New [x/timex](x/timex) sub-package, helps working with weekdays.

- Minor improvements to the [JSON Kitchen Time](x/jsonx/kitchen_time.go).
- A session database can now implement the `EndRequest(ctx *context.Context, session *Session)` method which will be fired at the end of the request-response lifecycle. 
- Improvements on JSON and ReadJSON when `Iris.Configuration.EnableOptimizations` is true. The request's Context is used whenever is necessary.
- New [monitor](_examples/monitor/monitor-middleware/main.go) middleware.

- New `RegisterRequestHandler` package-level and client methods to the new `x/client` package. Control or log the request-response lifecycle.
- New `RateLimit` and `Debug` HTTP Client options to the new `x/client` package.

- Push a security fix reported by [Kirill Efimov](https://github.com/kirill89) for older go runtimes.

- New `Configuration.Timeout` and `Configuration.TimeoutMessage` fields. Use it to set HTTP timeouts. Note that your http server's (`Application.ConfigureHost`) Read/Write timeouts should be a bit higher than the `Configuration.Timeout` in order to give some time to http timeout handler to kick in and be able to send the `Configuration.TimeoutMessage` properly.

- New `apps.OnApplicationRegistered` method which listens on new Iris applications hosted under the same binary. Use it on your `init` functions to configure Iris applications by any spot in your project's files.

- `Context.JSON` respects any object implements the `easyjson.Marshaler` interface and renders the result using the [easyjon](https://github.com/mailru/easyjson)'s writer. **Set** the `Configuration.EnableProtoJSON` and `Configuration.EnableEasyJSON` to true in order to enable this feature.

- minor: `Context` structure implements the standard go Context interface now (includes: Deadline, Done, Err and Value methods). Handlers can now just pass the `ctx iris.Context` as a shortcut of `ctx.Request().Context()` when needed.

- New [x/jsonx](x/jsonx) sub-package for JSON type helpers.

- New [x/mathx](x/mathx) sub-package for math related functions.

- New [/x/client](x/client) HTTP Client sub-package.

- New `email` builtin path parameter type. Example:

```go
// +------------------------+
// | {param:email}           |
// +------------------------+
// Email + mx look up path parameter validation. Use it on production.

// http://localhost:8080/user/kataras2006@hotmail.com -> OK
// http://localhost:8080/user/b-c@invalid_domain      -> NOT FOUND
app.Get("/user/{user_email:email}", func(ctx iris.Context) {
    email := ctx.Params().Get("user_email")
    ctx.WriteString(email)
})

// +------------------------+
// | {param:mail}           |
// +------------------------+
// Simple email path parameter validation.

// http://localhost:8080/user/kataras2006@hotmail.com    -> OK
// http://localhost:8080/user/b-c@invalid_domainxxx1.com -> NOT FOUND
app.Get("/user/{local_email:mail}", func(ctx iris.Context) {
    email := ctx.Params().Get("local_email")
    ctx.WriteString(email)
})
```

- New `iris.IsErrEmptyJSON(err) bool` which reports whether the given "err" is caused by a
`Context.ReadJSON` call when the request body didn't start with { (or it was totally empty). 

Example Code:

```go
func handler(ctx iris.Context) {
    var opts SearchOptions
    if err := ctx.ReadJSON(&opts); err != nil && !iris.IsErrEmptyJSON(err) {
        ctx.StopWithJSON(iris.StatusBadRequest, iris.Map{"message": "unable to parse body"})
        return
    }

    // [...continue with default values of "opts" struct if the client didn't provide some]
}
```

That means that the client can optionally set a JSON body.
	
- New `APIContainer.EnableStrictMode(bool)` to disable automatic payload binding and panic on missing dependencies for exported struct'sfields or function's input parameters on MVC controller or hero function or PartyConfigurator.

- New `Party.PartyConfigure(relativePath string, partyReg ...PartyConfigurator) Party` helper, registers a children Party like `Party` and `PartyFunc` but instead it accepts a structure value which may contain one or more of the dependencies registered by `RegisterDependency` or `ConfigureContainer().RegisterDependency` methods and fills the unset/zero exported struct's fields respectfully (useful when the api's dependencies amount are too much to pass on a function).

- **New feature:** add the ability to set custom error handlers on path type parameters errors (existing or custom ones). Example Code:

```go
app.Macros().Get("uuid").HandleError(func(ctx iris.Context, paramIndex int, err error) {
    ctx.StatusCode(iris.StatusBadRequest)

    param := ctx.Params().GetEntryAt(paramIndex)
    ctx.JSON(iris.Map{
        "error":     err.Error(),
        "message":   "invalid path parameter",
        "parameter": param.Key,
        "value":     param.ValueRaw,
    })
})

app.Get("/users/{id:uuid}", getUser)
```

- Improve the performance and fix `:int, :int8, :int16, :int32, :int64, :uint, :uint8, :uint16, :uint32, :uint64` path type parameters couldn't accept a positive number written with the plus symbol or with a leading zeroes, e.g. `+42` and `021`.

- The `iris.WithEmptyFormError` option is respected on `context.ReadQuery` method too, as requested at [#1727](https://github.com/kataras/iris/issues/1727). [Example comments](https://github.com/kataras/iris/blob/main/_examples/request-body/read-query/main.go) were updated.

- New `httptest.Strict` option setter to enable the `httpexpect.RequireReporter` instead of the default `httpexpect.AssetReporter. Use that to enable complete test failure on the first error. As requested at: [#1722](https://github.com/kataras/iris/issues/1722).

- New `uuid` builtin path parameter type. Example:

```go
// +------------------------+
// | {param:uuid}           |
// +------------------------+
// UUIDv4 (and v1) path parameter validation.

// http://localhost:8080/user/bb4f33e4-dc08-40d8-9f2b-e8b2bb615c0e -> OK
// http://localhost:8080/user/dsadsa-invalid-uuid                  -> NOT FOUND
app.Get("/user/{id:uuid}", func(ctx iris.Context) {
    id := ctx.Params().Get("id")
    ctx.WriteString(id)
})
```

- New `Configuration.KeepAlive` and `iris.WithKeepAlive(time.Duration) Configurator` added as helpers to start the server using a tcp listener featured with keep-alive.

- New `DirOptions.ShowHidden bool` is added by [@tuhao1020](https://github.com/tuhao1020) at [PR #1717](https://github.com/kataras/iris/pull/1717) to show or hide the hidden files when `ShowList` is set to true.

- New `Context.ReadJSONStream` method and `JSONReader` options for `Context.ReadJSON` and `Context.ReadJSONStream`, see the [example](_examples/request-body/read-json-stream/main.go).

- New `FallbackView` feature, per-party or per handler chain. Example can be found at: [_examples/view/fallback](_examples/view/fallback).

```go
    app.FallbackView(iris.FallbackViewFunc(func(ctx iris.Context, err iris.ErrViewNotExist) error {
        // err.Name is the previous template name.
        // err.IsLayout reports whether the failure came from the layout template.
        // err.Data is the template data provided to the previous View call.
        // [...custom logic e.g. ctx.View("fallback.html", err.Data)]
        return err
    }))
```

- New `versioning.Aliases` middleware and up to 80% faster version resolve. Example Code:

```go
app := iris.New()

api := app.Party("/api")
api.Use(Aliases(map[string]string{
    versioning.Empty: "1", // when no version was provided by the client.
    "beta": "4.0.0",
    "stage": "5.0.0-alpha"
}))

v1 := NewGroup(api, ">=1.0.0 <2.0.0")
v1.Get/Post...

v4 := NewGroup(api, ">=4.0.0 <5.0.0")
v4.Get/Post...

stage := NewGroup(api, "5.0.0-alpha")
stage.Get/Post...
```

- New [Basic Authentication](https://github.com/kataras/iris/tree/main/middleware/basicauth) middleware. Its `Default` function has not changed, however, the rest, e.g. `New` contains breaking changes as the new middleware features new functionalities.
- Add `iris.DirOptions.SPA bool` field to allow [Single Page Applications](https://github.com/kataras/iris/tree/main/_examples/file-server/single-page-application/basic/main.go) under a file server.
- A generic User interface, see the `Context.SetUser/User` methods in the New Context Methods section for more. In-short, the basicauth middleware's stored user can now be retrieved through `Context.User()` which provides more information than the native `ctx.Request().BasicAuth()` method one. Third-party authentication middleware creators can benefit of these two methods, plus the Logout below. 
- A `Context.Logout` method is added, can be used to invalidate [basicauth](https://github.com/kataras/iris/blob/main/_examples/auth/basicauth/basic/main.go) or [jwt](https://github.com/kataras/iris/blob/main/_examples/auth/jwt/blocklist/main.go) client credentials.
- Add the ability to [share functions](https://github.com/kataras/iris/tree/main/_examples/routing/writing-a-middleware/share-funcs) between handlers chain and add an [example](https://github.com/kataras/iris/tree/main/_examples/routing/writing-a-middleware/share-services) on sharing Go structures (aka services).

- Add the new `Party.UseOnce` method to the `*Route`
- Add a new `*Route.RemoveHandler(...interface{}) int` and `Party.RemoveHandler(...interface{}) Party` methods, delete a handler based on its name or the handler pc function.

```go
func middleware(ctx iris.Context) {
    // [...]
}

func main() {
    app := iris.New()

    // Register the middleware to all matched routes.
    app.Use(middleware)

    // Handlers = middleware, other
    app.Get("/", index)

    // Handlers = other
    app.Get("/other", other).RemoveHandler(middleware)
}
```

- Redis Driver is now based on the [go-redis](https://github.com/go-redis/redis/) module. Radix and redigo removed entirely. Sessions are now stored in hashes which fixes [issue #1610](https://github.com/kataras/iris/issues/1610). The only breaking change on default configuration is that the `redis.Config.Delim` option was removed. The redis sessions database driver is now defaults to the `&redis.GoRedisDriver{}`. End-developers can implement their own implementations too. The `Database#Close` is now automatically called on interrupt signals, no need to register it by yourself.

- Add builtin support for **[i18n pluralization](https://github.com/kataras/iris/tree/main/_examples/i18n/plurals)**. Please check out the [following yaml locale example](https://github.com/kataras/iris/tree/main/_examples/i18n/plurals/locales/en-US/welcome.yml) to see an overview of the supported formats.
- Fix [#1650](https://github.com/kataras/iris/issues/1650)
- Fix [#1649](https://github.com/kataras/iris/issues/1649)
- Fix [#1648](https://github.com/kataras/iris/issues/1648)
- Fix [#1641](https://github.com/kataras/iris/issues/1641)

- Add `Party.SetRoutesNoLog(disable bool) Party` to disable (the new) verbose logging of next routes.
- Add `mvc.Application.SetControllersNoLog(disable bool) *mvc.Application` to disable (the new) verbose logging of next controllers. As requested at [#1630](https://github.com/kataras/iris/issues/1630).

- Fix [#1621](https://github.com/kataras/iris/issues/1621) and add a new `cache.WithKey` to customize the cached entry key.

- Add a `Response() *http.Response` to the Response Recorder.
- Fix Response Recorder `Flush` when transfer-encoding is `chunked`.
- Fix Response Recorder `Clone` concurrent access afterwards.

- Add a `ParseTemplate` method on view engines to manually parse and add a template from a text as [requested](https://github.com/kataras/iris/issues/1617). [Examples](https://github.com/kataras/iris/tree/main/_examples/view/parse-template).
- Full `http.FileSystem` interface support for all **view** engines as [requested](https://github.com/kataras/iris/issues/1575). The first argument of the functions(`HTML`, `Blocks`, `Pug`, `Ace`, `Jet`, `Django`, `Handlebars`) can now be either a directory of `string` type (like before) or a value which completes the `http.FileSystem` interface. The `.Binary` method of all view engines was removed: pass the go-bindata's latest version `AssetFile()` exported function as the first argument instead of string.

- Add `Route.ExcludeSitemap() *Route` to exclude a route from sitemap as requested in [chat](https://chat.iris-go.com), also offline routes are excluded automatically now.

- Improved tracing (with `app.Logger().SetLevel("debug")`) for routes. Screens:

#### DBUG Routes (1)

![DBUG routes 1](https://iris-go.com/images/v12.2.0-dbug.png?v=0)

#### DBUG Routes (2)

![DBUG routes 2](https://iris-go.com/images/v12.2.0-dbug2.png?v=0)

#### DBUG Routes (3)

![DBUG routes with Controllers](https://iris-go.com/images/v12.2.0-dbug3.png?v=0)

- Update the [pprof middleware](https://github.com/kataras/iris/tree/main/middleware/pprof).

- New `Controller.HandleHTTPError(mvc.Code) <T>` optional Controller method to handle http errors as requested at: [MVC - More Elegent OnErrorCode registration?](https://github.com/kataras/iris/issues/1595). Example can be found [here](https://github.com/kataras/iris/tree/main/_examples/mvc/error-handler-http/main.go).

![MVC: HTTP Error Handler Method](https://user-images.githubusercontent.com/22900943/90948989-e04cd300-e44c-11ea-8c97-54d90fb0cbb6.png)

- New [Rewrite Engine Middleware](https://github.com/kataras/iris/tree/main/middleware/rewrite). Set up redirection rules for path patterns using the syntax we all know. [Example Code](https://github.com/kataras/iris/tree/main/_examples/routing/rewrite).

```yml
RedirectMatch: # REDIRECT_CODE_DIGITS | PATTERN_REGEX | TARGET_REPL
  # Redirects /seo/* to /*
  - 301 /seo/(.*) /$1

  # Redirects /docs/v12* to /docs
  - 301 /docs/v12(.*) /docs

  # Redirects /old(.*) to /
  - 301 /old(.*) /

  # Redirects http or https://test.* to http or https://newtest.*
  - 301 ^(http|https)://test.(.*) $1://newtest.$2

  # Handles /*.json or .xml as *?format=json or xml,
  # without redirect. See /users route.
  # When Code is 0 then it does not redirect the request,
  # instead it changes the request URL
  # and leaves a route handle the request.
  - 0 /(.*).(json|xml) /$1?format=$2

# Redirects root domain to www.
# Creation of a www subdomain inside the Application is unnecessary,
# all requests are handled by the root Application itself.
PrimarySubdomain: www
```

- New `TraceRoute bool` on [middleware/logger](https://github.com/kataras/iris/tree/main/middleware/logger) middleware. Displays information about the executed route. Also marks the handlers executed. Screenshot:

![logger middleware: TraceRoute screenshot](https://iris-go.com/images/github/logger-trace-route.png)

- Implement feature request [Log when I18n Translation Fails?](https://github.com/kataras/iris/issues/1593) by using the new `Application.I18n.DefaultMessageFunc` field **before** `I18n.Load`. [Example of usage](https://github.com/kataras/iris/blob/main/_examples/i18n/basic/main.go#L28-L50).

- Fix [#1594](https://github.com/kataras/iris/issues/1594) and add a new `PathAfterHandler` which can be set to true to enable the old behavior (not recommended though).

- New [apps](https://github.com/kataras/iris/tree/main/apps) subpackage. [Example of usage](https://github.com/kataras/iris/tree/main/_examples/routing/subdomains/redirect/multi-instances).

![apps image example](https://user-images.githubusercontent.com/22900943/90459288-8a54f400-e109-11ea-8dea-20631975c9fc.png)

- Fix `AutoTLS` when used with `iris.TLSNoRedirect` [*](https://github.com/kataras/iris/issues/1577). The `AutoTLS` runner can be customized through the new `iris.AutoTLSNoRedirect` instead, read its go documentation. Example of having both TLS and non-TLS versions of the same application without conflicts with letsencrypt `./well-known` path:

![](https://iris-go.com/images/github/autotls-1.png)

```go
package main

import (
	"net/http"
	"time"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"time": time.Now().Unix(),
			"tls":  ctx.Request().TLS != nil,
		})
	})

	var fallbackServer = func(acme func(http.Handler) http.Handler) *http.Server {
		srv := &http.Server{Handler: acme(app)}
		go srv.ListenAndServe()
		return srv
	}

	app.Run(iris.AutoTLS(":443", "example.com", "mail@example.com",
		iris.AutoTLSNoRedirect(fallbackServer)))
}
```

- `iris.Minify` middleware to minify responses based on their media/content-type.

- `Context.OnCloseErr` and `Context.OnConnectionCloseErr` - to call a function of `func() error`  instead of an `iris.Handler` when request is closed or manually canceled.

- `Party.UseError(...Handler)` - to register handlers to run before any http errors (e.g. before `OnErrorCode/OnAnyErrorCode` or default error codes when no handler is responsible to handle a specific http status code).

- `Party.UseRouter(...Handler) and Party.ResetRouterFilters()` - to register handlers before the main router, useful on handlers that should control whether the router itself should ran or not. Independently of the incoming request's method and path values. These handlers will be executed ALWAYS against ALL incoming matched requests. Example of use-case: CORS.

- `*versioning.Group` type is a full `Party` now.

- `Party.UseOnce` - either inserts a middleware, or on the basis of the middleware already existing, replace that existing middleware instead.

- Ability to register a view engine per group of routes or for the current chain of handlers through `Party.RegisterView` and `Context.ViewEngine` respectfully.

- Add [Blocks](_examples/view/template_blocks_0) template engine. <!-- Reminder for @kataras: follow https://github.com/flosch/pongo2/pull/236#issuecomment-668950566 discussion so we can get back on using the original pongo2 repository as they fixed the issue about an incompatible 3rd party package (although they need more fixes, that's why I commented there) -->

- Add [Ace](_examples/view/template_ace_0) template parser to the view engine and other minor improvements.

- Fix huge repo size of 55.7MB, which slows down the overall Iris installation experience. Now, go-get performs ~3 times faster. I 've managed it using the [bfg-repo-cleaner](https://github.com/rtyley/bfg-repo-cleaner) tool - an alternative to  git-filter-branch command. Watch the small gif below to learn how:

[![](https://media.giphy.com/media/U8560aiWTurW4iAOLn/giphy.gif)](https://media.giphy.com/media/U8560aiWTurW4iAOLn/giphy.gif)

- [gRPC](https://grpc.io/) features:
    - New Router [Wrapper](middleware/grpc).
    - New MVC `.Handle(ctrl, mvc.GRPC{...})` option which allows to register gRPC services per-party (without the requirement of a full wrapper) and optionally strict access to gRPC clients only, see the [example here](_examples/mvc/grpc-compatible).

- Add `Configuration.RemoteAddrHeadersForce bool` to force `Context.RemoteAddr() string` to return the first entry of request headers as a fallback instead of the `Request.RemoteAddr` one, as requested at: [1567#issuecomment-663972620](https://github.com/kataras/iris/issues/1567#issuecomment-663972620).

- Fix [#1569#issuecomment-663739177](https://github.com/kataras/iris/issues/1569#issuecomment-663739177).

- Fix [#1564](https://github.com/kataras/iris/issues/1564).

- Fix [#1553](https://github.com/kataras/iris/issues/1553).

- New `DirOptions.Cache` to cache assets in-memory among with their compressed contents (in order to be ready to served if client ask). Learn more about this feature by reading [all #1556 comments](https://github.com/kataras/iris/issues/1556#issuecomment-661057446). Usage:

```go
var dirOpts = DirOptions{
    // [...other options]
    Cache: DirCacheOptions{
        Enable: true,
        // Don't compress files smaller than 300 bytes.
        CompressMinSize: 300,
        // Ignore compress already compressed file types
        // (some images and pdf).
        CompressIgnore: iris.MatchImagesAssets,
        // Gzip, deflate, br(brotli), snappy.
        Encodings: []string{"gzip", "deflate", "br", "snappy"},
        // Log to the stdout the total reduced file size.
        Verbose: 1,
    },
}
```

- New `DirOptions.PushTargets` and `PushTargetsRegexp` to push index' assets to the client without additional requests. Inspirated by issue [#1562](https://github.com/kataras/iris/issues/1562). Example matching all `.js, .css and .ico` files (recursively):

```go
var dirOpts = iris.DirOptions{
    // [...other options]
    IndexName: "/index.html",
    PushTargetsRegexp: map[string]*regexp.Regexp{
        "/": regexp.MustCompile("((.*).js|(.*).css|(.*).ico)$"),
        // OR:
        // "/": iris.MatchCommonAssets,
    },
    Compress: true,
}
```

- Update jet parser to v5.0.2, closes [#1551](https://github.com/kataras/iris/issues/1551). It contains two breaking changes by its author:
    - Relative paths on `extends, import, include...` tmpl functions, e.g. `{{extends "../layouts/application.jet"}}` instead of `layouts/application.jet`
    - the new [jet.Ranger](https://github.com/CloudyKit/jet/pull/165) interface now requires a `ProvidesIndex() bool` method too
    - Example has been [updated](https://github.com/kataras/iris/tree/main/_examples/view/template_jet_0)

- Fix [#1552](https://github.com/kataras/iris/issues/1552).

- Proper listing of root directories on `Party.HandleDir` when its `DirOptions.ShowList` was set to true.
    - Customize the file/directory listing page through views, see [example](https://github.com/kataras/iris/tree/main/_examples/file-server/file-server).

- Socket Sharding as requested at [#1544](https://github.com/kataras/iris/issues/1544). New `iris.WithSocketSharding` Configurator and `SocketSharding bool` setting.

- Versioned Controllers feature through the new `mvc.Version` option. See [_examples/mvc/versioned-controller](https://github.com/kataras/iris/blob/main/_examples/mvc/versioned-controller/main.go).

- Fix [#1539](https://github.com/kataras/iris/issues/1539).

- New [rollbar example](https://github.com/kataras/iris/blob/main/_examples/logging/rollbar/main.go).

- New builtin [requestid](https://github.com/kataras/iris/tree/main/middleware/requestid) middleware.

- New builtin [JWT](https://github.com/kataras/iris/tree/main/middleware/jwt) middleware based on the fastest JWT implementation; [kataras/jwt](https://github.com/kataras/jwt) featured with optional wire encryption to set claims with sensitive data when necessary.

- New `iris.RouteOverlap` route registration rule. `Party.SetRegisterRule(iris.RouteOverlap)` to allow overlapping across multiple routes for the same request subdomain, method, path. See [1536#issuecomment-643719922](https://github.com/kataras/iris/issues/1536#issuecomment-643719922). This allows two or more **MVC Controllers** to listen on the same path based on one or more registered dependencies (see [_examples/mvc/authenticated-controller](https://github.com/kataras/iris/tree/main/_examples/mvc/authenticated-controller)).

- `Context.ReadForm` now can return an `iris.ErrEmptyForm` instead of `nil` when the new `Configuration.FireEmptyFormError` is true  (when `iris.WithEmptyFormError` is set) on missing form body to read from.

- `Configuration.EnablePathIntelligence | iris.WithPathIntelligence` to enable path intelligence automatic path redirection on the most closest path (if any), [example]((https://github.com/kataras/iris/blob/main/_examples/routing/intelligence/main.go)

- Enhanced cookie security and management through new `Context.AddCookieOptions` method and new cookie options (look on New Package-level functions section below), [securecookie](https://github.com/kataras/iris/tree/main/_examples/cookies/securecookie) example has been updated.
- `Context.RemoveCookie` removes also the Request's specific cookie of the same request lifecycle when `iris.CookieAllowReclaim` is set to cookie options, [example](https://github.com/kataras/iris/tree/main/_examples/cookies/options).

- `iris.TLS` can now accept certificates in form of raw `[]byte` contents too.
- `iris.TLS` registers a secondary http server which redirects "http://" to their "https://" equivalent requests, unless the new `iris.TLSNoRedirect` host Configurator is provided on `iris.TLS`, e.g. `app.Run(iris.TLS("127.0.0.1:443", "mycert.cert", "mykey.key", iris.TLSNoRedirect))`. There is `iris.AutoTLSNoRedirect` option for `AutoTLS` too.

- Fix an [issue](https://github.com/kataras/i18n/issues/1) about i18n loading from path which contains potential language code.

- Server will not return neither log the `ErrServerClosed` error if `app.Shutdown` was called manually via interrupt signal(CTRL/CMD+C), note that if the server closed by any other reason the error will be fired as previously (unless `iris.WithoutServerError(iris.ErrServerClosed)`).

- Finally, Log level's and Route debug information colorization is respected across outputs. Previously if the application used more than one output destination (e.g. a file through `app.Logger().AddOutput`) the color support was automatically disabled from all, including the terminal one, this problem is fixed now. Developers can now see colors in their terminals while log files are kept with clear text.

- New `iris.WithLowercaseRouting` option which forces all routes' paths to be lowercase and converts request paths to their lowercase for matching.

- New `app.Validator { Struct(interface{}) error }` field and `app.Validate` method were added. The `app.Validator = ` can be used to integrate a 3rd-party package such as [go-playground/validator](https://github.com/go-playground/validator). If set-ed then Iris `Context`'s `ReadJSON`, `ReadXML`, `ReadMsgPack`, `ReadYAML`, `ReadForm`, `ReadQuery`, `ReadBody` methods will return the validation error on data validation failures. The [read-json-struct-validation](_examples/request-body/read-json-struct-validation) example was updated.

- A result of <T> can implement the new `hero.PreflightResult` interface which contains a single method of `Preflight(iris.Context) error`. If this method exists on a custom struct value which is returned from a handler then it will fire that `Preflight` first and if not errored then it will cotninue by sending the struct value as JSON(by-default) response body.

- `ctx.JSON, JSONP, XML`: if `iris.WithOptimizations` is NOT passed on `app.Run/Listen` then the indentation defaults to `"    "` (four spaces) and `"  "` respectfully otherwise it is empty or the provided value.

- Hero Handlers (and `app.ConfigureContainer().Handle`) do not have to require `iris.Context` just to call `ctx.Next()` anymore, this is done automatically now.

- Improve Remote Address parsing as requested at: [#1453](https://github.com/kataras/iris/issues/1453). Add `Configuration.RemoteAddrPrivateSubnets` to exclude those addresses when fetched by `Configuration.RemoteAddrHeaders` through `context.RemoteAddr() string`.

- Fix [#1487](https://github.com/kataras/iris/issues/1487).

- Fix [#1473](https://github.com/kataras/iris/issues/1473).

## New Package-level Variables

- `iris.DirListRichOptions` to pass on `iris.DirListRich` method.
- `iris.DirListRich` to override the default look and feel if the `DirOptions.ShowList` was set to true, can be passed to `DirOptions.DirList` field.
- `DirOptions.PushTargets` for http/2 push on index [*](https://github.com/kataras/iris/tree/main/_examples/file-server/http2push/main.go).
- `iris.Compression` middleware to compress responses and decode compressed request data respectfully.
- `iris.B, KB, MB, GB, TB, PB, EB` for byte units.
- `TLSNoRedirect` to disable automatic "http://" to "https://" redirections (see below)
- `CookieAllowReclaim`, `CookieAllowSubdomains`, `CookieSameSite`, `CookieSecure` and `CookieEncoding` to bring previously sessions-only features to all cookies in the request.

## New Context Methods

- `Context.FormFiles(key string, before ...func(*Context, *multipart.FileHeader) bool) (files []multipart.File, headers []*multipart.FileHeader, err error)` method.
- `Context.ReadURL(ptr interface{}) error` shortcut of `ReadParams` and `ReadQuery`. Binds URL dynamic path parameters and URL query parameters to the given "ptr" pointer of a struct value.
- `Context.SetUser(User)` and `Context.User() User` to store and retrieve an authenticated client. Read more [here](https://github.com/iris-contrib/middleware/issues/63).
- `Context.SetLogoutFunc(fn interface{}, persistenceArgs ...interface{})` and `Logout(args ...interface{}) error` methods to allow different kind of auth middlewares to be able to set a "logout" a user/client feature with a single function, the route handler may not be aware of the implementation of the authentication used.
- `Context.SetFunc(name string, fn interface{}, persistenceArgs ...interface{})` and `Context.CallFunc(name string, args ...interface{}) ([]reflect.Value, error)` to allow middlewares to share functions dynamically when the type of the function is not predictable, see the [example](https://github.com/kataras/iris/tree/main/_examples/routing/writing-a-middleware/share-funcs) for more.
- `Context.TextYAML(interface{}) error` same as `Context.YAML` but with set the Content-Type to `text/yaml` instead (Google Chrome renders it as text). 
- `Context.IsDebug() bool` reports whether the application is running under debug/development mode. It is a shortcut of Application.Logger().Level >= golog.DebugLevel.
- `Context.IsRecovered() bool` reports whether the current request was recovered from the [recover middleware](https://github.com/kataras/iris/tree/main/middleware/recover). Also the `Context.GetErrPublic() (bool, error)`, `Context.SetErrPrivate(err error)` methods and `iris.ErrPrivate` interface have been introduced. 
- `Context.RecordRequestBody(bool)` same as the Application's `DisableBodyConsumptionOnUnmarshal` configuration field but registers per chain of handlers. It makes the request body readable more than once.
- `Context.IsRecordingBody() bool` reports whether the request body can be readen multiple times.
- `Context.ReadHeaders(ptr interface{}) error` binds request headers to "ptr". [Example](https://github.com/kataras/iris/blob/main/_examples/request-body/read-headers/main.go).
- `Context.ReadParams(ptr interface{}) error` binds dynamic path parameters to "ptr". [Example](https://github.com/kataras/iris/blob/main/_examples/request-body/read-params/main.go).
- `Context.SaveFormFile(fh *multipart.FileHeader, dest string) (int64, error)` previously unexported. Accepts a result file of `Context.FormFile` and saves it to the disk.
- `Context.URLParamSlice(name string) []string` is a a shortcut of `ctx.Request().URL.Query()[name]`. Like `URLParam` but it returns all values as a string slice instead of a single string separated by commas. Note that it skips any empty values (e.g. https://iris-go.com?values=).
- `Context.PostValueMany(name string) (string, error)` returns the post data of a given key. The returned value is a single string separated by commas on multiple values. It also reports whether the form was empty or when the "name" does not exist or whether the available values are empty. It strips any empty key-values from the slice before return. See `ErrEmptyForm`, `ErrNotFound` and `ErrEmptyFormField` respectfully. The `PostValueInt`, `PostValueInt64`, `PostValueFloat64` and `PostValueBool` now respect the above errors too (the `PostValues` method now returns a second output argument of `error` too, see breaking changes below). 
- `Context.URLParamsSorted() []memstore.StringEntry` returns a sorted (by key) slice of key-value entries of the URL Query parameters.
- `Context.ViewEngine(ViewEngine)` to set a view engine on-fly for the current chain of handlers, responsible to render templates through `ctx.View`. [Example](_examples/view/context-view-engine).
- `Context.SetErr(error)` and `Context.GetErr() error` helpers.
- `Context.CompressWriter(bool) error` and `Context.CompressReader(bool) error`.
- `Context.Clone() Context` returns a copy of the Context safe for concurrent access.
- `Context.IsCanceled() bool` reports whether the request has been canceled by the client.
- `Context.IsSSL() bool` reports whether the request is under HTTPS SSL (New `Configuration.SSLProxyHeaders` and `HostProxyHeaders` fields too).
- `Context.CompressReader(enable bool)` method and `iris.CompressReader` middleware to enable future request read body calls to decompress data, [example](_examples/compression/main.go).
- `Context.RegisterDependency(v interface{})` and `Context.UnregisterDependency(typ reflect.Type)` to register/remove struct dependencies on serve-time through a middleware.
- `Context.SetID(id interface{})` and `Context.GetID() interface{}` added to register a custom unique indetifier to the Context, if necessary.
- `Context.Scheme() string` returns the full scheme of the request URL.
- `Context.SubdomainFull() string` returns the full subdomain(s) part of the host (`host[0:rootLevelDomain]`).
- `Context.Domain() string` returns the root level domain.
- `Context.AddCookieOptions(...CookieOption)` adds options for `SetCookie`, `SetCookieKV, UpsertCookie` and `RemoveCookie` methods for the current request.
- `Context.ClearCookieOptions()` clears any cookie options registered through `AddCookieOptions`.
- `Context.SetLanguage(langCode string)` force-sets a language code from inside a middleare, similar to the `app.I18n.ExtractFunc`
- `Context.ServeContentWithRate`, `ServeFileWithRate` and `SendFileWithRate` methods to throttle the "download" speed of the client
- `Context.IsHTTP2() bool` reports whether the protocol version for incoming request was HTTP/2
- `Context.IsGRPC() bool` reports whether the request came from a gRPC client
- `Context.UpsertCookie(*http.Cookie, cookieOptions ...context.CookieOption)` upserts a cookie, fixes [#1485](https://github.com/kataras/iris/issues/1485) too
- `Context.StopWithStatus(int)` stops the handlers chain and writes the status code
- `StopWithText(statusCode int, format string, args ...interface{})` stops the handlers chain, writes thre status code and a plain text message
- `Context.StopWithError(int, error)` stops the handlers chain, writes thre status code and the error's message
- `Context.StopWithJSON(int, interface{})` stops the handlers chain, writes the status code and sends a JSON response
- `Context.StopWithProblem(int, iris.Problem)` stops the handlers, writes the status code and sends an `application/problem+json` response
- `Context.Protobuf(proto.Message)` sends protobuf to the client (note that the `Context.JSON` is able to send protobuf as JSON)
- `Context.MsgPack(interface{})` sends msgpack format data to the client
- `Context.ReadProtobuf(ptr)` binds request body to a proto message
- `Context.ReadJSONProtobuf(ptr, ...options)` binds JSON request body to a proto message
- `Context.ReadMsgPack(ptr)` binds request body of a msgpack format to a struct
- `Context.ReadBody(ptr)` binds the request body to the "ptr" depending on the request's Method and Content-Type
- `Context.ReflectValue() []reflect.Value` stores and returns the `[]reflect.ValueOf(ctx)`
- `Context.Controller() reflect.Value` returns the current MVC Controller value.

## MVC & Dependency Injection

The new release contains a fresh new and awesome feature....**a function dependency can accept previous registered dependencies and update or return a new value of any type**.

The new implementation is **faster** on both design and serve-time.

The most common scenario from a route to handle is to:
- accept one or more path parameters and request data, a payload
- send back a response, a payload (JSON, XML,...)

The new Iris Dependency Injection feature is about **33.2% faster** than its predecessor on the above case. This drops down even more the performance cost between native handlers and dynamic handlers with dependencies. This reason itself brings us, with safety and performance-wise, to the new `Party.ConfigureContainer(builder ...func(*iris.APIContainer)) *APIContainer` method which returns methods such as `Handle(method, relativePath string, handlersFn ...interface{}) *Route` and `RegisterDependency`.

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
    app.ConfigureContainer(func(api *iris.APIContainer) {
        api.Post("/{id:int}", handler)
    })
    app.Listen(":5000", iris.WithOptimizations)
}
```

Your eyes don't lie you. You read well, no `ctx.ReadJSON(&v)` and `ctx.JSON(send)` neither `error` handling are presented. It is a huge relief but if you ever need, you still have the control over those, even errors from dependencies. Here is a quick list of the new Party.ConfigureContainer()'s fields and methods:

```go
// Container holds the DI Container of this Party featured Dependency Injection.
// Use it to manually convert functions or structs(controllers) to a Handler.
Container *hero.Container
```

```go
// OnError adds an error handler for this Party's DI Hero Container and its handlers (or controllers).
// The "errorHandler" handles any error may occurred and returned
// during dependencies injection of the Party's hero handlers or from the handlers themselves.
OnError(errorHandler func(iris.Context, error))
```

```go
// RegisterDependency adds a dependency.
// The value can be a single struct value or a function.
// Follow the rules:
// * <T> {structValue}
// * func(accepts <T>)                                 returns <D> or (<D>, error)
// * func(accepts iris.Context)                        returns <D> or (<D>, error)
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
RegisterDependency(dependency interface{})

// UseResultHandler adds a result handler to the Container.
// A result handler can be used to inject the returned struct value
// from a request handler or to replace the default renderer.
UseResultHandler(handler func(next iris.ResultHandler) iris.ResultHandler)
```

<details><summary>ResultHandler</summary>

```go
type ResultHandler func(ctx iris.Context, v interface{}) error
```
</details>

```go
// Use same as a common Party's "Use" but it accepts dynamic functions as its "handlersFn" input.
Use(handlersFn ...interface{})
// Done same as a common Party's but it accepts dynamic functions as its "handlersFn" input.
Done(handlersFn ...interface{})
```

```go
// Handle same as a common Party's `Handle` but it accepts one or more "handlersFn" functions which each one of them
// can accept any input arguments that match with the Party's registered Container's `Dependencies` and
// any output result; like custom structs <T>, string, []byte, int, error,
// a combination of the above, hero.Result(hero.View | hero.Response) and more.
//
// It's common from a hero handler to not even need to accept a `Context`, for that reason,
// the "handlersFn" will call `ctx.Next()` automatically when not called manually.
// To stop the execution and not continue to the next "handlersFn"
// the end-developer should output an error and return `iris.ErrStopExecution`.
Handle(method, relativePath string, handlersFn ...interface{}) *Route

// Get registers a GET route, same as `Handle("GET", relativePath, handlersFn....)`.
Get(relativePath string, handlersFn ...interface{}) *Route
// and so on...
```

Prior to this version the `iris.Context` was the only one dependency that has been automatically binded to the handler's input or a controller's fields and methods, read below to see what types are automatically binded:

| Type | Maps To |
|------|:---------|
| [*mvc.Application](https://pkg.go.dev/github.com/kataras/iris/v12/mvc?tab=doc#Application) | Current MVC Application |
| [iris.Context](https://pkg.go.dev/github.com/kataras/iris/v12/context?tab=doc#Context) | Current Iris Context |
| [*sessions.Session](https://pkg.go.dev/github.com/kataras/iris/v12/sessions?tab=doc#Session) | Current Iris Session |
| [context.Context](https://golang.org/pkg/context/#Context) | [ctx.Request().Context()](https://golang.org/pkg/net/http/#Request.Context) |
| [*http.Request](https://golang.org/pkg/net/http/#Request) | `ctx.Request()` |
| [http.ResponseWriter](https://golang.org/pkg/net/http/#ResponseWriter) | `ctx.ResponseWriter()` |
| [http.Header](https://golang.org/pkg/net/http/#Header) | `ctx.Request().Header` |
| [time.Time](https://golang.org/pkg/time/#Time) | `time.Now()` |
| [*golog.Logger](https://pkg.go.dev/github.com/kataras/golog) | Iris Logger |
| [net.IP](https://golang.org/pkg/net/#IP) | `net.ParseIP(ctx.RemoteAddr())` |
| [mvc.Code](https://pkg.go.dev/github.com/kataras/iris/v12/mvc?tab=doc#Code) | `ctx.GetStatusCode() int` |
| [mvc.Err](https://pkg.go.dev/github.com/kataras/iris/v12/mvc?tab=doc#Err) | `ctx.GetErr() error` |
| [iris/context.User](https://pkg.go.dev/github.com/kataras/iris/v12/context?tab=doc#User) | `ctx.User()` |
| `string`, | |
| `int, int8, int16, int32, int64`, | |
| `uint, uint8, uint16, uint32, uint64`, | |
| `float, float32, float64`, | |
| `bool`, | |
| `slice` | [Path Parameter](https://github.com/kataras/iris/blob/main/_examples/routing/dynamic-path/main.go#L20) |
| Struct | [Request Body](https://github.com/kataras/iris/tree/main/_examples/request-body) of `JSON`, `XML`, `YAML`, `Form`, `URL Query`, `Protobuf`, `MsgPack` |

Here is a preview of what the new Hero handlers look like:

### Request & Response & Path Parameters

**1.** Declare Go types for client's request body and a server's response.

```go
type (
	request struct {
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
	}

	response struct {
		ID      uint64 `json:"id"`
		Message string `json:"message"`
	}
)
```

**2.** Create the route handler.

Path parameters and request body are binded automatically.
- **id uint64** binds to "id:uint64"
- **input request** binds to client request data such as JSON

```go
func updateUser(id uint64, input request) response {
	return response{
		ID:      id,
		Message: "User updated successfully",
	}
}
```

**3.** Configure the container per group and register the route.

```go
app.Party("/user").ConfigureContainer(container)

func container(api *iris.APIContainer) {
    api.Put("/{id:uint64}", updateUser)
}
```

**4.** Simulate a [client](https://curl.haxx.se/download.html) request which sends data to the server and displays the response.

```sh
curl --request PUT -d '{"firstanme":"John","lastname":"Doe"}' http://localhost:8080/user/42
```

```json
{
    "id": 42,
    "message": "User updated successfully"
}
```

### Custom Preflight

Before we continue to the next section, register dependencies, you may want to learn how a response can be customized through the `iris.Context` right before sent to the client.

The server will automatically execute the `Preflight(iris.Context) error` method of a function's output struct value right before send the response to the client.

Take for example that you want to fire different HTTP status codes depending on the custom logic inside your handler and also modify the value(response body) itself before sent to the client. Your response type should contain a `Preflight` method like below.

```go
type response struct {
    ID      uint64 `json:"id,omitempty"`
    Message string `json:"message"`
    Code    int    `json:"code"`
    Timestamp int64 `json:"timestamp,omitempty"`
}

func (r *response) Preflight(ctx iris.Context) error {
    if r.ID > 0 {
        r.Timestamp = time.Now().Unix()
    }

    if r.Code > 0 {
        ctx.StatusCode(r.Code)
    }

    return nil
}
```

Now, each handler that returns a `*response` value will call the `response.Preflight` method automatically.

```go
func deleteUser(db *sql.DB, id uint64) *response {
    // [...custom logic]

    return &response{
        Message: "User has been marked for deletion",
        Code: iris.StatusAccepted,
    }
}
```

If you register the route and fire a request you should see an output like this, the timestamp is filled and the HTTP status code of the response that the client will receive is 202 (Status Accepted).

```json
{
  "message": "User has been marked for deletion",
  "code": 202,
  "timestamp": 1583313026
}
```

### Register Dependencies

**1.** Import packages to interact with a database.
The go-sqlite3 package is a database driver for [SQLite](https://www.sqlite.org/index.html).

```go
import "database/sql"
import _ "github.com/mattn/go-sqlite3"
```

**2.** Configure the container ([see above](#request--response--path-parameters)), register your dependencies. Handler expects an *sql.DB instance.

```go
localDB, _ := sql.Open("sqlite3", "./foo.db")
api.RegisterDependency(localDB)
```

**3.** Register a route to create a user.

```go
api.Post("/{id:uint64}", createUser)
```

**4.** The create user Handler.

The handler accepts a database and some client request data such as JSON, Protobuf, Form, URL Query and e.t.c. It Returns a response.

```go
func createUser(db *sql.DB, user request) *response {
    // [custom logic using the db]
    userID, err := db.CreateUser(user)
    if err != nil {
        return &response{
            Message: err.Error(),
            Code: iris.StatusInternalServerError,
        }
    }

	return &response{
		ID:      userID,
		Message: "User created",
		Code:    iris.StatusCreated,
	}
}
```

**5.** Simulate a [client](https://curl.haxx.se/download.html) to create a user.

```sh
# JSON
curl --request POST -d '{"firstname":"John","lastname":"Doe"}' \
--header 'Content-Type: application/json' \
http://localhost:8080/user
```

```sh
# Form (multipart)
curl --request POST 'http://localhost:8080/users' \
--header 'Content-Type: multipart/form-data' \
--form 'firstname=John' \
--form 'lastname=Doe'
```

```sh
# Form (URL-encoded)
curl --request POST 'http://localhost:8080/users' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'firstname=John' \
--data-urlencode 'lastname=Doe'
```

```sh
# URL Query
curl --request POST 'http://localhost:8080/users?firstname=John&lastname=Doe'
```

Response: 

```json
{
    "id": 42,
    "message": "User created",
    "code": 201,
    "timestamp": 1583313026
}
```

## Breaking Changes

- The `versioning.NewMatcher` has been removed entirely in favor of `NewGroup`. Strict versions format on `versioning.NewGroup` is required. E.g. `"1"` is not valid anymore, you have to specify `"1.0.0"`. Example: `NewGroup(api, ">=1.0.0 <2.0.0")`. The [routing/versioning](_examples/routing/versioning) examples have been updated.
- Now that `RegisterView` can be used to register different view engines per-Party, there is no need to support registering multiple engines under the same Party. The `app.RegisterView` now upserts the given Engine instead of append. You can now render templates **without file extension**, e.g. `index` instead of `index.ace`, both forms are valid now.
- The `Context.ContentType` does not accept filenames to resolve the mime type anymore (caused issues with  vendor-specific(vnd) MIME types).
- The `Configuration.RemoteAddrPrivateSubnets.IPRange.Start and End` are now type of `string` instead of `net.IP`. The `WithRemoteAddrPrivateSubnet` option remains as it is, already accepts `string`s.
- The `i18n#LoaderConfig.FuncMap template.FuncMap` field was replaced with `Funcs func(iris.Locale) template.FuncMap` in order to give current locale access to the template functions. A new `app.I18n.Loader` was introduced too, in order to make it easier for end-developers to customize the translation key values.
- Request Logger's `Columns bool` field has been removed. Use the new [accesslog](https://github.com/kataras/iris/tree/main/_examples/logging/request-logger/accesslog/main.go) middleware instead.
- The `.Binary` method of all view engines was removed: pass the go-bindata's latest version `AssetFile()` exported function as the first argument instead of string. All examples updated.
- `ContextUploadFormFiles(destDirectory string, before ...func(*Context, *multipart.FileHeader) bool) (uploaded []*multipart.FileHeader, n int64, err error)` now returns the total files uploaded too (as its first parameter) and the "before" variadic option should return a boolean, if false then the specific file is skipped.
- `Context.PostValues(name string) ([]string, error)` now returns a second output argument of `error` type too, which reports `ErrEmptyForm` or `ErrNotFound` or `ErrEmptyFormField`. The single post value getters now returns the **last value** if multiple was given instead of the first one (this allows clients to append values on flow updates).
- `Party.GetReporter()` **removed**. The `Application.Build` returns the first error now and the API's errors are logged, this allows the server to run even if some of the routes are invalid but not fatal to the entire application (it was a request from a company).
- `versioning.NewGroup(string)` now accepts a `Party` as its first input argument: `NewGroup(Party, string)`.
- `versioning.RegisterGroups` is **removed** as it is no longer necessary.
- `Configuration.RemoteAddrHeaders` from `map[string]bool` to `[]string`. If you used `With(out)RemoteAddrHeader` then you are ready to proceed without any code changes for that one.
- `ctx.Gzip(boolean)` replaced with `ctx.CompressWriter(boolean) error`.
- `ctx.GzipReader(boolean) error` replaced with `ctx.CompressReader(boolean) error`.
- `iris.Gzip` and `iris.GzipReader` replaced with `iris.Compression` (middleware).
- `ctx.ClientSupportsGzip() bool` replaced with `ctx.ClientSupportsEncoding("gzip", "br" ...) bool`.
- `ctx.GzipResponseWriter()` is **removed**.
- `Party.HandleDir/iris.FileServer` now accepts both `http.FileSystem` and `string` and returns a list of `[]*Route` (GET and HEAD) instead of GET only. You can write: both `app.HandleDir("/", iris.Dir("./assets"))` and `app.HandleDir("/", "./assets")` and `DirOptions.Asset, AssetNames, AssetInfo` removed, use `go-bindata -fs [..]` and `app.HandleDir("/", AssetFile())` instead.
- `Context.OnClose` and `Context.OnCloseConnection` now both accept an `iris.Handler` instead of a simple `func()` as their callback.
- `Context.StreamWriter(writer func(w io.Writer) bool)` changed to `StreamWriter(writer func(w io.Writer) error) error` and it's now the `Context.Request().Context().Done()` channel that is used to receive any close connection/manual cancel signals, instead of the deprecated `ResponseWriter().CloseNotify()` one. Same for the `Context.OnClose` and `Context.OnCloseConnection` methods.
- Fixed handler's error response not be respected when response recorder was used instead of the common writer. Fixes [#1531](https://github.com/kataras/iris/issues/1531). It contains a **BREAKING CHANGE** of: the new `Configuration.ResetOnFireErrorCode` field should be set **to true** in order to behave as it used before this update (to reset the contents on recorder).
- `Context.String()` (rarely used by end-developers) it does not return a unique string anymore, to achieve the old representation you must call the new `Context.SetID` method first.
- `iris.CookieEncode` and `CookieDecode` are replaced with the `iris.CookieEncoding`.
- `sessions#Config.Encode` and `Decode` are removed in favor of (the existing) `Encoding` field.
- `versioning.GetVersion` now returns an empty string if version wasn't found.
- Change the MIME type of `Javascript .js` and `JSONP` as the HTML specification now recommends to `"text/javascript"` instead of the obselete `"application/javascript"`. This change was pushed to the `Go` language itself as well. See <https://go-review.googlesource.com/c/go/+/186927/>.
- Remove the last input argument of `enableGzipCompression` in `Context.ServeContent`, `ServeFile` methods. This was deprecated a few versions ago. A middleware (`app.Use(iris.CompressWriter)`) or a prior call to `Context.CompressWriter(true)` will enable compression. Also these two methods and `Context.SendFile` one now support `Content-Range` and `Accept-Ranges` correctly out of the box (`net/http` had a bug, which is now fixed).
- `Context.ServeContent` no longer returns an error, see `ServeContentWithRate`, `ServeFileWithRate` and `SendFileWithRate` new methods too.
- `route.Trace() string` changed to `route.Trace(w io.Writer)`, to achieve the same result just pass a `bytes.Buffer`
- `var mvc.AutoBinding` removed as the default behavior now resolves such dependencies automatically (see [[FEATURE REQUEST] MVC serving gRPC-compatible controller](https://github.com/kataras/iris/issues/1449)).
- `mvc#Application.SortByNumMethods()` removed as the default behavior now binds the "thinnest"  empty `interface{}` automatically (see [MVC: service injecting fails](https://github.com/kataras/iris/issues/1343)).
- `mvc#BeforeActivation.Dependencies().Add` should be replaced with `mvc#BeforeActivation.Dependencies().Register` instead
- **REMOVE** the `kataras/iris/v12/typescript` package in favor of the new [iris-cli](https://github.com/kataras/iris-cli). Also, the alm typescript online editor was removed as it is deprecated by its author, please consider using the [designtsx](https://designtsx.com/) instead.

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

- [_examples/routing/route-register-rule](_examples/routing/route-register-rule)

# We, 05 February 2020 | v12.1.6

Fixes:

- [jet.View - urlpath error](https://github.com/kataras/iris/issues/1438)
- [Context.ServeFile send 'application/wasm' with a wrong extra field](https://github.com/kataras/iris/issues/1440)

# Su, 02 February 2020 | v12.1.5

Various improvements and linting.

# Su, 29 December 2019 | v12.1.4

Minor fix on serving embedded files.

# We, 25 December 2019 | v12.1.3

Fix [[BUG] [iris.Default] RegisterView](https://github.com/kataras/iris/issues/1410)

# Th, 19 December 2019 | v12.1.2

Fix [[BUG]Session works incorrectly when meets the multi-level TLDs](https://github.com/kataras/iris/issues/1407).

# Mo, 16 December 2019 | v12.1.1

Add [Context.FindClosest(n int) []string](https://github.com/kataras/iris/blob/main/_examples/routing/intelligence/manual/main.go#L22)

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
- If you used [Iris Django](https://github.com/kataras/iris/tree/main/_examples/view/template_django_0) view engine with `import _ github.com/flosch/pongo2-addons` you **must change** the import path to `_ github.com/iris-contrib/pongo2-addons` or add a [go mod replace](https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive) to your `go.mod` file, e.g. `replace github.com/flosch/pongo2-addons => github.com/iris-contrib/pongo2-addons v0.0.1`.

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

Support for i18n is now a **builtin feature** and is being respected across your entire application, per say [sitemap](https://github.com/kataras/iris/blob/main/_examples/routing/sitemap/main.go) and [views](https://github.com/kataras/iris/blob/main/_examples/i18n/basic/main.go#L50).

### Sitemaps

Iris generates and serves one or more [sitemap.xml](https://www.sitemaps.org/protocol.html) for your static routes.

Navigate through: https://github.com/kataras/iris/blob/main/_examples/routing/sitemap/main.go for more.

## New Examples

2. [_examples/i18n](_examples/i18n)
1. [_examples/sitemap](_examples/routing/sitemap)
3. [_examples/desktop/blink](_examples/desktop/blink)
4. [_examples/desktop/lorca](_examples/desktop/lorca)
5. [_examples/desktop/webview](_examples/desktop/webview)

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
- Add and update all [experimental handlers](https://github.com/iris-contrib/middleware) 
- New `XMLMap` function which wraps a `map[string]interface{}` and converts it to a valid xml content to render through `Context.XML` method
- Add new `ProblemOptions.XML` and `RenderXML` fields to render the `Problem` as XML(application/problem+xml) instead of JSON("application/problem+json) and enrich the `Negotiate` to easily accept the `application/problem+xml` mime.

Commit log: https://github.com/kataras/iris/compare/v11.2.7...v11.2.8

# Th, 15 August 2019 | v11.2.7

This minor version contains improvements on the Problem Details for HTTP APIs implemented on [v11.2.5](#mo-12-august-2019--v1125).

- Fix https://github.com/kataras/iris/issues/1335#issuecomment-521319721
- Add `ProblemOptions` with `RetryAfter` as requested at: https://github.com/kataras/iris/issues/1335#issuecomment-521330994.
- Add `iris.JSON` alias for `context#JSON` options type.

[Example](https://github.com/kataras/iris/blob/45d7c6fedb5adaef22b9730592255f7bb375e809/_examples/routing/http-errors/main.go#L85) updated. 

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
- [Context.ReadYAML](https://github.com/kataras/iris/tree/main/_examples/request-body/read-yaml)
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
