package client

import (
	"net/http"
)

// The re-exported surface, referenced once each so that a wrapper dropped by
// accident fails this package's build rather than a downstream one.
var (
	_ = []any{
		New, NoOption,
		BaseURL, Timeout, DialTimeout, Transport, Handler, OAuth2, PersistentRequestOptions,
		RateLimit, RateLimitPerMinute, RateLimiter, RateLimitFor, RateLimitForPerMinute, RateLimiterFor,
		NewRateLimiter, NewRateLimiterPerMinute,
		Retry, RedactQueryParams, RedactHeaders, RedactURL,
		JSONOptions, JSONMarshalOptions, JSONUnmarshalOptions, Debug,
		RequestHeader, RequestAuthorization, RequestAuthorizationBearer, RequestQuery, RequestParam,
		RequestRateLimit, ClientTrace,
		DrainResponseBody, RegisterRequestHandler,
		ExtractError, GetError, GetErrorCode, DecodeError, IsErrEmptyJSON,
		ErrUploaderClosed, DefaultDebugLogger,
		PageSize, ListMethod, MethodQuery,
	}

	_ = Bind[string]
	_ = BindError[string]
	_ = List[string]
	_ = Items[string]

	_ Client
	_ Option
	_ RequestOption
	_ RequestHandler
	_ APIError
	_ RetryPolicy
	_ Limiter
	_ Uploader
	_ DebugLogger
	_ PageOptions
	_ Page[string]
	_ CountResponse[int64]
	_ ListFilter

	// Methods reachable through the alias.
	_ = (*Client).Do
	_ = (*Client).JSON
	_ = (*Client).Form
	_ = (*Client).Call
	_ = (*Client).ReadJSON
	_ = (*Client).ReadPlain
	_ = (*Client).GetPlainUnquote
	_ = (*Client).WriteTo
	_ = (*Client).NewUploader
	_ = (*Client).Clone
	_ = (*Client).RegisterRequestHandler
	_ = (*Client).DrainResponseBody
	_ = (*Client).BindJSON[string]
	_ = (*Client).BindPlain[string]

	_ APIError = APIError{Response: &http.Response{}}
)
