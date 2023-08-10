package main

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestSamePatternDifferentFuncUseGlobal(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	expectedResultFmt := "Called first middleware\nCalled second middleware\n%s\nCalled done: %s"
	tests := map[string]string{
		"/one-num":   "first route",
		"/two-num":   "second route",
		"/three-num": "third route",
	}

	for path, mainBody := range tests {
		result := fmt.Sprintf(expectedResultFmt, mainBody, path[1:])
		e.GET(path).Expect().Status(httptest.StatusOK).Body().IsEqual(result)
	}
}
