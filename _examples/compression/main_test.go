package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"
)

func TestCompression(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	var expectedReply = payload{Username: "Makis"}
	body := e.GET("/").WithHeader(context.AcceptEncodingHeaderKey, context.GZIP).Expect().
		Status(httptest.StatusOK).
		ContentEncoding(context.GZIP).
		ContentType(context.ContentJSONHeaderValue).Body().Raw()

	// Note that .Expect() consumes the response body
	// and stores it to unexported "contents" field
	// therefore, we retrieve it as string and put it to a new buffer.
	r := strings.NewReader(body)
	cr, err := context.NewCompressReader(r, context.GZIP)
	if err != nil {
		t.Fatal(err)
	}
	defer cr.Close()

	var got payload
	if err = json.NewDecoder(cr).Decode(&got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedReply, got) {
		t.Fatalf("expected %#+v but got %#+v", expectedReply, got)
	}
}
