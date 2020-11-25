// file: web/middleware/basicauth.go

package middleware

import "github.com/kataras/iris/v12/middleware/basicauth"

// BasicAuth middleware sample.
var BasicAuth = basicauth.Default(map[string]string{
	"admin": "password",
})
