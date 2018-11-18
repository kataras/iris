# Versioning

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