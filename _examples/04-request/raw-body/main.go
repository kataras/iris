// ctx.ReadBody: the raw body, and one struct for every content type.
//
// ReadBody has two special cases: a *string or a *[]byte target skips
// decoding entirely and receives the body as it arrived. Anything else is
// dispatched by the request: a GET reads the URL query (falling back to the
// form), other methods switch on the media type - XML, YAML, url-encoded or
// multipart form, JSON, Protobuf, MessagePack. An unknown type falls back to
// the query when there is one, then to JSON. A multipart/related body is
// refused: read those with ctx.ReadMultipartRelated.
//
// ReadBody returns its error rather than answering the client, unlike
// ctx.BindJSON, so every handler here forwards it to
// errors.HandleRequestError: an unrecognized failure becomes a 400 and a body
// past ContextOptions.BodyMaxMemory stays a 413.
//
// Learn: read a body as raw text or bytes, or let ReadBody pick the decoder
// from the content type, and forward its error with the status it implies.
// Before: ../read-json-optional
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -X POST http://localhost:8080/raw/text -H 'Content-Type: text/plain' -d 'hello raw world'   # 200, length and text
//	curl -X POST http://localhost:8080/raw/bytes --data-binary @main.go                              # 200, length and sha256
//	curl -X POST http://localhost:8080/notes -H 'Content-Type: application/json' \
//	  -d '{"title":"iris","body":"read the body"}'                                                   # 200, decoded_from application/json
//	curl -X POST http://localhost:8080/notes -H 'Content-Type: application/xml' \
//	  -d '<note><title>iris</title><body>read the body</body></note>'                              # 200, decoded_from application/xml
//	curl -X POST http://localhost:8080/notes -d 'title=iris&body=read+the+body'                     # 200, decoded from the form
//	curl 'http://localhost:8080/notes?title=iris&body=read+the+body'                                # 200, decoded from the query
//
// Next: ../strict-query
package main

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Post("/raw/text", readText)
	app.Post("/raw/bytes", readBytes)

	app.Post("/notes", readNote)
	app.Get("/notes", readNote) // a GET reads the query instead.

	app.Listen(":8080")
}

// Note carries one tag per source, because ReadBody delegates to the
// matching reader: json/xml/yaml for bodies, form for form posts and url
// for the query string.
type Note struct {
	Title string `json:"title" xml:"title" yaml:"title" form:"title" url:"title"`
	Body  string `json:"body" xml:"body" yaml:"body" form:"body" url:"body"`
}

// readText fills a *string with the undecoded body.
func readText(ctx iris.Context) {
	var text string
	if err := ctx.ReadBody(&text); err != nil {
		errors.HandleRequestError(ctx, err)
		return
	}

	ctx.OK(iris.Map{
		"media_type": ctx.MediaType(),
		"length":     len(text),
		"text":       text,
	}, nil)
}

// readBytes fills a *[]byte the same way - binary safe, no decoding.
func readBytes(ctx iris.Context) {
	var raw []byte
	if err := ctx.ReadBody(&raw); err != nil {
		errors.HandleRequestError(ctx, err)
		return
	}

	sum := sha256.Sum256(raw)
	ctx.OK(iris.Map{
		"length": len(raw),
		"sha256": hex.EncodeToString(sum[:]),
	}, nil)
}

// readNote lets ReadBody pick the decoder from the request itself.
func readNote(ctx iris.Context) {
	var note Note
	if err := ctx.ReadBody(&note); err != nil && !iris.IsErrPath(err) {
		errors.HandleRequestError(ctx, err)
		return
	}

	ctx.OK(iris.Map{"decoded_from": ctx.MediaType(), "note": note}, nil)
}
