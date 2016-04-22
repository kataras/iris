package fasthttpadaptor

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestNewFastHTTPHandler(t *testing.T) {
	expectedMethod := "POST"
	expectedProto := "HTTP/1.1"
	expectedProtoMajor := 1
	expectedProtoMinor := 1
	expectedRequestURI := "/foo/bar?baz=123"
	expectedBody := "body 123 foo bar baz"
	expectedContentLength := len(expectedBody)
	expectedHost := "foobar.com"
	expectedRemoteAddr := "1.2.3.4:6789"
	expectedHeader := map[string]string{
		"Foo-Bar":         "baz",
		"Abc":             "defg",
		"XXX-Remote-Addr": "123.43.4543.345",
	}
	expectedURL, err := url.ParseRequestURI(expectedRequestURI)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	callsCount := 0
	nethttpH := func(w http.ResponseWriter, r *http.Request) {
		callsCount++
		if r.Method != expectedMethod {
			t.Fatalf("unexpected method %q. Expecting %q", r.Method, expectedMethod)
		}
		if r.Proto != expectedProto {
			t.Fatalf("unexpected proto %q. Expecting %q", r.Proto, expectedProto)
		}
		if r.ProtoMajor != expectedProtoMajor {
			t.Fatalf("unexpected protoMajor %d. Expecting %d", r.ProtoMajor, expectedProtoMajor)
		}
		if r.ProtoMinor != expectedProtoMinor {
			t.Fatalf("unexpected protoMinor %d. Expecting %d", r.ProtoMinor, expectedProtoMinor)
		}
		if r.RequestURI != expectedRequestURI {
			t.Fatalf("unexpected requestURI %q. Expecting %q", r.RequestURI, expectedRequestURI)
		}
		if r.ContentLength != int64(expectedContentLength) {
			t.Fatalf("unexpected contentLength %d. Expecting %d", r.ContentLength, expectedContentLength)
		}
		if r.Host != expectedHost {
			t.Fatalf("unexpected host %q. Expecting %q", r.Host, expectedHost)
		}
		if r.RemoteAddr != expectedRemoteAddr {
			t.Fatalf("unexpected remoteAddr %q. Expecting %q", r.RemoteAddr, expectedRemoteAddr)
		}
		body, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			t.Fatalf("unexpected error when reading request body: %s", err)
		}
		if string(body) != expectedBody {
			t.Fatalf("unexpected body %q. Expecting %q", body, expectedBody)
		}
		if !reflect.DeepEqual(r.URL, expectedURL) {
			t.Fatalf("unexpected URL: %#v. Expecting %#v", r.URL, expectedURL)
		}

		for k, expectedV := range expectedHeader {
			v := r.Header.Get(k)
			if v != expectedV {
				t.Fatalf("unexpected header value %q for key %q. Expecting %q", v, k, expectedV)
			}
		}

		w.Header().Set("Header1", "value1")
		w.Header().Set("Header2", "value2")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "request body is %q", body)
	}
	fasthttpH := NewFastHTTPHandler(http.HandlerFunc(nethttpH))

	var ctx fasthttp.RequestCtx
	var req fasthttp.Request

	req.Header.SetMethod(expectedMethod)
	req.SetRequestURI(expectedRequestURI)
	req.Header.SetHost(expectedHost)
	req.BodyWriter().Write([]byte(expectedBody))
	for k, v := range expectedHeader {
		req.Header.Set(k, v)
	}

	remoteAddr, err := net.ResolveTCPAddr("tcp", expectedRemoteAddr)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx.Init(&req, remoteAddr, nil)

	fasthttpH(&ctx)

	if callsCount != 1 {
		t.Fatalf("unexpected callsCount: %d. Expecting 1", callsCount)
	}

	resp := &ctx.Response
	if resp.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("unexpected statusCode: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusBadRequest)
	}
	if string(resp.Header.Peek("Header1")) != "value1" {
		t.Fatalf("unexpected header value: %q. Expecting %q", resp.Header.Peek("Header1"), "value1")
	}
	if string(resp.Header.Peek("Header2")) != "value2" {
		t.Fatalf("unexpected header value: %q. Expecting %q", resp.Header.Peek("Header2"), "value2")
	}
	expectedResponseBody := fmt.Sprintf("request body is %q", expectedBody)
	if string(resp.Body()) != expectedResponseBody {
		t.Fatalf("unexpected response body %q. Expecting %q", resp.Body(), expectedResponseBody)
	}
}
