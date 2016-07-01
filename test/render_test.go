package test

// Contains tests for render/rest & render/template

import (
	"encoding/xml"
	"strconv"
	"testing"

	"github.com/kataras/iris"
)

type renderTestInformationType struct {
	XMLName    xml.Name `xml:"info"`
	FirstAttr  string   `xml:"first,attr"`
	SecondAttr string   `xml:"second,attr"`
	Name       string   `xml:"name",json:"name"`
	Birth      string   `xml:"birth",json:"birth"`
	Stars      int      `xml:"stars",json:"stars"`
}

func TestRenderRest(t *testing.T) {
	api := iris.New()

	dataContents := []byte("Some binary data here.")
	textContents := "Plain text here"
	JSONPContents := map[string]string{"hello": "jsonp"}
	JSONPCallback := "callbackName"
	JSONXMLContents := renderTestInformationType{
		XMLName:    xml.Name{Local: "info", Space: "info"}, // only need to verify that later
		FirstAttr:  "this is the first attr",
		SecondAttr: "this is the second attr",
		Name:       "Iris web framework",
		Birth:      "13 March 2016",
		Stars:      4064,
	}
	markdownContents := "# Hello dynamic markdown from Iris"

	api.Get("/data", func(ctx *iris.Context) {
		ctx.Data(iris.StatusOK, dataContents)
	})

	api.Get("/text", func(ctx *iris.Context) {
		ctx.Text(iris.StatusOK, textContents)
	})

	api.Get("/jsonp", func(ctx *iris.Context) {
		ctx.JSONP(iris.StatusOK, JSONPCallback, JSONPContents)
	})

	api.Get("/json", func(ctx *iris.Context) {
		ctx.JSON(iris.StatusOK, JSONXMLContents)
	})
	api.Get("/xml", func(ctx *iris.Context) {
		ctx.XML(iris.StatusOK, JSONXMLContents)
	})

	api.Get("/markdown", func(ctx *iris.Context) {
		ctx.Markdown(iris.StatusOK, markdownContents)
	})

	e := tester(api, t)
	dataT := e.GET("/data").Expect()
	dataT.Header("Content-Type").Equal("application/octet-stream")
	dataT.Body().Equal(string(dataContents))

	textT := e.GET("/text").Expect()
	textT.Header("Content-Type").Equal("text/plain; charset=UTF-8")
	textT.Body().Equal(textContents)

	JSONPT := e.GET("/jsonp").Expect()
	JSONPT.Header("Content-Type").Equal("application/javascript; charset=UTF-8")
	JSONPT.Body().Equal(JSONPCallback + `({"hello":"jsonp"});`)

	JSONT := e.GET("/json").Expect()
	JSONT.Header("Content-Type").Equal("application/json; charset=UTF-8")
	JSONT.JSON().Object().Equal(JSONXMLContents)

	XMLT := e.GET("/xml").Expect()
	XMLT.Header("Content-Type").Equal("text/xml; charset=UTF-8")
	XMLT.Body().Equal(`<` + JSONXMLContents.XMLName.Local + ` first="` + JSONXMLContents.FirstAttr + `" second="` + JSONXMLContents.SecondAttr + `"><name>` + JSONXMLContents.Name + `</name><birth>` + JSONXMLContents.Birth + `</birth><stars>` + strconv.Itoa(JSONXMLContents.Stars) + `</stars></info>`)

	markdownT := e.GET("/markdown").Expect()
	markdownT.Header("Content-Type").Equal("text/html; charset=UTF-8")
	markdownT.Body().Equal("<h1>" + markdownContents[2:] + "</h1>\n")

}
