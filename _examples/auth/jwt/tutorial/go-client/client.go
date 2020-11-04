package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Client is the default http client instance used by the following methods.
var Client = http.DefaultClient

// RequestOption is a function which can be used to modify
// a request instance before Do.
type RequestOption func(*http.Request) error

// WithAccessToken sets the given "token" to the authorization request header.
func WithAccessToken(token []byte) RequestOption {
	bearer := "Bearer " + string(token)
	return func(req *http.Request) error {
		req.Header.Add("Authorization", bearer)
		return nil
	}
}

// WithContentType sets the content-type request header.
func WithContentType(cType string) RequestOption {
	return func(req *http.Request) error {
		req.Header.Set("Content-Type", cType)
		return nil
	}
}

// WithContentLength sets the content-length request header.
func WithContentLength(length int) RequestOption {
	return func(req *http.Request) error {
		req.Header.Set("Content-Length", strconv.Itoa(length))
		return nil
	}
}

// Do fires a request to the server.
func Do(method, url string, body io.Reader, opts ...RequestOption) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err = opt(req); err != nil {
			return nil, err
		}
	}

	return Client.Do(req)
}

// JSON fires a request with "v" as client json data.
func JSON(method, url string, v interface{}, opts ...RequestOption) (*http.Response, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}

	opts = append(opts, WithContentType("application/json; charset=utf-8"))
	return Do(method, url, buf, opts...)
}

// Form fires a request with "formData" as client form data.
func Form(method, url string, formData url.Values, opts ...RequestOption) (*http.Response, error) {
	encoded := formData.Encode()
	body := strings.NewReader(encoded)

	opts = append([]RequestOption{
		WithContentType("application/x-www-form-urlencoded"),
		WithContentLength(len(encoded)),
	}, opts...)

	return Do(method, url, body, opts...)
}

// BindResponse binds a response body to the "dest" pointer and closes the body.
func BindResponse(resp *http.Response, dest interface{}) error {
	contentType := resp.Header.Get("Content-Type")
	if idx := strings.IndexRune(contentType, ';'); idx > 0 {
		contentType = contentType[0:idx]
	}

	switch contentType {
	case "application/json":
		defer resp.Body.Close()
		return json.NewDecoder(resp.Body).Decode(dest)
	default:
		return fmt.Errorf("unsupported content type: %s", contentType)
	}
}

// RawResponse simply returns the raw response body.
func RawResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
