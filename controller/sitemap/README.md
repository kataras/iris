# Sitemap

## Overview

The [sitemap](https://github.com/kataras/iris/tree/main/controller/sitemap) controller for the [Iris web framework](https://www.iris-go.com) provides a mechanism to automatically generate and serve [sitemap](https://www.sitemaps.org/protocol.html) XML files based on the registered routes of your application. This controller helps improve your website's SEO by providing search engines with a structured list of URLs to crawl.

## Features

- **Automatic Sitemap Generation**: Generates sitemap XML files based on the registered routes.
- **Localization Support**: Supports localized links for different languages.
- **Customizable URL Properties**: Allows customization of URL properties such as `lastmod`, `changefreq`, and `priority`.
- **Sitemap Indexing**: Automatically splits sitemaps and serves a sitemap index if more than 50,000 static routes are registered.

## Usage

### Installation

To use the Sitemap controller, you need to import it into your Iris application:

```go
import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/sitemap"
)
```

### Basic Usage

To enable the Sitemap controller, simply use it in your Iris application:

```go
app := iris.New()
app.Router("/", sitemap.New("http://localhost"))
```

### Localization Support

You can add localization support for the generated links in the sitemap by using the `WithLocalization` method:

```go
import (
	"github.com/kataras/iris/v14/middleware/i18n"
)

app := iris.New()

// Load locales.
locales := i18n.New()
locales.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
locales.SetDefault("en-US")
i18nHandler := i18n.NewHandler(locales, i18n.HandlerOptions{
    // Retrieve the language from the URL parameter, e.g. "/?lang=el-GR".
    URLParameter: "lang", 
    // Retrieve the language from the cookie, e.g. "lang=el-GR".
    Cookie:       "lang",
})
app.Use(i18nHandler)

sitemapRouter := sitemap.New("http://localhost").WithLocalization(i18nHandler, nil)
app.Router("/", sitemapRouter)
```

## Example

Here is a complete example demonstrating the usage of the Sitemap controller:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/sitemap"
)

func main() {
	app := iris.New()
	app.Router("/", sitemap.New("http://localhost"))

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Index")
	})

	app.Get("/about", func(ctx iris.Context) {
		ctx.WriteString("About")
	})

	app.Get("/signup", func(ctx iris.Context) {
		ctx.WriteString("Signup Form")
	})
	app.Post("/signup", func(ctx iris.Context) {
		ctx.WriteString("Signup")
	})

	app.Listen(":8080")
}
```

### Example Output

When you access the `/sitemap.xml` endpoint, the generated sitemap XML will look like this:

```xml
<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>http://localhost/</loc>
  </url>
  <url>
    <loc>http://localhost/about</loc>
  </url>
  <url>
    <loc>http://localhost/signup</loc>
  </url>
</urlset>
```

The Sitemap controller reads the registered routes and generates the sitemap XML from them. It excludes dynamic routes, subdomain routes, offline routes, and any route that called `ExcludeSitemap`.
