package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// the base client
type Client struct {
	HTTPClient *http.Client

	// BaseURL prepends to all requests.
	BaseURL string

	// A list of persistent request options.
	PersistentRequestOptions []RequestOption
}

func New(opts ...Option) *Client {
	c := &Client{
		HTTPClient:               &http.Client{},
		PersistentRequestOptions: defaultRequestOptions,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type RequestOption func(*http.Request) error

// We always add the following request headers, unless they're removed by custom ones.
var defaultRequestOptions = []RequestOption{
	RequestHeader(false, acceptKey, contentTypeJSON),
}

func RequestHeader(overridePrev bool, key string, values ...string) RequestOption {
	key = http.CanonicalHeaderKey(key)

	return func(req *http.Request) error {
		if overridePrev { // upsert.
			req.Header[key] = values
		} else { // just insert.
			req.Header[key] = append(req.Header[key], values...)
		}

		return nil
	}
}

// RequestAuthorization sets an Authorization request header.
// Note that we could do the same with a Transport RoundDrip too.
func RequestAuthorization(value string) RequestOption {
	return RequestHeader(true, "Authorization", value)
}

// RequestAuthorizationBearer sets an Authorization: Bearer $token request header.
func RequestAuthorizationBearer(accessToken string) RequestOption {
	headerValue := "Bearer " + accessToken
	return RequestAuthorization(headerValue)
}

// RequestQuery adds a set of URL query parameters to the request.
func RequestQuery(query url.Values) RequestOption {
	return func(req *http.Request) error {
		q := req.URL.Query()
		for k, v := range query {
			q[k] = v
		}
		req.URL.RawQuery = q.Encode()

		return nil
	}
}

// RequestParam sets a single URL query parameter to the request.
func RequestParam(key string, values ...string) RequestOption {
	return RequestQuery(url.Values{
		key: values,
	})
}

// Do sends an HTTP request and returns an HTTP response.
//
// The payload can be:
// - io.Reader
// - raw []byte
// - JSON raw message
// - string
// - struct (JSON).
//
// If method is empty then it defaults to "GET".
// The final variadic, optional input argument sets
// the custom request options to use before the request.
//
// Any HTTP returned error will be of type APIError
// or a timeout error if the given context was canceled.
func (c *Client) Do(ctx context.Context, method, urlpath string, payload interface{}, opts ...RequestOption) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Method defaults to GET.
	if method == "" {
		method = http.MethodGet
	}

	// Find the payload, if any.
	var body io.Reader
	if payload != nil {
		switch v := payload.(type) {
		case io.Reader:
			body = v
		case []byte:
			body = bytes.NewBuffer(v)
		case json.RawMessage:
			body = bytes.NewBuffer(v)
		case string:
			body = strings.NewReader(v)
		case url.Values:
			body = strings.NewReader(v.Encode())
		default:
			w := new(bytes.Buffer)
			// We assume it's a struct, we wont make use of reflection to find out though.
			err := json.NewEncoder(w).Encode(v)
			if err != nil {
				return nil, err
			}
			body = w
		}
	}

	if c.BaseURL != "" {
		urlpath = c.BaseURL + urlpath // note that we don't do any special checks here, the caller is responsible.
	}

	// Initialize the request.
	req, err := http.NewRequestWithContext(ctx, method, urlpath, body)
	if err != nil {
		return nil, err
	}

	// We separate the error for the default options for now.
	for i, opt := range c.PersistentRequestOptions {
		if opt == nil {
			continue
		}

		if err = opt(req); err != nil {
			return nil, fmt.Errorf("client.Do: default request option[%d]: %w", i, err)
		}
	}

	// Apply any custom request options (e.g. content type, accept headers, query...)
	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if err = opt(req); err != nil {
			return nil, err
		}
	}

	// Caller is responsible for closing the response body.
	// Also note that the gzip compression is handled automatically nowadays.
	return c.HTTPClient.Do(req)
}

const (
	acceptKey                 = "Accept"
	contentTypeKey            = "Content-Type"
	contentLengthKey          = "Content-Length"
	contentTypePlainText      = "plain/text"
	contentTypeJSON           = "application/json"
	contentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

func (c *Client) JSON(ctx context.Context, method, urlpath string, payload interface{}, opts ...RequestOption) (*http.Response, error) {
	opts = append(opts, RequestHeader(true, contentTypeKey, contentTypeJSON))
	return c.Do(ctx, method, urlpath, payload, opts...)
}

func (c *Client) Form(ctx context.Context, method, urlpath string, formValues url.Values, opts ...RequestOption) (*http.Response, error) {
	payload := formValues.Encode()

	opts = append(opts,
		RequestHeader(true, contentTypeKey, contentTypeFormURLEncoded),
		RequestHeader(true, contentLengthKey, strconv.Itoa(len(payload))),
	)

	return c.Do(ctx, method, urlpath, payload, opts...)
}

type Uploader struct {
	client *Client

	body   *bytes.Buffer
	Writer *multipart.Writer
}

func (u *Uploader) AddField(key, value string) error {
	f, err := u.Writer.CreateFormField(key)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, strings.NewReader(value))
	return err
}

func (u *Uploader) AddFileSource(key, filename string, source io.Reader) error {
	f, err := u.Writer.CreateFormFile(key, filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, source)
	return err
}

func (u *Uploader) AddFile(key, filename string) error {
	source, err := os.Open(filename)
	if err != nil {
		return err
	}

	return u.AddFileSource(key, filename, source)
}

func (u *Uploader) Upload(ctx context.Context, method, urlpath string, opts ...RequestOption) (*http.Response, error) {
	err := u.Writer.Close()
	if err != nil {
		return nil, err
	}

	payload := bytes.NewReader(u.body.Bytes())
	opts = append(opts, RequestHeader(true, contentTypeKey, u.Writer.FormDataContentType()))

	return u.client.Do(ctx, method, urlpath, payload, opts...)
}

func (c *Client) NewUploader() *Uploader {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	return &Uploader{
		client: c,
		body:   body,
		Writer: writer,
	}
}

func (c *Client) ReadJSON(ctx context.Context, dest interface{}, method, urlpath string, payload interface{}, opts ...RequestOption) error {
	if payload != nil {
		opts = append(opts, RequestHeader(true, contentTypeKey, contentTypeJSON))
	}

	resp, err := c.Do(ctx, method, urlpath, payload, opts...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return ExtractError(resp)
	}

	// DBUG
	// b, _ := ioutil.ReadAll(resp.Body)
	// println(string(b))
	// return json.Unmarshal(b, &dest)

	return json.NewDecoder(resp.Body).Decode(&dest)
}

// ReadPlain like ReadJSON but it accepts a pointer to a string or byte slice or integer
// and it reads the body as plain text.
func (c *Client) ReadPlain(ctx context.Context, dest interface{}, method, urlpath string, payload interface{}, opts ...RequestOption) error {
	resp, err := c.Do(ctx, method, urlpath, payload, opts...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return ExtractError(resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch ptr := dest.(type) {
	case *[]byte:
		*ptr = body
		return nil
	case *string:
		*ptr = string(body)
		return nil
	case *int:
		*ptr, err = strconv.Atoi(string(body))
		return err
	default:
		return fmt.Errorf("unsupported response body type: %T", ptr)
	}
}

// GetPlainUnquote reads the response body as raw text and tries to unquote it,
// useful when the remote server sends a single key as a value but due to backend mistake
// it sends it as JSON (quoted) instead of plain text.
func (c *Client) GetPlainUnquote(ctx context.Context, method, urlpath string, payload interface{}, opts ...RequestOption) (string, error) {
	var bodyStr string
	if err := c.ReadPlain(ctx, &bodyStr, method, urlpath, payload, opts...); err != nil {
		return "", err
	}

	s, err := strconv.Unquote(bodyStr)
	if err == nil {
		bodyStr = s
	}

	return bodyStr, nil
}

// WriteTo reads the response and then copies its data to the "dest" writer.
// If the "dest" is a type of HTTP response writer then it writes the
// content-type and content-length of the original request.
//
// Returns the amount of bytes written to "dest".
func (c *Client) WriteTo(ctx context.Context, dest io.Writer, method, urlpath string, payload interface{}, opts ...RequestOption) (int64, error) {
	if payload != nil {
		opts = append(opts, RequestHeader(true, contentTypeKey, contentTypeJSON))
	}

	resp, err := c.Do(ctx, method, urlpath, payload, opts...)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if w, ok := dest.(http.ResponseWriter); ok {
		// Copy the content type and content-length.
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		if resp.ContentLength > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
		}
	}

	return io.Copy(dest, resp.Body)
}

// BindResponse consumes the response's body and binds the result to the "dest" pointer,
// closing the response's body is up to the caller.
//
// The "dest" will be binded based on the response's content type header.
// Note that this is strict in order to catch bad actioners fast,
// e.g. it wont try to read plain text if not specified on
// the response headers and the dest is a *string.
func BindResponse(resp *http.Response, dest interface{}) (err error) {
	contentType := trimHeader(resp.Header.Get(contentTypeKey))
	switch contentType {
	case contentTypeJSON: // the most common scenario on successful responses.
		return json.NewDecoder(resp.Body).Decode(&dest)
	case contentTypePlainText:
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		switch v := dest.(type) {
		case *string:
			*v = string(b)
		case *[]byte:
			*v = b
		default:
			return fmt.Errorf("plain text response should accept a *string or a *[]byte")
		}

	default:
		acceptContentType := trimHeader(resp.Request.Header.Get(acceptKey))
		msg := ""
		if acceptContentType == contentType {
			// Here we make a special case, if the content type
			// was explicitly set by the request but we cannot handle it.
			msg = fmt.Sprintf("current implementation can not handle the received (and accepted) mime type: %s", contentType)
		} else {
			msg = fmt.Sprintf("unexpected mime type received: %s", contentType)
		}
		err = errors.New(msg)
	}

	return
}

func trimHeader(v string) string {
	for i, char := range v {
		if char == ' ' || char == ';' {
			return v[:i]
		}
	}
	return v
}
