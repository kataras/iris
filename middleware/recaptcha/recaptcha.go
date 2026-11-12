// Package recaptcha verifies the Google reCAPTCHA answer that a form submission
// carries.
//
// Its entry point is New(Options), which returns a Handler that calls SiteVerify
// and runs Options.FailureHandler when the answer does not pass. Read the stored
// Response with Get.
package recaptcha

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/kataras/iris/v14/client"
	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/recaptcha.*", "iris.reCAPTCHA")
}

const (
	// responseFormValue = "g-recaptcha-response"
	apiURL = "https://www.google.com/recaptcha/api/siteverify"
)

// Response is the google's recaptcha response as JSON.
type Response struct {
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Success     bool      `json:"success"`
}

// Client is the default `github.com/kataras/iris/v14/client#Client` instance which
// is used to send requests to the Google API. It is created with a 20s dial timeout.
//
// Change Client only if you know what you're doing.
var Client = client.New(client.DialTimeout(20 * time.Second))

// Options is the configuration for the recaptcha middleware.
// The zero value is valid but useless: Secret is required.
type Options struct {
	// Secret is the google's recaptcha secret key, obtained by
	// https://www.google.com/recaptcha. Used for communication between your
	// site and Google. Be sure to keep it a secret.
	// Required.
	Secret string `yaml:"Secret"`

	// FailureHandler, when set, is called instead of the default 429
	// (Too Many Requests) response after a failed verification.
	// Read the reason with `Get`.
	//
	// Defaults to DefaultFailureHandler.
	FailureHandler context.Handler `yaml:"-"`
}

// DefaultFailureHandler is the default handler that is fired on recaptcha
// failures. It stops the chain with 429 (Too Many Requests).
// See `Options.FailureHandler`.
var DefaultFailureHandler = func(ctx *context.Context) {
	ctx.StopWithStatus(http.StatusTooManyRequests)
}

// New returns a middleware that verifies the request by sending a response to
// the google's API(V2-latest), using the given Options.
//
// Use `SiteVerify` to verify a request inside another handler if needed.
// A failed verification stops the chain with 429 (Too Many Requests) and stores
// the provider's response on the context, retrievable with `Get`.
//
// Note(@kataras): the failure branch is not optional. This used to have no else
// at all, so a failed captcha answered an empty "200 OK" - which every client
// and every monitor reads as success - and discarded the reason in ErrorCodes.
func New(opts Options) context.Handler {
	failureHandler := opts.FailureHandler
	if failureHandler == nil {
		failureHandler = DefaultFailureHandler
	}

	return func(ctx *context.Context) {
		response := SiteVerify(ctx, opts.Secret)
		if response.Success {
			ctx.Next()
			return
		}

		ctx.Values().Set(responseContextKey, response)
		failureHandler(ctx)
	}
}

const responseContextKey = "iris.recaptcha.response"

// Get returns the verification response of the current request, as stored by the
// middleware returned from `New`.
func Get(ctx *context.Context) (Response, bool) {
	v, ok := ctx.Values().Get(responseContextKey).(Response)
	return v, ok
}

// SiteVerify accepts  context and the secret key(https://www.google.com/recaptcha) and
// returns the google's recaptcha response, if `response.Success` is true then validation passed.
//
// Use `New` for middleware use instead.
func SiteVerify(ctx *context.Context, secret string) (response Response) {
	generatedResponseID := ctx.Form().Get("g-recaptcha-response")
	if generatedResponseID == "" {
		response.ErrorCodes = append(response.ErrorCodes,
			"form value[g-recaptcha-response] is empty")
		return
	}

	r, err := Client.Form(ctx, "POST", apiURL,
		url.Values{
			"secret":   {secret},
			"response": {generatedResponseID},
			// optional: let's no track our users "remoteip": {ctx.RemoteAddr()},
		},
	)
	if err != nil {
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		response.ErrorCodes = append(response.ErrorCodes, err.Error())
		return
	}

	return response
}

var recaptchaForm = `<form action="%s" method="POST">
	    <script src="https://www.google.com/recaptcha/api.js"></script>
		<div class="g-recaptcha" data-sitekey="%s"></div>
    	<input type="submit" name="button" value="OK">
</form>`

// GetFormHTML can be used on pages where empty form
// is enough to verify that the client is not a "robot".
// i.e: GetFormHTML("public_key", "/contact")
// will return form tag which imports the google API script,
// with a simple submit button where redirects to the
// "postActionRelativePath".
//
// The "postActionRelativePath" MUST use the `SiteVerify` or
// followed by the `New()`'s context.Handler  (with the secret key this time)
// in order to validate if the recaptcha verified.
//
// The majority of applications will use a custom form,
// this function is here for ridiculous simple use cases.
//
// Example Code:
//
// Method: "POST" | Path: "/contact"
//
//	func postContact(ctx *context.Context) {
//		// [...]
//		response := recaptcha.SiteVerify(ctx, recaptchaSecret)
//
//		if response.Success {
//			// [your action here, i.e sendEmail(...)]
//		}
//
//		ctx.JSON(response)
//	}
//
// Method: "GET" | Path: "/contact"
//
//	func getContact(ctx *context.Context) {
//		// render the recaptcha form
//		ctx.HTML(recaptcha.GetFormHTML(recaptchaPublic, "/contact"))
//	}
func GetFormHTML(dataSiteKey string, postActionRelativePath string) string {
	return fmt.Sprintf(recaptchaForm, postActionRelativePath, dataSiteKey)
}
