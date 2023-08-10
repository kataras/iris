package main

import (
	"testing"

	"github.com/kataras/iris/v12/core/memstore"
	"github.com/kataras/iris/v12/httptest"
)

func TestSameParameterTypeDifferentMacroFunctions(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	type resp struct {
		Handler string         `json:"handler"`
		Params  memstore.Store `json:"params"`
	}

	var (
		expectedIndex = resp{
			Handler: "iris/_examples/routing/dynamic-path/same-pattern-different-func.handler1",
			Params:  nil,
		}
		expectedHTMLPage = resp{
			Handler: "iris/_examples/routing/dynamic-path/same-pattern-different-func.handler1",
			Params: memstore.Store{
				{Key: "page", ValueRaw: "random.html"},
			},
		}
		expectedZipName = resp{
			Handler: "iris/_examples/routing/dynamic-path/same-pattern-different-func.handler2",
			Params: memstore.Store{
				{Key: "name", ValueRaw: "random.zip"},
			},
		}
	)

	e.GET("/").Expect().Status(httptest.StatusOK).JSON().IsEqual(expectedIndex)
	e.GET("/api/random.html").Expect().Status(httptest.StatusOK).JSON().IsEqual(expectedHTMLPage)
	e.GET("/api/random.zip").Expect().Status(httptest.StatusOK).JSON().IsEqual(expectedZipName)
}
