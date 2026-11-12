# Response Rendering and Request Body Reading

The `iris.Context` carries one request and its response, and every read and write goes through it. This document covers the essential methods available in the `iris.Context` struct for rendering various content types, providing detailed explanations, usage examples, and output examples.

- [JSON](#json)
  - [What is JSON?](#what-is-json)
  - [When to Use JSON?](#when-to-use-json)
  - [Usage](#usage)
  - [Output Example](#output-example)
  - [Reading JSON Request Body](#reading-json-request-body)
  - [Input Example](#input-example)
  - [Reading JSON Stream Request Body](#reading-json-stream-request-body)
  - [Stream Input Example](#stream-input-example)
- [YAML](#yaml)
  - [What is YAML?](#what-is-yaml)
  - [When to Use YAML?](#when-to-use-yaml)
  - [Usage](#usage-1)
  - [Output Example](#output-example-1)
  - [Reading YAML Request Body](#reading-yaml-request-body)
  - [Input Example](#input-example-1)
- [Binary](#binary)
  - [What is Binary?](#what-is-binary)
  - [When to Use Binary?](#when-to-use-binary)
  - [Usage](#usage-2)
  - [Output Example](#output-example-2)
  - [Reading Binary Request Body](#reading-binary-request-body)
  - [Input Example](#input-example-2)
- [Text](#text)
  - [What is Text?](#what-is-text)
  - [When to Use Text?](#when-to-use-text)
  - [Usage](#usage-3)
  - [Output Example](#output-example-3)
  - [Reading Text Request Body](#reading-text-request-body)
  - [Input Example](#input-example-3)
- [JSONP](#jsonp)
  - [What is JSONP?](#what-is-jsonp)
  - [When to Use JSONP?](#when-to-use-jsonp)
  - [Usage](#usage-4)
  - [Output Example](#output-example-4)
  - [Reading JSONP Request Body](#reading-jsonp-request-body)
  - [Input Example](#input-example-4)
- [XML](#xml)
  - [What is XML?](#what-is-xml)
  - [When to Use XML?](#when-to-use-xml)
  - [Usage](#usage-5)
  - [Output Example](#output-example-5)
  - [Reading XML Request Body](#reading-xml-request-body)
  - [Input Example](#input-example-5)
- [Markdown](#markdown)
  - [What is Markdown?](#what-is-markdown)
  - [When to Use Markdown?](#when-to-use-markdown)
  - [Usage](#usage-6)
  - [Output Example](#output-example-6)
  - [Reading Markdown Request Body](#reading-markdown-request-body)
  - [Input Example](#input-example-6)
- [MsgPack](#msgpack)
  - [What is MsgPack?](#what-is-msgpack)
  - [When to Use MsgPack?](#when-to-use-msgpack)
  - [Usage](#usage-7)
  - [Output Example](#output-example-7)
  - [Reading MsgPack Request Body](#reading-msgpack-request-body)
  - [Input Example](#input-example-7)
- [Custom Fast-Path JSON (MarshalerTo)](#custom-fast-path-json-marshalerto)
  - [What is MarshalerTo?](#what-is-marshalerto)
  - [When to Use MarshalerTo?](#when-to-use-marshalerto)
  - [Usage](#usage-8)
  - [Output Example](#output-example-8)
- [Protobuf](#protobuf)
  - [What is Protobuf?](#what-is-protobuf)
  - [When to Use Protobuf?](#when-to-use-protobuf)
  - [Usage](#usage-9)
  - [Output Example](#output-example-9)
  - [Reading Protobuf Request Body](#reading-protobuf-request-body)
  - [Input Example](#input-example-9)
- [Protobuf JSON](#protobuf-json)
  - [What is Protobuf JSON?](#what-is-protobuf-json)
  - [When to Use Protobuf JSON?](#when-to-use-protobuf-json)
  - [Usage](#usage-10)
  - [Output Example](#output-example-10)
  - [Reading Protobuf JSON Request Body](#reading-protobuf-json-request-body)
  - [Input Example](#input-example-10)
- [Form](#form)
  - [What is Form?](#what-is-form)
  - [When to Use Form?](#when-to-use-form)
  - [Usage](#usage-11)
  - [Reading Form Request Body](#reading-form-request-body)
  - [Input Example](#input-example-11)
- [HTML](#html-data)
  - [When to Use HTML?](#when-to-use-html)
  - [Usage](#usage-12)
  - [Output Example](#output-example-11)
- [RenderResponse](#renderresponse)
  - [What is RenderResponse?](#what-is-renderresponse)
  - [When to Use RenderResponse?](#when-to-use-renderresponse)
  - [Usage](#usage-13)
  - [Output Example](#output-example-12)
- [RenderComponent](#rendercomponent)
  - [What is RenderComponent?](#what-is-rendercomponent)
  - [When to Use RenderComponent?](#when-to-use-rendercomponent)
  - [Usage with templ Library](#usage-with-templ-library)
  - [Output Example](#output-example-13)

## JSON

### What is JSON?

JSON (JavaScript Object Notation) is a lightweight data interchange format that is easy for humans to read and write and easy for machines to parse and generate. It is commonly used for transmitting data in web applications.

### When to Use JSON?

Use JSON when you need to send structured data between a client and a server, especially in web APIs.

### Usage

```go
type Message struct {
    Hello string `json:"hello"`
}
```

```go
app.Get("/json", func(ctx iris.Context) {
    ctx.JSON(Message{Hello: "json"})
})
```

### Output Example

```json
{
    "hello": "json"
}
```

### Reading JSON Request Body

```go
type Message struct {
    Hello string `json:"hello"`
}

func readJSON(ctx iris.Context) {
    var msg Message
    if err := ctx.ReadJSON(&msg); err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    ctx.Writef("Received: %#+v\n", msg)
}

app.Post("/", readJSON)
```

### Input Example

```json
{
    "hello": "json"
}
```

### Reading JSON Stream Request Body

```go
type User struct {
	Username string `json:"username"`
}

func postIndexStream(ctx iris.Context) {
	var users []User
	job := func(decode iris.DecodeFunc) error {
		var u User
		if err := decode(&u); err != nil {
			return err
		}
		users = append(users, u)
		return nil
	}

	err := ctx.ReadJSONStream(job, iris.JSONReader{
		Optimize:              true,
		DisallowUnknownFields: true,
		ArrayStream:           false,
	})
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.JSON(iris.Map{
		"code":        iris.StatusOK,
		"users_count": len(users),
		"users":       users,
	})
}

app.Post("/stream", postIndexStream)
```

### Input Example

```json
{"username":"john"}
{"username": "makis"}
{"username": "george"}
```

## YAML

### What is YAML?

YAML (YAML Ain't Markup Language) is a human-readable data serialization format that is commonly used for configuration files and data exchange between languages with different data structures.

### When to Use YAML?

Use YAML when you need a human-readable format for configuration files or data exchange.

### Usage

```go
type Message struct {
    Hello string `yaml:"hello"`
}

app.Get("/yaml", func(ctx iris.Context) {
    ctx.YAML(Message{Hello: "yaml"})
})
```

### Output Example

```yaml
hello: yaml
```

### Reading YAML Request Body

```go
type Message struct {
    Hello string `yaml:"hello"`
}

func readYAML(ctx iris.Context) {
    var msg Message
    if err := ctx.ReadYAML(&msg); err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    ctx.Writef("Received: %#+v\n", msg)
}

app.Post("/", readYAML)
```

### Input Example

```yaml
hello: yaml
```

## Binary

### What is Binary?

Binary data refers to any data that is represented in binary (0s and 1s) form. It is commonly used for transmitting non-text data such as images, audio, and video files.

### When to Use Binary?

Use Binary when you need to send non-text data such as files, images, or any other binary data.

### Usage

```go
app.Get("/binary", func(ctx iris.Context) {
    ctx.Binary([]byte("Some binary data here."))
})
```

### Output Example

```
Some binary data here.
```

### Reading Binary Request Body

```go
func readBinary(ctx iris.Context) {
    body, err := ctx.Body()
    if err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    ctx.Writef("Received binary data: %v", body)
}

app.Post("/", readBinary)
```

### Input Example

```
Some binary data here.
```

## Text

### What is Text?

Text data refers to any data that is represented in plain text form. It is commonly used for transmitting human-readable data.

### When to Use Text?

Use Text when you need to send plain text data to the client.

### Usage

```go
app.Get("/text", func(ctx iris.Context) {
    ctx.Text("Plain text here")
})
```

### Output Example

```
Plain text here
```

### Reading Text Request Body

```go
func readText(ctx iris.Context) {
    body, err := ctx.Body()
    if err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    ctx.Writef("Received text data: %s", string(body))
}

app.Post("/", readText)
```

### Input Example

```
Plain text here
```

## JSONP

### What is JSONP?

JSONP (JSON with Padding) predates CORS as a way around the browser's same-origin restriction. It fetches data from another domain by using a `<script>` tag.

### When to Use JSONP?

Use JSONP when you need to make cross-domain requests and the server supports JSONP.

### Usage

```go
type Message struct {
    Hello string `json:"hello"`
}

app.Get("/jsonp", func(ctx iris.Context) {
    ctx.JSONP(Message{Hello: "jsonp"}, iris.JSONP{Callback: "callbackName"})
})
```

### Output Example

```javascript
callbackName({"hello": "jsonp"})
```

### Reading JSONP Request Body

```go
type Message struct {
    Hello string `json:"hello"`
}

func readJSONP(ctx iris.Context) {
    var msg Message
    if err := ctx.ReadJSON(&msg); err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    ctx.Writef("Received: %#+v\n", msg)
}

app.Post("/", readJSONP)
```

### Input Example

```json
{
    "hello": "jsonp"
}
```

## XML

### What is XML?

XML (eXtensible Markup Language) is a markup language that defines a set of rules for encoding documents in a format that is both human-readable and machine-readable.

### When to Use XML?

Use XML when you need to send structured data that needs to be both human-readable and machine-readable, especially in legacy systems.

### Usage

```go
type Message struct {
    Hello string `xml:"hello"`
}

app.Get("/xml", func(ctx iris.Context) {
    ctx.XML(Message{Hello: "xml"})
})
```

### Output Example

```xml
<message>
    <hello>xml</hello>
</message>
```

### Reading XML Request Body

```go
type Message struct {
    Hello string `xml:"hello"`
}

func readXML(ctx iris.Context) {
    var msg Message
    if err := ctx.ReadXML(&msg); err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    ctx.Writef("Received: %#+v\n", msg)
}

app.Post("/", readXML)
```

### Input Example

```xml
<message>
    <hello>xml</hello>
</message>
```

## Markdown

### What is Markdown?

Markdown is a lightweight markup language with plain text formatting syntax. It is commonly used for formatting readme files, writing messages in online discussion forums, and creating rich text using a plain text editor.

### When to Use Markdown?

Use Markdown when you need to send formatted text that can be easily converted to HTML.

### Usage

```go
app.Get("/markdown", func(ctx iris.Context) {
    ctx.Markdown([]byte("# Hello Dynamic Markdown -- iris"))
})
```

### Output Example

```html
<h1>Hello Dynamic Markdown -- iris</h1>
```

## MsgPack

### What is MsgPack?

MsgPack (MessagePack) is a binary format that is used to serialize and deserialize data. It is more efficient than JSON in terms of both size and speed.

### When to Use MsgPack?

Use MsgPack when you need a more efficient alternative to JSON for data serialization and deserialization.

### Usage

```go
type Message struct {
    Hello string `msgpack:"hello"`
}

app.Get("/msgpack", func(ctx iris.Context) {
    ctx.MsgPack(Message{Hello: "msgpack"})
})
```

### Output Example

The output will be in binary format, which is not human-readable. However, when deserialized, it will represent the following data:

```json
{
    "hello": "msgpack"
}
```

### Reading MsgPack Request Body

```go
type Message struct {
    Hello string `msgpack:"hello"`
}

func readMsgPack(ctx iris.Context) {
    var msg Message
    err := ctx.ReadMsgPack(&msg)
    if err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    ctx.Writef("Received: %#+v\n", msg)
}

app.Post("/", readMsgPack)
```

### Input Example

```json
{
    "hello": "msgpack"
}
```

## Custom Fast-Path JSON (MarshalerTo)

### What is MarshalerTo?

Since Go 1.27 the standard library ships `encoding/json/v2`, and Iris encodes every JSON response through it. A type that implements the `json.MarshalerTo` interface (`MarshalJSONTo(*jsontext.Encoder) error`) streams itself straight into the response encoder with no intermediate `[]byte` — the role the third-party easyjson code generator used to play, now with no code generation and no dependency.

### When to Use MarshalerTo?

Implement it on hot-path response types when profiling shows the reflection-based encoding is a cost; everything else should use plain `ctx.JSON`.

### Usage

```go
import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
)

type Message struct {
	Hello string `json:"hello"`
}

func (m Message) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("hello")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(m.Hello)); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

app.Get("/fast", func(ctx iris.Context) {
	ctx.JSON(Message{Hello: "world"}) // MarshalJSONTo is picked up automatically.
})
```

The matching read-side interface is `json.UnmarshalerFrom` (`UnmarshalJSONFrom(*jsontext.Decoder) error`), honored by `ctx.ReadJSON` the same way.

### Output Example

```json
{
    "hello": "world"
}
```

## Protobuf

### What is Protobuf?

Protobuf (Protocol Buffers) is a language-neutral, platform-neutral extensible mechanism for serializing structured data. It is used to serialize and deserialize data efficiently.

### When to Use Protobuf?

Use Protobuf when you need a more efficient and flexible alternative to JSON for data serialization and deserialization.

### Usage

```go
// Protobuf example struct.
type HelloReply struct {
    Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

app.Get("/protobuf", func(ctx iris.Context) {
    response := &HelloReply{Message: "Hello, World!"}
    ctx.Protobuf(response)
})
```

### Output Example

The output will be in binary format, which is not human-readable. However, when deserialized, it will represent the following data:

```json
{
    "message": "Hello, World!"
}
```

### Reading Protobuf Request Body

```go
type HelloRequest struct {
    Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func readProtobuf(ctx iris.Context) {
    var req HelloRequest
    err := ctx.ReadProtobuf(&req)
    if err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    ctx.Writef("Received: %#+v\n", req)
}

app.Post("/", readProtobuf)
```

### Input Example

The input will be in binary format, which is not human-readable. However, when deserialized, it will represent the following data:

```
0a 08 4a 6f 68 6e 20 44 6f 65
```

This binary data corresponds to the following Protobuf message:

```json
{
    "name": "John Doe"
}
```

### Protobuf JSON

### What is Protobuf JSON?

Protobuf JSON is a way to serialize and deserialize Protobuf messages to and from JSON format.

### When to Use Protobuf JSON?

Use Protobuf JSON when you need to serialize and deserialize Protobuf messages to and from JSON format.

### Usage

```go
// Protobuf example struct.
type HelloReply struct {
	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}

app.Get("/protobufjson", func(ctx iris.Context) {
    response := &HelloReply{Message: "Hello, World!"}
    ctx.ProtobufJSON(response)
})
```

### Output Example

```json
{
    "message": "Hello, World!"
}
```

### Reading Protobuf JSON Request Body

```go
type HelloRequest struct {
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func readProtobufJSON(ctx iris.Context) {
	var req HelloRequest
	err := ctx.ReadProtobufJSON(&req)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.Writef("Received: %#+v\n", req)
}

app.Post("/", readProtobufJSON)
```

### Input Example

```json
{
    "name": "John Doe"
}
```

## Form

### What is Form?

Form data refers to data submitted through HTML forms. It is commonly used for transmitting user input data from web forms to the server.

### When to Use Form?

Use Form when you need to handle data submitted through HTML forms.

### Usage

```go
type User struct {
	Name  string `form:"name"`
	Email string `form:"email"`
}

func readForm(ctx iris.Context) {
	var user User
	err := ctx.ReadForm(&user)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.Writef("Received: %#+v\n", user)
}

app.Post("/form", readForm)
```

### Input Example

```
name=John Doe&email=john.doe@example.com
```

## HTML Data

HTML data refers to any data that is represented in HTML (HyperText Markup Language) form. It is commonly used for transmitting web pages and other content that can be rendered by a web browser.

### When to Use HTML?

Use HTML when you need to send HTML content to the client, such as web pages or fragments of web pages.

### Usage

```go
app.Get("/html", func(ctx iris.Context) {
    ctx.HTML("<h1>Hello, World!</h1>")
})
```

### Output Example

```html
<h1>Hello, World!</h1>
```

The `ctx.HTML` method sets the `Content-Type` header to `text/html; charset=UTF-8` and writes the provided HTML string to the response body.

## RenderResponse

### What is RenderResponse?

`RenderResponse` is a method that writes the response headers and calls the given renderer to render data. This method can be used while migrating from other frameworks.

### When to Use RenderResponse?

Use `RenderResponse` when you need to render custom response data using a custom response renderer.

### Usage

```go
type CustomRenderer struct {
    Content string
}

func (r CustomRenderer) Render(w http.ResponseWriter) error {
    _, err := w.Write([]byte(r.Content))
    return err
}

func (r CustomRenderer) WriteContentType(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "text/plain")
}

app.Get("/custom", func(ctx iris.Context) {
    renderer := CustomRenderer{Content: "Hello, Custom Renderer!"}
    ctx.RenderResponse(iris.StatusOK, renderer)
})
```

### Output Example

```
Hello, Custom Renderer!
```

## RenderComponent

### What is RenderComponent?

`RenderComponent` is a method that renders a component to the client. It sets the "Content-Type" header to "text/html; charset=utf-8" and reports any component render errors back to the caller.

### When to Use RenderComponent?

Use `RenderComponent` when you need to render a component as HTML through `Context.RenderComponent`.

### Usage with templ Library

First, install the templ library:

```sh
go get github.com/a-h/templ@latest
```

Create a templ component:

```go
package main

import (
    "context"
    "io"

    "github.com/kataras/iris/v14"

    "github.com/a-h/templ"
)

type CustomComponent struct {
    Content string
}

func (c CustomComponent) Render(ctx context.Context, w io.Writer) error {
    t := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
        _, err := w.Write([]byte("<div>" + c.Content + "</div>"))
        return err
    })
    return t.Render(ctx, w)
}

func main() {
    app := iris.New()

    app.Get("/component", func(ctx iris.Context) {
        component := CustomComponent{Content: "Hello, Component!"}
        ctx.RenderComponent(component)
    })

    app.Listen(":8080")
}
```

### Output Example

```html
<div>Hello, Component!</div>
```
