package hcaptcha

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("iris/middleware/hcaptcha.*", "iris.hCaptcha")
}

var (
	// ResponseContextKey is the default request's context key that response of a hcaptcha request is kept.
	ResponseContextKey string = "iris.hcaptcha"
	// DefaultFailureHandler is the default HTTP handler that is fired on hcaptcha failures.
	// See `Client.FailureHandler`.
	DefaultFailureHandler = func(ctx *context.Context) {
		ctx.StopWithStatus(http.StatusTooManyRequests)
	}
)

// Client represents the hcaptcha client.
type Client struct {
	// FailureHandler if specified, fired when user does not complete hcaptcha successfully.
	// Failure and error codes information are kept as `Response` type
	// at the Request's Context key of "hcaptcha".
	//
	// Defaults to a handler that writes a status code of 429 (Too Many Requests)
	// and without additional information.
	FailureHandler context.Handler

	secret string
}

// Option declares an option for the hcaptcha client.
// See `New` package-level function.
type Option func(*Client)

// Response is the hcaptcha JSON response.
type Response struct {
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Success     bool     `json:"success"`
	Credit      bool     `json:"credit,omitempty"`
}

// New accepts a hpcatcha secret key and returns a new hcaptcha HTTP Client.
//
// Instructions at: https://docs.hcaptcha.com/.
//
// See its `Handler` and `SiteVerify` for details.
func New(secret string, options ...Option) context.Handler {
	c := &Client{
		FailureHandler: DefaultFailureHandler,
		secret:         secret,
	}

	for _, opt := range options {
		opt(c)
	}

	return c.Handler
}

// Handler is the HTTP route middleware featured hcaptcha validation.
// It calls the `SiteVerify` method and fires the "next" when user completed the hcaptcha successfully,
//  otherwise it calls the Client's `FailureHandler`.
// The hcaptcha's `Response` (which contains any `ErrorCodes`)
// is saved on the Request's Context (see `GetResponseFromContext`).
func (c *Client) Handler(ctx *context.Context) {
	v := SiteVerify(ctx, c.secret)
	ctx.Values().Set(ResponseContextKey, v)
	if v.Success {
		ctx.Next()
		return
	}

	if c.FailureHandler != nil {
		c.FailureHandler(ctx)
	}
}

// responseFormValue = "h-captcha-response"
const apiURL = "https://hcaptcha.com/siteverify"

// SiteVerify accepts an Iris Context and a secret key (https://dashboard.hcaptcha.com/settings).
// It returns the hcaptcha's `Response`.
// The `response.Success` reports whether the validation passed.
// Any errors are passed through the `response.ErrorCodes` field.
func SiteVerify(ctx *context.Context, secret string) (response Response) {
	generatedResponseID := ctx.FormValue("h-captcha-response")

	if generatedResponseID == "" {
		response.ErrorCodes = append(response.ErrorCodes,
			"form[h-captcha-response] is empty")
		return
	}

	resp, err := http.DefaultClient.PostForm(apiURL,
		url.Values{
			"secret":   {secret},
			"response": {generatedResponseID},
		},
	)
	if err != nil {
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
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
	v := ctx.Values().Get(ResponseContextKey)
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

// RenderForm writes the `HTMLForm` to "w" response writer.
// See `_examples/auth/hcaptcha/templates/register_form.html` example for a custom form instead.
func RenderForm(ctx *context.Context, dataSiteKey, postActionRelativePath string) (int, error) {
	return ctx.HTML(ParseForm(dataSiteKey, postActionRelativePath))
}
