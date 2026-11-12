package versioning

import (
	"strings"

	"github.com/kataras/iris/v14/context"
)

// Skipper is a function which reports whether the versioning middleware should run or not on specific requests.
// It can be used to skip the versioning middleware on specific requests (e.g. by path prefix).
// If it returns true then the versioning middleware must not run.
// Usage:
//
//	versioning.Options{
//		Skipper: func(ctx iris.Context) bool {
//			return !strings.HasPrefix(ctx.Path(), "/api")
//		},
//	}
//
// See `OnlyPath` and `OnlyAPI` for ready-to-use Skipper functions.
type Skipper = func(ctx *context.Context) bool

var (
	// OnlyAPI is a helper function which can be used as a Skipper to run the versioning middleware only on specific requests by the "/api" path prefix.
	// Usage:
	//
	//	versioning.Options{
	//		Skipper: versioning.OnlyAPI,
	//	    [...]
	//	}
	OnlyAPI Skipper = OnlyPath("/api")
	// OnlyPath is a helper function which can be used as a Skipper to run the versioning middleware only on specific requests by the path prefix.
	// Usage:
	//
	//	versioning.Options{
	//		Skipper: versioning.OnlyPath("/api"),
	//		[...]
	OnlyPath = func(pathPrefix string) Skipper {
		return func(ctx *context.Context) bool {
			path := ctx.Request().URL.Path

			// Allow versioning middleware to only /pathPrefix requests and trim the /pathPrefix prefix, so for example /api/v1 redirects to the /v1.
			if after, ok := strings.CutPrefix(path, pathPrefix); ok {
				ctx.Request().URL.Path = after
				return false
			}

			return true
			// return !strings.HasPrefix(ctx.Path(), pathPrefix)
		}
	}
)
