package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"
)

func TestCompression(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	var expectedReply = payload{Username: "Makis"}
	testBody(t, e.GET("/"), expectedReply)
}

func TestCompressionAfterRecorder(t *testing.T) {
	var expectedReply = payload{Username: "Makis"}

	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Record()
		ctx.Next()
	})
	app.Use(iris.Compression)

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(expectedReply)
	})

	e := httptest.New(t, app)
	testBody(t, e.GET("/"), expectedReply)
}

func TestCompressionBeforeRecorder(t *testing.T) {
	var expectedReply = payload{Username: "Makis"}

	app := iris.New()
	app.Use(iris.Compression)
	app.Use(func(ctx iris.Context) {
		ctx.Record()
		ctx.Next()
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(expectedReply)
	})

	e := httptest.New(t, app)
	testBody(t, e.GET("/"), expectedReply)
}

func testBody(t *testing.T, req *httptest.Request, expectedReply interface{}) {
	t.Helper()

	body := req.WithHeader(context.AcceptEncodingHeaderKey, context.GZIP).Expect().
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
