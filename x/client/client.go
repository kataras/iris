package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"golang.org/x/time/rate"
)

// A Client is an HTTP client. Initialize with the New package-level function.
type Client struct {
	opts []Option // keep for clones.

	HTTPClient *http.Client

	// BaseURL prepends to all requests.
	BaseURL string

	// A list of persistent request options.
	PersistentRequestOptions []RequestOption

	// Optional rate limiter instance initialized by the RateLimit method.
	rateLimiter *rate.Limiter

	// Optional handlers that are being fired before and after each new request.
	requestHandlers []RequestHandler

	// store it here for future use.
	keepAlive bool
}

// New returns a new Iris HTTP Client.
// Available options:
// - BaseURL
// - Timeout
// - PersistentRequestOptions
// - RateLimit
//
// Look the Client.Do/JSON/... methods to send requests and
// ReadXXX methods to read responses.
//
// The default content type to send and receive data is JSON.
func New(opts ...Option) *Client {
	c := &Client{
		opts: opts,

		HTTPClient:               &http.Client{},
		PersistentRequestOptions: defaultRequestOptions,
		requestHandlers:          defaultRequestHandlers,
	}

	for _, opt := range c.opts { // c.opts in order to make with `NoOption` work.
		opt(c)
	}

	if transport, ok := c.HTTPClient.Transport.(*http.Transport); ok {
		c.keepAlive = !transport.DisableKeepAlives
	}

	return c
}

// NoOption is a helper function that clears the previous options in the chain.
// See `Client.Clone` method.
var NoOption = func(c *Client) { c.opts = make([]Option, 0) /* clear previous options */ }

// Clone returns a new Client with the same options as the original.
// If you want to override the options from the base "c" Client,
// use the `NoOption` variable as the 1st argument.
func (c *Client) Clone(opts ...Option) *Client {
	return New(append(c.opts, opts...)...)
}

// RegisterRequestHandler registers one or more request handlers
// to be ran before and after of each new request.
//
// Request handler's BeginRequest method run after each request constructed
// and right before sent to the server.
//
// Request handler's EndRequest method run after response each received
// and right before methods return back to the caller.
//
// Any request handlers MUST be set right after the Client's initialization.
func (c *Client) RegisterRequestHandler(reqHandlers ...RequestHandler) {
	reqHandlersToRegister := make([]RequestHandler, 0, len(reqHandlers))
	for _, h := range reqHandlers {
		if h == nil {
			continue
		}

		reqHandlersToRegister = append(reqHandlersToRegister, h)
	}

	c.requestHandlers = append(c.requestHandlers, reqHandlersToRegister...)
}

func (c *Client) emitBeginRequest(ctx context.Context, req *http.Request) error {
	if len(c.requestHandlers) == 0 {
		return nil
	}

	for _, h := range c.requestHandlers {
		if hErr := h.BeginRequest(ctx, req); hErr != nil {
			return hErr
		}
	}

	return nil
}

func (c *Client) emitEndRequest(ctx context.Context, resp *http.Response, err error) error {
	if len(c.requestHandlers) == 0 {
		return nil
	}

	for _, h := range c.requestHandlers {
		if hErr := h.EndRequest(ctx, resp, err); hErr != nil {
			return hErr
		}
	}

	return err
}

// RequestOption declares the type of option one can pass
// to the Do methods(JSON, Form, ReadJSON...).
// Request options run before request constructed.
type RequestOption = func(*http.Request) error

// We always add the following request headers, unless they're removed by custom ones.
var defaultRequestOptions = []RequestOption{
	RequestHeader(false, acceptKey, contentTypeJSON),
}

// RequestHeader adds or sets (if overridePrev is true) a header to the request.
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

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
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

	if err = c.emitBeginRequest(ctx, req); err != nil {
		return nil, err
	}

	// Caller is responsible for closing the response body.
	// Also note that the gzip compression is handled automatically nowadays.
	resp, respErr := c.HTTPClient.Do(req)

	if err = c.emitEndRequest(ctx, resp, respErr); err != nil {
		return nil, err
	}

	return resp, respErr
}

// DrainResponseBody drains response body and close it, allowing the transport to reuse TCP connections.
// It's automatically called on Client.ReadXXX methods on the end.
func (c *Client) DrainResponseBody(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

const (
	acceptKey                 = "Accept"
	contentTypeKey            = "Content-Type"
	contentLengthKey          = "Content-Length"
	contentTypePlainText      = "plain/text"
	contentTypeJSON           = "application/json"
	contentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

// JSON writes data as JSON to the server.
func (c *Client) JSON(ctx context.Context, method, urlpath string, payload interface{}, opts ...RequestOption) (*http.Response, error) {
	opts = append(opts, RequestHeader(true, contentTypeKey, contentTypeJSON))
	return c.Do(ctx, method, urlpath, payload, opts...)
}

// JSON writes form data to the server.
func (c *Client) Form(ctx context.Context, method, urlpath string, formValues url.Values, opts ...RequestOption) (*http.Response, error) {
	payload := formValues.Encode()

	opts = append(opts,
		RequestHeader(true, contentTypeKey, contentTypeFormURLEncoded),
		RequestHeader(true, contentLengthKey, strconv.Itoa(len(payload))),
	)

	return c.Do(ctx, method, urlpath, payload, opts...)
}

// Uploader holds the necessary information for upload requests.
//
// Look the Client.NewUploader method.
type Uploader struct {
	client *Client

	body   *bytes.Buffer
	Writer *multipart.Writer
}

// AddFileSource adds a form field to the uploader with the given key.
func (u *Uploader) AddField(key, value string) error {
	f, err := u.Writer.CreateFormField(key)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, strings.NewReader(value))
	return err
}

// AddFileSource adds a form file to the uploader with the given key.
func (u *Uploader) AddFileSource(key, filename string, source io.Reader) error {
	f, err := u.Writer.CreateFormFile(key, filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, source)
	return err
}

// AddFile adds a local form file to the uploader with the given key.
func (u *Uploader) AddFile(key, filename string) error {
	source, err := os.Open(filename)
	if err != nil {
		return err
	}

	return u.AddFileSource(key, filename, source)
}

// Uploads sends local data to the server.
func (u *Uploader) Upload(ctx context.Context, method, urlpath string, opts ...RequestOption) (*http.Response, error) {
	err := u.Writer.Close()
	if err != nil {
		return nil, err
	}

	payload := bytes.NewReader(u.body.Bytes())
	opts = append(opts, RequestHeader(true, contentTypeKey, u.Writer.FormDataContentType()))

	return u.client.Do(ctx, method, urlpath, payload, opts...)
}

// NewUploader returns a structure which is responsible for sending
// file and form data to the server.
func (c *Client) NewUploader() *Uploader {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	return &Uploader{
		client: c,
		body:   body,
		Writer: writer,
	}
}

// ReadJSON binds "dest" to the response's body.
// After this call, the response body reader is closed.
func (c *Client) ReadJSON(ctx context.Context, dest interface{}, method, urlpath string, payload interface{}, opts ...RequestOption) error {
	if payload != nil {
		opts = append(opts, RequestHeader(true, contentTypeKey, contentTypeJSON))
	}

	resp, err := c.Do(ctx, method, urlpath, payload, opts...)
	if err != nil {
		return err
	}
	defer c.DrainResponseBody(resp)

	if resp.StatusCode >= http.StatusBadRequest {
		return ExtractError(resp)
	}

	// DBUG
	// b, _ := io.ReadAll(resp.Body)
	// println(string(b))
	// return json.Unmarshal(b, &dest)

	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(&dest)
	}

	return json.NewDecoder(resp.Body).Decode(&dest)
}

// ReadPlain like ReadJSON but it accepts a pointer to a string or byte slice or integer
// and it reads the body as plain text.
func (c *Client) ReadPlain(ctx context.Context, dest interface{}, method, urlpath string, payload interface{}, opts ...RequestOption) error {
	resp, err := c.Do(ctx, method, urlpath, payload, opts...)
	if err != nil {
		return err
	}
	defer c.DrainResponseBody(resp)

	if resp.StatusCode >= http.StatusBadRequest {
		return ExtractError(resp)
	}

	body, err := io.ReadAll(resp.Body)
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
		b, err := io.ReadAll(resp.Body)
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
