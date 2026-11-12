# reCAPTCHA

reCAPTCHA protects a website from bots and spam by putting a challenge in front of the visitor. This middleware verifies that challenge on the server side, so a request that did not pass it never reaches your handler.

## What is reCAPTCHA?

reCAPTCHA is a service that provides a widget that can be embedded in any web page. The widget displays a challenge that requires human intelligence to solve, such as identifying objects in images or typing words. The widget communicates with the reCAPTCHA server and verifies if the user has passed the challenge or not. If the user passes the challenge, the widget generates a token that can be used to validate the user's request on the server side.

reCAPTCHA is widely used to protect websites from spam and abuse. It is developed by Google and is available in two versions: reCAPTCHA v2 and reCAPTCHA v3. reCAPTCHA v2 presents a challenge to the user, while reCAPTCHA v3 provides a score based on the user's interactions with the site.

For more information, you can refer to the [reCAPTCHA Wikipedia page](https://en.wikipedia.org/wiki/ReCAPTCHA) and the [official reCAPTCHA documentation](https://developers.google.com/recaptcha).

## How to use reCAPTCHA with Iris?

To use reCAPTCHA with Iris, we need to do two things:

- Import the `github.com/kataras/iris/v14/middleware/recaptcha` package, which provides an Iris middleware for reCAPTCHA.
- Use the `recaptcha.New` function to create an `iris.Handler` and register it in our Iris app.

First, we need to install the Iris Web Framework, which contains the middleware too:

```sh
go get github.com/kataras/iris/v14@latest
```

Then, we need to import it in our `main.go` file:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/recaptcha"
)
```

Next, we need to use the `recaptcha.New` function to create an `iris.Handler`. The function takes a `recaptcha.Options` value. `Secret` is required: a string that we can obtain from the reCAPTCHA website after creating an account and registering our website. `FailureHandler` is optional and runs when verification does not pass; the default replies with 429.

We can use the following code to create the handler:

```go
// Create a recaptcha middleware with the secret key.
recaptchaMiddleware := recaptcha.New(recaptcha.Options{
    // Replace with your own secret key.
    Secret: "6Lf3WywUAAAAAJpArb8nW_LCL_PuPuokmEABFfgw",
})
```

The secret key can be found through: https://www.google.com/recaptcha. Please store it in a secure and private place.

Finally, we need to register the recaptcha middleware in our Iris app. We can use the `app.UseRouter` and `app.Use` method to apply the middleware on every route. For example:

```go
// Create an Iris app instance.
app := iris.New()

// Apply the recaptcha middleware.
app.Use(recaptchaMiddleware)

// Or apply the recaptcha middleware to specific routes.
app.Get("/protected", recaptchaMiddleware, func(ctx iris.Context) {
    // This route is protected by recaptcha.
    ctx.WriteString("Hello, human!")
})
```

## Example

Here is a complete example of how to use reCAPTCHA with Iris:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/recaptcha"
)

// keys should be obtained by https://www.google.com/recaptcha
const (
    recaptchaPublic = "6Lf3WywUAAAAAKNfAm5DP2J5ahqedtZdHTYaKkJ6"
    recaptchaSecret = "6Lf3WywUAAAAAJpArb8nW_LCL_PuPuokmEABFfgw"
)

func showRecaptchaForm(ctx iris.Context, path string) {
    ctx.HTML(recaptcha.GetFormHTML(recaptchaPublic, path))
}

func main() {
    app := iris.New()

    // On both Get and Post on this example, so you can easily
    // use a single route to show a form and the main subject if recaptcha's validation result succeed.
    app.Handle("GET POST", "/", func(ctx iris.Context) {
        if ctx.Method() == iris.MethodGet {
            showRecaptchaForm(ctx, "/")
            return
        }

        result := recaptcha.SiteVerify(ctx, recaptchaSecret)
        if !result.Success {
            /* redirect here if you want or do nothing */
            ctx.HTML("<b> failed please try again </b>")
            return
        }

        ctx.Writef("succeed.")
    })

    app.Listen(":8080")
}
```
