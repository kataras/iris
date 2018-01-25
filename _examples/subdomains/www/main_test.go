package main

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/httptest"
)

type testRoute struct {
	path      string
	method    string
	subdomain string
}

func (r testRoute) response() string {
	msg := fmt.Sprintf("\nInfo\n\nMethod: %s\nSubdomain: %s\nPath: %s", r.method, r.subdomain, r.path)
	return msg
}

func TestSubdomainWWW(t *testing.T) {
	app := newApp()

	tests := []testRoute{
		// host
		{"/", "GET", ""},
		{"/about", "GET", ""},
		{"/contact", "GET", ""},
		{"/api/users", "GET", ""},
		{"/api/users/42", "GET", ""},
		{"/api/users", "POST", ""},
		{"/api/users/42", "PUT", ""},
		// www sub domain
		{"/", "GET", "www"},
		{"/about", "GET", "www"},
		{"/contact", "GET", "www"},
		{"/api/users", "GET", "www"},
		{"/api/users/42", "GET", "www"},
		{"/api/users", "POST", "www"},
		{"/api/users/42", "PUT", "www"},
	}

	host := "localhost:1111"
	e := httptest.New(t, app, httptest.URL("http://"+host), httptest.Debug(false))

	for _, test := range tests {

		req := e.Request(test.method, test.path)
		if subdomain := test.subdomain; subdomain != "" {
			req.WithURL("http://" + subdomain + "." + host)
		}

		req.Expect().
			Status(httptest.StatusOK).
			Body().Equal(test.response())
	}

}
