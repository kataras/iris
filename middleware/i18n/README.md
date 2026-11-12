# Iris Internationalization and Localization

The `i18n` middleware picks the visitor's language and serves your content in it. It reads the language from the `Accept-Language` header, a URL parameter, a cookie, or a subdomain, loads translations from files, embedded assets, or maps, and handles plural forms.

## Features

- **Automatic Language Detection**: Detects the user's language based on the `Accept-Language` header, URL parameter, cookie, or subdomain.
- **Flexible Translation Loading**: Supports loading translations from various sources including files, embedded assets, and maps.
- **Template Integration**: Easily integrate translations into your HTML templates.
- **Pluralization with Dynamic Variables**: Supports dynamic plural forms in translations.
- **Template Functions and Local Variables**: Allows custom template functions and local variables in translations.
- **Customizable**: Allows custom functions and message formats.

## Installation

To use the `i18n` middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/i18n"
```

## Basic Usage

Here is a basic example of how to use the `i18n` middleware in an Iris application:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/i18n"
)

func main() {
    app := iris.New()
    locales := i18n.New()
    locales.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
    locales.SetDefault("en-US")

    app.Use(i18n.NewHandler(locales, i18n.HandlerOptions{
        // Retrieve the language from the URL parameter, e.g. "/?lang=el-GR".
        URLParameter: "lang", 
        // Retrieve the language from the cookie, e.g. "lang=el-GR".
        Cookie:       "lang",
        // Retrieve and resolve the language from the subdomain, e.g. "en.example.com".
        Subdomain:    true, 
    }))

    app.Get("/", func(ctx iris.Context) {
        locale := i18n.GetLocale(ctx)
        hi := locale.GetMessage("hi", "iris")
        // OR:
        // hi := i18n.GetMessage(ctx, "hi", "iris")
        ctx.Writef("From the language %s translated output: %s", locale.Language(), hi)
    })

    app.Listen(":8080")
}
```

**./locales/en-US.ini**

```ini
hi = γεια, %s
```

**./locales/el-GR.ini**

```ini
hi = γεια, %s
```

**./locales/zh-CN.ini**

```ini
hi = 你好，%s
```

The supported file formats are `.ini`, `.json`, `.toml` and `.yaml/.yml`. Each of them can be used to store translations in the specified format and optionally can be splitted into multiple files or/and sections in the same file.


```sh
curl -i -H "Accept-Language: el-GR" http://localhost:8080
```

Output:

```sh
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Date: Sun, 01 Dec 2024 11:36:22 GMT
Content-Length: 33

From the language el-GR translated output: γεια, iris
```

## Advanced Usage

### Loading Translations

You can load translations from various sources such as files, embedded assets, or maps.

#### From Files

```go
locales.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
```

#### From Embedded Assets

```go
locales.LoadAssets(assetNames, asset, "en-US", "el-GR", "zh-CN")
```

#### From Maps

```go
langMap := i18n.LangMap{
    "en-US": map[string]any{
        "hello": "Hello",
    },
    "el-GR": map[string]any{
        "hello": "Γειά",
    },
}
locales.LoadKV(langMap)
```

#### From File System

```go
import "embed"

//go:embed locales/*
var localesFS embed.FS

locales.LoadFS(localesFS, "locales/*", "en-US", "el-GR", "zh-CN")
```

### Template Integration

You can integrate translations into your Blocks HTML templates using the `tr` and `trUnsafe` functions.

```go
templates := blocks.New("./views")
templates.AddFunc("tr", func(lang, key string, args ...any) string {
    if locale, ok := locales.GetLocale(lang); ok {
        return locale.GetMessage(key, args...)
    }
    return "[INVALID LOCALE] " + lang
})
templates.Load()
app.UseRouter(view.Handler(templates))
```

## Pluralization and Variables

Translations can carry plural forms and variables, so one key covers "1 message" and "5 messages" and the text can name the user.

### Pluralization

Pluralization is the ability to provide different translations based on a numeric value. The `i18n` middleware supports various plural forms such as `one`, `other`, and specific counts like `=x`.

#### Example

```yaml
FreeDay:
  one: "You have a day off"
  other: "You have %[1]d free days"
  "=3": "You have three days and %[2]d minutes off."
```

In this example:
- `FreeDay` with a count of 1 will use the `one` form.
- `FreeDay` with a count of 5 will use the `other` form.
- `FreeDay` with a count of 3 and an additional argument will use the `=3` form.

### Variables

Variables put dynamic content into a translation. You declare them in the locale file and fill them in at render time.

#### Example

```yaml
Vars:
  - Minutes:
      one: "minute"
      other: "minutes"
  - Houses:
      one: "house"
      other: "houses"
  - Gender:
      "=1": "She"
      "=2": "He"

HouseCount: "${Gender} (%[3]s) has %[2]d ${Houses}"
```

In this example:
- The `Gender` variable will be replaced based on the provided value (1 for female, 2 for male).
- The `Houses` variable will be replaced based on the count (singular or plural form).

### Using Variables in Translations

You can use variables in your translations by referencing them with `${VariableName}` syntax.

#### Example

```yaml
YouLate: "You are %[1]d ${Minutes} late."
```

In this example:
- The `Minutes` variable will be replaced based on the provided count (singular or plural form).

### Combining Pluralization and Variables

You can combine pluralization and variables to create complex translations.

#### Example

```yaml
VarTemplatePlural:
  one: "${Gender} is awesome"
  other: "other (${Gender}) has %[3]d ${Houses}"
  "=5": "{{call .InlineJoin .Names}} are awesome"
```

In this example:
- The `VarTemplatePlural` translation will use the appropriate form based on the count and replace the `Gender` and `Houses` variables accordingly.
- The `=5` form uses a template function to join names.
- The `call .InlineJoin .Names` can be a custom template function that joins names.

### Local Variables

Local variables can be defined within a specific translation section.

#### Example

```yaml
LocalVarsHouseCount:
  Text: "${Gender} has %[2]d ${Houses}"
  Vars:
    - Gender:
        "=3": "She"
        "=4": "He"
    - Houses:
        one: "house"
        other: "houses"
```

In this example:
- The `LocalVarsHouseCount` translation uses local variables `Gender` and `Houses` specific to this section.

### Using Pluralization and Variables in Code

You can use the `i18n.GetMessage` function to retrieve translations with pluralization and variables.

#### Example

```go
i18n.GetMessage(ctx, "YouLate", 1)
i18n.GetMessage(ctx, "HouseCount", female, 2, "Maria")
i18n.GetMessage(ctx, "VarTemplatePlural", 2, male, 1)
```

In this example:

- `i18n.GetMessage(ctx, "YouLate", 1)` will output "You are 1 minute late."
- `i18n.GetMessage(ctx, "HouseCount", female, 2, "Maria")` will output "She (Maria) has 2 houses."
- `i18n.GetMessage(ctx, "VarTemplatePlural", 2, male, 1)` will output "other (He) has 1 house."

> For more examples and detailed usage of pluralization and variables, refer to the [_examples/06-middleware/i18n](https://github.com/kataras/iris/tree/main/_examples/06-middleware/i18n).

## Customization

You can customize the middleware by providing custom functions and message formats, **before any Load** method call.

```go
locales.SetDefaultMessageFunc(func(langMatched, key string, args ...any) string {
    return fmt.Sprintf("lang: %s: not found key: %s: args: %v", langMatched, key, args)
})
```

## Router Integration

The `i18n` middleware can be integrated with the router to handle language-specific routes and perform redirections based on the language. Make sure to register the Handler as a middleware to retrieve the Locale from the `i18n.GetLocale` or the translated message from the `i18n.GetMessage` functions.

### Path-Based Redirection

You can configure the router to redirect requests based on the language specified in the URL path.

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/i18n"
)

func main() {
    app := iris.New()
    locales := i18n.New()
    locales.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
    locales.SetDefault("en-US")

    app.UseRouter(i18n.NewRouter(locales, i18n.RouterOptions{
        Cookie:    "lang",
        // Disable subdomain redirection, so only path-based redirection is enabled.
        Subdomain: false,
    }))

    app.Use(i18n.NewHandler(locales, i18n.HandlerOptions{
        URLParameter: "lang",
        Cookie:       "lang",
        // Retrieve and resolve the language from the subdomain, e.g. "en.example.com".
        Subdomain:    false,
    }))

    app.Get("/", func(ctx iris.Context) {
        hi := i18n.GetMessage(ctx, "hi", "iris")
        ctx.Writef("Translated output: %s", hi)
    })

    app.Listen(":8080")
}
```

In this example, the router will redirect requests with a language prefix in the URL path (e.g., `/en-US/`, `/el-GR/`, `/zh-CN/`) to the appropriate language-specific content.

### Subdomain-Based Redirection

You can configure the router to redirect requests based on the language specified in the subdomain.

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/i18n"
)

func main() {
    app := iris.New()
    locales := i18n.New()
    locales.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
    locales.SetDefault("en-US")

    app.UseRouter(i18n.NewRouter(locales, i18n.RouterOptions{
        Cookie:    "lang",
        Subdomain: true, // Enable subdomain redirection.
    }))

    app.Use(i18n.NewHandler(locales, i18n.HandlerOptions{
        // Retrieve the language from the URL parameter, e.g. "/?lang=el-GR".
        URLParameter: "lang", 
        // Retrieve the language from the cookie, e.g. "lang=el-GR".
        Cookie:       "lang",
        // Do not include subdomain identifier as the language,
        // we already have the subdomain redirection enabled in the Router level.
        Subdomain:    false,
    }))

    app.Get("/", func(ctx iris.Context) {
        hi := i18n.GetMessage(ctx, "hi", "iris")
        ctx.Writef("Translated output: %s", hi)
    })

    app.Listen(":8080")
}
```

In this example, the router will redirect requests with a language subdomain (e.g., `en.example.com`, `el.example.com`, `zh.example.com`) to the appropriate language-specific content.

### Combining Path and Subdomain Redirection

You can also combine both path and subdomain redirection.

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/i18n"
)

func main() {
    app := iris.New()
    locales := i18n.New()
    locales.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
    locales.SetDefault("en-US")

    app.UseRouter(i18n.NewRouter(locales, i18n.RouterOptions{
        Cookie:    "lang", // Store the language in a cookie.
        Subdomain: true, // Enable subdomain redirection.
    }))

    app.Use(i18n.NewHandler(locales, i18n.HandlerOptions{
        URLParameter: "lang", // Retrieve the language from the URL parameter.
        Cookie:       "lang", // Retrieve the language from the cookie.
    }))

    app.Get("/", func(ctx iris.Context) {
        hi := i18n.GetMessage(ctx, "hi", "iris")
        ctx.Writef("Translated output: %s", hi)
    })

    app.Listen(":8080")
}
```

In this example, the router will handle both path-based and subdomain-based redirections, allowing for flexible language-specific routing.

## References

- [RFC 4646 - Tags for Identifying Languages](https://tools.ietf.org/html/rfc4646)
- [Wikipedia: Internationalization and Localization](https://en.wikipedia.org/wiki/Internationalization_and_localization)

For more examples and detailed usage, refer to the [official Iris i18n examples](https://github.com/kataras/iris/tree/main/_examples/06-middleware/i18n).
