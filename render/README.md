# Package information

Provides functionality for easily, one line, rendering JSON, XML, text, binary data, and HTML templates.

All functions are inside Context, options declaration at the configuration state.


## Usage
The rendering functions simply wraps Go's existing functionality for marshaling and rendering data.

- HTML/Render: Uses the [html/template](http://golang.org/pkg/html/template/) package to render HTML templates.
- JSON: Uses the [encoding/json](http://golang.org/pkg/encoding/json/) package to marshal data into a JSON-encoded response.
- XML: Uses the [encoding/xml](http://golang.org/pkg/encoding/xml/) package to marshal data into an XML-encoded response.
- Binary data: Passes the incoming data straight through to the `iris.Context.Response`.
- Text: Passes the incoming string straight through to the ``iris.Context.Response``.

~~~ go
// main.go
 package main

  import (
      "encoding/xml"
      "github.com/kataras/iris"
  )

  type ExampleXml struct {
      XMLName xml.Name `xml:"example"`
      One     string   `xml:"one,attr"`
      Two     string   `xml:"two,attr"`
  }

  func main() {
      iris.Get("/data", func(ctx *iris.Context) {
         ctx.Data(iris.StatusOK, []byte("Some binary data here."))
      })

      iris.Get("/text", func(ctx *iris.Context) {
          ctx.Text(iris.StatusOK, "Plain text here")
      })

      iris.Get("/json", func(ctx *iris.Context) {
          ctx.JSON(iris.StatusOK, map[string]string{"hello": "json"})
      })

      iris.Get("/jsonp", func(ctx *iris.Context) {
          ctx.JSONP(iris.StatusOK, "callbackName", map[string]string{"hello": "jsonp"})
      })

      iris.Get("/xml", func(ctx *iris.Context) {
          ctx.XML(iris.StatusOK, ExampleXml{One: "hello", Two: "xml"})
      })

      iris.Get("/html", func(ctx *iris.Context) {
          // Assumes you have a template in ./templates called "example.tmpl".
          // $ mkdir -p templates && echo "<h1>Hello HTML world.</h1>" > templates/example.tmpl
          ctx.HTML(iris.StatusOK, "example", nil)
      })
      
      // ctx.Render is the same as ctx.HTML but with default 200 status OK
     iris.Get("/html2", func(ctx *iris.Context) {
          // Assumes you have a template in ./templates called "example.tmpl".
          // $ mkdir -p templates && echo "<h1>Hello HTML world.</h1>" > templates/example.tmpl
          ctx.Render("example", nil)
      })

      iris.Listen(":8080")
~~~

~~~ html
<!-- templates/example.tmpl -->
<h1>Hello {{.}}.</h1>
~~~

### Available Options
Render comes with a variety of configuration options _(Note: these are not the default option values. See the defaults below.)_:

~~~ go
// ...
renderOptions := &iris.RenderConfig{
    Directory: "templates", // Specify what path to load the templates from.
    Asset: func(name string) ([]byte, error) { // Load from an Asset function instead of file.
      return []byte("template content"), nil
    },
    AssetNames: func() []string { // Return a list of asset names for the Asset function
      return []string{"filename.tmpl"}
    },
    Layout: "layout", // Specify a layout template. Layouts can call {{ yield }} to render the current template or {{ partial "css" }} to render a partial from the current template.
    Extensions: []string{".tmpl", ".html"}, // Specify extensions to load for templates.
    Funcs: []template.FuncMap{AppHelpers}, // Specify helper function maps for templates to access.
    Delims: iris.Delims{"{[{", "}]}"}, // Sets delimiters to the specified strings.
    Charset: "UTF-8", // Sets encoding for json and html content-types. Default is "UTF-8".
    IndentJSON: true, // Output human readable JSON.
    IndentXML: true, // Output human readable XML.
    PrefixJSON: []byte(")]}',\n"), // Prefixes JSON responses with the given bytes.
    PrefixXML: []byte("<?xml version='1.0' encoding='UTF-8'?>"), // Prefixes XML responses with the given bytes.
    HTMLContentType: "application/xhtml+xml", // Output XHTML content type instead of default "text/html".
    IsDevelopment: true, // Render will now recompile the templates on every HTML response.
    UnEscapeHTML: true, // Replace ensure '&<>' are output correctly (JSON only).
    StreamingJSON: true, // Streams the JSON response via json.Encoder.
    RequirePartials: true, // Return an error if a template is missing a partial used in a layout.
    DisableHTTPErrorRendering: true, // Disables automatic rendering of iris.StatusInternalServerError when an error occurs.
})
// ...
~~~

### Default Options
These are the preset options for Render:

~~~ go
// Is the same as the default configuration options:

renderOptions = &iris.RenderConfig{
    Directory: "templates",
    Asset: nil,
    AssetNames: nil,
    Layout: "",
    Extensions: []string{".html"},
    Funcs: []template.FuncMap{},
    Delims: iris.Delims{"{{", "}}"},
    Charset: "UTF-8",
    IndentJSON: false,
    IndentXML: false,
    PrefixJSON: []byte(""),
    PrefixXML: []byte(""),
    HTMLContentType: "text/html",
    IsDevelopment: false,
    UnEscapeHTML: false,
    StreamingJSON: false,
    RequirePartials: false,
    DisableHTTPErrorRendering: false,
})
~~~

### JSON vs Streaming JSON
By default, Render does **not** stream JSON to the `iris.Context.Response`. It instead marshalls your object into a byte array, and if no errors occurred, writes that byte array to the `iris.Context.Response`. This is ideal as you can catch errors before sending any data.

If however you have the need to stream your JSON response (ie: dealing with massive objects), you can set the `StreamingJSON` option to true. This will use the `json.Encoder` to stream the output to the `iris.Context.Response`. If an error occurs, you will receive the error in your code, but the response will have already been sent. Also note that streaming is only implemented in `render.JSON` and not `render.JSONP`, and the `UnEscapeHTML` and `Indent` options are ignored when streaming.

### Loading Templates
By default Render will attempt to load templates with a '.html' extension from the "templates" directory. Templates are found by traversing the templates directory and are named by path and basename. For instance, the following directory structure:

~~~
templates/
  |
  |__ admin/
  |      |
  |      |__ index.html
  |      |
  |      |__ edit.html
  |
  |__ home.html
~~~

Will provide the following templates:
~~~
admin/index
admin/edit
home
~~~

You can also load templates from memory by providing the Asset and AssetNames options,
e.g. when generating an asset file using [go-bindata](https://github.com/jteeuwen/go-bindata).

### Layouts
Render provides `yield` and `partial` functions for layouts to access:
~~~ go
// ...

renderOptions := &iris.RenderConfig{
    Layout: "layout",
}

iris.SetRenderConfig(renderOptions)
// or api := iris.New(Render: renderOptions)

// ...
~~~

~~~ html
<!-- templates/layout.html -->
<html>
  <head>
    <title>My Layout</title>
    <!-- Render the partial template called `css-$current_template` here -->
    {{ partial "css" }}
  </head>
  <body>
    <!-- render the partial template called `header-$current_template` here -->
    {{ partial "header" }}
    <!-- Render the current template here -->
    {{ yield }}
    <!-- render the partial template called `footer-$current_template` here -->
    {{ partial "footer" }}
  </body>
</html>
~~~

`current` can also be called to get the current template being rendered.
~~~ html
<!-- templates/layout.html -->
<html>
  <head>
    <title>My Layout</title>
  </head>
  <body>
    This is the {{ current }} page.
  </body>
</html>
~~~

Partials are defined by individual templates as seen below. The partial template's
name needs to be defined as "{partial name}-{template name}".
~~~ html
<!-- templates/home.html -->
{{ define "header-home" }}
<h1>Home</h1>
{{ end }}

{{ define "footer-home"}}
<p>The End</p>
{{ end }}
~~~

By default, the template is not required to define all partials referenced in the
layout. If you want an error to be returned when a template does not define a
partial, set `RenderConfig.RequirePartials = true`.

### Character Encodings
Render will automatically set the proper Content-Type header based on which function you call.

In order to change the charset, you can set the `Charset` within the `RenderConfig` to your encoding value, or ```Iris.DefaultCharset = "UTF-8"```

~~~ go
// main.go
package main

import (
    "encoding/xml"
    "github.com/kataras/iris"

)

type ExampleXml struct {
    XMLName xml.Name `xml:"example"`
    One     string   `xml:"one,attr"`
    Two     string   `xml:"two,attr"`
}

func main() {
    iris.DefaultCharset = "ISO-8859-1"
    // or iris.SetRenderConfig(&iris.RenderConfig{ Charset: "ISO-8859-1"})
  

    //...
}

~~~

### Error Handling

The rendering functions return any errors from the rendering engine.
By default, they will also write the error to the HTTP response and set the status code to 500. You can disable
this behavior so that you can handle errors yourself by setting
`RenderConfig.DisableHTTPErrorRendering: true`.

~~~go
renderOptions := &iris.RenderConfig{
  DisableHTTPErrorRendering: true,
}

iris.SetRenderConfig(renderOptions)

//...
func (ctx *iris.Context) {
  err := ctx.HTML(iris.StatusOK "example", "World")
  if err != nil{
    ctx.Redirect("/my-custom-500", iris.StatusFound)
  }
}


~~~