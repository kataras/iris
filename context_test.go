package iris

import (
	"net/http"
	"testing"
)

func TestContext_GetCookie(t *testing.T) {

	request, _ := http.NewRequest("GET", "/", nil)

	cookie := &http.Cookie{Name: "cookie-name", Value: "cookie-value"}
	request.AddCookie(cookie)

	context := &Context{Request: request}

	value := context.GetCookie("cookie-name")

	if value != "cookie-value" {
		t.Fatal("GetCookie should return \"cookie-value\", but returned: \"", value, "\"")
	}

}

func TestContext_GetCookie_Err(t *testing.T) {

	request, _ := http.NewRequest("GET", "/", nil)
	context := &Context{Request: request}

	value := context.GetCookie("cookie-name")

	if value != "" {
		t.Fatal("GetCookie should be empty, but returned: \"", value, "\"")
	}

}
