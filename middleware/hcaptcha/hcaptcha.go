// Package hcaptcha verifies the hCaptcha answer that a form submission carries.
//
// Its entry point is New(Options), which returns a Handler that calls SiteVerify
// and runs Options.FailureHandler when the answer does not pass. Read the stored
// Response with Get.
package hcaptcha

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/hcaptcha.*", "iris.hCaptcha")
}

// responseContextKey is the request's context key that keeps the response of a hcaptcha request.
// Read the value with Get.
const responseContextKey = "iris.hcaptcha"

var (
	// DefaultFailureHandler is the default HTTP handler that is fired on hcaptcha failures.
	// See `Options.FailureHandler`.
	DefaultFailureHandler = func(ctx *context.Context) {
		ctx.StopWithStatus(http.StatusTooManyRequests)
	}
)

// Options is the configuration for the hcaptcha middleware.
// The zero value is valid but useless: Secret is required.
type Options struct {
	// Secret is the hcaptcha secret key, from https://dashboard.hcaptcha.com/settings.
	// Required.
	Secret string `yaml:"Secret"`

	// FailureHandler if specified, fired when user does not complete hcaptcha successfully.
	// Failure and error codes information are kept as a `Response` value
	// in the request's context, retrieved through the `Get` package-level function.
	//
	// Defaults to DefaultFailureHandler, which writes a status code of
	// 429 (Too Many Requests) and no additional information.
	FailureHandler context.Handler `yaml:"-"`

	// RemoteIP is an optional check for siteverify:
	// the user's remote IP address, sent as the "remoteip" form field.
	//
	// Defaults to empty, not sent.
	RemoteIP string `yaml:"RemoteIP"`

	// SiteKey is an optional check for siteverify:
	// the sitekey form field you expect to see.
	//
	// Defaults to empty, not sent.
	SiteKey string `yaml:"SiteKey"`
}

// Response is the hcaptcha JSON response.
type Response struct {
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Success     bool     `json:"success"`
	Credit      bool     `json:"credit,omitzero"`
}

// New returns a route middleware Handler which verifies the hcaptcha response
// of each incoming request against the given Options.
//
// It calls the `SiteVerify` function and fires the "next" handler when the user
// completed the hcaptcha successfully, otherwise it calls Options.FailureHandler.
// The hcaptcha's `Response` (which contains any `ErrorCodes`) is saved on the
// Request's Context (see the `Get` package-level function).
//
// Instructions at: https://docs.hcaptcha.com/.
func New(opts Options) context.Handler {
	failureHandler := opts.FailureHandler
	if failureHandler == nil {
		failureHandler = DefaultFailureHandler
	}

	return func(ctx *context.Context) {
		v := SiteVerify(ctx, opts.Secret, opts.RemoteIP, opts.SiteKey)
		ctx.Values().Set(responseContextKey, v)
		if v.Success {
			ctx.Next()
			return
		}

		failureHandler(ctx)
	}
}

// responseFormValue = "h-captcha-response"
const apiURL = "https://hcaptcha.com/siteverify"

// VerifyTimeout bounds how long a single verification call may take.
// Set it before the first verification; the HTTP client is built once,
// on first use, and later assignments have no effect.
var VerifyTimeout = 10 * time.Second

// verifyClient is used instead of http.DefaultClient, which has no timeout.
// It is built on first use rather than at package init, so a VerifyTimeout
// assignment in the program's main is honored.
var verifyClient = sync.OnceValue(func() *http.Client {
	return &http.Client{Timeout: VerifyTimeout}
})

// SiteVerify accepts an Iris Context and a secret key (https://dashboard.hcaptcha.com/settings).
// It returns the hcaptcha's `Response`.
// The `response.Success` reports whether the validation passed.
// Any errors are passed through the `response.ErrorCodes` field.
//
// The remoteIP and siteKey input arguments are optional.
func SiteVerify(ctx *context.Context, secret, remoteIP, siteKey string) (response Response) {
	generatedResponseID := ctx.Form().Get("h-captcha-response")

	if generatedResponseID == "" {
		response.ErrorCodes = append(response.ErrorCodes,
			"form[h-captcha-response] is empty")
		return
	}

	values := url.Values{
		"secret":   {secret},
		"response": {generatedResponseID},
	}
	if remoteIP != "" {
		values.Add("remoteip", remoteIP)
	}
	if siteKey != "" {
		values.Add("sitekey", siteKey)
	}

	// Note(@kataras): a timeout and the request's context, not http.DefaultClient.
	//
	// DefaultClient has NO timeout. If hcaptcha.com is slow, or an on-path
	// attacker simply holds the connection open, every request goroutine that
	// reaches this captcha-protected route blocks indefinitely - connection and
	// goroutine exhaustion from a dependency the application does not control.
	// Threading the request context also makes a client disconnect cancel the
	// outbound call instead of leaving it running.
	req, err := http.NewRequestWithContext(ctx.Request().Context(), http.MethodPost, apiURL, strings.NewReader(values.Encode()))
	if err != nil {
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := verifyClient().Do(req)
	if err != nil {
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return
	}

	return
}

// Get returns the hcaptcha `Response` of the current request and reports whether was found or not.
func Get(ctx *context.Context) (Response, bool) {
	v := ctx.Values().Get(responseContextKey)
	if v != nil {
		if response, ok := v.(Response); ok {
			return response, true
		}
	}

	return Response{}, false
}

// Script is the hCaptcha's javascript source file that should be incldued in the HTML head or body.
const Script = "https://hcaptcha.com/1/api.js"

// HTMLForm is the default HTML form for clients.
// It's totally optional, use your own code for the best possible result depending on your web application.
// See `ParseForm` and `RenderForm` for more.
var HTMLForm = `<form action="%s" method="POST">
	    <script src="%s"></script>
		<div class="h-captcha" data-sitekey="%s"></div>
    	<input type="submit" name="button" value="OK">
</form>`

// ParseForm parses the `HTMLForm` with the necessary parameters and returns
// its result for render.
func ParseForm(dataSiteKey, postActionRelativePath string) string {
	return fmt.Sprintf(HTMLForm, postActionRelativePath, Script, dataSiteKey)
}

// RenderForm writes the `HTMLForm` to the Context's response writer.
// Use `ParseForm` or write your own HTML form for a custom result instead.
func RenderForm(ctx *context.Context, dataSiteKey, postActionRelativePath string) error {
	return ctx.HTML(ParseForm(dataSiteKey, postActionRelativePath))
}
