package httpexpect

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestCompactPrinter(t *testing.T) {
	printer := NewCompactPrinter(t)

	body1 := bytes.NewBufferString("body1")
	body2 := bytes.NewBufferString("body2")

	req1, _ := http.NewRequest("GET", "http://example.com", body1)
	req2, _ := http.NewRequest("GET", "http://example.com", nil)

	printer.Request(req1)
	printer.Request(req2)
	printer.Request(nil)

	printer.Response(&http.Response{Body: ioutil.NopCloser(body2)}, 0)
	printer.Response(&http.Response{}, 0)
	printer.Response(nil, 0)
}

func TestDebugPrinter(t *testing.T) {
	printer := NewDebugPrinter(t, true)

	body1 := bytes.NewBufferString("body1")
	body2 := bytes.NewBufferString("body2")

	req1, _ := http.NewRequest("GET", "http://example.com", body1)
	req2, _ := http.NewRequest("GET", "http://example.com", nil)

	printer.Request(req1)
	printer.Request(req2)
	printer.Request(nil)

	printer.Response(&http.Response{Body: ioutil.NopCloser(body2)}, 0)
	printer.Response(&http.Response{}, 0)
	printer.Response(nil, 0)
}
