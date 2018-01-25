package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kataras/iris/httptest"
)

func TestSubdomainRedirectWWW(t *testing.T) {
	app := newApp()
	root := strings.TrimSuffix(addr, ":80")

	e := httptest.New(t, app)

	tests := []struct {
		path     string
		response string
	}{
		{"/", fmt.Sprintf("This is the www.%s endpoint.", root)},
		{"/users", fmt.Sprintf("This is the www.%s/users endpoint.", root)},
		{"/users/login", fmt.Sprintf("This is the www.%s/users/login endpoint.", root)},
	}

	for _, test := range tests {
		e.GET(test.path).Expect().Status(httptest.StatusOK).Body().Equal(test.response)
	}

}
