package client

import (
	"context"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/kataras/golog"
	"golang.org/x/time/rate"
)

// All the builtin client options should live here, for easy discovery.

type Option = func(*Client)

// BaseURL registers the base URL of this client.
// All of its methods will prepend this url.
func BaseURL(uri string) Option {
	return func(c *Client) {
		c.BaseURL = uri
	}
}

// Timeout specifies a time limit for requests made by this
// Client. The timeout includes connection time, any
// redirects, and reading the response body.
// A Timeout of zero means no timeout.
//
// Defaults to 15 seconds.
func Timeout(d time.Duration) Option {
	return func(c *Client) {
		c.HTTPClient.Timeout = d
	}
}

// Handler specifies an iris.Application or any http.Handler
// instance which can be tested using this Client.
//
// It registers a custom HTTP client transport
// which allows "fake calls" to the "h" server. Use it for testing.
func Handler(h http.Handler) Option {
	return func(c *Client) {
		c.HTTPClient.Transport = new(handlerTransport)
	}
}

// PersistentRequestOptions adds one or more persistent request options
// that all requests made by this Client will respect.
func PersistentRequestOptions(reqOpts ...RequestOption) Option {
	return func(c *Client) {
		c.PersistentRequestOptions = append(c.PersistentRequestOptions, reqOpts...)
	}
}

// RateLimit configures the rate limit for requests.
//
// Defaults to zero which disables rate limiting.
func RateLimit(requestsPerSecond int) Option {
	return func(c *Client) {
		c.rateLimiter = rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond)
	}
}

// Debug enables the client's debug logger.
// It fires right before request is created
// and right after a response from the server is received.
//
// Example Output for request:
//
//	[DBUG] 2022/03/01 21:54 Iris HTTP Client: POST / HTTP/1.1
//	Host: 127.0.0.1:50948
//	User-Agent: Go-http-client/1.1
//	Content-Length: 22
//	Accept: application/json
//	Content-Type: application/json
//	Accept-Encoding: gzip
//
//	{"firstname":"Makis"}
//
// Example Output for response:
//
//	[DBUG] 2022/03/01 21:54 Iris HTTP Client: HTTP/1.1 200 OK
//	Content-Length: 27
//	Content-Type: application/json; charset=utf-8
//	Date: Tue, 01 Mar 2022 19:54:03 GMT
//
//	{
//	    "firstname": "Makis"
//	}
func Debug(c *Client) {
	handler := &debugRequestHandler{
		logger: golog.Child("Iris HTTP Client: ").SetLevel("debug"),
	}
	c.requestHandlers = append(c.requestHandlers, handler)
}

type debugRequestHandler struct {
	logger *golog.Logger
}

func (h *debugRequestHandler) BeginRequest(ctx context.Context, req *http.Request) error {
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return err
	}

	h.logger.Debug(string(dump))
	return nil
}

func (h *debugRequestHandler) EndRequest(ctx context.Context, resp *http.Response, err error) error {
	if err != nil {
		h.logger.Debugf("%s: %s: ERR: %s", resp.Request.Method, resp.Request.URL.String(), err.Error())
	} else {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}

		h.logger.Debug(string(dump))
	}

	return err
}
