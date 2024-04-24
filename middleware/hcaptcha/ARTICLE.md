# How to use hCAPTCHA with Iris

In this article, we will learn how to use hCAPTCHA with Iris, a web framework for Go that provides fast and easy development of web applications. hCAPTCHA is a service that protects websites from bots and spam by presenting challenges to human visitors. By using hCAPTCHA, we can ensure that only real users can access our website and prevent automated attacks.

## What is hCAPTCHA?

hCAPTCHA is a service that provides a widget that can be embedded in any web page. The widget displays a challenge that requires human intelligence to solve, such as identifying objects in images or typing words. The widget communicates with the hCAPTCHA server and verifies if the user has passed the challenge or not. If the user passes the challenge, the widget generates a token that can be used to validate the user's request on the server side.

hCAPTCHA is similar to reCAPTCHA, another popular service that offers similar functionality. However, hCAPTCHA claims to have some advantages over reCAPTCHA, such as:

- Better privacy: hCAPTCHA does not track users across websites or collect personal data.
- Better performance: hCAPTCHA uses less resources and loads faster than reCAPTCHA.
- Better rewards: hCAPTCHA pays website owners for using their service and supports various causes and charities.

## How to use hCAPTCHA with Iris?

To use hCAPTCHA with Iris, we need to do two things:

- Import the `github.com/kataras/iris/v12/middleware/hcaptcha` package, which provides an Iris middleware for hCAPTCHA.
- Use the `hcaptcha.New` function to create an `iris.Handler` and register it in our Iris app.

First, we need to install the Iris Web Framework, which contains the middleware too:

```sh
$ go get github.com/kataras/iris/v12@latest
```

Then, we need to import it in our main.go file:

```go
import (
    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/middleware/hcaptcha"
)
```

Next, we need to use the `hcaptcha.New` function to create an `iris.Handler`. The function takes two arguments: a secret key and, optionally, one or more functions to configure the hCAPTCHA Client instance. The secret key is a string that we can obtain from the hCAPTCHA website after creating an account and registering our website. We'll register an error handler through the second article. The error handler (`FailureHandler`) is a function that handles any errors that may occur during the validation process.

We can use the following code to create the handler:

```go
// Replace with your own secret key.
secret := "0x123456789abcdef"

// Create a hcaptcha middleware with the secret key and a custom error handler.
hcaptchaMiddleware := hcaptcha.New(secret, func(c *hcaptcha.Client) {
    c.FailureHandler = func(ctx iris.Context) {
        // Handle the error as you wish, for example:
        ctx.StopWithText(iris.StatusBadRequest, "hcaptcha verification failed: %v", err)
    }
})
```

The secret key can be found through: https://dashboard.hcaptcha.com. Please store it on a secure and private place. To test hCAPTCHA on a local environment please read the following instructions at: https://docs.hcaptcha.com/#localdev.

Finally, we need to register the hcaptcha middleware in our Iris app. We can use the `app.UseRouter` method to apply the middleware to all routes, or the `app.Use/UseGlobal` method to apply it to all non-error routes. For example:

```go
// Create an Iris app instance.
app := iris.New()

// Apply the hcaptcha middleware to all non-error routes.
app.Use(hcaptchaMiddleware)

// Or apply the hcaptcha middleware to specific routes.
app.Get("/protected", hcaptchaMiddleware, func(ctx iris.Context) {
    // This route is protected by hcaptcha.
    ctx.WriteString("Hello, human!")
})
```

## Conclusion

In this article, we have learned how to use basic hCAPTCHA middleware with Iris. For a more complete example please navigate through: https://github.com/kataras/iris/tree/main/_examples/auth/hcaptcha. By using these kind of features, we can create a more secure, user-friendly, and robust web application.

I hope you enjoyed this article and found it useful. If you have any questions or feedback, please feel free to leave a comment below. Thank you for reading! 😊
