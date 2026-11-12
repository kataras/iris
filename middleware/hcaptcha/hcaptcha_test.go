package hcaptcha_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/hcaptcha"
)

// A submission without an "h-captcha-response" form value is rejected by
// SiteVerify before it would contact hcaptcha.com, so no network call happens
// here: the middleware must stop the chain and run the failure handler.
func TestHCaptchaFailureStopsTheChain(t *testing.T) {
	app := iris.New()
	app.Post("/submit", hcaptcha.New(hcaptcha.Options{Secret: "test-secret"}), func(ctx iris.Context) {
		ctx.WriteString("protected")
	})

	e := httptest.New(t, app)

	e.POST("/submit").Expect().
		Status(httptest.StatusTooManyRequests).Body().NotEqual("protected")
}

// Options.FailureHandler replaces the default 429 and can read the provider's
// response, error codes included, from the request context.
func TestHCaptchaOptionsFailureHandler(t *testing.T) {
	var gotErrorCodes []string

	app := iris.New()
	app.Post("/submit", hcaptcha.New(hcaptcha.Options{
		Secret: "test-secret",
		FailureHandler: func(ctx iris.Context) {
			if response, ok := hcaptcha.Get(ctx); ok {
				gotErrorCodes = response.ErrorCodes
			}
			ctx.StopWithStatus(iris.StatusBadRequest)
		},
	}), func(ctx iris.Context) {
		ctx.WriteString("protected")
	})

	e := httptest.New(t, app)
	e.POST("/submit").Expect().Status(httptest.StatusBadRequest)

	if len(gotErrorCodes) == 0 {
		t.Fatal("expected the failure handler to receive the provider's error codes")
	}
}
