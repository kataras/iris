package httpexpect

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

func TestExpectMethods(t *testing.T) {
	client := &mockClient{}

	reporter := NewAssertReporter(t)

	config := Config{
		BaseURL:  "http://example.com",
		Client:   client,
		Reporter: reporter,
	}

	var reqs [8]*Request

	e := WithConfig(config)

	reqs[0] = e.Request("METHOD", "/url")
	reqs[1] = e.OPTIONS("/url")
	reqs[2] = e.HEAD("/url")
	reqs[3] = e.GET("/url")
	reqs[4] = e.POST("/url")
	reqs[5] = e.PUT("/url")
	reqs[6] = e.PATCH("/url")
	reqs[7] = e.DELETE("/url")

	assert.Equal(t, "METHOD", reqs[0].http.Method)
	assert.Equal(t, "OPTIONS", reqs[1].http.Method)
	assert.Equal(t, "HEAD", reqs[2].http.Method)
	assert.Equal(t, "GET", reqs[3].http.Method)
	assert.Equal(t, "POST", reqs[4].http.Method)
	assert.Equal(t, "PUT", reqs[5].http.Method)
	assert.Equal(t, "PATCH", reqs[6].http.Method)
	assert.Equal(t, "DELETE", reqs[7].http.Method)
}

func TestExpectBuilders(t *testing.T) {
	client := &mockClient{}

	reporter := NewAssertReporter(t)

	config := Config{
		Client:   client,
		Reporter: reporter,
	}

	e := WithConfig(config)

	var reqs1 []*Request

	e1 := e.Builder(func(r *Request) {
		reqs1 = append(reqs1, r)
	})

	var reqs2 []*Request

	e2 := e1.Builder(func(r *Request) {
		reqs2 = append(reqs2, r)
	})

	e.Request("METHOD", "/url")

	r1 := e1.Request("METHOD", "/url")
	r2 := e2.Request("METHOD", "/url")

	assert.Equal(t, 2, int(len(reqs1)))
	assert.Equal(t, 1, int(len(reqs2)))

	assert.Equal(t, r1, reqs1[0])
	assert.Equal(t, r2, reqs1[1])
	assert.Equal(t, r1, reqs2[0])
}

func TestExpectValues(t *testing.T) {
	client := &mockClient{}

	r := NewAssertReporter(t)

	config := Config{
		Client:   client,
		Reporter: r,
	}

	e := WithConfig(config)

	m := map[string]interface{}{}
	a := []interface{}{}
	s := ""
	n := 0.0
	b := false

	assert.Equal(t, NewValue(r, m), e.Value(m))
	assert.Equal(t, NewObject(r, m), e.Object(m))
	assert.Equal(t, NewArray(r, a), e.Array(a))
	assert.Equal(t, NewString(r, s), e.String(s))
	assert.Equal(t, NewNumber(r, n), e.Number(n))
	assert.Equal(t, NewBoolean(r, b), e.Boolean(b))
}

func TestExpectTraverse(t *testing.T) {
	client := &mockClient{}

	reporter := NewAssertReporter(t)

	config := Config{
		BaseURL:  "http://example.com",
		Client:   client,
		Reporter: reporter,
	}

	data := map[string]interface{}{
		"foo": []interface{}{"bar", 123, false, nil},
		"bar": "hello",
		"baz": 456,
	}

	resp := WithConfig(config).GET("/url").WithJSON(data).Expect()

	m := resp.JSON().Object()

	m.Equal(data)

	m.ContainsKey("foo")
	m.ContainsKey("bar")
	m.ContainsKey("foo")

	m.ValueEqual("foo", data["foo"])
	m.ValueEqual("bar", data["bar"])
	m.ValueEqual("baz", data["baz"])

	m.Keys().ContainsOnly("foo", "bar", "baz")
	m.Values().ContainsOnly(data["foo"], data["bar"], data["baz"])

	m.Value("foo").Array().Elements("bar", 123, false, nil)
	m.Value("bar").String().Equal("hello")
	m.Value("baz").Number().Equal(456)

	m.Value("foo").Array().Element(2).Boolean().False()
	m.Value("foo").Array().Element(3).Null()
}

func TestExpectBranches(t *testing.T) {
	client := &mockClient{}

	config := Config{
		BaseURL:  "http://example.com",
		Client:   client,
		Reporter: newMockReporter(t),
	}

	data := map[string]interface{}{
		"foo": []interface{}{"bar", 123, false, nil},
		"bar": "hello",
		"baz": 456,
	}

	resp := WithConfig(config).GET("/url").WithJSON(data).Expect()

	m1 := resp.JSON().Array()
	m2 := resp.JSON().Object()

	e1 := m2.Value("foo").Object()
	e2 := m2.Value("foo").Array().Element(999).String()
	e3 := m2.Value("foo").Array().Element(0).Number()
	e4 := m2.Value("foo").Array().Element(0).String()

	e4.Equal("bar")

	m1.chain.assertFailed(t)
	m2.chain.assertOK(t)

	e1.chain.assertFailed(t)
	e2.chain.assertFailed(t)
	e3.chain.assertFailed(t)
	e4.chain.assertOK(t)
}

func TestExpectStdCompat(_ *testing.T) {
	New(&testing.T{}, "")
	New(&testing.B{}, "")
	New(testing.TB(&testing.T{}), "")
}

type testRequestFactory struct {
	lastreq *http.Request
	fail    bool
}

func (f *testRequestFactory) NewRequest(method, urlStr string, body io.Reader) (*http.Request, error) {
	if f.fail {
		return nil, errors.New("testRequestFactory")
	}
	f.lastreq = httptest.NewRequest(method, urlStr, body)
	return f.lastreq, nil
}

func TestExpectRequestFactory(t *testing.T) {
	e1 := WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
	})
	r1 := e1.Request("GET", "/")
	r1.chain.assertOK(t)
	assert.NotNil(t, r1.http)

	f2 := &testRequestFactory{}
	e2 := WithConfig(Config{
		BaseURL:        "http://example.com",
		Reporter:       NewAssertReporter(t),
		RequestFactory: f2,
	})
	r2 := e2.Request("GET", "/")
	r2.chain.assertOK(t)
	assert.NotNil(t, f2.lastreq)
	assert.True(t, f2.lastreq == r2.http)

	f3 := &testRequestFactory{
		fail: true,
	}
	e3 := WithConfig(Config{
		BaseURL:        "http://example.com",
		Reporter:       newMockReporter(t),
		RequestFactory: f3,
	})
	r3 := e3.Request("GET", "/")
	r3.chain.assertFailed(t)
	assert.Nil(t, f3.lastreq)
}

func createBasicHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"foo":123}`))
	})

	mux.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		w.Write([]byte(`field1=` + r.FormValue("field1")))
		w.Write([]byte(`&field2=` + r.PostFormValue("field2")))
	})

	mux.HandleFunc("/baz/qux", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[true, false]`))

		case "PUT":
			decoder := json.NewDecoder(r.Body)
			var m map[string]interface{}
			if err := decoder.Decode(&m); err != nil {
				w.WriteHeader(http.StatusBadRequest)
			} else if m["test"] != "ok" {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`ok`))
			}
		}
	})

	mux.HandleFunc("/wee", func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); ok {
			w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
			w.Write([]byte(`username=` + u))
			w.Write([]byte(`&password=` + p))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	return mux
}

func testBasicHandler(e *Expect) {
	e.GET("/foo")
	e.GET("/foo").Expect()
	e.GET("/foo").Expect().Status(http.StatusOK)

	e.GET("/foo").Expect().
		Status(http.StatusOK).JSON().Object().ValueEqual("foo", 123)

	e.PUT("/bar").WithQuery("field1", "hello").WithFormField("field2", "world").
		Expect().
		Status(http.StatusOK).
		Form().ValueEqual("field1", "hello").ValueEqual("field2", "world")

	e.GET("/{a}/{b}", "baz", "qux").
		Expect().
		Status(http.StatusOK).JSON().Array().Elements(true, false)

	e.PUT("/{a}/{b}").
		WithPath("a", "baz").
		WithPath("b", "qux").
		WithJSON(map[string]string{"test": "ok"}).
		Expect().
		Status(http.StatusOK).Body().Equal("ok")

	auth := e.Builder(func(req *Request) {
		req.WithBasicAuth("john", "secret")
	})

	auth.PUT("/wee").
		Expect().
		Status(http.StatusOK).
		Form().ValueEqual("username", "john").ValueEqual("password", "secret")
}

func TestExpectBasicHandlerLiveDefault(t *testing.T) {
	handler := createBasicHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	testBasicHandler(New(t, server.URL))
}

func TestExpectBasicHandlerLiveConfig(t *testing.T) {
	handler := createBasicHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	testBasicHandler(WithConfig(Config{
		BaseURL:  server.URL,
		Reporter: NewAssertReporter(t),
		Printers: []Printer{
			NewCurlPrinter(t),
			NewDebugPrinter(t, true),
		},
	}))
}

func TestExpectBasicHandlerLiveTLS(t *testing.T) {
	handler := createBasicHandler()

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	testBasicHandler(WithConfig(Config{
		BaseURL:  server.URL,
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}))
}

func TestExpectBasicHandlerLiveLongRun(t *testing.T) {
	if testing.Short() {
		return
	}

	handler := createBasicHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	e := New(t, server.URL)

	for i := 0; i < 2; i++ {
		testBasicHandler(e)
	}
}

func TestExpectBasicHandlerBinderStandard(t *testing.T) {
	handler := createBasicHandler()

	testBasicHandler(WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewBinder(handler),
		},
	}))
}

func TestExpectBasicHandlerBinderFast(t *testing.T) {
	handler := fasthttpadaptor.NewFastHTTPHandler(createBasicHandler())

	testBasicHandler(WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewFastBinder(handler),
		},
	}))
}

func createRedirectHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`hello`))
	})

	mux.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/foo", http.StatusFound)
	})

	return mux
}

func createRedirectFastHandler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/foo":
			ctx.SetBody([]byte(`hello`))

		case "/bar":
			ctx.Redirect("/foo", http.StatusFound)
		}
	}
}

func testRedirectHandler(e *Expect) {
	e.POST("/bar").
		Expect().
		Status(http.StatusOK).Body().Equal(`hello`)
}

func TestExpectRedirectHandlerLive(t *testing.T) {
	handler := createRedirectHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	testRedirectHandler(New(t, server.URL))
}

func TestExpectRedirectHandlerBinderStandard(t *testing.T) {
	handler := createRedirectHandler()

	testRedirectHandler(WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewBinder(handler),
		},
	}))
}

func TestExpectRedirectHandlerBinderFast(t *testing.T) {
	handler := createRedirectFastHandler()

	testRedirectHandler(WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewFastBinder(handler),
		},
	}))
}

func createAutoTLSHandler(https string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/tls", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			w.Write([]byte(`no`))
		} else {
			w.Write([]byte(`yes`))
		}
	})

	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			http.Redirect(w, r, https+r.RequestURI, http.StatusFound)
		} else {
			w.Write([]byte(`hello`))
		}
	})

	return mux
}

func createAutoTLSFastHandler(https string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/tls":
			if !ctx.IsTLS() {
				ctx.SetBody([]byte(`no`))
			} else {
				ctx.SetBody([]byte(`yes`))
			}

		case "/protected":
			if !ctx.IsTLS() {
				ctx.Redirect(https+string(ctx.Request.RequestURI()), http.StatusFound)
			} else {
				ctx.SetBody([]byte(`hello`))
			}
		}
	}
}

func testAutoTLSHandler(config Config) {
	e := WithConfig(config)

	tls := e.POST("/tls").
		Expect().
		Status(http.StatusOK).Body()

	if strings.HasPrefix(config.BaseURL, "https://") {
		tls.Equal(`yes`)
	} else {
		tls.Equal(`no`)
	}

	e.POST("/protected").
		Expect().
		Status(http.StatusOK).Body().Equal(`hello`)
}

func TestExpectAutoTLSHandlerLive(t *testing.T) {
	httpsServ := httptest.NewTLSServer(createAutoTLSHandler(""))
	defer httpsServ.Close()

	httpServ := httptest.NewServer(createAutoTLSHandler(httpsServ.URL))
	defer httpServ.Close()

	assert.True(t, strings.HasPrefix(httpsServ.URL, "https://"))
	assert.True(t, strings.HasPrefix(httpServ.URL, "http://"))

	for _, url := range []string{httpsServ.URL, httpServ.URL} {
		testAutoTLSHandler(Config{
			BaseURL:  url,
			Reporter: NewRequireReporter(t),
			Printers: []Printer{
				NewDebugPrinter(t, true),
			},
			Client: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			},
		})
	}
}

func TestExpectAutoTLSHandlerBinderStandard(t *testing.T) {
	handler := createAutoTLSHandler("https://example.com")

	for _, url := range []string{"https://example.com", "http://example.com"} {
		testAutoTLSHandler(Config{
			BaseURL:  url,
			Reporter: NewRequireReporter(t),
			Printers: []Printer{
				NewDebugPrinter(t, true),
			},
			Client: &http.Client{
				Transport: &Binder{
					Handler: handler,
					TLS:     &tls.ConnectionState{},
				},
			},
		})
	}
}

func TestExpectAutoTLSHandlerBinderFast(t *testing.T) {
	handler := createAutoTLSFastHandler("https://example.com")

	for _, url := range []string{"https://example.com", "http://example.com"} {
		testAutoTLSHandler(Config{
			BaseURL:  url,
			Reporter: NewRequireReporter(t),
			Printers: []Printer{
				NewDebugPrinter(t, true),
			},
			Client: &http.Client{
				Transport: &FastBinder{
					Handler: handler,
					TLS:     &tls.ConnectionState{},
				},
			},
		})
	}
}

func createChunkedHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Proto != "HTTP/1.1" {
			w.WriteHeader(http.StatusBadRequest)
		} else if len(r.TransferEncoding) != 1 || r.TransferEncoding[0] != "chunked" {
			w.WriteHeader(http.StatusBadRequest)
		} else if r.PostFormValue("key") != "value" {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[1, `))
			w.(http.Flusher).Flush()
			w.Write([]byte(`2]`))
		}
	})

	return mux
}

func createChunkedFastHandler(t *testing.T) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		headers := map[string][]string{}

		ctx.Request.Header.VisitAll(func(k, v []byte) {
			headers[string(k)] = append(headers[string(k)], string(v))
		})

		assert.Equal(t, []string{"chunked"}, headers["Transfer-Encoding"])
		assert.Equal(t, "value", string(ctx.FormValue("key")))
		assert.Equal(t, "key=value", string(ctx.Request.Body()))

		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
			w.WriteString(`[1, `)
			w.Flush()
			w.WriteString(`2]`)
		})
	}
}

func testChunkedHandler(e *Expect) {
	e.PUT("/").
		WithHeader("Content-Type", "application/x-www-form-urlencoded").
		WithChunked(strings.NewReader("key=value")).
		Expect().
		Status(http.StatusOK).
		ContentType("application/json").
		TransferEncoding("chunked").
		JSON().Array().Elements(1, 2)
}

func TestExpectChunkedHandlerLive(t *testing.T) {
	handler := createChunkedHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	testChunkedHandler(New(t, server.URL))
}

func TestExpectChunkedHandlerBinderStandard(t *testing.T) {
	handler := createChunkedHandler()

	testChunkedHandler(WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewBinder(handler),
		},
	}))
}

func TestExpectChunkedHandlerBinderFast(t *testing.T) {
	handler := createChunkedFastHandler(t)

	testChunkedHandler(WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewFastBinder(handler),
		},
	}))
}

func createCookieHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:    "myname",
			Value:   "myvalue",
			Path:    "/",
			Expires: time.Date(3000, 0, 0, 0, 0, 0, 0, time.UTC),
		})
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("myname")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(cookie.Value))
		}
	})

	return mux
}

func testCookieHandler(e *Expect, enabled bool) {
	r := e.PUT("/set").Expect().Status(http.StatusNoContent)

	r.Cookies().ContainsOnly("myname")
	c := r.Cookie("myname")
	c.Value().Equal("myvalue")
	c.Path().Equal("/")
	c.Expires().Equal(time.Date(3000, 0, 0, 0, 0, 0, 0, time.UTC))

	if enabled {
		e.GET("/get").Expect().Status(http.StatusOK).Text().Equal("myvalue")
	} else {
		e.GET("/get").Expect().Status(http.StatusBadRequest)
	}
}

func TestExpectCookieHandlerLiveDisabled(t *testing.T) {
	handler := createCookieHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	e := WithConfig(Config{
		BaseURL:  server.URL,
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Jar: nil,
		},
	})

	testCookieHandler(e, false)
}

func TestExpecCookieHandlerLiveEnabled(t *testing.T) {
	handler := createCookieHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	e := WithConfig(Config{
		BaseURL:  server.URL,
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Jar: NewJar(),
		},
	})

	testCookieHandler(e, true)
}

func TestExpectCookieHandlerBinderStandardDisabled(t *testing.T) {
	handler := createCookieHandler()

	e := WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewBinder(handler),
			Jar:       nil,
		},
	})

	testCookieHandler(e, false)
}

func TestExpectCookieHandlerBinderStandardEnabled(t *testing.T) {
	handler := createCookieHandler()

	e := WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewBinder(handler),
			Jar:       NewJar(),
		},
	})

	testCookieHandler(e, true)
}

func TestExpectCookieHandlerBinderFastDisabled(t *testing.T) {
	handler := fasthttpadaptor.NewFastHTTPHandler(createCookieHandler())

	e := WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewFastBinder(handler),
			Jar:       nil,
		},
	})

	testCookieHandler(e, false)
}

func TestExpectCookieHandlerBinderFastEnabled(t *testing.T) {
	handler := fasthttpadaptor.NewFastHTTPHandler(createCookieHandler())

	e := WithConfig(Config{
		BaseURL:  "http://example.com",
		Reporter: NewAssertReporter(t),
		Client: &http.Client{
			Transport: NewFastBinder(handler),
			Jar:       NewJar(),
		},
	})

	testCookieHandler(e, true)
}

func TestExpectStaticFastBinder(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "httpexpect")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempdir)

	if err := ioutil.WriteFile(
		path.Join(tempdir, "hello"), []byte("hello, world!"), 0666); err != nil {
		t.Fatal(err)
	}

	fs := &fasthttp.FS{
		Root: tempdir,
	}

	handler := fs.NewRequestHandler()

	e := WithConfig(Config{
		Client: &http.Client{
			Transport: NewFastBinder(handler),
			Jar:       NewJar(),
		},
		Reporter: NewAssertReporter(t),
		Printers: []Printer{
			NewDebugPrinter(t, true),
		},
	})

	e.GET("/hello").
		Expect().
		Status(http.StatusOK).
		Text().Equal("hello, world!")
}
