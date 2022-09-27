# Iris Handler with Generics support

```go
package x

import (
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/x/errors"
)

var ErrorHandler context.ErrorHandler = context.ErrorHandlerFunc(errors.InvalidArgument.Err)

type (
	Handler[Request any | *context.Context, Response any] func(Request) (Response, error)
	HandlerWithCtx[Request any, Response any]             func(*context.Context, Request) (Response, error)
)

func HandleContext[Request any, Response any](handler HandlerWithCtx[Request, Response]) context.Handler {
	return func(ctx *context.Context) {
		var req Request
		if err := ctx.ReadJSON(&req); err != nil {
			errors.InvalidArgument.Details(ctx, "unable to parse body", err.Error())
			return
		}

		resp, err := handler(ctx, req)
		if err != nil {
			ErrorHandler.HandleContextError(ctx, err)
			return
		}

		if _, err = ctx.JSON(resp); err != nil {
			errors.Internal.Details(ctx, "unable to parse response", err.Error())
			return
		}
	}
}

func Handle[Request any, Response any](handler Handler[Request, Response]) context.Handler {
	return HandleContext(func(_ *context.Context, req Request) (Response, error) { return handler(req) })
}

```

Usage Code:

```go
import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x"
)

type (
	Req struct {
		Email string `json:"email"`
	}

	Res struct {
		Verified bool `json:"verified"`
	}
)

func main() {
	app := iris.New()
	app.Post("/", x.Handle(handler))
	app.Listen(":8080")
}

func handler(req Req) (Res, error){
	verified := req.Email == "iris-go@outlook.com"
	return Res{Verified: verified}, nil
}
```

Example response:

```json
{
    "verified": true
}
```
