package httpexpect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/ajg/form"
	"github.com/fatih/structs"
	"github.com/gavv/monotime"
	"github.com/google/go-querystring/query"
	"github.com/imkira/go-interpol"
)

// Request provides methods to incrementally build http.Request object,
// send it, and receive response.
type Request struct {
	config     Config
	chain      chain
	http       *http.Request
	path       string
	query      url.Values
	form       url.Values
	formbuf    *bytes.Buffer
	multipart  *multipart.Writer
	forcetype  bool
	typesetter string
	bodysetter string
}

// NewRequest returns a new Request object.
//
// method defines the HTTP method (GET, POST, PUT, etc.). path defines url path.
//
// Simple interpolation is allowed for {named} parameters in path:
//  - if pathargs is given, it's used to substitute first len(pathargs) parameters,
//    regardless of their names
//  - if WithPath() or WithPathObject() is called, it's used to substitute given
//    parameters by name
//
// For example:
//  req := NewRequest(config, "POST", "/repos/{user}/{repo}", "gavv", "httpexpect")
//  // path will be "/repos/gavv/httpexpect"
//
// Or:
//  req := NewRequest(config, "POST", "/repos/{user}/{repo}")
//  req.WithPath("user", "gavv")
//  req.WithPath("repo", "httpexpect")
//  // path will be "/repos/gavv/httpexpect"
//
// After interpolation, path is urlencoded and appended to Config.BaseURL,
// separated by slash. If BaseURL ends with a slash and path (after interpolation)
// starts with a slash, only single slash is inserted.
func NewRequest(config Config, method, path string, pathargs ...interface{}) *Request {
	if config.RequestFactory == nil {
		panic("config.RequestFactory == nil")
	}

	if config.Client == nil {
		panic("config.Client == nil")
	}

	chain := makeChain(config.Reporter)

	n := 0
	path, err := interpol.WithFunc(path, func(k string, w io.Writer) error {
		if n < len(pathargs) {
			if pathargs[n] == nil {
				chain.fail(
					"\nunexpected nil argument for url path format string:\n"+
						" Request(\"%s\", %v...)", method, pathargs)
			} else {
				w.Write([]byte(fmt.Sprint(pathargs[n])))
			}
		} else {
			w.Write([]byte("{"))
			w.Write([]byte(k))
			w.Write([]byte("}"))
		}
		n++
		return nil
	})
	if err != nil {
		chain.fail(err.Error())
	}

	hr, err := config.RequestFactory.NewRequest(method, config.BaseURL, nil)
	if err != nil {
		chain.fail(err.Error())
	}

	return &Request{
		config: config,
		chain:  chain,
		path:   path,
		http:   hr,
	}
}

// WithPath substitutes named parameters in url path.
//
// value is converted to string using fmt.Sprint(). If there is no named
// parameter '{key}' in url path, failure is reported.
//
// Named parameters are case-insensitive.
//
// Example:
//  req := NewRequest(config, "POST", "/repos/{user}/{repo}")
//  req.WithPath("user", "gavv")
//  req.WithPath("repo", "httpexpect")
//  // path will be "/repos/gavv/httpexpect"
func (r *Request) WithPath(key string, value interface{}) *Request {
	if r.chain.failed() {
		return r
	}
	ok := false
	path, err := interpol.WithFunc(r.path, func(k string, w io.Writer) error {
		if strings.EqualFold(k, key) {
			if value == nil {
				r.chain.fail(
					"\nunexpected nil argument for url path format string:\n"+
						" WithPath(\"%s\", %v)", key, value)
			} else {
				w.Write([]byte(fmt.Sprint(value)))
				ok = true
			}
		} else {
			w.Write([]byte("{"))
			w.Write([]byte(k))
			w.Write([]byte("}"))
		}
		return nil
	})
	if err == nil {
		r.path = path
	} else {
		r.chain.fail(err.Error())
		return r
	}
	if !ok {
		r.chain.fail("\nunexpected key for url path format string:\n"+
			" WithPath(\"%s\", %v)\n\npath:\n %q",
			key, value, r.path)
		return r
	}
	return r
}

// WithPathObject substitutes multiple named parameters in url path.
//
// object should be map or struct. If object is struct, it's converted
// to map using https://github.com/fatih/structs. Structs may contain
// "path" struct tag, similar to "json" struct tag for json.Marshal().
//
// Each map value is converted to string using fmt.Sprint(). If there
// is no named parameter for some map '{key}' in url path, failure is
// reported.
//
// Named parameters are case-insensitive.
//
// Example:
//  type MyPath struct {
//      Login string `path:"user"`
//      Repo  string
//  }
//
//  req := NewRequest(config, "POST", "/repos/{user}/{repo}")
//  req.WithPathObject(MyPath{"gavv", "httpexpect"})
//  // path will be "/repos/gavv/httpexpect"
//
//  req := NewRequest(config, "POST", "/repos/{user}/{repo}")
//  req.WithPathObject(map[string]string{"user": "gavv", "repo": "httpexpect"})
//  // path will be "/repos/gavv/httpexpect"
func (r *Request) WithPathObject(object interface{}) *Request {
	if r.chain.failed() {
		return r
	}
	if object == nil {
		return r
	}
	var (
		m  map[string]interface{}
		ok bool
	)
	if reflect.Indirect(reflect.ValueOf(object)).Kind() == reflect.Struct {
		s := structs.New(object)
		s.TagName = "path"
		m = s.Map()
	} else {
		m, ok = canonMap(&r.chain, object)
		if !ok {
			return r
		}
	}
	for k, v := range m {
		r.WithPath(k, v)
	}
	return r
}

// WithQuery adds query parameter to request URL.
//
// value is converted to string using fmt.Sprint() and urlencoded.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithQuery("a", 123)
//  req.WithQuery("b", "foo")
//  // URL is now http://example.com/path?a=123&b=foo
func (r *Request) WithQuery(key string, value interface{}) *Request {
	if r.chain.failed() {
		return r
	}
	if r.query == nil {
		r.query = make(url.Values)
	}
	r.query.Add(key, fmt.Sprint(value))
	return r
}

// WithQueryObject adds multiple query parameters to request URL.
//
// object is converted to query string using github.com/google/go-querystring
// if it's a struct or pointer to struct, or github.com/ajg/form otherwise.
//
// Various object types are supported. Structs may contain "url" struct tag,
// similar to "json" struct tag for json.Marshal().
//
// Example:
//  type MyURL struct {
//      A int    `url:"a"`
//      B string `url:"b"`
//  }
//
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithQueryObject(MyURL{A: 123, B: "foo"})
//  // URL is now http://example.com/path?a=123&b=foo
//
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithQueryObject(map[string]interface{}{"a": 123, "b": "foo"})
//  // URL is now http://example.com/path?a=123&b=foo
func (r *Request) WithQueryObject(object interface{}) *Request {
	if r.chain.failed() {
		return r
	}
	if object == nil {
		return r
	}
	var (
		q   url.Values
		err error
	)
	if reflect.Indirect(reflect.ValueOf(object)).Kind() == reflect.Struct {
		q, err = query.Values(object)
		if err != nil {
			r.chain.fail(err.Error())
			return r
		}
	} else {
		q, err = form.EncodeToValues(object)
		if err != nil {
			r.chain.fail(err.Error())
			return r
		}
	}
	if r.query == nil {
		r.query = make(url.Values)
	}
	for k, v := range q {
		r.query[k] = append(r.query[k], v...)
	}
	return r
}

// WithQueryString parses given query string and adds it to request URL.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithQuery("a", 11)
//  req.WithQueryString("b=22&c=33")
//  // URL is now http://example.com/path?a=11&bb=22&c=33
func (r *Request) WithQueryString(query string) *Request {
	if r.chain.failed() {
		return r
	}
	v, err := url.ParseQuery(query)
	if err != nil {
		r.chain.fail(err.Error())
		return r
	}
	if r.query == nil {
		r.query = make(url.Values)
	}
	for k, v := range v {
		r.query[k] = append(r.query[k], v...)
	}
	return r
}

// WithURL sets request URL.
//
// This URL overwrites Config.BaseURL. Request path passed to NewRequest()
// is appended to this URL, separated by slash if necessary.
//
// Example:
//  req := NewRequest(config, "PUT", "/path")
//  req.WithURL("http://example.com")
//  // URL is now http://example.com/path
func (r *Request) WithURL(urlStr string) *Request {
	if r.chain.failed() {
		return r
	}
	if u, err := url.Parse(urlStr); err == nil {
		r.http.URL = u
	} else {
		r.chain.fail(err.Error())
	}
	return r
}

// WithHeaders adds given headers to request.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithHeaders(map[string]string{
//      "Content-Type": "application/json",
//  })
func (r *Request) WithHeaders(headers map[string]string) *Request {
	if r.chain.failed() {
		return r
	}
	for k, v := range headers {
		r.WithHeader(k, v)
	}
	return r
}

// WithHeader adds given single header to request.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithHeader("Content-Type": "application/json")
func (r *Request) WithHeader(k, v string) *Request {
	if r.chain.failed() {
		return r
	}
	switch http.CanonicalHeaderKey(k) {
	case "Host":
		r.http.Host = v
	case "Content-Type":
		if !r.forcetype {
			delete(r.http.Header, "Content-Type")
		}
		r.forcetype = true
		r.typesetter = "WithHeader"
		r.http.Header.Add(k, v)
	default:
		r.http.Header.Add(k, v)
	}
	return r
}

// WithCookies adds given cookies to request.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithCookies(map[string]string{
//      "foo": "aa",
//      "bar": "bb",
//  })
func (r *Request) WithCookies(cookies map[string]string) *Request {
	if r.chain.failed() {
		return r
	}
	for k, v := range cookies {
		r.WithCookie(k, v)
	}
	return r
}

// WithCookie adds given single cookie to request.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithCookie("name", "value")
func (r *Request) WithCookie(k, v string) *Request {
	if r.chain.failed() {
		return r
	}
	r.http.AddCookie(&http.Cookie{
		Name:  k,
		Value: v,
	})
	return r
}

// WithBasicAuth sets the request's Authorization header to use HTTP
// Basic Authentication with the provided username and password.
//
// With HTTP Basic Authentication the provided username and password
// are not encrypted.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithBasicAuth("john", "secret")
func (r *Request) WithBasicAuth(username, password string) *Request {
	if r.chain.failed() {
		return r
	}
	r.http.SetBasicAuth(username, password)
	return r
}

// WithProto sets HTTP protocol version.
//
// proto should have form of "HTTP/{major}.{minor}", e.g. "HTTP/1.1".
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithProto("HTTP/2.0")
func (r *Request) WithProto(proto string) *Request {
	if r.chain.failed() {
		return r
	}
	major, minor, ok := http.ParseHTTPVersion(proto)
	if !ok {
		r.chain.fail(
			"\nunexpected protocol version %q, expected \"HTTP/{major}.{minor}\"",
			proto)
		return r
	}
	r.http.ProtoMajor = major
	r.http.ProtoMinor = minor
	return r
}

// WithChunked enables chunked encoding and sets request body reader.
//
// Expect() will read all available data from given reader. Content-Length
// is not set, and "chunked" Transfer-Encoding is used.
//
// If protocol version is not at least HTTP/1.1 (required for chunked
// encoding), failure is reported.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/upload")
//  fh, _ := os.Open("data")
//  defer fh.Close()
//  req.WithHeader("Content-Type": "application/octet-stream")
//  req.WithChunked(fh)
func (r *Request) WithChunked(reader io.Reader) *Request {
	if r.chain.failed() {
		return r
	}
	if !r.http.ProtoAtLeast(1, 1) {
		r.chain.fail("chunked Transfer-Encoding requires at least \"HTTP/1.1\","+
			"but \"HTTP/%d.%d\" is enabled", r.http.ProtoMajor, r.http.ProtoMinor)
		return r
	}
	r.setBody("WithChunked", reader, -1, false)
	return r
}

// WithBytes sets request body to given slice of bytes.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithHeader("Content-Type": "application/json")
//  req.WithBytes([]byte(`{"foo": 123}`))
func (r *Request) WithBytes(b []byte) *Request {
	if r.chain.failed() {
		return r
	}
	if b == nil {
		r.setBody("WithBytes", nil, 0, false)
	} else {
		r.setBody("WithBytes", bytes.NewReader(b), len(b), false)
	}
	return r
}

// WithText sets Content-Type header to "text/plain; charset=utf-8" and
// sets body to given string.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithText("hello, world!")
func (r *Request) WithText(s string) *Request {
	if r.chain.failed() {
		return r
	}
	r.setType("WithText", "text/plain; charset=utf-8", false)
	r.setBody("WithText", strings.NewReader(s), len(s), false)
	return r
}

// WithJSON sets Content-Type header to "application/json; charset=utf-8"
// and sets body to object, marshaled using json.Marshal().
//
// Example:
//  type MyJSON struct {
//      Foo int `json:"foo"`
//  }
//
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithJSON(MyJSON{Foo: 123})
//
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithJSON(map[string]interface{}{"foo": 123})
func (r *Request) WithJSON(object interface{}) *Request {
	if r.chain.failed() {
		return r
	}
	b, err := json.Marshal(object)
	if err != nil {
		r.chain.fail(err.Error())
		return r
	}

	r.setType("WithJSON", "application/json; charset=utf-8", false)
	r.setBody("WithJSON", bytes.NewReader(b), len(b), false)

	return r
}

// WithForm sets Content-Type header to "application/x-www-form-urlencoded"
// or (if WithMultipart() was called) "multipart/form-data", converts given
// object to url.Values using github.com/ajg/form, and adds it to request body.
//
// Various object types are supported, including maps and structs. Structs may
// contain "form" struct tag, similar to "json" struct tag for json.Marshal().
// See https://github.com/ajg/form for details.
//
// Multiple WithForm(), WithFormField(), and WithFile() calls may be combined.
// If WithMultipart() is called, it should be called first.
//
// Example:
//  type MyForm struct {
//      Foo int `form:"foo"`
//  }
//
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithForm(MyForm{Foo: 123})
//
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithForm(map[string]interface{}{"foo": 123})
func (r *Request) WithForm(object interface{}) *Request {
	if r.chain.failed() {
		return r
	}

	f, err := form.EncodeToValues(object)
	if err != nil {
		r.chain.fail(err.Error())
		return r
	}

	if r.multipart != nil {
		r.setType("WithForm", "multipart/form-data", false)

		var keys []string
		for k := range f {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := r.multipart.WriteField(k, f[k][0]); err != nil {
				r.chain.fail(err.Error())
				return r
			}
		}
	} else {
		r.setType("WithForm", "application/x-www-form-urlencoded", false)

		if r.form == nil {
			r.form = make(url.Values)
		}
		for k, v := range f {
			r.form[k] = append(r.form[k], v...)
		}
	}

	return r
}

// WithFormField sets Content-Type header to "application/x-www-form-urlencoded"
// or (if WithMultipart() was called) "multipart/form-data", converts given
// value to string using fmt.Sprint(), and adds it to request body.
//
// Multiple WithForm(), WithFormField(), and WithFile() calls may be combined.
// If WithMultipart() is called, it should be called first.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithFormField("foo", 123).
//      WithFormField("bar", 456)
func (r *Request) WithFormField(key string, value interface{}) *Request {
	if r.chain.failed() {
		return r
	}
	if r.multipart != nil {
		r.setType("WithFormField", "multipart/form-data", false)

		err := r.multipart.WriteField(key, fmt.Sprint(value))
		if err != nil {
			r.chain.fail(err.Error())
			return r
		}
	} else {
		r.setType("WithFormField", "application/x-www-form-urlencoded", false)

		if r.form == nil {
			r.form = make(url.Values)
		}
		r.form[key] = append(r.form[key], fmt.Sprint(value))
	}
	return r
}

// WithFile sets Content-Type header to "multipart/form-data", reads given
// file and adds its contents to request body.
//
// If reader is given, it's used to read file contents. Otherwise, os.Open()
// is used to read a file with given path.
//
// Multiple WithForm(), WithFormField(), and WithFile() calls may be combined.
// WithMultipart() should be called before WithFile(), otherwise WithFile()
// fails.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithFile("avatar", "./john.png")
//
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  fh, _ := os.Open("./john.png")
//  req.WithMultipart().
//      WithFile("avatar", "john.png", fh)
//  fh.Close()
func (r *Request) WithFile(key, path string, reader ...io.Reader) *Request {
	if r.chain.failed() {
		return r
	}

	r.setType("WithFile", "multipart/form-data", false)

	if r.multipart == nil {
		r.chain.fail("WithFile requires WithMultipart to be called first")
		return r
	}

	wr, err := r.multipart.CreateFormFile(key, path)
	if err != nil {
		r.chain.fail(err.Error())
		return r
	}

	var rd io.Reader
	if len(reader) != 0 && reader[0] != nil {
		rd = reader[0]
	} else {
		f, err := os.Open(path)
		if err != nil {
			r.chain.fail(err.Error())
			return r
		}
		rd = f
		defer f.Close()
	}

	if _, err := io.Copy(wr, rd); err != nil {
		r.chain.fail(err.Error())
		return r
	}

	return r
}

// WithFileBytes is like WithFile, but uses given slice of bytes as the
// file contents.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  fh, _ := os.Open("./john.png")
//  b, _ := ioutil.ReadAll(fh)
//  req.WithMultipart().
//      WithFileBytes("avatar", "john.png", b)
//  fh.Close()
func (r *Request) WithFileBytes(key, path string, data []byte) *Request {
	if r.chain.failed() {
		return r
	}
	return r.WithFile(key, path, bytes.NewReader(data))
}

// WithMultipart sets Content-Type header to "multipart/form-data".
//
// After this call, WithForm() and WithFormField() switch to multipart
// form instead of urlencoded form.
//
// If WithMultipart() is called, it should be called before WithForm(),
// WithFormField(), and WithFile().
//
// WithFile() always requires WithMultipart() to be called first.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithMultipart().
//      WithForm(map[string]interface{}{"foo": 123})
func (r *Request) WithMultipart() *Request {
	if r.chain.failed() {
		return r
	}

	r.setType("WithMultipart", "multipart/form-data", false)

	if r.multipart == nil {
		r.formbuf = new(bytes.Buffer)
		r.multipart = multipart.NewWriter(r.formbuf)
		r.setBody("WithMultipart", r.formbuf, 0, false)
	}

	return r
}

// Expect constructs http.Request, sends it, receives http.Response, and
// returns a new Response object to inspect received response.
//
// Request is sent using Config.Client interface.
//
// Example:
//  req := NewRequest(config, "PUT", "http://example.com/path")
//  req.WithJSON(map[string]interface{}{"foo": 123})
//  resp := req.Expect()
//  resp.Status(http.StatusOK)
func (r *Request) Expect() *Response {
	r.encodeRequest()

	resp, elapsed := r.sendRequest()

	return makeResponse(r.chain, resp, elapsed)
}

func (r *Request) encodeRequest() {
	if r.chain.failed() {
		return
	}

	r.http.URL.Path = concatPaths(r.http.URL.Path, r.path)

	if r.query != nil {
		r.http.URL.RawQuery = r.query.Encode()
	}

	if r.multipart != nil {
		if err := r.multipart.Close(); err != nil {
			r.chain.fail(err.Error())
			return
		}

		r.setType("Expect", r.multipart.FormDataContentType(), true)
		r.setBody("Expect", r.formbuf, r.formbuf.Len(), true)
	} else if r.form != nil {
		s := r.form.Encode()
		r.setBody("WithForm or WithFormField", strings.NewReader(s), len(s), false)
	}
}

func (r *Request) sendRequest() (resp *http.Response, elapsed time.Duration) {
	if r.chain.failed() {
		return
	}

	for _, printer := range r.config.Printers {
		printer.Request(r.http)
	}

	start := monotime.Now()

	resp, err := r.config.Client.Do(r.http)

	elapsed = monotime.Since(start)

	if err != nil {
		r.chain.fail(err.Error())
		return
	}

	for _, printer := range r.config.Printers {
		printer.Response(resp, elapsed)
	}

	return
}

func (r *Request) setType(newSetter, newType string, overwrite bool) {
	if r.forcetype {
		return
	}

	if !overwrite {
		previousType := r.http.Header.Get("Content-Type")

		if previousType != "" && previousType != newType {
			r.chain.fail(
				"\nambiguous request \"Content-Type\" header values:\n %q (set by %s)\n\n"+
					"and:\n %q (wanted by %s)",
				previousType, r.typesetter,
				newType, newSetter)
			return
		}
	}

	r.typesetter = newSetter
	r.http.Header["Content-Type"] = []string{newType}
}

func (r *Request) setBody(setter string, reader io.Reader, len int, overwrite bool) {
	if !overwrite && r.bodysetter != "" {
		r.chain.fail(
			"\nambiguous request body contents:\n  set by %s\n  overwritten by %s",
			r.bodysetter, setter)
		return
	}

	if len > 0 && reader == nil {
		panic("invalid length")
	}

	if reader == nil {
		r.http.Body = nil
		r.http.ContentLength = 0
	} else {
		r.http.Body = ioutil.NopCloser(reader)
		r.http.ContentLength = int64(len)
	}

	r.bodysetter = setter
}

func concatPaths(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	a = strings.TrimSuffix(a, "/")
	b = strings.TrimPrefix(b, "/")
	return a + "/" + b
}
