package main

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestSecureCookie(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.URL("http://example.com"))

	cookieName, cookieValue := "my_cookie_name", "my_cookie_value"

	// Test set a Cookie.
	t1 := e.GET(fmt.Sprintf("/cookies/%s/%s", cookieName, cookieValue)).Expect().Status(httptest.StatusOK)
	// note that this will not work because it doesn't always returns the same value:
	// cookieValueEncoded, _ := sc.Encode(cookieName, cookieValue)
	t1.Cookie(cookieName).Value().NotEqual(cookieValue) // validate cookie's existence and value is not on its raw form.
	t1.Body().Contains(cookieValue)

	// Test retrieve a Cookie.
	t2 := e.GET(fmt.Sprintf("/cookies/%s", cookieName)).Expect().Status(httptest.StatusOK)
	t2.Body().IsEqual(cookieValue)

	// Test remove a Cookie.
	t3 := e.GET(fmt.Sprintf("/cookies/remove/%s", cookieName)).Expect().Status(httptest.StatusOK)
	t3.Body().Contains(cookieName)

	t4 := e.GET(fmt.Sprintf("/cookies/%s", cookieName)).Expect().Status(httptest.StatusOK)
	t4.Cookies().Empty()
	t4.Body().IsEmpty()
}
