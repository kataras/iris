package httpexpect

import (
	"bufio"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

type mockHandler struct {
	t       *testing.T
	https   bool
	http10  bool
	chunked bool
}

func (c *mockHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	assert.True(c.t, err == nil)

	if c.http10 {
		assert.Equal(c.t, "HTTP/1.0", req.Proto)
		assert.Equal(c.t, 1, req.ProtoMajor)
		assert.Equal(c.t, 0, req.ProtoMinor)
	} else {
		assert.Equal(c.t, "HTTP/1.1", req.Proto)
		assert.Equal(c.t, 1, req.ProtoMajor)
		assert.Equal(c.t, 1, req.ProtoMinor)
	}
	if c.https {
		assert.Equal(c.t, "https://example.com/path", req.URL.String())
	} else {
		assert.Equal(c.t, "http://example.com/path", req.URL.String())
	}
	assert.Equal(c.t, "/path", req.URL.RequestURI())
	assert.Equal(c.t, "/path", req.RequestURI)
	assert.Equal(c.t, "GET", req.Method)
	assert.Equal(c.t, "body", string(body))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(`{"hello":"world"}`))
	if c.chunked {
		w.(http.Flusher).Flush()
	}

	assert.True(c.t, err == nil)
}

func TestBinder(t *testing.T) {
	handler := &mockHandler{
		t:       t,
		http10:  true,
		chunked: false,
	}

	client := &http.Client{
		Transport: NewBinder(handler),
	}

	req, err := http.NewRequest("GET", "http://example.com/path", strings.NewReader("body"))
	if err != nil {
		t.Fatal(err)
	}

	req.ProtoMajor = 1
	req.ProtoMinor = 0
	req.Proto = ""

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	header := http.Header{
		"Content-Type": {"application/json"},
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, header, resp.Header)
	assert.Equal(t, `{"hello":"world"}`, string(b))

	assert.Equal(t, []string(nil), resp.TransferEncoding)
}

func TestBinderTLS(t *testing.T) {
	handler := &mockHandler{t: t}

	httpClient := &http.Client{
		Transport: &Binder{
			Handler: handler,
			TLS:     nil,
		},
	}

	tlsState := &tls.ConnectionState{}

	httpsClient := &http.Client{
		Transport: &Binder{
			Handler: handler,
			TLS:     tlsState,
		},
	}

	handler.https = false
	req, _ := http.NewRequest("GET", "http://example.com/path", strings.NewReader("body"))
	resp, err := httpClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Nil(t, req.TLS)

	handler.https = true
	req, _ = http.NewRequest("GET", "https://example.com/path", strings.NewReader("body"))
	resp, err = httpClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Nil(t, req.TLS)

	handler.https = false
	req, _ = http.NewRequest("GET", "http://example.com/path", strings.NewReader("body"))
	resp, err = httpsClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Nil(t, req.TLS)

	handler.https = true
	req, _ = http.NewRequest("GET", "https://example.com/path", strings.NewReader("body"))
	resp, err = httpsClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, req.TLS)
	assert.Equal(t, tlsState, req.TLS)
}

func TestBinderChunked(t *testing.T) {
	handler := &mockHandler{
		t:       t,
		chunked: true,
	}

	client := &http.Client{
		Transport: NewBinder(handler),
	}

	req, err := http.NewRequest("GET", "http://example.com/path", strings.NewReader("body"))
	if err != nil {
		t.Fatal(err)
	}

	req.ContentLength = -1

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"chunked"}, req.TransferEncoding)
	assert.Equal(t, []string{"chunked"}, resp.TransferEncoding)
}

func TestFastBinder(t *testing.T) {
	handler := func(ctx *fasthttp.RequestCtx) {
		assert.Equal(t, "POST", string(ctx.Request.Header.Method()))
		assert.Equal(t, "http://example.com/path", string(ctx.Request.Header.RequestURI()))

		assert.Equal(t, "application/x-www-form-urlencoded",
			string(ctx.Request.Header.ContentType()))

		headers := map[string][]string{}

		ctx.Request.Header.VisitAll(func(k, v []byte) {
			headers[string(k)] = append(headers[string(k)], string(v))
		})

		expected := map[string][]string{
			"Content-Type":   {"application/x-www-form-urlencoded"},
			"Content-Length": {"7"},
			"Some-Header":    {"foo", "bar"},
		}

		assert.Equal(t, expected, headers)

		assert.Equal(t, "bar", string(ctx.FormValue("foo")))
		assert.Equal(t, "foo=bar", string(ctx.Request.Body()))

		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.SetBody([]byte(`{"hello":"world"}`))
	}

	client := &http.Client{
		Transport: NewFastBinder(handler),
	}

	req, err := http.NewRequest("POST", "http://example.com/path", strings.NewReader("foo=bar"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Some-Header", "foo")
	req.Header.Add("Some-Header", "bar")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	header := http.Header{
		"Content-Type": {"application/json"},
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, header, resp.Header)
	assert.Equal(t, `{"hello":"world"}`, string(b))

	assert.Equal(t, []string(nil), resp.TransferEncoding)
}

func TestFastBinderTLS(t *testing.T) {
	var isHTTPS, isTLS bool

	tlsState := &tls.ConnectionState{}

	handler := func(ctx *fasthttp.RequestCtx) {
		isHTTPS = strings.HasPrefix(string(ctx.Request.Header.RequestURI()), "https://")
		isTLS = ctx.IsTLS()
		if isTLS {
			assert.Equal(t, *tlsState, *ctx.TLSConnectionState())
		}
	}

	httpClient := &http.Client{
		Transport: &FastBinder{
			Handler: handler,
			TLS:     nil,
		},
	}

	httpsClient := &http.Client{
		Transport: &FastBinder{
			Handler: handler,
			TLS:     tlsState,
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com/path", strings.NewReader("body"))
	resp, err := httpClient.Do(req)
	assert.Nil(t, err)
	assert.False(t, isHTTPS)
	assert.False(t, isTLS)

	req, _ = http.NewRequest("GET", "https://example.com/path", strings.NewReader("body"))
	resp, err = httpClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.True(t, isHTTPS)
	assert.False(t, isTLS)

	req, _ = http.NewRequest("GET", "http://example.com/path", strings.NewReader("body"))
	resp, err = httpsClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.False(t, isHTTPS)
	assert.False(t, isTLS)

	req, _ = http.NewRequest("GET", "https://example.com/path", strings.NewReader("body"))
	resp, err = httpsClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.True(t, isHTTPS)
	assert.True(t, isTLS)
}

func TestFastBinderChunked(t *testing.T) {
	handler := func(ctx *fasthttp.RequestCtx) {
		assert.Equal(t, "POST", string(ctx.Request.Header.Method()))
		assert.Equal(t, "http://example.com/path", string(ctx.Request.Header.RequestURI()))

		assert.Equal(t, "application/x-www-form-urlencoded",
			string(ctx.Request.Header.ContentType()))

		headers := map[string][]string{}

		ctx.Request.Header.VisitAll(func(k, v []byte) {
			headers[string(k)] = append(headers[string(k)], string(v))
		})

		expected := map[string][]string{
			"Content-Type":      {"application/x-www-form-urlencoded"},
			"Transfer-Encoding": {"chunked"},
		}

		assert.Equal(t, expected, headers)

		assert.Equal(t, "bar", string(ctx.FormValue("foo")))
		assert.Equal(t, "foo=bar", string(ctx.Request.Body()))

		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
			w.WriteString(`[1, `)
			w.Flush()
			w.WriteString(`2]`)
		})
	}

	client := &http.Client{
		Transport: NewFastBinder(handler),
	}

	req, err := http.NewRequest("POST", "http://example.com/path", strings.NewReader("foo=bar"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.ContentLength = -1

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"chunked"}, resp.TransferEncoding)
}

func TestFastBinderEmptyResponse(t *testing.T) {
	handler := func(*fasthttp.RequestCtx) {}

	client := &http.Client{
		Transport: NewFastBinder(handler),
	}

	req, err := http.NewRequest("POST", "http://example.com/path", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, resp.Body == nil)

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", string(b))
}
