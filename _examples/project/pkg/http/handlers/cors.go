package handlers

import "github.com/kataras/iris/v12"

// CORS set ups a cors allow-all.
// We may need to edit it before deployment.
func CORS(allowedOrigin string) iris.Handler { // or "github.com/iris-contrib/middleware/cors"
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}

	return func(ctx iris.Context) {
		ctx.Header("Access-Control-Allow-Origin", allowedOrigin)
		ctx.Header("Access-Control-Allow-Credentials", "true")
		// July 2021 Mozzila updated the following document: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
		ctx.Header("Referrer-Policy", "no-referrer-when-downgrade")
		ctx.Header("Access-Control-Expose-Headers", "*, Authorization, X-Authorization")
		if ctx.Method() == iris.MethodOptions {
			ctx.Header("Access-Control-Allow-Methods", "*")
			ctx.Header("Access-Control-Allow-Headers", "*")
			ctx.Header("Access-Control-Max-Age", "86400")
			ctx.StatusCode(iris.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
