package compress_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/compress"

	"github.com/klauspost/compress/zstd"
)

func TestHandler(t *testing.T) {
	// Server.
	app := iris.New()
	app.Use(compress.Handler)

	app.Post("/", func(ctx iris.Context) {
		body, _ := ctx.Body()
		ctx.Writef("Received: %s", string(body))
	})

	// Test gzip request and response.
	reqBody := "Hello, World!"
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(reqBody))
	if err != nil {
		t.Fatalf("Failed to write gzip: %v", err)
	}
	gz.Close()

	e := httptest.New(t, app)
	res := e.POST("/").WithBytes(buf.Bytes()).
		WithHeader("Accept-Encoding", "gzip").
		WithHeader("Content-Encoding", "gzip").
		Expect().Status(httptest.StatusOK).HasContentEncoding("gzip").Raw()

	gzRes, err := gzip.NewReader(res.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzRes.Close()

	resBody, err := io.ReadAll(gzRes)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expected := "Received: " + reqBody
	if strings.TrimSpace(string(resBody)) != expected {
		t.Errorf("Expected response body %q, got %q", expected, resBody)
	}
}

func TestHandlerZstd(t *testing.T) {
	// Server.
	app := iris.New()
	app.Use(compress.Handler)

	app.Post("/", func(ctx iris.Context) {
		body, _ := ctx.Body()
		ctx.Writef("Received: %s", string(body))
	})

	// Test zstd request and response.
	reqBody := "Hello, World!"
	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("Failed to create zstd writer: %v", err)
	}
	if _, err = zw.Write([]byte(reqBody)); err != nil {
		t.Fatalf("Failed to write zstd: %v", err)
	}
	zw.Close()

	e := httptest.New(t, app)
	res := e.POST("/").WithBytes(buf.Bytes()).
		WithHeader("Accept-Encoding", "zstd").
		WithHeader("Content-Encoding", "zstd").
		Expect().Status(httptest.StatusOK).HasContentEncoding("zstd").Raw()

	zr, err := zstd.NewReader(res.Body)
	if err != nil {
		t.Fatalf("Failed to create zstd reader: %v", err)
	}
	defer zr.Close()

	resBody, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expected := "Received: " + reqBody
	if strings.TrimSpace(string(resBody)) != expected {
		t.Errorf("Expected response body %q, got %q", expected, resBody)
	}
}
