package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestGzipReader(t *testing.T) {
	app := newApp()

	expected := payload{Message: "test"}
	b, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	w := gzip.NewWriter(buf)
	_, err = w.Write(b)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}

	e := httptest.New(t, app)
	// send gzip compressed.
	e.POST("/").WithHeader("Content-Encoding", "gzip").WithHeader("Content-Type", "application/json").
		WithBytes(buf.Bytes()).Expect().Status(httptest.StatusOK).Body().Equal(expected.Message)
	// raw.
	e.POST("/").WithJSON(expected).Expect().Status(httptest.StatusOK).Body().Equal(expected.Message)
}
