# hCAPTCHA

hCAPTCHA protects a website from bots and spam by putting a challenge in front of the visitor. This middleware verifies that challenge on the server side, so a request that did not pass it never reaches your handler.

## What is hCAPTCHA?

hCAPTCHA is a service that provides a widget that can be embedded in any web page. The widget displays a challenge that requires human intelligence to solve, such as identifying objects in images or typing words. The widget communicates with the hCAPTCHA server and verifies if the user has passed the challenge or not. If the user passes the challenge, the widget generates a token that can be used to validate the user's request on the server side.

hCAPTCHA is similar to reCAPTCHA, another popular service that offers similar functionality. However, hCAPTCHA claims to have some advantages over reCAPTCHA, such as:

- Better privacy: hCAPTCHA does not track users across websites or collect personal data.
- Better performance: hCAPTCHA uses less resources and loads faster than reCAPTCHA.
- Better rewards: hCAPTCHA pays website owners for using their service and supports various causes and charities.

## How to use hCAPTCHA with Iris?

To use hCAPTCHA with Iris, we need to do two things:

- Import the `github.com/kataras/iris/v14/middleware/hcaptcha` package, which provides an Iris middleware for hCAPTCHA.
- Use the `hcaptcha.New` function to create an `iris.Handler` and register it in our Iris app.

First, we need to install the Iris Web Framework, which contains the middleware too:

```sh
go get github.com/kataras/iris/v14@latest
```

Then, we need to import it in our main.go file:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/hcaptcha"
    "github.com/kataras/iris/v14/rest/errors"
)
```

`hcaptcha.New` takes an `hcaptcha.Options` value. `Secret` is required; you get it from the hCAPTCHA dashboard after registering your site. `FailureHandler` runs when verification does not pass; the default replies with 429. `RemoteIP` and `SiteKey` are optional extra checks sent to siteverify.

We can use the following code to create the handler:

```go
// Create a hcaptcha middleware with the secret key and a custom failure handler.
hcaptchaMiddleware := hcaptcha.New(hcaptcha.Options{
    // Replace with your own secret key.
    Secret: "0x123456789abcdef",
    FailureHandler: func(ctx iris.Context) {
        // The verification response, including any error codes,
        // is available from the request context.
        resp, _ := hcaptcha.Get(ctx)
        // The error code carries the 400 and stops the chain.
        errors.InvalidArgument.Detailf(ctx, "hcaptcha verification failed",
            "%v", resp.ErrorCodes)
    },
})
```

The secret key can be found through: https://dashboard.hcaptcha.com. Please store it on a secure and private place. To test hCAPTCHA on a local environment please read the following instructions at: https://docs.hcaptcha.com/#localdev.

Finally, we need to register the hcaptcha middleware in our Iris app. We can use the `app.UseRouter` and `app.Use` method to apply the middleware on every route. For example:

```go
// Create an Iris app instance.
app := iris.New()

// Apply the hcaptcha middleware.
app.Use(hcaptchaMiddleware)

// Or apply the hcaptcha middleware to specific routes.
app.Get("/protected", hcaptchaMiddleware, func(ctx iris.Context) {
    // This route is protected by hcaptcha.
    ctx.WriteString("Hello, human!")
})
```
