package recaptcha

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/netutil"
)

const (
	responseFormValue = "g-recaptcha-response"
	apiURL            = "https://www.google.com/recaptcha/api/siteverify"
)

// Response is the google's recaptcha response as JSON.
type Response struct {
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
	Success     bool      `json:"success"`
}

// Client is the default `net/http#Client` instance which
// is used to send requests to the Google API.
//
// Change Client only if you know what you're doing.
var Client = netutil.Client(time.Duration(20 * time.Second))

// New accepts the google's recaptcha secret key and returns
// a middleware that verifies the request by sending a response to the google's API(V2-latest).
// Secret key can be obtained by https://www.google.com/recaptcha.
//
// Used for communication between your site and Google. Be sure to keep it a secret.
//
// Use `SiteVerify` to verify a request inside another handler if needed.
func New(secret string) context.Handler {
	return func(ctx context.Context) {
		if SiteFerify(ctx, secret).Success {
			ctx.Next()
		}
	}
}

// SiteFerify accepts  context and the secret key(https://www.google.com/recaptcha)
//  and returns the google's recaptcha response, if `response.Success` is true
// then validation passed.
//
// Use `New` for middleware use instead.
func SiteFerify(ctx context.Context, secret string) (response Response) {
	generatedResponseID := ctx.FormValue("g-recaptcha-response")
	if generatedResponseID == "" {
		response.ErrorCodes = append(response.ErrorCodes,
			"form value[g-recaptcha-response] is empty")
		return
	}

	r, err := Client.PostForm(apiURL,
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

	body, err := ioutil.ReadAll(r.Body)
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
// func postContact(ctx context.Context) {
// 	// [...]
// 	response := recaptcha.SiteFerify(ctx, recaptchaSecret)
//
// 	if response.Success {
// 		// [your action here, i.e sendEmail(...)]
// 	}
//
// 	ctx.JSON(response)
// }
//
// Method: "GET" | Path: "/contact"
// func getContact(ctx context.Context) {
// 	// render the recaptcha form
// 	ctx.HTML(recaptcha.GetFormHTML(recaptchaPublic, "/contact"))
// }
func GetFormHTML(dataSiteKey string, postActionRelativePath string) string {
	return fmt.Sprintf(recaptchaForm, postActionRelativePath, dataSiteKey)
}
