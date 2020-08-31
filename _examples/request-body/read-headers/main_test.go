package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestReadHeaders(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)

	expectedOKBody := `myHeaders: main.myHeaders{RequestID:"373713f0-6b4b-42ea-ab9f-e2e04bc38e73", Authentication:"Bearer my-token"}`

	tests := []struct {
		headers map[string]string
		code    int
		body    string
	}{
		{headers: map[string]string{
			"X-Request-Id":   "373713f0-6b4b-42ea-ab9f-e2e04bc38e73",
			"Authentication": "Bearer my-token",
		}, code: 200, body: expectedOKBody},
		{headers: map[string]string{
			"x-request-id":   "373713f0-6b4b-42ea-ab9f-e2e04bc38e73",
			"authentication": "Bearer my-token",
		}, code: 200, body: expectedOKBody},
		{headers: map[string]string{
			"X-Request-ID":   "373713f0-6b4b-42ea-ab9f-e2e04bc38e73",
			"Authentication": "Bearer my-token",
		}, code: 200, body: expectedOKBody},
		{headers: map[string]string{
			"Authentication": "Bearer my-token",
		}, code: 500, body: "X-Request-Id is empty"},
		{headers: map[string]string{
			"X-Request-ID": "373713f0-6b4b-42ea-ab9f-e2e04bc38e73",
		}, code: 500, body: "Authentication is empty"},
		{headers: map[string]string{}, code: 500, body: "X-Request-Id is empty (and 1 other error)"},
	}

	for _, tt := range tests {
		e.GET("/").WithHeaders(tt.headers).Expect().Status(tt.code).Body().Equal(tt.body)
	}
}
