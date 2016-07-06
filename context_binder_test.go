package iris

import (
	"encoding/xml"
	"net/url"
	"strconv"
	"testing"
)

// Contains tests for context.ReadJSON/ReadXML/ReadFORM

type testBinderData struct {
	Username string
	Mail     string
	Data     []string `form:"mydata" json:"mydata"`
}

type testBinderXMLData struct {
	XMLName    xml.Name `xml:"info"`
	FirstAttr  string   `xml:"first,attr"`
	SecondAttr string   `xml:"second,attr"`
	Name       string   `xml:"name",json:"name"`
	Birth      string   `xml:"birth",json:"birth"`
	Stars      int      `xml:"stars",json:"stars"`
}

func TestBindForm(t *testing.T) {
	initDefault()

	Post("/form", func(ctx *Context) {
		obj := testBinderData{}
		err := ctx.ReadForm(&obj)
		if err != nil {
			t.Fatalf("Error when parsing the FORM: %s", err.Error())
		}
		ctx.JSON(StatusOK, obj)
	})

	e := Tester(t)

	passed := map[string]interface{}{"Username": "myusername", "Mail": "mymail@iris-go.com", "mydata": url.Values{"[0]": []string{"mydata1"},
		"[1]": []string{"mydata2"}}}

	expectedObject := testBinderData{Username: "myusername", Mail: "mymail@iris-go.com", Data: []string{"mydata1", "mydata2"}}

	e.POST("/form").WithForm(passed).Expect().Status(StatusOK).JSON().Object().Equal(expectedObject)
}

func TestBindJSON(t *testing.T) {
	initDefault()
	Post("/json", func(ctx *Context) {
		obj := testBinderData{}
		err := ctx.ReadJSON(&obj)
		if err != nil {
			t.Fatalf("Error when parsing the JSON body: %s", err.Error())
		}
		ctx.JSON(StatusOK, obj)
	})

	e := Tester(t)
	passed := map[string]interface{}{"Username": "myusername", "Mail": "mymail@iris-go.com", "mydata": []string{"mydata1", "mydata2"}}
	expectedObject := testBinderData{Username: "myusername", Mail: "mymail@iris-go.com", Data: []string{"mydata1", "mydata2"}}

	e.POST("/json").WithJSON(passed).Expect().Status(StatusOK).JSON().Object().Equal(expectedObject)
}

func TestBindXML(t *testing.T) {
	initDefault()

	Post("/xml", func(ctx *Context) {
		obj := testBinderXMLData{}
		err := ctx.ReadXML(&obj)
		if err != nil {
			t.Fatalf("Error when parsing the XML body: %s", err.Error())
		}
		ctx.XML(StatusOK, obj)
	})

	e := Tester(t)
	expectedObj := testBinderXMLData{
		XMLName:    xml.Name{Local: "info", Space: "info"},
		FirstAttr:  "this is the first attr",
		SecondAttr: "this is the second attr",
		Name:       "Iris web framework",
		Birth:      "13 March 2016",
		Stars:      4064,
	}
	// so far no WithXML or .XML like WithJSON and .JSON on httpexpect I added a feature request as post issue and we're waiting
	expectedBody := `<` + expectedObj.XMLName.Local + ` first="` + expectedObj.FirstAttr + `" second="` + expectedObj.SecondAttr + `"><name>` + expectedObj.Name + `</name><birth>` + expectedObj.Birth + `</birth><stars>` + strconv.Itoa(expectedObj.Stars) + `</stars></info>`
	e.POST("/xml").WithText(expectedBody).Expect().Status(StatusOK).Body().Equal(expectedBody)
}
