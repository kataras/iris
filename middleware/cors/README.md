# CORS Middleware for Iris

This middleware answers CORS preflight requests and sets the access-control headers on your responses. You declare which origins may reach your resources; a request from anywhere else is rejected before your handler runs.

## What is CORS?

CORS is a mechanism that allows web servers to specify who can access their resources and how the resources can be accessed. It is implemented through a set of HTTP headers that dictate how browsers and servers interact. The primary headers involved in CORS are:

- **Access-Control-Allow-Origin**: Specifies which origins are allowed to access the resource.
- **Access-Control-Allow-Methods**: Specifies the HTTP methods that are allowed when accessing the resource.
- **Access-Control-Allow-Headers**: Specifies the headers that can be used when making a request to the resource.
- **Access-Control-Allow-Credentials**: Indicates whether or not the response to the request can be exposed when the credentials flag is true.
- **Access-Control-Expose-Headers**: Specifies which headers can be exposed as part of the response.
- **Access-Control-Max-Age**: Indicates how long the results of a preflight request can be cached.

For more detailed information on CORS, you can refer to the [Mozilla Developer Network (MDN) documentation](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS).

## Features of Iris CORS Middleware

The Iris CORS middleware offers a variety of features to control CORS behavior. These features include:

1. **Allowing Specific Origins**: You can specify which origins are allowed to access your resources.
2. **Custom Origin Extraction**: You can define custom functions to extract the origin from the request.
3. **Custom Allow Origin Functions**: You can define custom functions to determine if an origin is allowed.
4. **Disallowing Credentials**: You can specify whether or not credentials are allowed.
5. **Exposing Headers**: You can specify which headers can be exposed as part of the response.
6. **Allowing Headers**: You can specify which headers are allowed when making a request to the resource.
7. **Setting Referrer Policy**: You can specify the referrer policy for the resource.
8. **Setting Max Age**: You can specify how long the results of a preflight request can be cached.

## Installation

To use the Iris CORS middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/cors"
```

## Usage

### Basic Setup

To use the CORS middleware, you need to import the necessary packages and configure the middleware in your Iris application. Here is a basic example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Allowing Specific Origins

You can specify which origins are allowed to access your resources using the `AllowOrigins` method. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().AllowOrigins("https://example.com", "https://another-example.com").Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Custom Origin Extraction

You can define custom functions to extract the origin from the request using the `ExtractOriginFunc` method. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().ExtractOriginFunc(func(ctx iris.Context) (string, bool) {
		origin := ctx.Header("X-Custom-Origin")
		return origin, origin != ""
	}).Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Custom Allow Origin Functions

You can define custom functions to determine if an origin is allowed using the `AllowOriginFunc` method. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().AllowOriginFunc(func(ctx iris.Context, origin string) bool {
		return origin == "https://example.com"
	}).Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Disallowing Credentials

You can specify whether or not credentials are allowed using the `DisallowCredentials` method. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().DisallowCredentials().Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Exposing Headers

You can specify which headers can be exposed as part of the response using the `ExposeHeaders` method. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().ExposeHeaders("X-Custom-Header").Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Allowing Headers

You can specify which headers are allowed when making a request to the resource using the `AllowHeaders` method. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().AllowHeaders("X-Custom-Header").Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Setting Referrer Policy

You can specify the referrer policy for the resource using the `ReferrerPolicy` method. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().ReferrerPolicy(cors.NoReferrer).Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

### Setting Max Age

You can specify how long the results of a preflight request can be cached using the `MaxAge` method. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/cors"
)

func main() {
	app := iris.New()
	app.UseRouter(cors.New().MaxAge(6*time.Minute).Handler())

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	app.Listen(":8080")
}
```

## JavaScript Client Examples

### Using AJAX (HTML5)

Here is an example of making a CORS request using AJAX:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CORS AJAX Example</title>
</head>
<body>
    <button id="fetchData">Fetch Data</button>
    <script>
        document.getElementById('fetchData').addEventListener('click', function() {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', 'http://localhost:8080', true);
            xhr.withCredentials = true;
            xhr.onload = function() {
                if (xhr.status === 200) {
                    console.log(xhr.responseText);
                }
            };
            xhr.send();
        });
    </script>
</body>
</html>
```

### Using Fetch API

Here is an example of making a CORS request using the Fetch API:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CORS Fetch API Example</title>
</head>
<body>
    <button id="fetchData">Fetch Data</button>
    <script>
        document.getElementById('fetchData').addEventListener('click', function() {
            fetch('http://localhost:8080', {
                method: 'GET',
                credentials: 'include'
            })
            .then(response => response.text())
            .then(data => console.log(data))
            .catch(error => console.error('Error:', error));
        });
    </script>
</body>
</html>
```

### Using jQuery

Here is an example of making a CORS request using jQuery:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CORS jQuery Example</title>
    <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
</head>
<body>
    <button id="fetchData">Fetch Data</button>
    <script>
        $('#fetchData').on('click', function() {
            $.ajax({
                url: 'http://localhost:8080',
                method: 'GET',
                xhrFields: {
                    withCredentials: true
                },
                success: function(data) {
                    console.log(data);
                },
                error: function(error) {
                    console.error('Error:', error);
                }
            });
        });
    </script>
</body>
</html>
```

### Using Axios

Here is an example of making a CORS request using Axios:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CORS Axios Example</title>
    <script src="https://cdn.jsdelivr.net/npm/axios/dist/axios.min.js"></script>
</head>
<body>
    <button id="fetchData">Fetch Data</button>
    <script>
        document.getElementById('fetchData').addEventListener('click', function() {
            axios.get('http://localhost:8080', { withCredentials: true })
                .then(response => console.log(response.data))
                .catch(error => console.error('Error:', error));
        });
    </script>
</body>
</html>
```

## Credentials and Wildcard Origins (v14)

As of v14 the middleware does not send `Access-Control-Allow-Credentials: true` together with a wildcard `Access-Control-Allow-Origin: *`, because browsers reject that combination. Credentialed CORS continues to work for specific origins.

If you deliberately need credentials with a wildcard origin (for example a non-browser API client), opt in with `AllowCredentials`:

```go
app.UseRouter(cors.New().AllowCredentials().Handler())
```

Use `DisallowCredentials` to turn credentials off entirely.
