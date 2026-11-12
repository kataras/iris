package errors

import (
	"github.com/kataras/iris/v14/context"
)

// NotFoundID renders NOT_FOUND with the detail "<resource> <id> not found",
// so the whole "this row is not there" answer is one line:
//
//	func getPlan(ctx iris.Context) {
//		plan, err := svc.GetPlan(ctx, id)
//		if errors.HandleError(ctx, err) {
//			return
//		}
//
//		if plan == nil {
//			errors.NotFoundID(ctx, "plan", id)
//			return
//		}
//
//		ctx.OK(plan, nil)
//	}
//
// The result follows the Context.OK convention, true meaning the handler
// succeeded, so it is always false here; return it directly only from a helper
// which itself reports that. Note that HandleError in this package reports the
// opposite, true when it wrote the response, which is why the example above
// tests it and ignores this one.
//
// The id is formatted with %v, so a string, an integer and a uuid.UUID all
// read the way they print.
func NotFoundID(ctx *context.Context, resource string, id any) bool {
	NotFound.Detailf(ctx, "resource not found", "%s %v not found", resource, id)
	return false
}
