# Extract Referrer

This middleware parses the `Referer` header, or a `referrer` query parameter, and tells you where the visit came from: a search engine, social media, email, or a direct link. For a Google search it also separates organic traffic from adwords. Background reading: [HTTP referer](https://en.wikipedia.org/wiki/HTTP_referer) on Wikipedia and [Referrer-Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy) on MDN.

## Features

- **Extract Referrer Information**: Extracts referrer information from the "Referer" (or "Referrer") header and URL query parameters.
- **Referrer Types**: Identifies different types of referrers such as search engines, social media, email, and direct links.
- **Google Search Type**: Differentiates between organic and adwords Google search types.

## Installation

To use the `referrer` middleware, you need to import it in your Iris application:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/referrer"
)
```

## Usage

### Example Code

Here is a basic example of how to use the `referrer` middleware in an Iris application:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/referrer"
)

func main() {
    app := iris.New()
    app.Use(referrer.Handler)

    app.Get("/", func(ctx iris.Context) {
        r := referrer.Get(ctx)
        switch r.Type {
        case referrer.ReferrerSearch:
            ctx.Writef("Search %s: %s\n", r.Label, r.Query)
            ctx.Writef("Google: %s\n", r.GoogleType)
        case referrer.ReferrerSocial:
            ctx.Writef("Social %s\n", r.Label)
        case referrer.ReferrerIndirect:
            ctx.Writef("Indirect: %s\n", r.URL)
        }
    })

    // Example URLs:
    // http://localhost:8080?referrer=https://twitter.com/Xinterio/status/1023566830974251008
    // http://localhost:8080?referrer=https://www.google.com/search?q=Top+6+golang+web+frameworks&oq=Top+6+golang+web+frameworks
    app.Listen(":8080")
}
```

When you access the `/` endpoint with a referrer query parameter, you will see an output similar to the following:

```
Search Google: Top 6 golang web frameworks
Google: organic
```
