package httpexpect

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestFailed(t *testing.T) {
	client := &mockClient{}

	chain := makeChain(newMockReporter(t))

	chain.fail("fail")

	config := Config{
		Client: client,
	}

	req := &Request{
		config: config,
		chain:  chain,
		http:   nil,
	}

	req.WithPath("foo", "bar")
	req.WithPathObject(map[string]interface{}{"foo": "bar"})
	req.WithQuery("foo", "bar")
	req.WithQueryObject(map[string]interface{}{"foo": "bar"})
	req.WithQueryString("foo=bar")
	req.WithURL("http://example.com")
	req.WithHeaders(map[string]string{"foo": "bar"})
	req.WithHeader("foo", "bar")
	req.WithCookies(map[string]string{"foo": "bar"})
	req.WithCookie("foo", "bar")
	req.WithBasicAuth("foo", "bar")
	req.WithProto("HTTP/1.1")
	req.WithChunked(strings.NewReader("foo"))
	req.WithBytes([]byte("foo"))
	req.WithText("foo")
	req.WithJSON(map[string]string{"foo": "bar"})
	req.WithForm(map[string]string{"foo": "bar"})
	req.WithFormField("foo", "bar")
	req.WithFile("foo", "bar", strings.NewReader("baz"))
	req.WithFileBytes("foo", "bar", []byte("baz"))
	req.WithMultipart()

	resp := req.Expect()
	assert.False(t, resp == nil)

	req.chain.assertFailed(t)
	resp.chain.assertFailed(t)
}

func TestRequestEmpty(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "", "")

	resp := req.Expect()

	req.chain.assertOK(t)
	resp.chain.assertOK(t)
}

func TestRequestTime(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	for n := 0; n < 10; n++ {
		req := NewRequest(config, "", "")
		resp := req.Expect()
		assert.True(t, resp.time >= 0)
	}
}

func TestRequestProto(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "/")

	assert.Equal(t, 1, req.http.ProtoMajor)
	assert.Equal(t, 1, req.http.ProtoMinor)

	req.WithProto("HTTP/2.0")

	assert.Equal(t, 2, req.http.ProtoMajor)
	assert.Equal(t, 0, req.http.ProtoMinor)

	req.WithProto("HTTP/1.0")

	assert.Equal(t, 1, req.http.ProtoMajor)
	assert.Equal(t, 0, req.http.ProtoMinor)

	req.WithProto("bad")
	req.chain.assertFailed(t)

	assert.Equal(t, 1, req.http.ProtoMajor)
	assert.Equal(t, 0, req.http.ProtoMinor)
}

func TestRequestURLConcatenate(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := NewAssertReporter(t)

	config1 := Config{
		RequestFactory: factory,
		BaseURL:        "",
		Client:         client,
		Reporter:       reporter,
	}

	config2 := Config{
		RequestFactory: factory,
		BaseURL:        "http://example.com",
		Client:         client,
		Reporter:       reporter,
	}

	config3 := Config{
		RequestFactory: factory,
		BaseURL:        "http://example.com/",
		Client:         client,
		Reporter:       reporter,
	}

	reqs := []*Request{
		NewRequest(config2, "METHOD", "path"),
		NewRequest(config2, "METHOD", "/path"),
		NewRequest(config3, "METHOD", "path"),
		NewRequest(config3, "METHOD", "/path"),
		NewRequest(config3, "METHOD", "{arg}", "/path"),
		NewRequest(config3, "METHOD", "{arg}").WithPath("arg", "/path"),
	}

	for _, req := range reqs {
		req.Expect().chain.assertOK(t)
		assert.Equal(t, "http://example.com/path", client.req.URL.String())
	}

	empty1 := NewRequest(config1, "METHOD", "")
	empty2 := NewRequest(config2, "METHOD", "")
	empty3 := NewRequest(config3, "METHOD", "")

	empty1.Expect().chain.assertOK(t)
	empty2.Expect().chain.assertOK(t)
	empty3.Expect().chain.assertOK(t)

	assert.Equal(t, "", empty1.http.URL.String())
	assert.Equal(t, "http://example.com", empty2.http.URL.String())
	assert.Equal(t, "http://example.com/", empty3.http.URL.String())
}

func TestRequestURLOverwrite(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := NewAssertReporter(t)

	config1 := Config{
		RequestFactory: factory,
		BaseURL:        "",
		Client:         client,
		Reporter:       reporter,
	}

	config2 := Config{
		RequestFactory: factory,
		BaseURL:        "http://foobar.com",
		Client:         client,
		Reporter:       reporter,
	}

	reqs := []*Request{
		NewRequest(config1, "METHOD", "/path").WithURL("http://example.com"),
		NewRequest(config1, "METHOD", "path").WithURL("http://example.com"),
		NewRequest(config1, "METHOD", "/path").WithURL("http://example.com/"),
		NewRequest(config1, "METHOD", "path").WithURL("http://example.com/"),
		NewRequest(config2, "METHOD", "/path").WithURL("http://example.com"),
		NewRequest(config2, "METHOD", "path").WithURL("http://example.com"),
		NewRequest(config2, "METHOD", "/path").WithURL("http://example.com/"),
		NewRequest(config2, "METHOD", "path").WithURL("http://example.com/"),
	}

	for _, req := range reqs {
		req.Expect().chain.assertOK(t)
		assert.Equal(t, "http://example.com/path", client.req.URL.String())
	}
}

func TestRequestURLInterpolate(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	var reqs [3]*Request

	config := Config{
		RequestFactory: factory,
		BaseURL:        "http://example.com/",
		Client:         client,
		Reporter:       reporter,
	}

	reqs[0] = NewRequest(config, "METHOD", "/foo/{arg}", "bar")
	reqs[1] = NewRequest(config, "METHOD", "{arg}foo{arg}", "/", "/bar")
	reqs[2] = NewRequest(config, "METHOD", "{arg}", "/foo/bar")

	for _, req := range reqs {
		req.Expect().chain.assertOK(t)
		assert.Equal(t, "http://example.com/foo/bar", client.req.URL.String())
	}

	r1 := NewRequest(config, "METHOD", "/{arg1}/{arg2}", "foo")
	r1.Expect().chain.assertOK(t)
	assert.Equal(t, "http://example.com/foo/%7Barg2%7D",
		client.req.URL.String())

	r2 := NewRequest(config, "METHOD", "/{arg1}/{arg2}/{arg3}")
	r2.WithPath("ARG3", "foo")
	r2.WithPath("arg2", "bar")
	r2.Expect().chain.assertOK(t)
	assert.Equal(t, "http://example.com/%7Barg1%7D/bar/foo",
		client.req.URL.String())

	r3 := NewRequest(config, "METHOD", "/{arg1}.{arg2}.{arg3}")
	r3.WithPath("arg2", "bar")
	r3.WithPathObject(map[string]string{"ARG1": "foo", "arg3": "baz"})
	r3.WithPathObject(nil)
	r3.Expect().chain.assertOK(t)
	assert.Equal(t, "http://example.com/foo.bar.baz",
		client.req.URL.String())

	type S struct {
		Arg1 string
		A2   int `path:"arg2"`
		A3   int `path:"-"`
	}

	r4 := NewRequest(config, "METHOD", "/{arg1}{arg2}")
	r4.WithPathObject(S{"foo", 1, 2})
	r4.Expect().chain.assertOK(t)
	assert.Equal(t, "http://example.com/foo1", client.req.URL.String())

	r5 := NewRequest(config, "METHOD", "/{arg1}{arg2}")
	r5.WithPathObject(&S{"foo", 1, 2})
	r5.Expect().chain.assertOK(t)
	assert.Equal(t, "http://example.com/foo1", client.req.URL.String())

	r6 := NewRequest(config, "GET", "{arg}", nil)
	r6.chain.assertFailed(t)

	r7 := NewRequest(config, "GET", "{arg}")
	r7.chain.assertOK(t)
	r7.WithPath("arg", nil)
	r7.chain.assertFailed(t)

	r8 := NewRequest(config, "GET", "{arg}")
	r8.chain.assertOK(t)
	r8.WithPath("bad", "value")
	r8.chain.assertFailed(t)

	r9 := NewRequest(config, "GET", "{arg")
	r9.chain.assertFailed(t)
	r9.WithPath("arg", "foo")
	r9.chain.assertFailed(t)

	r10 := NewRequest(config, "GET", "{arg}")
	r10.chain.assertOK(t)
	r10.WithPathObject(func() {})
	r10.chain.assertFailed(t)
}

func TestRequestURLQuery(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req1 := NewRequest(config, "METHOD", "http://example.com/path").
		WithQuery("aa", "foo").WithQuery("bb", 123).WithQuery("cc", "*&@")

	q := map[string]interface{}{
		"bb": 123,
		"cc": "*&@",
	}

	req2 := NewRequest(config, "METHOD", "http://example.com/path").
		WithQuery("aa", "foo").
		WithQueryObject(q)

	type S struct {
		Bb int    `url:"bb"`
		Cc string `url:"cc"`
		Dd string `url:"-"`
	}

	req3 := NewRequest(config, "METHOD", "http://example.com/path").
		WithQueryObject(S{123, "*&@", "dummy"}).WithQuery("aa", "foo")

	req4 := NewRequest(config, "METHOD", "http://example.com/path").
		WithQueryObject(&S{123, "*&@", "dummy"}).WithQuery("aa", "foo")

	req5 := NewRequest(config, "METHOD", "http://example.com/path").
		WithQuery("bb", 123).
		WithQueryString("aa=foo&cc=%2A%26%40")

	req6 := NewRequest(config, "METHOD", "http://example.com/path").
		WithQueryString("aa=foo&cc=%2A%26%40").
		WithQuery("bb", 123)

	for _, req := range []*Request{req1, req2, req3, req4, req5, req6} {
		client.req = nil
		req.Expect()
		req.chain.assertOK(t)
		assert.Equal(t, "http://example.com/path?aa=foo&bb=123&cc=%2A%26%40",
			client.req.URL.String())
	}

	req7 := NewRequest(config, "METHOD", "http://example.com/path").
		WithQuery("foo", "bar").
		WithQueryObject(nil)

	req7.Expect()
	req7.chain.assertOK(t)
	assert.Equal(t, "http://example.com/path?foo=bar", client.req.URL.String())

	NewRequest(config, "METHOD", "http://example.com/path").
		WithQueryObject(func() {}).chain.assertFailed(t)

	NewRequest(config, "METHOD", "http://example.com/path").
		WithQueryString("%").chain.assertFailed(t)
}

func TestRequestHeaders(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithHeader("first-header", "foo")

	req.WithHeaders(map[string]string{
		"Second-Header": "bar",
		"content-Type":  "baz",
		"HOST":          "example.com",
	})

	expectedHeaders := map[string][]string{
		"First-Header":  {"foo"},
		"Second-Header": {"bar"},
		"Content-Type":  {"baz"},
	}

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "example.com", client.req.Host)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, http.Header(expectedHeaders), client.req.Header)

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestCookies(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithCookie("foo", "1")
	req.WithCookie("bar", "2 ")

	req.WithCookies(map[string]string{
		"baz": " 3",
	})

	expectedHeaders := map[string][]string{
		"Cookie": {`foo=1; bar="2 "; baz=" 3"`},
	}

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, http.Header(expectedHeaders), client.req.Header)

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestBasicAuth(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithBasicAuth("Aladdin", "open sesame")
	req.chain.assertOK(t)

	assert.Equal(t, "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==",
		req.http.Header.Get("Authorization"))
}

func TestRequestBodyChunked(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithChunked(bytes.NewBufferString("body"))

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.False(t, client.req.Body == nil)
	assert.Equal(t, int64(-1), client.req.ContentLength)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, make(http.Header), client.req.Header)
	assert.Equal(t, "body", string(resp.content))

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestBodyChunkedNil(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithChunked(nil)

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.True(t, client.req.Body == nil)
	assert.Equal(t, int64(0), client.req.ContentLength)
}

func TestRequestBodyChunkedProto(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req1 := NewRequest(config, "METHOD", "url")

	req1.WithProto("HTTP/1.0")
	assert.Equal(t, 1, req1.http.ProtoMajor)
	assert.Equal(t, 0, req1.http.ProtoMinor)

	req1.WithChunked(bytes.NewBufferString("body"))
	req1.chain.assertFailed(t)

	req2 := NewRequest(config, "METHOD", "url")

	req2.WithProto("HTTP/2.0")
	assert.Equal(t, 2, req2.http.ProtoMajor)
	assert.Equal(t, 0, req2.http.ProtoMinor)

	req2.WithChunked(bytes.NewBufferString("body"))
	assert.Equal(t, 2, req2.http.ProtoMajor)
	assert.Equal(t, 0, req2.http.ProtoMinor)
}

func TestRequestBodyBytes(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithBytes([]byte("body"))

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.False(t, client.req.Body == nil)
	assert.Equal(t, int64(len("body")), client.req.ContentLength)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, make(http.Header), client.req.Header)
	assert.Equal(t, "body", string(resp.content))

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestBodyBytesNil(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithBytes(nil)

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.True(t, client.req.Body == nil)
	assert.Equal(t, int64(0), client.req.ContentLength)
}

func TestRequestBodyText(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	expectedHeaders := map[string][]string{
		"Content-Type": {"text/plain; charset=utf-8"},
		"Some-Header":  {"foo"},
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithHeaders(map[string]string{
		"Some-Header": "foo",
	})

	req.WithText("some text")

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, http.Header(expectedHeaders), client.req.Header)
	assert.Equal(t, "some text", string(resp.content))

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestBodyForm(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	expectedHeaders := map[string][]string{
		"Content-Type": {"application/x-www-form-urlencoded"},
		"Some-Header":  {"foo"},
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithHeaders(map[string]string{
		"Some-Header": "foo",
	})

	req.WithForm(map[string]interface{}{
		"a": 1,
		"b": "2",
	})

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, http.Header(expectedHeaders), client.req.Header)
	assert.Equal(t, `a=1&b=2`, string(resp.content))

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestBodyField(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	expectedHeaders := map[string][]string{
		"Content-Type": {"application/x-www-form-urlencoded"},
		"Some-Header":  {"foo"},
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithHeaders(map[string]string{
		"Some-Header": "foo",
	})

	req.WithFormField("a", 1)
	req.WithFormField("b", "2")

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, http.Header(expectedHeaders), client.req.Header)
	assert.Equal(t, `a=1&b=2`, string(resp.content))

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestBodyFormStruct(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	expectedHeaders := map[string][]string{
		"Content-Type": {"application/x-www-form-urlencoded"},
	}

	req := NewRequest(config, "METHOD", "url")

	type S struct {
		A string `form:"a"`
		B int    `form:"b"`
		C int    `form:"-"`
	}

	req.WithForm(S{"1", 2, 3})

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, http.Header(expectedHeaders), client.req.Header)
	assert.Equal(t, `a=1&b=2`, string(resp.content))

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestBodyFormCombined(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	expectedHeaders := map[string][]string{
		"Content-Type": {"application/x-www-form-urlencoded"},
	}

	req := NewRequest(config, "METHOD", "url")

	type S struct {
		A int `form:"a"`
	}

	req.WithForm(S{A: 1})
	req.WithForm(map[string]string{"b": "2"})
	req.WithFormField("c", 3)

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, http.Header(expectedHeaders), client.req.Header)
	assert.Equal(t, `a=1&b=2&c=3`, string(resp.content))

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestBodyMultipart(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "POST", "url")

	req.WithMultipart()
	req.WithForm(map[string]string{"b": "1", "c": "2"})
	req.WithFormField("a", 3)

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "POST", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())

	mediatype, params, err := mime.ParseMediaType(client.req.Header.Get("Content-Type"))

	assert.True(t, err == nil)
	assert.Equal(t, "multipart/form-data", mediatype)
	assert.True(t, params["boundary"] != "")

	reader := multipart.NewReader(bytes.NewReader(resp.content), params["boundary"])

	part1, _ := reader.NextPart()
	assert.Equal(t, "b", part1.FormName())
	assert.Equal(t, "", part1.FileName())
	b1, _ := ioutil.ReadAll(part1)
	assert.Equal(t, "1", string(b1))

	part2, _ := reader.NextPart()
	assert.Equal(t, "c", part2.FormName())
	assert.Equal(t, "", part2.FileName())
	b2, _ := ioutil.ReadAll(part2)
	assert.Equal(t, "2", string(b2))

	part3, _ := reader.NextPart()
	assert.Equal(t, "a", part3.FormName())
	assert.Equal(t, "", part3.FileName())
	b3, _ := ioutil.ReadAll(part3)
	assert.Equal(t, "3", string(b3))

	eof, _ := reader.NextPart()
	assert.True(t, eof == nil)
}

func TestRequestBodyMultipartFile(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "POST", "url")

	fh, _ := ioutil.TempFile("", "httpexpect")
	filename2 := fh.Name()
	fh.WriteString("2")
	fh.Close()
	defer os.Remove(filename2)

	req.WithMultipart()
	req.WithForm(map[string]string{"a": "1"})
	req.WithFile("b", filename2)
	req.WithFile("c", "filename3", strings.NewReader("3"))
	req.WithFileBytes("d", "filename4", []byte("4"))

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "POST", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())

	mediatype, params, err := mime.ParseMediaType(client.req.Header.Get("Content-Type"))

	assert.True(t, err == nil)
	assert.Equal(t, "multipart/form-data", mediatype)
	assert.True(t, params["boundary"] != "")

	reader := multipart.NewReader(bytes.NewReader(resp.content), params["boundary"])

	part1, _ := reader.NextPart()
	assert.Equal(t, "a", part1.FormName())
	assert.Equal(t, "", part1.FileName())
	b1, _ := ioutil.ReadAll(part1)
	assert.Equal(t, "1", string(b1))

	part2, _ := reader.NextPart()
	assert.Equal(t, "b", part2.FormName())
	assert.Equal(t, filename2, part2.FileName())
	b2, _ := ioutil.ReadAll(part2)
	assert.Equal(t, "2", string(b2))

	part3, _ := reader.NextPart()
	assert.Equal(t, "c", part3.FormName())
	assert.Equal(t, "filename3", part3.FileName())
	b3, _ := ioutil.ReadAll(part3)
	assert.Equal(t, "3", string(b3))

	part4, _ := reader.NextPart()
	assert.Equal(t, "d", part4.FormName())
	assert.Equal(t, "filename4", part4.FileName())
	b4, _ := ioutil.ReadAll(part4)
	assert.Equal(t, "4", string(b4))

	eof, _ := reader.NextPart()
	assert.True(t, eof == nil)
}

func TestRequestBodyJSON(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	expectedHeaders := map[string][]string{
		"Content-Type": {"application/json; charset=utf-8"},
		"Some-Header":  {"foo"},
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithHeaders(map[string]string{
		"Some-Header": "foo",
	})

	req.WithJSON(map[string]interface{}{"key": "value"})

	resp := req.Expect()
	resp.chain.assertOK(t)

	assert.Equal(t, "METHOD", client.req.Method)
	assert.Equal(t, "url", client.req.URL.String())
	assert.Equal(t, http.Header(expectedHeaders), client.req.Header)
	assert.Equal(t, `{"key":"value"}`, string(resp.content))

	assert.Equal(t, &client.resp, resp.Raw())
}

func TestRequestContentLength(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req1 := NewRequest(config, "METHOD", "url")
	req1.WithChunked(bytes.NewReader([]byte("12345")))
	req1.Expect().chain.assertOK(t)
	assert.Equal(t, int64(-1), client.req.ContentLength)

	req2 := NewRequest(config, "METHOD", "url")
	req2.WithBytes([]byte("12345"))
	req2.Expect().chain.assertOK(t)
	assert.Equal(t, int64(5), client.req.ContentLength)

	req3 := NewRequest(config, "METHOD", "url")
	req3.WithText("12345")
	req3.Expect().chain.assertOK(t)
	assert.Equal(t, int64(5), client.req.ContentLength)

	j, _ := json.Marshal(map[string]string{"a": "b"})
	req4 := NewRequest(config, "METHOD", "url")
	req4.WithJSON(map[string]string{"a": "b"})
	req4.Expect().chain.assertOK(t)
	assert.Equal(t, int64(len(j)), client.req.ContentLength)

	f := `a=b`
	req5 := NewRequest(config, "METHOD", "url")
	req5.WithForm(map[string]string{"a": "b"})
	req5.Expect().chain.assertOK(t)
	assert.Equal(t, int64(len(f)), client.req.ContentLength)

	req6 := NewRequest(config, "METHOD", "url")
	req6.WithFormField("a", "b")
	req6.Expect().chain.assertOK(t)
	assert.Equal(t, int64(len(f)), client.req.ContentLength)

	req7 := NewRequest(config, "METHOD", "url")
	req7.WithMultipart()
	req7.WithFileBytes("a", "b", []byte("12345"))
	req7.Expect().chain.assertOK(t)
	assert.True(t, client.req.ContentLength > 0)
}

func TestRequestContentTypeOverwrite(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req1 := NewRequest(config, "METHOD", "url")
	req1.WithText("hello")
	req1.WithHeader("Content-Type", "foo")
	req1.Expect().chain.assertOK(t)
	assert.Equal(t, http.Header{"Content-Type": {"foo"}}, client.req.Header)

	req2 := NewRequest(config, "METHOD", "url")
	req2.WithHeader("Content-Type", "foo")
	req2.WithText("hello")
	req2.Expect().chain.assertOK(t)
	assert.Equal(t, http.Header{"Content-Type": {"foo"}}, client.req.Header)

	req3 := NewRequest(config, "METHOD", "url")
	req3.WithJSON(map[string]interface{}{"a": "b"})
	req3.WithHeader("Content-Type", "foo")
	req3.WithHeader("Content-Type", "bar")
	req3.Expect().chain.assertOK(t)
	assert.Equal(t, http.Header{"Content-Type": {"foo", "bar"}}, client.req.Header)

	req4 := NewRequest(config, "METHOD", "url")
	req4.WithForm(map[string]interface{}{"a": "b"})
	req4.WithHeader("Content-Type", "foo")
	req4.WithHeader("Content-Type", "bar")
	req4.Expect().chain.assertOK(t)
	assert.Equal(t, http.Header{"Content-Type": {"foo", "bar"}}, client.req.Header)

	req5 := NewRequest(config, "METHOD", "url")
	req5.WithMultipart()
	req5.WithForm(map[string]interface{}{"a": "b"})
	req5.WithHeader("Content-Type", "foo")
	req5.WithHeader("Content-Type", "bar")
	req5.Expect().chain.assertOK(t)
	assert.Equal(t, http.Header{"Content-Type": {"foo", "bar"}}, client.req.Header)
}

func TestRequestErrorMarshalForm(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithForm(func() {})

	resp := req.Expect()
	resp.chain.assertFailed(t)

	assert.True(t, resp.Raw() == nil)
}

func TestRequestErrorMarshalJSON(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithJSON(func() {})

	resp := req.Expect()
	resp.chain.assertFailed(t)

	assert.True(t, resp.Raw() == nil)
}

func TestRequestErrorReadFile(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{
		err: errors.New("error"),
	}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	req.WithMultipart()
	req.WithFile("", "")

	resp := req.Expect()
	resp.chain.assertFailed(t)

	assert.True(t, resp.Raw() == nil)
}

func TestRequestErrorSend(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{
		err: errors.New("error"),
	}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req := NewRequest(config, "METHOD", "url")

	resp := req.Expect()
	resp.chain.assertFailed(t)

	assert.True(t, resp.Raw() == nil)
}

func TestRequestErrorConflictBody(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{
		err: errors.New("error"),
	}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req1 := NewRequest(config, "METHOD", "url")
	req1.WithChunked(nil)
	req1.chain.assertOK(t)
	req1.WithChunked(nil)
	req1.chain.assertFailed(t)

	req2 := NewRequest(config, "METHOD", "url")
	req2.WithChunked(nil)
	req2.chain.assertOK(t)
	req2.WithBytes(nil)
	req2.chain.assertFailed(t)

	req3 := NewRequest(config, "METHOD", "url")
	req3.WithChunked(nil)
	req3.chain.assertOK(t)
	req3.WithText("")
	req3.chain.assertFailed(t)

	req4 := NewRequest(config, "METHOD", "url")
	req4.WithChunked(nil)
	req4.chain.assertOK(t)
	req4.WithJSON(map[string]interface{}{"a": "b"})
	req4.chain.assertFailed(t)

	req5 := NewRequest(config, "METHOD", "url")
	req5.WithChunked(nil)
	req5.chain.assertOK(t)
	req5.WithForm(map[string]interface{}{"a": "b"})
	req5.Expect()
	req5.chain.assertFailed(t)

	req6 := NewRequest(config, "METHOD", "url")
	req6.WithChunked(nil)
	req6.chain.assertOK(t)
	req6.WithFormField("a", "b")
	req6.Expect()
	req6.chain.assertFailed(t)

	req7 := NewRequest(config, "METHOD", "url")
	req7.WithChunked(nil)
	req7.chain.assertOK(t)
	req7.WithMultipart()
	req7.chain.assertFailed(t)
}

func TestRequestErrorConflictType(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{
		err: errors.New("error"),
	}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req1 := NewRequest(config, "METHOD", "url")
	req1.WithText("")
	req1.chain.assertOK(t)
	req1.WithJSON(map[string]interface{}{"a": "b"})
	req1.chain.assertFailed(t)

	req2 := NewRequest(config, "METHOD", "url")
	req2.WithText("")
	req2.chain.assertOK(t)
	req2.WithForm(map[string]interface{}{"a": "b"})
	req2.chain.assertFailed(t)

	req3 := NewRequest(config, "METHOD", "url")
	req3.WithText("")
	req3.chain.assertOK(t)
	req3.WithFormField("a", "b")
	req3.chain.assertFailed(t)

	req4 := NewRequest(config, "METHOD", "url")
	req4.WithText("")
	req4.chain.assertOK(t)
	req4.WithMultipart()
	req4.chain.assertFailed(t)
}

func TestRequestErrorConflictMultipart(t *testing.T) {
	factory := DefaultRequestFactory{}

	client := &mockClient{
		err: errors.New("error"),
	}

	reporter := newMockReporter(t)

	config := Config{
		RequestFactory: factory,
		Client:         client,
		Reporter:       reporter,
	}

	req1 := NewRequest(config, "METHOD", "url")
	req1.WithForm(map[string]interface{}{"a": "b"})
	req1.chain.assertOK(t)
	req1.WithMultipart()
	req1.chain.assertFailed(t)

	req2 := NewRequest(config, "METHOD", "url")
	req2.WithFormField("a", "b")
	req2.chain.assertOK(t)
	req2.WithMultipart()
	req2.chain.assertFailed(t)

	req3 := NewRequest(config, "METHOD", "url")
	req3.WithFileBytes("a", "a", []byte("a"))
	req3.chain.assertFailed(t)
}
