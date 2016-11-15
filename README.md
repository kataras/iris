<p align="center">

 <a href="https://www.gitbook.com/book/kataras/iris/details"><img  width="600" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_6_flat_alpha.png"></a>

<br/>


<a href="https://travis-ci.org/kataras/iris"><img src="https://img.shields.io/travis/kataras/iris.svg?style=flat-square" alt="Build Status"></a>

<a href="http://goreportcard.com/report/kataras/iris"><img src="https://img.shields.io/badge/report%20card%20-a%2B-F44336.svg?style=flat-square" alt="http://goreportcard.com/report/kataras/iris"></a>

<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-go-6362c2.svg?style=flat-square" alt="Built with GoLang"></a>


<a href="https://golang.org"><img src="https://img.shields.io/badge/platform-any-ec2eb4.svg?style=flat-square" alt="Cross framework"></a>


<a href="https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted"><img src="https://img.shields.io/badge/open-%20source-thisismycolor.svg?logo=data:image%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAwIDEwMDAiPjxwYXRoIGZpbGw9InJnYigyMjAsMjIwLDIyMCkiIGQ9Ik04ODYuNiwzMDUuM2MtNDUuNywyMDMuMS0xODcsMzEwLjMtNDA5LjYsMzEwLjNoLTc0LjFsLTUxLjUsMzI2LjloLTYybC0zLjIsMjEuMWMtMi4xLDE0LDguNiwyNi40LDIyLjYsMjYuNGgxNTguNWMxOC44LDAsMzQuNy0xMy42LDM3LjctMzIuMmwxLjUtOGwyOS45LTE4OS4zbDEuOS0xMC4zYzIuOS0xOC42LDE4LjktMzIuMiwzNy43LTMyLjJoMjMuNWMxNTMuNSwwLDI3My43LTYyLjQsMzA4LjktMjQyLjdDOTIxLjYsNDA2LjgsOTE2LjcsMzQ4LjYsODg2LjYsMzA1LjN6Ii8%2BPHBhdGggZmlsbD0icmdiKDIyMCwyMjAsMjIwKSIgZD0iTTc5MS45LDgzLjlDNzQ2LjUsMzIuMiw2NjQuNCwxMCw1NTkuNSwxMEgyNTVjLTIxLjQsMC0zOS44LDE1LjUtNDMuMSwzNi44TDg1LDg1MWMtMi41LDE1LjksOS44LDMwLjIsMjUuOCwzMC4ySDI5OWw0Ny4zLTI5OS42bC0xLjUsOS40YzMuMi0yMS4zLDIxLjQtMzYuOCw0Mi45LTM2LjhINDc3YzE3NS41LDAsMzEzLTcxLjIsMzUzLjItMjc3LjVjMS4yLTYuMSwyLjMtMTIuMSwzLjEtMTcuOEM4NDUuMSwxODIuOCw4MzMuMiwxMzAuOCw3OTEuOSw4My45TDc5MS45LDgzLjl6Ii8%2BPC9zdmc%2B" alt="Donation"></a>

<br/>


<a href="https://github.com/kataras/iris/releases"><img src="https://img.shields.io/badge/%20version%20-%205.0.2%20-blue.svg?style=flat-square" alt="Releases"></a>

<a href="https://github.com/iris-contrib/examples"><img src="https://img.shields.io/badge/%20examples-repository-3362c2.svg?style=flat-square" alt="Examples"></a>

<a href="https://docs.iris-go.com"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Practical Guide/Docs"></a>

<a href="https://kataras.rocket.chat/channel/iris"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Chat"></a><br/>
<br/>


<b>iris</b> is a web application framework written in Go.
<b>Easy</b> to <a href="https://github.com/iris-contrib/examples">learn</a>,  while it's highly customizable. <br/>

Its only sin?
To be the <b>fastest</b> web framework was ever published under open-source circumstances.<br/>
<i>Companies' offers were rejected, free forever.</i><br/><br/>

<b>Enjoy</b> this framework now, without hesitation.<br/>
</p>



Quick Start
-----------

The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.7.

```bash
$ go get -u github.com/kataras/iris
```


> Production? Install the [stable version](https://github.com/kataras/iris/tree/4.0.0#versioning) instead
```bash
$ go get -u gopkg.in/kataras/iris.v4
```

### Hello, World!

```sh
$ cat helloworld.go
```

```go
package main

import "github.com/kataras/iris"

func main(){

  iris.Get("/", func(c *iris.Context){
    c.HTML(iris.StatusOK, "<h1> Hello World! </h1>")
  })


  iris.Listen(":8080")
}

```

```sh
$ go run helloworld.go
```

Open your browser at http://localhost:8080 and you should see the `Hello World!`.

### New

```go
// New with default configuration
app := iris.New()

app.Listen(....)

// New with configuration struct
app := iris.New(iris.Configuration{ DisablePathEscape: true})

app.Listen(...)

// Default station
iris.Listen(...)

// Default station with custom configuration
iris.Config.DisablePathEscape = true

iris.Listen(...)
```

### Listening
`Serve(ln net.Listener) error`
```go
ln, err := net.Listen("tcp4", ":8080")
if err := iris.Serve(ln); err != nil {
   panic(err)
}
```
`Listen(addr string)`
```go
iris.Listen(":8080")
```
`ListenTLS(addr string, certFile, keyFile string)`
```go
iris.ListenTLS(":8080", "./ssl/mycert.cert", "./ssl/mykey.key")
```
`ListenLETSENCRYPT(addr string, cacheFileOptional ...string)`
```go
iris.ListenLETSENCRYPT("mydomain.com")
```
```go
iris.Serve(iris.LETSENCRYPTPROD("myproductionwebsite.com"))
```

And

```go
ListenUNIX(addr string, mode os.FileMode)
Close() error
Reserve() error
IsRunning() bool
```

### Routing

```go
iris.Get("/products/:id", getProduct)
iris.Post("/products", saveProduct)
iris.Put("products/:id", editProduct)
iris.Delete("/products/:id", deleteProduct)
```

And

```go
iris.Patch("", ...)
iris.Connect("", ...)
iris.Options("", ...)
iris.Trace("", ...)
```

### Path Parameters

```go
func getProduct(ctx *iris.Context){
  // Get id from path '/products/:id'
  id := ctx.Param("id")
}

```

### Query Parameters

`/details?color=blue&weight=20`

```go
func details(ctx *iris.Context){
  color:= ctx.URLParam("color")
  weight:= ctx.URLParamInt("weight")
}

```

### Form `application/x-www-form-urlencoded`

`METHOD: POST | PATH: /save`

name | value
:--- | :---
name | Gerasimos Maropoulos
email | kataras2006@homail.com


```go
func save(ctx *iris.Context) {
	// Get name and email
	name := ctx.FormValueString("name")
	email := ctx.FormValueString("email")
}
```

### Form `multipart/form-data`

`POST` `/save`

name | value
:--- | :---
name | Gerasimos Maropoulos
email | kataras2006@hotmail.com
avatar | avatar

```go
func save(ctx *iris.Context)  {
	// Get name and email
	name := ctx.FormValueString("name")
	email := ctx.FormValueString("email")
	// Get avatar
	avatar, err := ctx.FormFile("avatar")
	if err != nil {
       ctx.EmitError(iris.StatusInternalServerError)
       return
	}

	// Source
	src, err := avatar.Open()
	if err != nil {
       ctx.EmitError(iris.StatusInternalServerError)
       return
	}
	defer src.Close()

	// Destination
	dst, err := os.Create(avatar.Filename)
	if err != nil {
       ctx.EmitError(iris.StatusInternalServerError)
       return
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
       ctx.EmitError(iris.StatusInternalServerError)
       return
	}

	ctx.HTML(iris.StatusOK, "<b>Thanks!</b>")
}
```

### Handling Request


- Bind `JSON` or `XML` or `form` payload into Go struct based on `Content-Type` request header.
- Render response as `JSON` or `XML` with status code.

```go
type User struct {
	Name  string `json:"name" xml:"name" form:"name"`
	Email string `json:"email" xml:"email" form:"email"`
}

iris.Post("/users", func(ctx *iris.Context) {
	u := new(User)
	if err := ctx.ReadJSON(u); err != nil {
       ctx.EmitError(iris.StatusInternalServerError)
       return
	}
	ctx.JSON(iris.StatusCreated, u)
   // or
   // ctx.XML(iris.StatusCreated, u)
   // ctx.JSONP(...)
   // ctx.HTML(iris.StatusCreated, "<b>Hi "+u.Name+"</b>")
   // ctx.Markdown(iris.StatusCreated, "## Name: "+u.Name)
})
```


| Name        | Description           | Usage  |
| ------------------|:---------------------:|-------:|
| [JSON ](https://github.com/kataras/go-serializer/tree/master/json)      | JSON Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/json_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/serialize_engines/json_2/main.go), [book section](https://docs.iris-go.com/serialize-engines.html)
| [JSONP ](https://github.com/kataras/go-serializer/tree/master/jsonp)      | JSONP Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/jsonp_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/serialize_engines/jsonp_2/main.go), [book section](https://docs.iris-go.com/serialize-engines.html)
| [XML ](https://github.com/kataras/go-serializer/tree/master/xml)      | XML Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/xml_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/serialize_engines/xml_2/main.go), [book section](https://docs.iris-go.com/serialize-engines.html)
| [Markdown ](https://github.com/kataras/go-serializer/tree/master/markdown)      | Markdown Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/markdown_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/serialize_engines/markdown_2/main.go), [book section](https://docs.iris-go.com/serialize-engines.html)
| [Text](https://github.com/kataras/go-serializer/tree/master/text)      | Text Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/text_1/main.go), [book section](https://docs.iris-go.com/serialize-engines.html)
| [Binary Data ](https://github.com/kataras/go-serializer/tree/master/data)      | Binary Data Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/data_1/main.go), [book section](https://docs.iris-go.com/serialize-engines.html)


### HTTP Errors

You can define your own handlers when http error occurs.

```go
package main

import (
	"github.com/kataras/iris"
)

func main() {

	iris.OnError(iris.StatusInternalServerError, func(ctx *iris.Context) {
        ctx.Write("CUSTOM 500 INTERNAL SERVER ERROR PAGE")
		// or ctx.Render, ctx.HTML any render method you want
		ctx.Log("http status: 500 happened!")
	})

	iris.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.Write("CUSTOM 404 NOT FOUND ERROR PAGE")
		ctx.Log("http status: 404 happened!")
	})

	// emit the errors to test them
	iris.Get("/500", func(ctx *iris.Context) {
		ctx.EmitError(iris.StatusInternalServerError) // ctx.Panic()
	})

	iris.Get("/404", func(ctx *iris.Context) {
		ctx.EmitError(iris.StatusNotFound) // ctx.NotFound()
	})

	iris.Listen(":80")

}


```

### Static Content

Serve files or directories, use the correct for your case, if you don't know which one, just use the `Static(relative string, systemPath string, stripSlashes int)`.

```go
// StaticHandler returns a HandlerFunc to serve static system directory
// Accepts 5 parameters
//
// first param is the systemPath (string)
// Path to the root directory to serve files from.
//
// second is the stripSlashes (int) level
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
//
// third is the compress (bool)
// Transparently compresses responses if set to true.
//
// The server tries minimizing CPU usage by caching compressed files.
// It adds FSCompressedFileSuffix suffix to the original file name and
// tries saving the resulting compressed file under the new file name.
// So it is advisable to give the server write access to Root
// and to all inner folders in order to minimze CPU usage when serving
// compressed responses.
//
// fourth is the generateIndexPages (bool)
// Index pages for directories without files matching IndexNames
// are automatically generated if set.
//
// Directory index generation may be quite slow for directories
// with many files (more than 1K), so it is discouraged enabling
// index pages' generation for such directories.
//
// fifth is the indexNames ([]string)
// List of index file names to try opening during directory access.
//
// For example:
//
//     * index.html
//     * index.htm
//     * my-super-index.xml
//
StaticHandler(systemPath string, stripSlashes int, compress bool,
                  generateIndexPages bool, indexNames []string) HandlerFunc

// Static registers a route which serves a system directory
// this doesn't generates an index page which list all files
// no compression is used also, for these features look at StaticFS func
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
Static(relative string, systemPath string, stripSlashes int)

// StaticFS registers a route which serves a system directory
// generates an index page which list all files
// uses compression which file cache, if you use this method it will generate compressed files also
// think this function as small fileserver with http
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
StaticFS(relative string, systemPath string, stripSlashes int)

// StaticWeb same as Static but if index.html e
// xists and request uri is '/' then display the index.html's contents
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
// third parameter is the level (int) of stripSlashes
// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
// * stripSlashes = 2, original path: "/foo/bar", result: ""
StaticWeb(relative string, systemPath string, stripSlashes int)

// StaticServe serves a directory as web resource
// it's the simpliest form of the Static* functions
// Almost same usage as StaticWeb
// accepts only one required parameter which is the systemPath
// (the same path will be used to register the GET&HEAD routes)
// if the second parameter is empty, otherwise the requestPath is the second parameter
// it uses gzip compression (compression on each request, no file cache)
StaticServe(systemPath string, requestPath ...string)

```

```go
iris.Static("/public", "./static/assets/", 1)
//-> /public/assets/favicon.ico
```

```go
iris.StaticFS("/ftp", "./myfiles/public", 1)
```

```go
iris.StaticWeb("/","./my_static_html_website", 1)
```

```go
StaticServe(systemPath string, requestPath ...string)
```

#### Manual static file serving

```go
// ServeFile serves a view file, to send a file
// to the client you should use the SendFile(serverfilename,clientfilename)
// receives two parameters
// filename/path (string)
// gzipCompression (bool)
//
// You can define your own "Content-Type" header also, after this function call
ServeFile(filename string, gzipCompression bool) error
```

Serve static individual file

```go

iris.Get("/txt", func(ctx *iris.Context) {
    ctx.ServeFile("./myfolder/staticfile.txt", false)
}
```

### Templates

**HTML Template Engine, defaulted**


```html
<!-- file ./templates/hi.html -->

<html>
<head>
<title>Hi Iris</title>
</head>
<body>
	<h1>Hi {{.Name}}
</body>
</html>
```

```go
// file ./main.go
package main

import "github.com/kataras/iris"

func main() {
	iris.Config.IsDevelopment = true // this will reload the templates on each request
	iris.Get("/hi", hi)
	iris.Listen(":8080")
}

func hi(ctx *iris.Context) {
	ctx.MustRender("hi.html", struct{ Name string }{Name: "iris"})
}

```

| Name        | Description           | Usage  |
| ------------------|:---------------------:|-------:|
| [HTML/Default Engine ](https://github.com/kataras/go-template/tree/master/html)      | HTML Template Engine (Default)                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_html_0/main.go), [book section](https://docs.iris-go.com/template-engines.html)
| [Django Engine ](https://github.com/kataras/go-template/tree/master/django)      | Django Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_django_1/main.go), [book section](https://docs.iris-go.com/template-engines.html)
| [Pug/Jade Engine ](https://github.com/kataras/go-template/tree/master/pug)      | Pug Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_pug_1/main.go), [book section](https://docs.iris-go.com/template-engines.html)
| [Handlebars Engine ](https://github.com/kataras/go-template/tree/master/handlebars)      | Handlebars Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_handlebars_1/main.go), [book section](https://docs.iris-go.com/template-engines.html)
| [Amber Engine ](https://github.com/kataras/go-template/tree/master/amber)      | Amber Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_amber_1/main.go), [book section](https://docs.iris-go.com/template-engines.html)
| [Markdown Engine ](https://github.com/kataras/go-template/tree/master/markdown)      | Markdown Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_markdown_1/main.go), [book section](https://docs.iris-go.com/template-engines.html)

> Each section of the README has its own - more advanced - subject on the book, so be sure to check book for any further research

[Read more](https://docs.iris-go.com/template-engines.html)

### Middleware ecosystem


```go

import (
  "github.com/iris-contrib/middleware/logger"
  "github.com/iris-contrib/middleware/cors"
  "github.com/iris-contrib/middleware/basicauth"
)
// Root level middleware
iris.Use(logger.New())
iris.Use(cors.Default())

// Group level middleware
authConfig := basicauth.Config{
    Users:      map[string]string{"myusername": "mypassword", "mySecondusername": "mySecondpassword"},
    Realm:      "Authorization Required", // if you don't set it it's "Authorization Required"
    ContextKey: "mycustomkey",            // if you don't set it it's "user"
    Expires:    time.Duration(30) * time.Minute,
}

authentication := basicauth.New(authConfig)

g := iris.Party("/admin")
g.Use(authentication)

// Route level middleware
logme := func(ctx *iris.Context)  {
		println("request to /products")
		ctx.Next()
}
iris.Get("/products", logme, func(ctx *iris.Context) {
	 ctx.Text(iris.StatusOK, "/products")
})
```


| Name        | Description           | Usage  |
| ------------------|:---------------------:|-------:|
| [Basicauth Middleware ](https://github.com/iris-contrib/middleware/tree/master/basicauth)      | HTTP Basic authentication                  |[example 1](https://github.com/iris-contrib/examples/blob/master/middleware_basicauth_1/main.go), [example 2](https://github.com/iris-contrib/examples/blob/master/middleware_basicauth_2/main.go), [book section](https://docs.iris-go.com/basic-authentication.html)  |
| [JWT Middleware ](https://github.com/iris-contrib/middleware/tree/master/jwt)      | JSON Web Tokens                  |[example ](https://github.com/iris-contrib/examples/blob/master/middleware_jwt/main.go), [book section](https://docs.iris-go.com/jwt.html)  |
| [Cors Middleware ](https://github.com/iris-contrib/middleware/tree/master/cors)      | Cross Origin Resource Sharing W3 specification   | [how to use ](https://github.com/iris-contrib/middleware/tree/master/cors#how-to-use)  |
| [Secure Middleware ](https://github.com/iris-contrib/middleware/tree/master/secure) |  Facilitates some quick security wins      | [example](https://github.com/iris-contrib/examples/blob/master/middleware_secure/main.go)  |
| [I18n Middleware ](https://github.com/iris-contrib/middleware/tree/master/i18n)      | Simple internationalization       | [example](https://github.com/iris-contrib/examples/tree/master/middleware_internationalization_i18n), [book section](https://docs.iris-go.com/middleware-internationalization-and-localization.html)  |
| [Recovery Middleware ](https://github.com/iris-contrib/middleware/tree/master/recovery) | Safety recover the station from panic       | [example](https://github.com/iris-contrib/examples/blob/master/middleware_recovery/main.go)  |
| [Logger Middleware ](https://github.com/iris-contrib/middleware/tree/master/logger)      | Logs every request       | [example](https://github.com/iris-contrib/examples/blob/master/middleware_logger/main.go), [book section](https://docs.iris-go.com/logger.html)  |
| [Profile Middleware ](https://github.com/iris-contrib/middleware/tree/master/pprof)      | Http profiling for debugging    | [example](https://github.com/iris-contrib/examples/blob/master/middleware_pprof/main.go)  |
| [Editor Plugin](https://github.com/iris-contrib/plugin/tree/master/editor)      | Alm-tools, a typescript online IDE/Editor | [book section](https://docs.iris-go.com/plugin-editor.html) |
| [Typescript Plugin](https://github.com/iris-contrib/plugin/tree/master/typescript)      | Auto-compile client-side typescript files      |   [book section](https://docs.iris-go.com/plugin-typescript.html) |
| [OAuth,OAuth2 Plugin](https://github.com/iris-contrib/plugin/tree/master/oauth) |  User Authentication was never be easier, supports >27 providers |    [example](https://github.com/iris-contrib/examples/tree/master/plugin_oauth_oauth2), [book section](https://docs.iris-go.com/plugin-oauth.html) |
| [Iris control Plugin](https://github.com/iris-contrib/plugin/tree/master/iriscontrol) |   Basic (browser-based) control over your Iris station |    [example](https://github.com/iris-contrib/examples/blob/master/plugin_iriscontrol/main.go), [book section](https://docs.iris-go.com/plugin-iriscontrol.html) |

### Sessions
If you notice a bug or issue [post it here](https://github.com/kataras/go-sessions).


- Cleans the temp memory when a session is idle, and re-allocates it to the temp memory when it's necessary.
The most used sessions are optimized to be in the front of the memory's list.

- Supports any type of database, currently only [Redis](https://github.com/kataras/go-sessions/tree/master/sessiondb/redis) and [LevelDB](https://github.com/kataras/go-sessions/tree/master/sessiondb/leveldb).


**A session can be defined as a server-side storage of information that is desired to persist throughout the user's interaction with the web application**.

Instead of storing large and constantly changing data via cookies in the user's browser (i.e. CookieStore),
**only a unique identifier is stored on the client side** called a "session id".
This session id is passed to the web server on every request.
The web application uses the session id as the key for retrieving the stored data from the database/memory. The session data is then available inside the iris.Context.

```go
iris.Get("/", func(ctx *iris.Context) {
		ctx.Write("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})

	iris.Get("/set", func(ctx *iris.Context) {

		//set session values
		ctx.Session().Set("name", "iris")

		//test if setted here
		ctx.Write("All ok session setted to: %s", ctx.Session().GetString("name"))
	})

	iris.Get("/get", func(ctx *iris.Context) {
		// get a specific key as a string.
		// returns an empty string if the key was not found.
		name := ctx.Session().GetString("name")

		ctx.Write("The name on the /set was: %s", name)
	})

	iris.Get("/delete", func(ctx *iris.Context) {
		// delete a specific key
		ctx.Session().Delete("name")
	})

	iris.Get("/clear", func(ctx *iris.Context) {
		// removes all entries
		ctx.Session().Clear()
	})

	iris.Get("/destroy", func(ctx *iris.Context) {
		// destroy/removes the entire session and cookie
		ctx.SessionDestroy()
		ctx.Log("You have to refresh the page to completely remove the session (on browsers), so the name should NOT be empty NOW, is it?\n ame: %s\n\nAlso check your cookies in your browser's cookies, should be no field for localhost/127.0.0.1 (or whatever you use)", ctx.Session().GetString("name"))
		ctx.Write("You have to refresh the page to completely remove the session (on browsers), so the name should NOT be empty NOW, is it?\nName: %s\n\nAlso check your cookies in your browser's cookies, should be no field for localhost/127.0.0.1 (or whatever you use)", ctx.Session().GetString("name"))
	})

	iris.Listen(":8080")

```

> Each section of the README has its own - more advanced - subject on the book, so be sure to check book for any further research

[Read more](https://docs.iris-go.com/package-sessions.html)

### Websockets

```go
// file ./main.go
package main

import (
    "fmt"
    "github.com/kataras/iris"
)

type clientPage struct {
    Title string
    Host  string
}

func main() {
    iris.Static("/js", "./static/js", 1)

    iris.Get("/", func(ctx *iris.Context) {
        ctx.Render("client.html", clientPage{"Client Page", ctx.HostString()})
    })

    // the path at which the websocket client should register itself to
    iris.Config.Websocket.Endpoint = "/my_endpoint"

    var myChatRoom = "room1"
    iris.Websocket.OnConnection(func(c iris.WebsocketConnection) {

        c.Join(myChatRoom)

        c.On("chat", func(message string) {
            // to all except this connection ->
            //c.To(iris.Broadcast).Emit("chat", "Message from: "+c.ID()+"-> "+message)

            // to the client ->
            //c.Emit("chat", "Message from myself: "+message)

            // send the message to the whole room,
            // all connections which are inside this room will receive this message
            c.To(myChatRoom).Emit("chat", "From: "+c.ID()+": "+message)
        })

        c.OnDisconnect(func() {
            fmt.Printf("\nConnection with ID: %s has been disconnected!", c.ID())
        })
    })

    iris.Listen(":8080")
}

```

```js
// file js/chat.js
var messageTxt;
var messages;

$(function () {

    messageTxt = $("#messageTxt");
    messages = $("#messages");


    ws = new Ws("ws://" + HOST + "/my_endpoint");
    ws.OnConnect(function () {
        console.log("Websocket connection enstablished");
    });

    ws.OnDisconnect(function () {
        appendMessage($("<div><center><h3>Disconnected</h3></center></div>"));
    });

    ws.On("chat", function (message) {
        appendMessage($("<div>" + message + "</div>"));
    })

    $("#sendBtn").click(function () {
        //ws.EmitMessage(messageTxt.val());
        ws.Emit("chat", messageTxt.val().toString());
        messageTxt.val("");
    })

})


function appendMessage(messageDiv) {
    var theDiv = messages[0]
    var doScroll = theDiv.scrollTop == theDiv.scrollHeight - theDiv.clientHeight;
    messageDiv.appendTo(messages)
    if (doScroll) {
        theDiv.scrollTop = theDiv.scrollHeight - theDiv.clientHeight;
    }
}
```

```html
<!-- file templates/client.html -->
<html>

<head>
    <title>My iris-ws</title>
</head>

<body>
    <div id="messages" style="border-width:1px;border-style:solid;height:400px;width:375px;">

    </div>
    <input type="text" id="messageTxt" />
    <button type="button" id="sendBtn">Send</button>
    <script type="text/javascript">
        var HOST = {{.Host}}
    </script>
    <script src="js/vendor/jquery-2.2.3.min.js" type="text/javascript"></script>
    <!-- /iris-ws.js is served automatically by the server -->
    <script src="/iris-ws.js" type="text/javascript"></script>
    <!-- -->
    <script src="js/chat.js" type="text/javascript"></script>
</body>

</html>

```

View a working example by navigating [here](https://github.com/iris-contrib/examples/tree/master/websocket) and if you need more than one websocket server [click here](https://github.com/iris-contrib/examples/tree/master/websocket_unlimited_servers).

> Each section of the README has its own - more advanced - subject on the book, so be sure to check book for any further research

[Read more](https://docs.iris-go.com/package-websocket.html)

### Need help?

 <a href="https://www.gitbook.com/book/kataras/iris/details"><img align="right" width="125" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_4.jpg"></a>


 - The most important is to read [the practical guide](https://docs.iris-go.com/).

 - Explore & download the [examples](https://github.com/iris-contrib/examples).

 - [HISTORY.md](https://github.com//kataras/iris/tree/master/HISTORY.md) file is your best friend.

#### FAQ

Explore [these questions](https://github.com/kataras/iris/issues?q=label%3Aquestion) or navigate to the [community chat][Chat].


Support
------------

Hi, my name is Gerasimos Maropoulos and I'm the author of this project, let me put a few words about me.

I started to design iris the night of the 13 March 2016, some weeks later, iris started to became famous and I have to fix many issues and implement new features, but I didn't have time to work on Iris because I had a part time job and the (software engineering) colleague which I studied.

I wanted to make iris' users proud of the framework they're using, so I decided to interrupt my studies and colleague, two days later I left from my part time job also.

Today I spend all my days and nights coding for Iris, and I'm happy about this, therefore I have zero incoming value.

- :star: the project
- [Donate](https://github.com/kataras/iris/blob/master/DONATIONS.md)
- :earth_americas: spread the word
- [Contribute](#contributing) to the project



Philosophy
------------

The Iris philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs. Keep note that, today, iris is faster than nginx itself.

Iris does not force you to use any specific ORM or template engine. With support for the most used template engines, you can quickly craft the perfect application.



Benchmarks
------------


This Benchmark test aims to compare the whole HTTP request processing between Go web frameworks.


![Benchmark Wizzard July 21, 2016- Processing Time Horizontal Graph](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

**The results have been updated on July 21, 2016**

The second is an article I just found(**3 October 2016**) which compares Iris vs Nginx vs Nodejs express, it was written in Thai, so I used google to translate it to english.

![Iris vs Nginx vs Nodejs express](https://github.com/iris-contrib/website/raw/gh-pages/assets/03Oct2016/iris_1.png)

The results showed that the req / sec iris do best at around 70k-50k, followed by nginx and nginx-php-fpm and nodejs respectively.
The error golang-iris and nginx work equally, followed by the final nginx and php-fpm at a ratio of 1: 1.

You can read the full article [here](https://translate.google.com/translate?sl=auto&tl=en&js=y&prev=_t&hl=en&ie=UTF-8&u=http://webcache.googleusercontent.com/search?q=cache:https%3A%2F%2Fwww.ntossapo.me%2F2016%2F08%2F13%2Fnginx-vs-nginx-php-fpm-vs-go-iris-vs-express-with-wrk%2F&edit-text=&act=url).


Testing
------------

I recommend writing your API tests using this new library, [httpexpect](https://github.com/gavv/httpexpect) which supports Iris and fasthttp now, after my request [here](https://github.com/gavv/httpexpect/issues/2). You can find Iris examples [here](https://github.com/gavv/httpexpect/blob/master/example/iris_test.go), [here](https://github.com/kataras/iris/blob/master/http_test.go) and [here](https://github.com/kataras/iris/blob/master/context_test.go).

Versioning
------------

Current: **v5.0.2**

Stable: **[v4 LTS](https://github.com/kataras/iris/tree/4.0.0#versioning)**


Todo
------------

- [ ] Server-side React render, as requested [here](https://github.com/kataras/iris/issues/503)

Iris is a **Community-Driven** Project, waiting for your suggestions and [feature requests](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=label%3A%22feature%20request%22)!

People
------------

The author of Iris is [@kataras](https://github.com/kataras).

If **you**'re willing to donate and you can afford the cost, feel **free** to navigate to the [DONATIONS PAGE](https://github.com/kataras/iris/blob/master/DONATIONS.md).


Contributing
------------

Iris is the work of hundreds of the community's [feature requests](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=%20label%3A%22feature%20request%22%20) and [reports](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=label%3Abug). I appreciate your help!

If you are interested in contributing to the Iris project, please see the document [CONTRIBUTING](https://github.com/kataras/iris/blob/master/.github/CONTRIBUTING.md).

##### Note that I do not accept pull requests and that I use the issue tracker for bug reports and proposals only. Please ask questions on the [https://kataras.rocket.chat/channel/iris][Chat] or [http://stackoverflow.com/](http://stackoverflow.com).

License
------------

Copyright &copy; 2016 Gerasimos Maropoulos <makis@ideopod.com>
This work is free. Unless otherwise noted, the iris source files are distributed under the
terms of the Do What The Fuck You Want To Public License, Version 2,
as published by [Sam Hocevar](http://www.wtfpl.net/author/admin/). Navigate [here](http://www.wtfpl.net/about/) for more details.



[Chat]: https://kataras.rocket.chat/channel/iris
