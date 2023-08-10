package main

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestCookieOptions(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.URL("http://example.com"))

	cookieName, cookieValue := "my_cookie_name", "my_cookie_value"

	// Test set a Cookie.
	t1 := e.GET(fmt.Sprintf("/set/%s/%s", cookieName, cookieValue)).Expect().Status(httptest.StatusOK)
	t1.Cookie(cookieName).Value().Equal(cookieValue)
	t1.Body().Contains(fmt.Sprintf("%s=%s", cookieName, cookieValue))

	// Test retrieve a Cookie.
	t2 := e.GET(fmt.Sprintf("/get/%s", cookieName)).Expect().Status(httptest.StatusOK)
	t2.Body().IsEqual(cookieValue)

	// Test remove a Cookie.
	t3 := e.GET(fmt.Sprintf("/remove/%s", cookieName)).Expect().Status(httptest.StatusOK)
	t3.Body().Contains(fmt.Sprintf("cookie %s removed, value should be empty=%s", cookieName, ""))

	t4 := e.GET(fmt.Sprintf("/get/%s", cookieName)).Expect().Status(httptest.StatusOK)
	t4.Cookies().Empty()
	t4.Body().IsEmpty()
}
