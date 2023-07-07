package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestReadYAML(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	expectedResponse := `Received: main.product{Invoice:34843, Tax:251.42, Total:4443.52, Comments:"Late afternoon is best. Backup contact is Nancy Billsmer @ 338-4338."}`
	send := `invoice: 34843
tax  : 251.42
total: 4443.52
comments: >
    Late afternoon is best.
    Backup contact is Nancy
    Billsmer @ 338-4338.`

	e.POST("/").WithHeader("Content-Type", "application/x-yaml").WithBytes([]byte(send)).Expect().
		Status(httptest.StatusOK).Body().IsEqual(expectedResponse)
}
