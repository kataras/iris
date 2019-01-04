package main

import (
	"encoding/xml"

	"github.com/kataras/iris"
)

func main() {
	app := newApp()

	// use Postman or whatever to do a POST request
	// to the http://localhost:8080 with RAW BODY:
	/*
		<person name="Winston Churchill" age="90">
			<description>Description of this person, the body of this inner element.</description>
		</person>
	*/
	// and Content-Type to application/xml (optionally but good practise)
	//
	// The response should be:
	// Received: main.person{XMLName:xml.Name{Space:"", Local:"person"}, Name:"Winston Churchill", Age:90, Description:"Description of this person, the body of this inner element."}
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed), iris.WithOptimizations)
}

func newApp() *iris.Application {
	app := iris.New()
	app.Post("/", handler)

	return app
}

// simple xml stuff, read more at https://golang.org/pkg/encoding/xml
type person struct {
	XMLName     xml.Name `xml:"person"`      // element name
	Name        string   `xml:"name,attr"`   // ,attr for attribute.
	Age         int      `xml:"age,attr"`    // ,attr attribute.
	Description string   `xml:"description"` // inner element name, value is its body.
}

func handler(ctx iris.Context) {
	var p person
	if err := ctx.ReadXML(&p); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.WriteString(err.Error())
		return
	}

	ctx.Writef("Received: %#+v", p)
}
