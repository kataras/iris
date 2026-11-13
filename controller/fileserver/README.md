# Iris File Server

This controller serves static files: images, stylesheets, scripts, downloads. The files can sit in a directory on disk or be compiled into the binary through an embedded file system. Directory listing, caching, compression, HTTP/2 Push, single-page-application fallback and WebDAV are all options on the same controller.

## Features of Iris File Server

The Iris file server offers a variety of features to handle static file serving efficiently. These features include:

1. **Serving Static Files**: Serve static files from a directory or an embedded file system.
2. **Directory Listing**: Optionally list the contents of directories with customizable options.
3. **File Caching**: Enable in-memory caching and pre-compression of files.
4. **HTTP/2 Push**: Push resources to the client without additional requests.
5. **Attachments**: Serve files as downloadable attachments with optional rate limiting and filename customization.
6. **Single Page Application (SPA) Support**: Serve the index file for all not-found routes.
7. **Favicon Serving**: Serve a favicon file with caching.
8. **Strip Prefix**: Serve files by stripping a specified prefix from the URL.
9. **WebDAV Support**: Serve files with WebDAV support for remote file management.
10. **Index Pages with Relative Links**: Serve a directory's index file at its own URL, so the relative links of the page load from that directory.

## Installation

To use the Iris file server, you need to have the Iris framework installed. You can install Iris using the following command:

```sh
go get github.com/kataras/iris/v14@latest
```

## Usage

### Basic Setup

To use the file server, you need to import the necessary packages and configure the controller in your Iris application. Here is a basic example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		IndexName: "index.html",
	}))

	app.Listen(":8080")
}
```

### Serving Static Files

Files can come from a directory on disk or from an embedded file system. Here are examples for each:

#### Using a Directory Path

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		IndexName: "index.html",
	}))

	app.Listen(":8080")
}
```

#### Using `http.FileSystem`

```go
package main

import (
	"net/http"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	fs := http.Dir("./assets")
	app.Router("/static", fileserver.New(fs, fileserver.Options{
		IndexName: "index.html",
	}))

	app.Listen(":8080")
}
```

#### Using `embed.FS`

```go
package main

import (
	"embed"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

//go:embed assets
var embeddedFiles embed.FS

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New(embeddedFiles, fileserver.Options{
		IndexName: "index.html",
	}))

	app.Listen(":8080")
}
```

#### Using `fs.FS`

```go
package main

import (
	"io/fs"
	"os"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	dirFS := os.DirFS("./assets")
	app.Router("/static", fileserver.New(dirFS, fileserver.Options{
		IndexName: "index.html",
	}))

	app.Listen(":8080")
}
```

#### Keeping symbolic links inside the directory

A directory given as a string becomes an `http.Dir`. `http.Dir` works on the request path as text: it cleans the path before it opens anything, so a `..` in `/static/../secret` cannot climb out of the directory. A symbolic link (or a Windows junction) that sits inside `./assets` and points somewhere else is a different matter: it is followed, and the file at the other end is served. To serve only what is really under the directory, open it with `os.OpenRoot` and pass its `fs.FS`:

```go
root, err := os.OpenRoot("./assets")
if err != nil {
	app.Logger().Fatal(err)
}
app.Router("/static", fileserver.New(root.FS(), fileserver.Options{}))
```

A link that leaves the root now answers 404 instead of the file it points to. This costs two things. The process holds the directory open for as long as it runs, so a deploy that replaces `./assets` with a new directory keeps serving the old one until the process restarts. And a link that leaves the root is no longer served at all, which breaks a layout that relies on one on purpose, for example `./assets/shared` linked to a directory outside it.

### Directory Listing

The file server can optionally list the contents of directories. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		ShowList: true,
	}))

	app.Listen(":8080")
}
```

#### Using `DirList`

The `DirList` function renders directories and files in a plain HTML format. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		ShowList: true,
		DirList:  fileserver.DirList,
	}))

	app.Listen(":8080")
}
```

#### Using `DirListRich`

The `DirListRich` function renders directories and files in a rich HTML format with customizable templates. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		ShowList: true,
		DirList:  fileserver.DirListRich(fileserver.DirListRichOptions{
			Tmpl:     fileserver.DirListRichTemplate,
			TmplName: "dirlist",
		}),
	}))

	app.Listen(":8080")
}
```

### File Caching

The file server supports in-memory caching and pre-compression of files. The `CompressMinSize` option specifies the minimum content size for compression in bytes. Files smaller than this size will not be compressed to avoid the overhead of compressing small files, which may not provide significant benefits. The supported compression algorithms are gzip, deflate, brotli, snappy, s2 and zstd. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		Cache: fileserver.DirCacheOptions{
			Enable:          true,
			CompressMinSize: 300, // Minimum content size for compression in bytes
			Encodings:       []string{"gzip", "deflate", "br", "snappy", "s2", "zstd"},
			Verbose:         1,
		},
	}))

	app.Listen(":8080")
}
```

### HTTP/2 Push

HTTP/2 Push is a feature that allows the server to send resources to the client proactively, without the client having to request them. This can improve page load times by reducing the number of round trips required to fetch resources. You can read more about HTTP/2 Push on [Wikipedia](https://en.wikipedia.org/wiki/HTTP/2_Server_Push).

With HTTP/2 Push the server sends a resource the client has not asked for yet, saving a round trip. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		PushTargets: map[string][]string{
			"/": {"favicon.ico", "js/main.js", "css/main.css"},
		},
	}))

	app.Listen(":8080")
}
```

### Attachments

A file can be served as a download rather than rendered, with an optional transfer rate limit and a filename of your choosing. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		Attachments: fileserver.Attachments{
			Enable: true,
			Limit:  1024, // 1 KB per second
			Burst:  2048, // Allow bursts up to 2 KB
			NameFunc: func(systemName string) string {
				return "download-" + systemName
			},
		},
	}))

	app.Listen(":8080")
}
```

### Single Page Application (SPA) Support

The file server supports serving the index file for all not-found routes, which is useful for Single Page Applications (SPAs). Make sure to set the `IndexName` field. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		SPA:       true,
		IndexName: "index.html",
	}))

	app.Listen(":8080")
}
```

### Index Pages with Relative Links

The router removes the trailing slash of a directory URL, so the index page of `./assets/doc` is served at `/static/doc`. From that URL a browser resolves a relative link against the parent path: it asks for `css/app.css` at `/static/css/app.css` and for `page2.html` at `/static/page2.html`. Root-relative links such as `/static/doc/css/app.css` work from any URL and need nothing.

Set `RedirectToIndex` to serve the index file at its own URL instead. `/static/doc` then redirects to `/static/doc/index.html`, and the relative links of the page resolve inside `/static/doc/`. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		IndexName:       "index.html",
		RedirectToIndex: true,
	}))

	app.Listen(":8080")
}
```

The option is off by default, and it does nothing when `SPA` is true or `IndexName` is empty. A directory that has no index file is not redirected. On a site that was live without it, a browser that cached the old redirect of `/static/doc/index.html` to its directory can loop until that cache entry is gone.

### Customizable MIME Types

Map your own MIME type onto a file extension when the default guess is wrong. Here is an example:

```go
package main

import (
	"mime"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	mime.AddExtensionType(".custom", "application/custom")

	app := iris.New()
	app.Router("/static", fileserver.New("./assets", fileserver.DefaultOptions))

	app.Listen(":8080")
}
```

### Favicon Serving

A favicon is a small icon that represents a website or web application. It is typically displayed in the browser's address bar, tab, or bookmark list. Favicons can also be used as app icons on mobile devices and tablets. The recommended size for a favicon is 16x16 pixels for browsers, but it can also include larger sizes such as 32x32, 48x48, 64x64, and 128x128 pixels for different platforms and applications.

Serve a favicon with caching in one call. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	app.Router("/", fileserver.Favicon("./assets/favicon.ico"))

	app.Listen(":8080")
}
```

### StripPrefix

The `StripPrefix` function returns a handler that serves HTTP requests by removing the given prefix from the request URL's path and invoking the handler `h`. This is useful when you want to serve files from a specific directory but want to strip the directory prefix from the URL. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
	app := iris.New()
	fileserverHandler := fileserver.Handler("./static_files", fileserver.DefaultOptions)
	h := fileserver.StripPrefix("/static", fileserverHandler)

	app.Get("/static/{file:path}", h)
	app.Head("/static/{file:path}", h)

	app.Listen(":8080")
}
```

### Differences Between `New` and `Handler`

The `New` and `Handler` functions both serve static files, but they are used in different contexts and have different purposes:

- **`New`**: This function returns a new file server controller and registers the necessary routes to serve files from the provided file system or directory (GET, HEAD). It is used to create a complete file server with routing capabilities. It is typically used when you want to serve static files from a specific route in your Iris application.

	```go
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		IndexName: "index.html",
		ShowList:  true,
	}))
	```

- **`Handler`**: This function returns a handler that serves files from a specific file system. It is used to create a file server handler that can be used with custom route(s). It is typically used when you want to serve static files as part of a more complex routing setup.

	```go
	fileserverHandler := fileserver.Handler("./static_files", fileserver.DefaultOptions)
	h := fileserver.StripPrefix("/static", fileserverHandler)

	app.Get("/static/{file:path}", h)
	app.Head("/static/{file:path}", h)
	```

### WebDAV Support

WebDAV (Web Distributed Authoring and Versioning) extends HTTP with methods for creating, moving and deleting files, so a client can manage the served directory remotely. Read more about it on [Wikipedia](https://en.wikipedia.org/wiki/WebDAV).

Here is an example of how to use WebDAV with the Iris file server:

```go
package main

import (
    "net/http"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/controller/fileserver"

    "golang.org/x/net/webdav"
)

func main() {
    app := iris.New()
    webdavHandler := &webdav.Handler{
        FileSystem: webdav.Dir("./"),
        LockSystem: webdav.NewMemLS(),
        Logger: func(r *http.Request, err error) {
            if err != nil {
                app.Logger().Error(err)
            }
        },
    }
    app.Router("/", fileserver.WebDAV(webdavHandler))

    app.Listen(":8080")
}
```

* List files:
	```
	curl --location --request PROPFIND 'http://localhost:8080'
	```
* Get File:
	```
	curl --location --request GET 'http://localhost:8080/test.txt'
	```
* Upload File:
	```
	curl --location --request PUT 'http://localhost:8080/newfile.txt' \
	--header 'Content-Type: text/plain' \
	--data-raw 'This is a new file!'
	```
* Copy File:
	```
	curl --location --request COPY 'http://localhost:8080/test.txt' \
	--header 'Destination: newdir/test.txt'
	```
* Create New Directory:
	```
	curl --location --request MKCOL 'http://localhost:8080/anewdir/'
	```

## Context Methods

### ServeContent

The `ServeContent` method replies to the request using the content in the provided `io.ReadSeeker`. It handles Range requests, sets the MIME type, and handles various conditional requests.

#### Example

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
	"os"
	"time"
)

func main() {
	app := iris.New()

	app.Get("/content", func(ctx iris.Context) {
		file, err := os.Open("example.txt")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}
		defer file.Close()

		ctx.ServeContent(file, "example.txt", time.Now())
	})

	app.Listen(":8080")
}
```

### ServeContentWithRate

The `ServeContentWithRate` method is similar to `ServeContent` but it can throttle the speed of reading and writing the content to the client.

#### Example

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
	"os"
	"time"
)

func main() {
	app := iris.New()

	app.Get("/content", func(ctx iris.Context) {
		file, err := os.Open("example.txt")
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}
		defer file.Close()

		ctx.ServeContentWithRate(file, "example.txt", time.Now(), 1024, 2048) // 1 KB/s with bursts up to 2 KB
	})

	app.Listen(":8080")
}
```

### ServeFile

The `ServeFile` method replies to the request with the contents of the named file or directory. Use it when you want to serve assets like CSS and JavaScript files. If the client should confirm and save the file, use the `SendFile` method instead.

#### Example

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/file", func(ctx iris.Context) {
		err := ctx.ServeFile("example.txt")
		if err != nil {
			errors.HandleError(ctx, err)
		}
	})

	app.Listen(":8080")
}
```

### ServeFileWithRate

The `ServeFileWithRate` method is similar to `ServeFile` but it can throttle the speed of reading and writing the file to the client.

#### Example

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/file", func(ctx iris.Context) {
		err := ctx.ServeFileWithRate("example.txt", 1024, 2048) // 1 KB/s with bursts up to 2 KB
		if err != nil {
			errors.HandleError(ctx, err)
		}
	})

	app.Listen(":8080")
}
```

### SendFile

The `SendFile` method sends a file as an attachment, which is downloaded and saved locally by the client. Use this method when you want the user to directly download a file, such as an uploaded file.

#### Example

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/download", func(ctx iris.Context) {
		err := ctx.SendFile("example.txt", "downloaded_example.txt")
		if err != nil {
			errors.HandleError(ctx, err)
		}
	})

	app.Listen(":8080")
}
```

### SendFileWithRate

The `SendFileWithRate` method is similar to `SendFile` but it can throttle the speed of reading and writing the file to the client.

#### Example

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/download", func(ctx iris.Context) {
		err := ctx.SendFileWithRate("example.txt", "downloaded_example.txt", 1024, 2048) // 1 KB/s with bursts up to 2 KB
		if err != nil {
			errors.HandleError(ctx, err)
		}
	})

	app.Listen(":8080")
}
```

### When to Use Serve Methods vs. Send Methods

- **Serve Methods (`ServeContent`, `ServeContentWithRate`, `ServeFile`, `ServeFileWithRate`)**: Use these methods when you want to serve static files like CSS, JavaScript, or HTML files directly to the client. These methods are suitable for serving assets that are part of your web application's frontend.

- **Send Methods (`SendFile`, `SendFileWithRate`)**: Use these methods when you want the client to download a file as an attachment. This is useful for scenarios where the user needs to download and save a file locally, such as downloading an uploaded file or exporting data.

## Blocking Hidden Files (v14)

By default the file server serves dotfiles such as `.git` or `.env` when they exist in the served directory. To deny them, set `fileserver.BlockHidden` as the `AssetValidator`:

```go
app.Router("/static", fileserver.New("./assets", fileserver.Options{
    AssetValidator: fileserver.BlockHidden,
}))
```

`BlockHidden` denies any request whose path contains a dot-prefixed segment. It also blocks conventional dot-directories such as `.well-known`; compose your own validator on top of it if you need to allow specific ones.
