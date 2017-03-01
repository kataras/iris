package iris_test

import (
	"encoding/xml"
	"strconv"
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/httptest"
)

type renderTestInformationType struct {
	XMLName    xml.Name `xml:"info"`
	FirstAttr  string   `xml:"first,attr"`
	SecondAttr string   `xml:"second,attr"`
	Name       string   `xml:"name" json:"name"`
	Birth      string   `xml:"birth" json:"birth"`
	Stars      int      `xml:"stars" json:"stars"`
}

func TestContextRenderRest(t *testing.T) {
	app := iris.New()
	app.Adapt(newTestNativeRouter())

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

	// data is not part of the render policy but let's test it here.
	app.Get("/data", func(ctx *iris.Context) {
		ctx.Data(iris.StatusOK, dataContents)
	})
	// text is not part of the render policy but let's test it here.
	app.Get("/text", func(ctx *iris.Context) {
		ctx.Text(iris.StatusOK, textContents)
	})

	app.Get("/jsonp", func(ctx *iris.Context) {
		ctx.JSONP(iris.StatusOK, JSONPCallback, JSONPContents)
	})

	app.Get("/json", func(ctx *iris.Context) {
		ctx.JSON(iris.StatusOK, JSONXMLContents)
	})
	app.Get("/xml", func(ctx *iris.Context) {
		ctx.XML(iris.StatusOK, JSONXMLContents)
	})

	app.Get("/markdown", func(ctx *iris.Context) {
		ctx.Markdown(iris.StatusOK, markdownContents)
	})

	e := httptest.New(app, t)
	dataT := e.GET("/data").Expect().Status(iris.StatusOK)
	dataT.Header("Content-Type").Equal("application/octet-stream")
	dataT.Body().Equal(string(dataContents))

	textT := e.GET("/text").Expect().Status(iris.StatusOK)
	textT.Header("Content-Type").Equal("text/plain; charset=UTF-8")
	textT.Body().Equal(textContents)

	JSONPT := e.GET("/jsonp").Expect().Status(iris.StatusOK)
	JSONPT.Header("Content-Type").Equal("application/javascript; charset=UTF-8")
	JSONPT.Body().Equal(JSONPCallback + `({"hello":"jsonp"});`)

	JSONT := e.GET("/json").Expect().Status(iris.StatusOK)
	JSONT.Header("Content-Type").Equal("application/json; charset=UTF-8")
	JSONT.JSON().Object().Equal(JSONXMLContents)

	XMLT := e.GET("/xml").Expect().Status(iris.StatusOK)
	XMLT.Header("Content-Type").Equal("text/xml; charset=UTF-8")
	XMLT.Body().Equal(`<` + JSONXMLContents.XMLName.Local + ` first="` + JSONXMLContents.FirstAttr + `" second="` + JSONXMLContents.SecondAttr + `"><name>` + JSONXMLContents.Name + `</name><birth>` + JSONXMLContents.Birth + `</birth><stars>` + strconv.Itoa(JSONXMLContents.Stars) + `</stars></info>`)

	markdownT := e.GET("/markdown").Expect().Status(iris.StatusOK)
	markdownT.Header("Content-Type").Equal("text/html; charset=UTF-8")
	markdownT.Body().Equal("<h1>" + markdownContents[2:] + "</h1>\n")
}
