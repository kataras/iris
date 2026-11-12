package errors

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/versioning"
)

// DefaultVersioningNotFoundHandler is the default not-found handler for versioning,
// it renders the versioning.ErrNotFound as Unimplemented in JSON format.
var DefaultVersioningNotFoundHandler = func(ctx *context.Context) {
	Unimplemented.Err(ctx, versioning.ErrNotFound)
}
