package jwt

import "github.com/kataras/iris/v12/context"

func init() {
	context.SetHandlerName("iris/middleware/jwt.*", "iris.jwt")
}
