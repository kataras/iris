// This file carries the error codes of this package into the sort policy of
// Context.BindPagination and Context.Paginated.
package errors

import (
	"github.com/kataras/iris/v14/context"
)

// DenySort rejects any sort request with this error code and the given
// message. It is the context.DenySort pagination option answered with a
// chosen status instead of the default 400 Bad Request.
//
// Use it where sorting is a permission and not a shape: a list a customer may
// read but not order answers 403, and the client can tell the two apart.
//
// Usage:
//
//	list, filter, ok := ctx.BindPagination[customer.CheatFoodFilterOptions](
//		errors.PermissionDenied.DenySort("sort feature is not available for customers"))
//	if !ok {
//		return
//	}
func (e ErrorCodeName) DenySort(message string) context.PaginationOption {
	return context.DenySort(message, func(ctx *context.Context, msg string) {
		e.Message(ctx, msg)
	})
}
