package main

import (
	"testing"

	"github.com/GoLandr/iris/httptest"
)

func TestReadCustomPerType(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	expectedResponse := `Received: main.config{Addr:"localhost:8080", ServerName:"Iris"}`

	e.POST("/").WithText("addr: localhost:8080\nserverName: Iris").Expect().
		Status(httptest.StatusOK).Body().Equal(expectedResponse)
}
