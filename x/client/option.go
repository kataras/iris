package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kataras/golog"
	"golang.org/x/time/rate"
)

// All the builtin client options should live here, for easy discovery.

type Option func(*Client)

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
func Debug(c *Client) {
	handler := &debugRequestHandler{
		logger: golog.Child("Iris HTTP Client: ").SetLevel("debug"),
	}
	c.requestHandlers = append(c.requestHandlers, handler)
}

type debugRequestHandler struct {
	logger *golog.Logger
}

func (h *debugRequestHandler) getHeadersLine(headers http.Header) (headersLine string) {
	for k, v := range headers {
		headersLine += fmt.Sprintf("%s(%s), ", k, strings.Join(v, ","))
	}

	headersLine = strings.TrimRight(headersLine, ", ")
	return
}

func (h *debugRequestHandler) BeginRequest(ctx context.Context, req *http.Request) error {
	format := "%s: %s: content length: %d: headers: %s"
	headersLine := h.getHeadersLine(req.Header)

	h.logger.Debugf(format, req.Method, req.URL.String(), req.ContentLength, headersLine)
	return nil
}

func (h *debugRequestHandler) EndRequest(ctx context.Context, resp *http.Response, err error) error {
	if err != nil {
		h.logger.Debugf("%s: %s: ERR: %s", resp.Request.Method, resp.Request.URL.String(), err.Error())
	} else {
		format := "%s: %s: content length: %d: headers: %s"
		headersLine := h.getHeadersLine(resp.Header)

		h.logger.Debugf(format, resp.Request.Method, resp.Request.URL.String(), resp.ContentLength, headersLine)
	}

	return err
}
