// The Context response writers: one format per route.
//
// Each writer sets its own Content-Type and writes the body. JSON, JSONP, XML
// and Markdown accept an optional options value as their last argument; YAML,
// TextYAML, Text, HTML and Binary take none.
//
// Learn: pick the writer for a response format and know which ones take an
// options value.
// Before: ../rest-helpers
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/json          # 200 application/json, compact
//	curl -i http://localhost:8080/json/pretty   # 200 application/json, indented, HTML unescaped
//	curl -i http://localhost:8080/jsonp         # 200 text/javascript, onBook({...})
//	curl -i http://localhost:8080/xml           # 200 text/xml
//	curl -i http://localhost:8080/xml/map       # 200 text/xml, a map under <Book>
//	curl -i http://localhost:8080/yaml          # 200 application/x-yaml
//	curl -i http://localhost:8080/yaml/text     # 200 text/yaml
//	curl -i http://localhost:8080/markdown      # 200 text/html, rendered from markdown
//	curl -i http://localhost:8080/text          # 200 text/plain
//	curl -i http://localhost:8080/html          # 200 text/html
//	curl -i http://localhost:8080/binary        # 200 application/octet-stream
//
// Next: ../content-negotiation
package main

import (
	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	// application/json, default encoder: compact, HTML kept as-is
	// (JSON parsers do not need the < escaping; opt back in with
	// EscapeHTML when the payload is ever inlined into HTML).
	app.Get("/json", func(ctx iris.Context) {
		ctx.JSON(book)
	})

	// application/json with options.
	app.Get("/json/pretty", func(ctx iris.Context) {
		ctx.JSON(iris.Map{"book": book, "note": "<b>indented</b>"}, iris.JSON{
			Indent:         "  ",  // pretty-print with two spaces.
			EscapeHTML:     false, // true escapes <, > and & like the old encoding/json did.
			Prefix:         "",    // written before the body.
			EscapeNonASCII: false, // escape non-ASCII runes when true.
			ArrayPrefix:    false, // prepend "while(1);" to top-level arrays.
		})
	})

	// text/javascript: the JSON wrapped in a function call (legacy format).
	app.Get("/jsonp", func(ctx iris.Context) {
		ctx.JSONP(book, iris.JSONP{Callback: "onBook"})
	})

	// text/xml from a struct.
	app.Get("/xml", func(ctx iris.Context) {
		ctx.XML(book, iris.XML{Indent: "  "})
	})

	// Maps have no element name of their own, so they need the XMLMap wrapper.
	app.Get("/xml/map", func(ctx iris.Context) {
		ctx.XML(iris.XMLMap("Book", iris.Map{"title": book.Title}), iris.XML{Indent: "  "})
	})

	// application/x-yaml.
	app.Get("/yaml", func(ctx iris.Context) {
		ctx.YAML(book)
	})

	// text/yaml: the same encoder, a different content type.
	app.Get("/yaml/text", func(ctx iris.Context) {
		ctx.TextYAML(book)
	})

	// text/html, rendered from markdown source.
	app.Get("/markdown", func(ctx iris.Context) {
		md := []byte("# " + book.Title + "\n\nBy **" + book.Author + "**.\n")
		ctx.Markdown(md, iris.Markdown{Sanitize: true}) // sanitize user-provided input.
	})

	// text/plain: a single string, no format arguments (use ctx.Writef for those).
	app.Get("/text", func(ctx iris.Context) {
		ctx.Text(book.Title + " by " + book.Author)
	})

	// text/html, written as-is.
	app.Get("/html", func(ctx iris.Context) {
		ctx.HTML("<h1>" + book.Title + "</h1>")
	})

	// application/octet-stream.
	app.Get("/binary", func(ctx iris.Context) {
		ctx.Binary([]byte(book.Title))
	})

	app.Listen(":8080")
}

// Book is rendered by every writer in main, so it carries both json and xml tags.
type Book struct {
	Title  string `json:"title" xml:"title" yaml:"title"`
	Author string `json:"author" xml:"author" yaml:"author"`
	Pages  int    `json:"pages" xml:"pages" yaml:"pages"`
}

var book = Book{Title: "Iris in Action", Author: "Jane Doe", Pages: 412}
