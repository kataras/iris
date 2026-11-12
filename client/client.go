package client

import (
	"net/http"
	"slices"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"

	"github.com/kataras/httpclient"
)

type (
	// A Client is an HTTP client. Initialize with the New package-level function.
	//
	// It is github.com/kataras/httpclient.Client, so every method of that type
	// is available: Do, JSON, Form, Call, BindJSON, BindPlain, ReadJSON,
	// ReadPlain, GetPlainUnquote, WriteTo, NewUploader, Clone,
	// RegisterRequestHandler and DrainResponseBody.
	Client = httpclient.Client

	// Option configures a Client. Pass options to New or Client.Clone.
	Option = httpclient.Option

	// RequestOption declares the type of option one can pass to the request
	// methods (Do, JSON, Form, BindJSON, ReadJSON...). Request options run
	// before the request is sent.
	RequestOption = httpclient.RequestOption

	// RequestHandler observes every request. Its BeginRequest fires right
	// before the client talks to the server and its EndRequest right after a
	// response is received. A non-nil error from either stops the call; an
	// EndRequest handler that returns the error it was given passes it through.
	RequestHandler = httpclient.RequestHandler

	// APIError is the error returned for a response with a status of 400 or
	// above. Response holds the response, Body its raw body, already read and
	// closed. Server code builds APIError values too, so Response, its Request
	// and the Body may each be nil.
	APIError = httpclient.APIError

	// RetryPolicy configures automatic retries. Register it with Retry.
	RetryPolicy = httpclient.RetryPolicy

	// A Limiter blocks until the next request may be sent. *rate.Limiter of
	// golang.org/x/time/rate satisfies it, and NewRateLimiter returns one.
	Limiter = httpclient.Limiter

	// Uploader collects multipart form fields and files. See Client.NewUploader.
	Uploader = httpclient.Uploader

	// A DebugLogger receives the request and response dumps of the Debug
	// option. *golog.Logger satisfies it.
	DebugLogger = httpclient.DebugLogger

	// PlainText lists the Go types Client.BindPlain can produce from a plain
	// text body: string, []byte, int, int64, float64 and named types over them.
	PlainText = httpclient.PlainText
)

// ErrUploaderClosed is returned when an Uploader is used for a second upload.
var ErrUploaderClosed = httpclient.ErrUploaderClosed

// irisDefaults applies the framework's JSON policy: jsonutil.DefaultMarshalOptions
// for request payloads and jsonutil.DefaultUnmarshalOptions for response bodies.
// New runs it before the caller's options, so a JSONOptions given by the caller
// replaces it.
func irisDefaults(c *Client) {
	httpclient.JSONMarshalOptions(jsonutil.DefaultMarshalOptions)(c)
	httpclient.JSONUnmarshalOptions(jsonutil.DefaultUnmarshalOptions)(c)
}

// New returns a new Iris HTTP Client with the framework's JSON policy applied,
// then the given options in order. See the package documentation for the
// available options.
//
// The default content type to send and receive data is JSON.
func New(opts ...Option) *Client {
	return httpclient.New(slices.Concat([]Option{irisDefaults}, opts)...)
}

// NoOption clears the options recorded before it, so a Clone that starts with
// it keeps none of the parent's, and then re-applies the Iris JSON defaults so
// they survive the reset:
//
//	admin := c.Clone(client.NoOption, client.BaseURL(adminURL))
//
// It is recorded in the option list like any other option, so a clone of that
// clone re-applies the defaults too.
var NoOption Option = func(c *Client) {
	httpclient.NoOption(c)
	irisDefaults(c)
}

// DrainResponseBody drains the response body and closes it, allowing the
// transport to reuse TCP connections. A nil response or a nil body is a no-op.
// The BindXXX, ReadXXX, Call and WriteTo methods do it for you.
func DrainResponseBody(resp *http.Response) error {
	return httpclient.DrainResponseBody(resp)
}

// RegisterRequestHandler registers one or more request handlers to run before
// and after each request on all Clients created afterwards, by this package or
// by github.com/kataras/httpclient directly. Useful for third-party libraries,
// e.g. to register request-response lifecycle logging on init.
func RegisterRequestHandler(reqHandlers ...RequestHandler) {
	httpclient.RegisterRequestHandler(reqHandlers...)
}
