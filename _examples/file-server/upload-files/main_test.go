package main

import (
	"os"
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestUploadFiles(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	// upload the file itself.
	fh, err := os.Open("main.go")
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()

	e.POST("/upload").WithMultipart().WithFile("files", "main.go", fh).
		Expect().Status(httptest.StatusOK)

	f, err := os.Open("uploads/main.go")
	if err != nil {
		t.Fatalf("expected file to get actually uploaded on the system directory but: %v", err)
	}
	f.Close()

	os.Remove(f.Name())
}
