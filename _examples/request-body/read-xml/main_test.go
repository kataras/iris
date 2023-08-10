package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestReadXML(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	expectedResponse := `Received: main.person{XMLName:xml.Name{Space:"", Local:"person"}, Name:"Winston Churchill", Age:90, Description:"Description of this person, the body of this inner element."}`
	send := `<person name="Winston Churchill" age="90"><description>Description of this person, the body of this inner element.</description></person>`

	e.POST("/").WithText(send).Expect().
		Status(httptest.StatusOK).Body().IsEqual(expectedResponse)
}
