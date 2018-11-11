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

## Grouping versioned routes

Impl & tests done, example not. **TODO**

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

