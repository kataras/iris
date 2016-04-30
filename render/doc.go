/*Package render is a package that provides functionality for easily rendering JSON, XML, binary data, and HTML templates.

  package main

  import (
      "encoding/xml"
      "github.com/kataras/iris"

      "github.com/iris-contrib/render"
  )

  type ExampleXml struct {
      XMLName xml.Name `xml:"example"`
      One     string   `xml:"one,attr"`
      Two     string   `xml:"two,attr"`
  }

  func main() {
      r := render.New()

      iris.Get("/data", func(ctx *iris.Context) {
          r.Data(ctx, iris.StatusOK, []byte("Some binary data here."))
      })

      iris.Get("/text", func(ctx *iris.Context) {
          r.Text(ctx, iris.StatusOK, "Plain text here")
      })

      iris.Get("/json", func(ctx *iris.Context) {
          r.JSON(ctx, iris.StatusOK, map[string]string{"hello": "json"})
      })

      iris.Get("/jsonp", func(ctx *iris.Context) {
          r.JSONP(ctx, iris.StatusOK, "callbackName", map[string]string{"hello": "jsonp"})
      })

      iris.Get("/xml", func(ctx *iris.Context) {
          r.XML(ctx, iris.StatusOK, ExampleXml{One: "hello", Two: "xml"})
      })

      iris.Get("/html", func(ctx *iris.Context) {
          // Assumes you have a template in ./templates called "example.tmpl".
          // $ mkdir -p templates && echo "<h1>Hello HTML world.</h1>" > templates/example.tmpl
          r.HTML(ctx, iris.StatusOK, "example", nil)
      })

      iris.Listen("0.0.0.0:3000")
  }
*/
package render
