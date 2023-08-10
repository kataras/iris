package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestReadBodyAndNegotiate(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)

	var (
		expectedPayload        = payload{Message: "a message"}
		expectedMsgPackPayload = "\x81\xa7message\xa9a message"
		expectedXMLPayload     = `<payload><message>a message</message></payload>`
		expectedYAMLPayload    = "Message: a message\n"
	)

	// Test send JSON and receive JSON.
	e.POST("/").WithJSON(expectedPayload).Expect().Status(httptest.StatusOK).
		JSON().IsEqual(expectedPayload)

	// Test send Form and receive XML.
	e.POST("/").WithForm(expectedPayload).
		WithHeader("Accept", "application/xml").
		Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedXMLPayload)

	// Test send URL Query and receive MessagePack.
	e.POST("/").WithQuery("message", expectedPayload.Message).
		WithHeader("Accept", "application/msgpack").
		Expect().Status(httptest.StatusOK).ContentType("application/msgpack").
		Body().IsEqual(expectedMsgPackPayload)

	// Test send MessagePack and receive MessagePack.
	e.POST("/").WithBytes([]byte(expectedMsgPackPayload)).
		WithHeader("Content-Type", "application/msgpack").
		WithHeader("Accept", "application/msgpack").
		Expect().Status(httptest.StatusOK).
		ContentType("application/msgpack").Body().IsEqual(expectedMsgPackPayload)

	// Test send YAML and receive YAML.
	e.POST("/").WithBytes([]byte(expectedYAMLPayload)).
		WithHeader("Content-Type", "application/x-yaml").
		WithHeader("Accept", "application/x-yaml").
		Expect().Status(httptest.StatusOK).
		ContentType("application/x-yaml").Body().IsEqual(expectedYAMLPayload)
}
