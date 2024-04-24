package main

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

// Shows a very basic usage of the httptest.
// The tests are written in a way to be easy to understand,
// for a more comprehensive testing examples check out the:
// _examples/routing/main_test.go,
// _examples/routing/subdomains/www/main_test.go
// _examples/file-server and e.t.c.
// Almost every example which covers
// a new feature from you to learn
// contains a test file as well.
func TestRoutingBasic(t *testing.T) {
	expectedUResponse := func(paramName, paramType, paramValue string) string {
		s := fmt.Sprintf("before %s (%s), current route name: GET/u/{%s:%s}\n", paramName, paramType, paramName, paramType)
		s += fmt.Sprintf("%s (%s): %s", paramName, paramType, paramValue)
		return s
	}

	var (
		expectedNotFoundResponse = "Custom route for 404 not found http code, here you can render a view, html, json <b>any valid response</b>."

		expectedIndexResponse = "Hello from /"
		expectedHomeResponse  = `Same as app.Handle("GET", "/", [...])`

		expectedUpathResponse         = ":string, :int, :uint, :alphabetical and :path in the same path pattern."
		expectedUStringResponse       = expectedUResponse("username", "string", "abcd123")
		expectedUIntResponse          = expectedUResponse("id", "int", "-1")
		expectedUUintResponse         = expectedUResponse("uid", "uint", "42")
		expectedUAlphabeticalResponse = expectedUResponse("firstname", "alphabetical", "abcd")

		expectedAPIUsersIndexResponse = map[string]interface{}{"user_id": 42}

		expectedAdminIndexResponse = "<h1>Hello from admin/</h1>"

		expectedSubdomainV1IndexResponse                  = `Version 1 API. go to <a href="/api/users">/api/users</a>`
		expectedSubdomainV1APIUsersIndexResponse          = "All users"
		expectedSubdomainV1APIUsersIndexWithParamResponse = "user with id: 42"

		expectedSubdomainWildcardIndexResponse = "Subdomain can be anything, now you're here from: any-subdomain-here"
	)

	app := newApp()
	e := httptest.New(t, app)

	e.GET("/anotfound").Expect().Status(httptest.StatusNotFound).
		Body().IsEqual(expectedNotFoundResponse)

	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedIndexResponse)
	e.GET("/home").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedHomeResponse)

	e.GET("/u/some/path/here").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedUpathResponse)
	e.GET("/u/abcd123").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedUStringResponse)
	e.GET("/u/-1").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedUIntResponse)
	e.GET("/u/42").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedUUintResponse)
	e.GET("/u/abcd").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedUAlphabeticalResponse)

	e.GET("/api/users/42").Expect().Status(httptest.StatusOK).
		JSON().IsEqual(expectedAPIUsersIndexResponse)

	e.GET("/admin").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedAdminIndexResponse)

	e.Request("GET", "/").WithURL("http://v1.example.com").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedSubdomainV1IndexResponse)

	e.Request("GET", "/api/users").WithURL("http://v1.example.com").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedSubdomainV1APIUsersIndexResponse)

	e.Request("GET", "/api/users/42").WithURL("http://v1.example.com").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedSubdomainV1APIUsersIndexWithParamResponse)

	e.Request("GET", "/").WithURL("http://any-subdomain-here.example.com").Expect().Status(httptest.StatusOK).
		Body().IsEqual(expectedSubdomainWildcardIndexResponse)
}
