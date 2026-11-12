package recaptcha_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/recaptcha"
)

// A failed verification used to answer an empty "200 OK" - no else branch, no
// StopWithStatus - which every client and every monitor reads as success.
//
// No network call happens here: an empty "g-recaptcha-response" form value is
// rejected by SiteVerify before it would contact Google.
func TestRecaptchaFailureDoesNotAnswerOK(t *testing.T) {
	app := iris.New()
	app.Post("/submit", recaptcha.New(recaptcha.Options{Secret: "test-secret"}), func(ctx iris.Context) {
		ctx.WriteString("protected")
	})

	e := httptest.New(t, app)

	resp := e.POST("/submit").Expect()
	if resp.Raw().StatusCode == httptest.StatusOK {
		t.Fatalf("a failed captcha answered 200 OK; body = %q", resp.Body().Raw())
	}
	resp.Body().NotEqual("protected")
}

// The provider's response is stored on the context so a FailureHandler can read
// why the verification failed.
func TestRecaptchaFailureResponseIsAvailable(t *testing.T) {
	var gotErrorCodes []string

	failureHandler := func(ctx iris.Context) {
		if response, ok := recaptcha.Get(ctx); ok {
			gotErrorCodes = response.ErrorCodes
		}
		ctx.StopWithStatus(iris.StatusTooManyRequests)
	}

	app := iris.New()
	app.Post("/submit", recaptcha.New(recaptcha.Options{
		Secret:         "test-secret",
		FailureHandler: failureHandler,
	}), func(ctx iris.Context) {
		ctx.WriteString("protected")
	})

	e := httptest.New(t, app)
	e.POST("/submit").Expect().Status(httptest.StatusTooManyRequests)

	if len(gotErrorCodes) == 0 {
		t.Fatal("expected the failure handler to receive the provider's error codes")
	}
}
