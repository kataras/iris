package main

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"io"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

func TestContentNegotiation(t *testing.T) {
	var (
		expectedJSONResponse = testdata{
			Name: "test name",
			Age:  26,
		}
		expectedXMLResponse, _ = xml.Marshal(expectedJSONResponse)
		expectedHTMLResponse   = "<h1>Test Name</h1><h2>Age 26</h2>"
	)

	app := newApp()
	app.Configure(iris.WithOptimizations)
	e := httptest.New(t, app)

	e.GET("/resource").WithHeader("Accept", "application/json").
		Expect().Status(httptest.StatusOK).
		ContentType("application/json", "utf-8").
		JSON().IsEqual(expectedJSONResponse)
	e.GET("/resource").WithHeader("Accept", "application/xml").WithHeader("Accept-Charset", "iso-8859-7").
		Expect().Status(httptest.StatusOK).
		ContentType("application/xml", "iso-8859-7").
		Body().IsEqual(string(expectedXMLResponse))

	e.GET("/resource2").WithHeader("Accept", "application/json").
		Expect().Status(httptest.StatusOK).
		ContentType("application/json", "utf-8").
		JSON().IsEqual(expectedJSONResponse)
	e.GET("/resource2").WithHeader("Accept", "application/xml").
		Expect().Status(httptest.StatusOK).
		ContentType("application/xml", "utf-8").
		Body().IsEqual(string(expectedXMLResponse))
	e.GET("/resource2").WithHeader("Accept", "text/html").
		Expect().Status(httptest.StatusOK).
		ContentType("text/html", "utf-8").
		Body().IsEqual(expectedHTMLResponse)

	e.GET("/resource3").WithHeader("Accept", "application/json").
		Expect().Status(httptest.StatusOK).
		ContentType("application/json", "utf-8").
		JSON().IsEqual(expectedJSONResponse)
	e.GET("/resource3").WithHeader("Accept", "application/xml").
		Expect().Status(httptest.StatusOK).
		ContentType("application/xml", "utf-8").
		Body().IsEqual(string(expectedXMLResponse))

	// test html with "gzip" encoding algorithm.
	rawGzipResponse := e.GET("/resource3").WithHeader("Accept", "text/html").
		WithHeader("Accept-Encoding", "gzip").
		Expect().Status(httptest.StatusOK).
		ContentType("text/html", "utf-8").
		ContentEncoding("gzip").
		Body().Raw()

	zr, err := gzip.NewReader(bytes.NewReader([]byte(rawGzipResponse)))
	if err != nil {
		t.Fatal(err)
	}

	rawResponse, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}

	if expected, got := expectedHTMLResponse, string(rawResponse); expected != got {
		t.Fatalf("expected response to be:\n%s but got:\n%s", expected, got)
	}
}
