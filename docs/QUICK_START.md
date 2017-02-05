Quick Start
-----------

```bash
go get -u github.com/kataras/iris/iris
```

```sh
cat app.go
```

```go
package iris_test

import (
	"github.com/kataras/go-template/html"
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()
	// 6 template engines are supported out-of-the-box:
	//
	// - standard html/template
	// - amber
	// - django
	// - handlebars
	// - pug(jade)
	// - markdown
	//
	// Use the html standard engine for all files inside "./views" folder with extension ".html"
	// Defaults to:
	app.UseTemplate(html.New()).Directory("./views", ".html")

	// http://localhost:6111
	// Method: "GET"
	// Render ./views/index.html
	app.Get("/", func(ctx *iris.Context) {
		ctx.Render("index.html", nil)
	})

	// Group routes, optionally: share middleware, template layout and custom http errors.
	userAPI := app.Party("/users", userAPIMiddleware).
		Layout("layouts/userLayout.html")
	{
		// Fire userNotFoundHandler when Not Found
		// inside http://localhost:6111/users/*anything
		userAPI.OnError(404, userNotFoundHandler)

		// http://localhost:6111/users
		// Method: "GET"
		userAPI.Get("/", getAllHandler)

		// http://localhost:6111/users/42
		// Method: "GET"
		userAPI.Get("/:id", getByIDHandler)

		// http://localhost:6111/users
		// Method: "POST"
		userAPI.Post("/", saveUserHandler)
	}

	// Start the server at 0.0.0.0:6111
	app.Listen(":6111")
}

func getByIDHandler(ctx *iris.Context) {
	// take the :id from the path, parse to integer
	// and set it to the new userID local variable.
	userID, _ := ctx.ParamInt("id")

	// userRepo, imaginary database service <- your only job.
	user := userRepo.GetByID(userID)

	// send back a response to the client,
	// .JSON: content type as application/json; charset="utf-8"
	// iris.StatusOK: with 200 http status code.
	//
	// send user as it is or make use of any json valid golang type,
	// like the iris.Map{"username" : user.Username}.
	ctx.JSON(iris.StatusOK, user)
}

```

> TIP: $ iris run main.go to enable hot-reload on .go source code changes.

> TIP: iris.Config.IsDevelopment = true to monitor the changes you make in the templates.

> TIP:  Want to change the default Router's behavior to something else like Gorilla's Mux?
Go [there](https://github.com/iris-contrib/examples/tree/master/plugin_gorillamux) to learn how.


### New

```go
// New with default configuration
app := iris.New()

app.Listen(....)

// New with configuration struct
app := iris.New(iris.Configuration{ IsDevelopment: true})

app.Listen(...)

// Default station
iris.Listen(...)

// Default station with custom configuration
// view the whole configuration at: ./configuration.go
iris.Config.IsDevelopment = true
iris.Config.Charset = "UTF-8"

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
  color := ctx.URLParam("color")
  weight,_ := ctx.URLParamInt("weight")
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
	name := ctx.FormValue("name")
	email := ctx.FormValue("email")
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
	name := ctx.FormValue("name")
	email := ctx.FormValue("email")
	// Get avatar
	avatar, info, err := ctx.FormFile("avatar")
	if err != nil {
       ctx.EmitError(iris.StatusInternalServerError)
       return
	}

	defer avatar.Close()

	// Destination
	dst, err := os.Create(avatar.Filename)
	if err != nil {
       ctx.EmitError(iris.StatusInternalServerError)
       return
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, avatar); err != nil {
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
    ctx.Writef("CUSTOM 500 INTERNAL SERVER ERROR PAGE")
		// or ctx.Render, ctx.HTML any render method you want
		ctx.Log("http status: 500 happened!")
	})

	iris.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.Writef("CUSTOM 404 NOT FOUND ERROR PAGE")
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

Serve files or directories, use the correct for your case, if you don't know which one, just use the `StaticWeb(reqPath string, systemPath string)`.

```go
// Favicon serves static favicon
// accepts 2 parameters, second is optional
// favPath (string), declare the system directory path of the __.ico
// requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
// you can declare your own path if you have more than one favicon (desktop, mobile and so on)
//
// this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico (nothing special that you can't handle by yourself)
// Note that you have to call it on every favicon you have to serve automatically (desktop, mobile and so on)
//
// panics on error
Favicon(favPath string, requestPath ...string) RouteNameFunc

// StaticHandler returns a new Handler which serves static files
StaticHandler(reqPath string, systemPath string, showList bool, enableGzip bool) HandlerFunc

// StaticWeb same as Static but if index.html e
// xists and request uri is '/' then display the index.html's contents
// accepts three parameters
// first parameter is the request url path (string)
// second parameter is the system directory (string)
StaticWeb(reqPath string, systemPath string) RouteNameFunc

// StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
// First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
// Second parameter is the (virtual) directory path, for example "./assets"
// Third parameter is the Asset function
// Forth parameter is the AssetNames function
//
// For more take a look at the
// example: https://github.com/iris-contrib/examples/tree/master/static_files_embedded
StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) RouteNameFunc

// StaticContent serves bytes, memory cached, on the reqPath
// a good example of this is how the websocket server uses that to auto-register the /iris-ws.js
StaticContent(reqPath string, cType string, content []byte) RouteNameFunc

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
iris.StaticWeb("/public", "./static/assets/")
//-> /public/assets/favicon.ico
```

```go
iris.StaticWeb("/","./my_static_html_website")
```

```go
context.StaticServe(systemPath string, requestPath ...string)
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
context.ServeFile(filename string, gzipCompression bool) error
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
| [LoggerZap Middleware ](https://github.com/iris-contrib/middleware/tree/master/loggerzap)      | Logs every request using zap | [example](https://github.com/iris-contrib/examples/blob/master/middleware_logger/main.go), [book section](https://docs.iris-go.com/logger.html)  |
| [Profile Middleware ](https://github.com/iris-contrib/middleware/tree/master/pprof)      | Http profiling for debugging    | [example](https://github.com/iris-contrib/examples/blob/master/middleware_pprof/main.go)  |
| [Editor Plugin](https://github.com/iris-contrib/plugin/tree/master/editor)      | Alm-tools, a typescript online IDE/Editor | [book section](https://docs.iris-go.com/plugin-editor.html) |
| [Typescript Plugin](https://github.com/iris-contrib/plugin/tree/master/typescript)      | Auto-compile client-side typescript files      |   [book section](https://docs.iris-go.com/plugin-typescript.html) |
| [OAuth,OAuth2 Plugin](https://github.com/iris-contrib/plugin/tree/master/oauth) |  User Authentication was never be easier, supports >27 providers |    [example](https://github.com/iris-contrib/examples/tree/master/plugin_oauth_oauth2), [book section](https://docs.iris-go.com/plugin-oauth.html) |
| [Iris control Plugin](https://github.com/iris-contrib/plugin/tree/master/iriscontrol) |   Basic (browser-based) control over your Iris station |    [example](https://github.com/iris-contrib/examples/blob/master/plugin_iriscontrol/main.go), [book section](https://docs.iris-go.com/plugin-iriscontrol.html) |

> NOTE: All net/http handlers and middleware that already created by other go developers are also compatible with Iris, even if they are not be documented here, read more [here](https://github.com/iris-contrib/middleware#can-i-use-standard-nethttp-handler-with-iris).


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
		ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
	})

	iris.Get("/set", func(ctx *iris.Context) {

		//set session values
		ctx.Session().Set("name", "iris")

		//test if setted here
		ctx.Writef("All ok session setted to: %s", ctx.Session().GetString("name"))
	})

	iris.Get("/get", func(ctx *iris.Context) {
		// get a specific key as a string.
		// returns an empty string if the key was not found.
		name := ctx.Session().GetString("name")

		ctx.Writef("The name on the /set was: %s", name)
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
		ctx.Writef("You have to refresh the page to completely remove the session (on browsers), so the name should NOT be empty NOW, is it?\nName: %s\n\nAlso check your cookies in your browser's cookies, should be no field for localhost/127.0.0.1 (or whatever you use)", ctx.Session().GetString("name"))
	})

	iris.Listen(":8080")

```

- `iris.DestroySessionByID(string)`

```go
// DestroySessionByID removes the session entry
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
//
// It's safe to use it even if you are not sure if a session with that id exists.
DestroySessionByID(string)
```

- `iris.DestroyAllSessions()`

```go
// DestroyAllSessions removes all sessions
// from the server-side memory (and database if registered).
// Client's session cookie will still exist but it will be reseted on the next request.
DestroyAllSessions()
```

> Each section of the README has its own - more advanced - subject on the book, so be sure to check book for any further research

[Read more](https://docs.iris-go.com/package-sessions.html)

### Websockets

Server configuration

```go
iris.Config.Websocket{
	// WriteTimeout time allowed to write a message to the connection.
	// Default value is 15 * time.Second
	WriteTimeout time.Duration
	// PongTimeout allowed to read the next pong message from the connection
	// Default value is 60 * time.Second
	PongTimeout time.Duration
	// PingPeriod send ping messages to the connection with this period. Must be less than PongTimeout
	// Default value is (PongTimeout * 9) / 10
	PingPeriod time.Duration
	// MaxMessageSize max message size allowed from connection
	// Default value is 1024
	MaxMessageSize int64
	// BinaryMessages set it to true in order to denotes binary data messages instead of utf-8 text
	// see https://github.com/kataras/iris/issues/387#issuecomment-243006022 for more
	// Defaults to false
	BinaryMessages bool
	// Endpoint is the path which the websocket server will listen for clients/connections
	// Default value is empty string, if you don't set it the Websocket server is disabled.
	Endpoint string
	// ReadBufferSize is the buffer size for the underline reader
	ReadBufferSize int
	// WriteBufferSize is the buffer size for the underline writer
	WriteBufferSize int
	// Error specifies the function for generating HTTP error responses.
	//
	// The default behavior is to store the reason in the context (ctx.Set(reason)) and fire any custom error (ctx.EmitError(status))
	Error func(ctx *Context, status int, reason error)
	// CheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, the host in the Origin header must not be set or
	// must match the host of the request.
	//
	// The default behavior is to allow all origins
	// you can change this behavior by setting the iris.Config.Websocket.CheckOrigin = iris.WebsocketCheckSameOrigin
	CheckOrigin func(r *http.Request) bool
	// IDGenerator used to create (and later on, set)
	// an ID for each incoming websocket connections (clients).
	// If empty then the ID is generated by the result of 64
	// random combined characters
	IDGenerator func(r *http.Request) string
}

```

Connection's methods

```go
ID() string

Request() *http.Request

// Receive from the client
On("anyCustomEvent", func(message string) {})
On("anyCustomEvent", func(message int){})
On("anyCustomEvent", func(message bool){})
On("anyCustomEvent", func(message anyCustomType){})
On("anyCustomEvent", func(){})

// Receive a native websocket message from the client
// compatible without need of import the iris-ws.js to the .html
OnMessage(func(message []byte){})

// Send to the client
Emit("anyCustomEvent", string)
Emit("anyCustomEvent", int)
Emit("anyCustomEvent", bool)
Emit("anyCustomEvent", anyCustomType)

// Send native websocket messages
// with config.BinaryMessages = true
// useful when you use proto or something like this.
EmitMessage([]byte("anyMessage"))

// Send to specific client(s)
To("otherConnectionId").Emit/EmitMessage...
To("anyCustomRoom").Emit/EmitMessage...

// Send to all opened connections/clients
To(websocket.All).Emit/EmitMessage...

// Send to all opened connections/clients EXCEPT this client
To(websocket.Broadcast).Emit/EmitMessage...

// Rooms, group of connections/clients
Join("anyCustomRoom")
Leave("anyCustomRoom")


// Fired when the connection is closed
OnDisconnect(func(){})

// Force-disconnect the client from the server-side
Disconnect() error
```

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
        ctx.Render("client.html", clientPage{"Client Page", ctx.Host()})
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



Benchmarks
------------

These benchmarks are for the previous Iris version(1month ago), new benchmarks are coming after the release of the Go version 1.8 in order to include the `Push` feature inside the tests.


This Benchmark test aims to compare the whole HTTP request processing between Go web frameworks.


![Benchmark Wizzard July 21, 2016- Processing Time Horizontal Graph](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

**The results have been updated on July 21, 2016**



Depends on:

- http protocol layer comes from [net/http](https://github.com/golang/go/tree/master/src/net/http), by Go Authors.
- rich and encoded responses support comes from [kataras/go-serializer](https://github.com/kataras/go-serializer/tree/0.0.4), by me.
- template support comes from [kataras/go-template](https://github.com/kataras/go-template/tree/0.0.3), by me.
- gzip support comes from [kataras/go-fs](https://github.com/kataras/go-fs/tree/0.0.5) and the super-fast compression library [klauspost/compress/gzip](https://github.com/klauspost/compress/tree/master/gzip), by me & Klaus Post.
- websockets support comes from [kataras/go-websocket](https://github.com/kataras/go-websocket/tree/0.0.2), by me.
- Base of the parameterized routing algorithm comes from [julienschmidt/httprouter](https://github.com/julienschmidt/httprouter), by Julien Schmidt, with some relative to performance edits by me.
- sessions support comes from [kataras/go-sessions](https://github.com/kataras/go-sessions/tree/0.0.6), by me.
- caching support comes from [geekypanda/httpcache](https://github.com/geekypanda/httpcache/tree/0.0.1), by me and GeekyPanda.
- end-to-end http test APIs comes from [gavv/httpexpect](https://github.com/gavv/httpexpect), by Victor Gaydov.
- hot-reload on source code changes comes from [kataras/rizla](https://github.com/kataras/rizla), by me.
- auto-updater (via github) comes from [kataras/go-fs](https://github.com/kataras/go-fs), by me.
- request body form binder is an [edited version](https://github.com/iris-contrib/formBinder) of the [monoculum/formam](https://github.com/monoculum/formam) library, by Monoculum Organisation.
- all other packages comes from the [Iris Contrib Organisation](https://github.com/iris-contrib) and the [Go standard library](https://github.com/golang/go), by me & The Go Authors.
