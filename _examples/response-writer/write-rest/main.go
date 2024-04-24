package main

import (
	"encoding/xml"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"
)

// User example struct for json and msgpack.
type User struct {
	Firstname string `json:"firstname" msgpack:"firstname"`
	Lastname  string `json:"lastname" msgpack:"lastname"`
	City      string `json:"city" msgpack:"city"`
	Age       int    `json:"age" msgpack:"age"`
}

// ExampleXML just a test struct to view represents xml content-type
type ExampleXML struct {
	XMLName xml.Name `xml:"example"`
	One     string   `xml:"one,attr"`
	Two     string   `xml:"two,attr"`
}

// ExampleYAML just a test struct to write yaml to the client.
type ExampleYAML struct {
	Name       string `yaml:"name"`
	ServerAddr string `yaml:"ServerAddr"`
}

func main() {
	app := iris.New()
	// Optionally, set a custom handler for JSON, JSONP, Protobuf, MsgPack, YAML, Markdown...
	// write errors.
	app.SetContextErrorHandler(new(errorHandler))
	// Read
	app.Post("/decode", func(ctx iris.Context) {
		// Read https://github.com/kataras/iris/blob/main/_examples/request-body/read-json/main.go as well.
		var user User
		err := ctx.ReadJSON(&user)
		if err != nil {
			errors.InvalidArgument.Details(ctx, "unable to parse body", err.Error())
			return
		}

		ctx.Writef("%s %s is %d years old and comes from %s!", user.Firstname, user.Lastname, user.Age, user.City)
	})

	// Write
	app.Get("/encode", func(ctx iris.Context) {
		u := User{
			Firstname: "John",
			Lastname:  "Doe",
			City:      "Neither FBI knows!!!",
			Age:       25,
		}

		// Manually setting a content type: ctx.ContentType("text/javascript")
		ctx.JSON(u)
	})

	// Use Secure field to prevent json hijacking.
	// It prepends `"while(1),"` to the body when the data is array.
	app.Get("/json_secure", func(ctx iris.Context) {
		response := []string{"val1", "val2", "val3"}
		options := iris.JSON{Indent: "", Secure: true}
		ctx.JSON(response, options)

		// Will output: while(1);["val1","val2","val3"]
	})

	// Use ASCII field to generate ASCII-only JSON
	// with escaped non-ASCII characters.
	app.Get("/json_ascii", func(ctx iris.Context) {
		response := iris.Map{"lang": "GO-虹膜", "tag": "<br>"}
		options := iris.JSON{Indent: "    ", ASCII: true}
		ctx.JSON(response, options)

		/* Will output:
		   {
		       "lang": "GO-\u8679\u819c",
		       "tag": "\u003cbr\u003e"
		   }
		*/
	})

	// Do not replace special HTML characters with their unicode entities
	// using the UnescapeHTML field.
	app.Get("/json_raw", func(ctx iris.Context) {
		options := iris.JSON{UnescapeHTML: true}
		ctx.JSON(iris.Map{
			"html": "<b>Hello, world!</b>",
		}, options)

		// Will output: {"html":"<b>Hello, world!</b>"}
	})

	// Other content types,

	app.Get("/binary", func(ctx iris.Context) {
		// useful when you want force-download of contents of raw bytes form.
		ctx.Binary([]byte("Some binary data here."))
	})

	app.Get("/text", func(ctx iris.Context) {
		ctx.Text("Plain text here")
	})

	app.Get("/json", func(ctx iris.Context) {
		ctx.JSON(map[string]string{"hello": "json"}) // or myjsonStruct{hello:"json}
	})

	app.Get("/jsonp", func(ctx iris.Context) {
		ctx.JSONP(map[string]string{"hello": "jsonp"}, iris.JSONP{Callback: "callbackName"})
	})

	app.Get("/xml", func(ctx iris.Context) {
		ctx.XML(ExampleXML{One: "hello", Two: "xml"})
		// OR:
		// ctx.XML(iris.XMLMap("keys", iris.Map{"key": "value"}))
	})

	app.Get("/markdown", func(ctx iris.Context) {
		ctx.Markdown([]byte("# Hello Dynamic Markdown -- iris"))
	})

	app.Get("/yaml", func(ctx iris.Context) {
		ctx.YAML(ExampleYAML{Name: "Iris", ServerAddr: "localhost:8080"})
		// OR:
		// ctx.YAML(iris.Map{"name": "Iris", "serverAddr": "localhost:8080"})
	})

	// app.Get("/protobuf", func(ctx iris.Context) {
	// 	ctx.Protobuf(proto.Message)
	// })

	app.Get("/msgpack", func(ctx iris.Context) {
		u := User{
			Firstname: "John",
			Lastname:  "Doe",
			City:      "Neither FBI knows!!!",
			Age:       25,
		}

		ctx.MsgPack(u)
	})

	// http://localhost:8080/decode
	// http://localhost:8080/encode
	// http://localhost:8080/json_secure
	// http://localhost:8080/json_ascii
	//
	// http://localhost:8080/binary
	// http://localhost:8080/text
	// http://localhost:8080/json
	// http://localhost:8080/jsonp
	// http://localhost:8080/xml
	// http://localhost:8080/markdown
	// http://localhost:8080/msgpack
	//
	// `iris.WithOptimizations` is an optional configurator,
	// if passed to the `Run` then it will ensure that the application
	// response to the client as fast as possible.
	//
	//
	// `iris.WithoutServerError` is an optional configurator,
	// if passed to the `Run` then it will not print its passed error as an actual server error.
	app.Listen(":8080", iris.WithOptimizations)
}

type errorHandler struct{}

func (h *errorHandler) HandleContextError(ctx iris.Context, err error) {
	errors.Internal.Err(ctx, err)
}
