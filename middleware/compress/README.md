# Compression Middleware for Iris

The `compress` middleware works in both directions. It decompresses a request body according to the request's `Content-Encoding`, and compresses the response with the best encoding the client advertised in `Accept-Encoding`.

If you need finer control than the middleware gives you, the same package lets you acquire a request reader or a response writer by hand. Both are covered below.

## Features of Iris Compress Middleware

The Iris compress middleware offers a variety of features to handle HTTP compression and decompression. These features include:

1. **Request Decompression**: Automatically decompresses incoming request bodies based on the `Content-Encoding` header.
2. **Response Compression**: Compresses outgoing response bodies based on the client's `Accept-Encoding` header.
3. **Support for Multiple Compression Algorithms**: Supports gzip, deflate, brotli, snappy, s2 and zstd compression algorithms.
4. **Custom Compression Levels**: Allows setting custom compression levels for supported algorithms.

### Compression Algorithms

The Iris compress middleware supports the following compression algorithms:

- **gzip**: A widely used compression algorithm that provides a good balance between compression ratio and speed.
- **deflate**: A compression algorithm similar to gzip but with slightly different compression characteristics.
- **brotli**: A modern compression algorithm that provides higher compression ratios than gzip and deflate.
- **snappy**: A fast compression algorithm that provides lower compression ratios but is very efficient in terms of speed.
- **s2**: A variant of snappy that offers faster decompression speeds.
- **zstd**: Zstandard, a modern algorithm that compresses better than gzip at comparable or better speed.

## Usage

To use the `compress` middleware, simply add it to your Iris application:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/compress"
)

func main() {
	app := iris.New()
	app.Use(compress.Handler)

	app.Post("/", func(ctx iris.Context) {
		body, _ := ctx.Body()
		ctx.Writef("Received: %s", string(body))
	})

	app.Listen(":8080")
}
```

### Manually Acquiring a Request Reader for Decompression

In some cases, you might want to manually acquire a request reader for decompression. The `NewRequestReader` function can be used for this purpose.

#### Example

```go
package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Post("/", func(ctx iris.Context) {
		r := ctx.Request()
		reader, ok, err := compress.NewRequestReader(r)
		if err != nil {
			errors.UnsupportedMediaType.Err(ctx, err) // the code carries the 415.
			return
		}
		if ok {
			defer reader.Close()
			r.Body = reader
		}

		body, _ := io.ReadAll(r.Body)
		ctx.Writef("Received: %s", string(body))
	})

	app.Listen(":8080")
}
```

### Manually Acquiring a Response Writer for Compression

Similarly, you might want to manually acquire a response writer for compression. The `AcquireResponseWriter` function can be used for this purpose.

#### Example

```go
package main

import (
	"net/http"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Post("/", func(ctx iris.Context) {
		r := ctx.Request()
		w := ctx.ResponseWriter()

		writer, ok, err := compress.AcquireResponseWriter(w, r, -1)
		if err != nil {
			errors.UnsupportedMediaType.Err(ctx, err) // the code carries the 415.
			return
		}
		if ok {
			ctx.SetResponseWriter(writer)
			// Put the previous writer back before releasing. A released writer
			// returns to a pool, and a middleware registered outside this
			// handler (an access log, say) reads the Context's writer after it.
			defer func() {
				ctx.SetResponseWriter(w)
				compress.ReleaseResponseWriter(writer)
			}()
		}

		body, _ := ctx.Body()
		ctx.Writef("Received: %s", string(body))
	})

	app.Listen(":8080")
}
```
