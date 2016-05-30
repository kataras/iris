# History

## 3.0.0-alpha.beta -> 3.0.0-beta


- New [iris.API] for easy API declaration, read more [here](https://kataras.gitbooks.io/iris/content/using-handlerapi.html), example [there](https://github.com/iris-contrib/examples/tree/master/api_handler_2).


- Add [example](https://github.com/iris-contrib/examples/tree/master/middleware_basicauth_2) and fix the Basic Authentication middleware

## 3.0.0-alpha.6 -> 3.0.0-alpha.beta

- [Implement feature request to add Globals on the pongo2](https://github.com/kataras/iris/issues/145)

- [Implement feature request for static Favicon ](https://github.com/kataras/iris/issues/141)

- Implement a unique easy only-websocket support:



```go
OnConnection(func(c websocket.Connection){})
```

websocket.Connection
```go

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

// Send via native websocket way, compatible without need of import the iris-ws.js to the .html
EmitMessage([]byte("anyMessage"))

// Send to specific client(s)
To("otherConnectionId").Emit/EmitMessage...
To("anyCustomRoom").Emit/EmitMessage...

// Send to all opened connections/clients
To(websocket.All).Emit/EmitMessage...

// Send to all opened connections/clients EXCEPT this client(c)
To(websocket.NotMe).Emit/EmitMessage...

// Rooms, group of connections/clients
Join("anyCustomRoom")
Leave("anyCustomRoom")


// Fired when the connection is closed
OnDisconnect(func(){})

```

- [Example](https://github.com/iris-contrib/examples/tree/master/websocket)
- [E-book section](https://kataras.gitbooks.io/iris/content/package-websocket.html)


We have some base-config's changed, these configs which are defaulted to true renamed to 'Disable+$oldName'
```go

		// DisablePathCorrection corrects and redirects the requested path to the registed path
		// for example, if /home/ path is requested but no handler for this Route found,
		// then the Router checks if /home handler exists, if yes,
		// (permant)redirects the client to the correct path /home
		//
		// Default is false
		DisablePathCorrection bool

		// DisablePathEscape when is false then its escapes the path, the named parameters (if any).
		// Change to true it if you want something like this https://github.com/kataras/iris/issues/135 to work
		//
		// When do you need to Disable(true) it:
		// accepts parameters with slash '/'
		// Request: http://localhost:8080/details/Project%2FDelta
		// ctx.Param("project") returns the raw named parameter: Project%2FDelta
		// which you can escape it manually with net/url:
		// projectName, _ := url.QueryUnescape(c.Param("project").
		// Look here: https://github.com/kataras/iris/issues/135 for more
		//
		// Default is false
		DisablePathEscape bool

		// DisableLog turn it to true if you want to disable logger,
		// Iris prints/logs ONLY errors, so be careful when you enable it
		DisableLog bool

		// DisableBanner outputs the iris banner at startup
		//
		// Default is false
		DisableBanner bool

```


## 3.0.0-alpha.5 -> 3.0.0-alpha.6

Changes:
	- config/iris.Config().Render.Template.HTMLTemplate.Funcs typeof `[]template.FuncMap` -> `template.FuncMap`


Added:
	- iris.AmberEngine [Amber](https://github.com/eknkc/amber). [View an example](https://github.com/iris-contrib/examples/tree/master/templates_7_html_amber)
	- iris.JadeEngine [Jade](https://github.com/Joker/jade). [View an example](https://github.com/iris-contrib/examples/tree/master/templates_6_html_jade)

Book section [Render/Templates updated](https://kataras.gitbooks.io/iris/content/render_templates.html)



## 3.0.0-alpha.4 -> 3.0.0-alpha.5

- [NoLayout support for particular templates](https://github.com/kataras/iris/issues/130#issuecomment-219754335)
- [Raw Markdown Template Engine](https://kataras.gitbooks.io/iris/content/render_templates.html)
- [Markdown to HTML](https://kataras.gitbooks.io/iris/content/render_rest.html) > `context.Markdown(statusCode int, markdown string)` , `context.MarkdownString(markdown string) (htmlReturn string)`
- [Simplify the plugin registration](https://github.com/kataras/iris/issues/126#issuecomment-219622481)

## 3.0.0-alpha.3 -> 3.0.0-alpha.4

Community suggestions implemented:

- [Request: Rendering html template to string](https://github.com/kataras/iris/issues/130)
	> New RenderString(name string, binding interface{}, layout ...string) added to the Context & the Iris' station (iris.Templates().RenderString)
- [Minify Templates](https://github.com/kataras/iris/issues/129)
	> New config field for minify, defaulted to true: iris.Config().Render.Template.Minify  = true
	> 3.0.0-alpha5+ this has been removed because the minify package has bugs, one of them is this: https://github.com/tdewolff/minify/issues/35.



Bugfixes and enhancements:

- [Static not allowing configuration of `IndexNames`](https://github.com/kataras/iris/issues/128)
- [Processing access error](https://github.com/kataras/iris/issues/125)
- [Invalid header](https://github.com/kataras/iris/issues/123)

## 3.0.0-alpha.2 -> 3.0.0-alpha.3

The only change here is a panic-fix on form bindings. Now **no need to make([]string,0)** before form binding, new example:

```go
 //./main.go

package main

import (
	"fmt"

	"github.com/kataras/iris"
)

type Visitor struct {
	Username string
	Mail     string
	Data     []string `form:"mydata"`
}

func main() {

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Render("form.html", nil)
	})

	iris.Post("/form_action", func(ctx *iris.Context) {
		visitor := Visitor{}
		err := ctx.ReadForm(&visitor)
		if err != nil {
			fmt.Println("Error when reading form: " + err.Error())
		}
		fmt.Printf("\n Visitor: %v", visitor)
	})

	fmt.Println("Server is running at :8080")
	iris.Listen(":8080")
}

```

```html

<!-- ./templates/form.html -->
<!DOCTYPE html>
<head>
<meta charset="utf-8">
</head>
<body>
<form action="/form_action" method="post">
<input type="text" name="Username" />
<br/>
<input type="text" name="Mail" /><br/>
<select multiple="multiple" name="mydata">
<option value='one'>One</option>
<option value='two'>Two</option>
<option value='three'>Three</option>
<option value='four'>Four</option>
</select>
<hr/>
<input type="submit" value="Send data" />

</form>
</body>
</html>

```



## 3.0.0-alpha.1 -> 3.0.0-alpha.2

*The e-book was updated, take a closer look [here](https://www.gitbook.com/book/kataras/iris/details)*


**Breaking changes**

**First**. Configuration owns a package now `github.com/kataras/iris/config` . I took this decision after a lot of thought and I ensure you that this is the best
architecture to easy:

- change the configs without need to re-write all of their fields.
	```go
	irisConfig := config.Iris { Profile: true, PathCorrection: false }
	api := iris.New(irisConfig)
	```

- easy to remember: `iris` type takes config.Iris, sessions takes config.Sessions`, `iris.Config().Render` is `config.Render`, `iris.Config().Render.Template` is `config.Template`, `Logger` takes `config.Logger` and so on...

- easy to find what features are exists and what you can change: just navigate to the config folder and open the type you want to learn about, for example `/iris.go` Iris' type configuration is on `/config/iris.go`

- default setted fields which you can use. They are already setted by iris, so don't worry too much, but if you ever need them you can find their default configs by this pattern: for example `config.Template` has `config.DefaultTemplate()`, `config.Rest` has `config.DefaultRest()`, `config.Typescript()` has `config.DefaultTypescript()`, note that only `config.Iris` has `config.Default()`. I wrote that all structs even the plugins have their default configs now, to make it easier for you, so you can do this without set a config by yourself: `iris.Config().Render.Template.Engine = config.PongoEngine` or `iris.Config().Render.Template.Pongo.Extensions = []string{".xhtml", ".html"}`.



**Second**. Template & rest package moved to the `render`, so

		*  a new config field named `render` of type `config.Render` which nests the `config.Template` & `config.Rest`
		-  `iris.Config().Templates` -> `iris.Config().Render.Template` of type `config.Template`
		- `iris.Config().Rest` -> `iris.Config().Render.Rest` of type `config.Rest`

**Third, sessions**.



Configuration instead of parameters. Before `sessions.New("memory","sessionid",time.Duration(42) * time.Minute)` -> Now:  `sessions.New(config.DefaultSessions())` of type `config.Sessions`

- Before this change the cookie's life was the same as the manager's Gc duration. Now added an Expires option for the cookie's life time which defaults to infinitive, as you (correctly) suggests me in the chat community.-

- Default Cookie's expiration date: from 42 minutes -> to  `infinitive/forever`
- Manager's Gc duration: from 42 minutes -> to '2 hours'
- Redis store's MaxAgeSeconds: from 42 minutes -> to '1 year`


**Four**. Typescript, Editor & IrisControl plugins now accept a config.Typescript/ config.Editor/ config.IrisControl as parameter

Bugfixes

- [can't open /xxx/ path when PathCorrection = false ](https://github.com/kataras/iris/issues/120)
- [Invalid content on links on debug page when custom ProfilePath is set](https://github.com/kataras/iris/issues/118)
- [Example with custom config not working ](https://github.com/kataras/iris/issues/115)
- [Debug Profiler writing escaped HTML?](https://github.com/kataras/iris/issues/107)
- [CORS middleware doesn't work](https://github.com/kataras/iris/issues/108)



## 2.3.2 -> 3.0.0-alpha.1

**Changed**
- `&render.Config` -> `&iris.RestConfig` . All related to the html/template are removed from there.
- `ctx.Render("index",...)` -> `ctx.Render("index.html",...)` or any extension you have defined in iris.Config().Templates.Extensions
- `iris.Config().Render.Layout = "layouts/layout"` -> `iris.Config().Templates.Layout = "layouts/layout.html"`
- `License BSD-3 Clause Open source` -> `MIT License`
**Added**

- Switch template engines via `IrisConfig`. Currently, HTMLTemplate is 'html/template'. Pongo is 'flosch/pongo2`. Refer to the Book, which is updated too, [read here](https://kataras.gitbooks.io/iris/content/render.html).
