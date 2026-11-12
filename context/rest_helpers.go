package context

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"uuid"
)

type (
	// IDType is a type which can be used as an unique identifier for a resource.
	// It can be a string or int or int64 or uint64 or the standard library's uuid.UUID.
	// Created wraps a response value of one of these types as an IDPayload.
	IDType interface {
		string | int | int64 | uint64 | uuid.UUID
	}

	// IDPayload is a simple struct which describes a json id value:
	// the {"id": ...} body Created renders for a bare identifier.
	IDPayload[T IDType] struct {
		ID T `json:"id"`
	}
)

// HandleError renders "err" through the error renderer (see
// rest/errors.HandleError) and reports whether it did, so a step that has an
// error but nothing to render on success is one guarded line:
//
//	func(ctx iris.Context) {
//		if ctx.HandleError(svc.DeactivatePlans(ctx, id)) {
//			return
//		}
//		// [... the next step]
//	}
//
// A nil error writes nothing and reports false, so the handler continues.
// An unrecognized error renders as 500; the mappings are the application's,
// declared through rest/errors.Options.
//
// It is the error half that OK, Created, NoContent, List, Count and Paginated
// each run before rendering their success body, on its own for the steps that
// have no such body.
func (ctx *Context) HandleError(err error) bool {
	return restErrorRenderer.HandleError(ctx, err)
}

// OK handles a generic response and error from a service call and sends a
// JSON response to the client. If the error is not nil, it renders it through
// the error renderer (see rest/errors.HandleError) and returns false. It
// leaves the status code untouched (defaults to 200 OK). A nil "resp" writes
// no body. It returns a boolean value indicating whether the handle was
// successful or not.
//
// Usage:
//
//	func(ctx iris.Context) {
//		users, err := svc.ListUsers(ctx)
//		ctx.OK(users, err)
//	}
func (ctx *Context) OK(resp any, err error) bool {
	if restErrorRenderer.HandleError(ctx, err) {
		return false
	}

	if resp != nil {
		return ctx.JSONOrError(resp)
	}

	return true
}

// Created handles a generic response and error from a service call and sends
// a 201 JSON response to the client. If the error is not nil, it renders it
// through the error renderer and returns false.
//
// A bare identifier is wrapped as an IDPayload: a string, int, int64 or
// uint64 value renders as {"id": value} and a uuid.UUID as {"id": "..."}
// (its string form). Any other value, a struct, a map or a slice, renders as
// it is; a []string is a JSON array of strings, not a list of id payloads.
// A nil "resp" writes no body.
//
// It returns a boolean value indicating whether the handle was successful or not.
//
// Usage:
//
//	func(ctx iris.Context) {
//		id, err := svc.CreateUser(ctx, input)
//		ctx.Created(id, err) // 201 {"id": ...}
//	}
func (ctx *Context) Created(resp any, err error) bool {
	if restErrorRenderer.HandleError(ctx, err) { // an error written to the client.
		return false
	}

	ctx.StatusCode(http.StatusCreated)

	if resp == nil {
		return true
	}

	switch value := resp.(type) { // it's an ID, let's set a JSON ID payload from it.
	case string:
		resp = IDPayload[string]{ID: value}
	case int:
		resp = IDPayload[int]{ID: value}
	case int64:
		resp = IDPayload[int64]{ID: value}
	case uint64:
		resp = IDPayload[uint64]{ID: value}
	case uuid.UUID:
		resp = IDPayload[string]{ID: value.String()}
	}

	return ctx.JSONOrError(resp)
}

// NoContent handles an error from a service call and sends an empty response
// to the client. If the error is not nil, it renders it through the error
// renderer and returns false.
//
// A true "modified" sends 204 No Content: the delete or the update changed
// something. A false one sends 304 Not Modified, with no body: the call
// succeeded but nothing changed. Both report true.
//
// It returns a boolean value indicating whether the handle was successful or not.
//
// Usage:
//
//	func(ctx iris.Context) {
//		deleted, err := svc.DeleteUser(ctx, id)
//		ctx.NoContent(deleted, err) // 204, 304, or the mapped error.
//	}
func (ctx *Context) NoContent(modified bool, err error) bool {
	if restErrorRenderer.HandleError(ctx, err) {
		return false
	}

	if modified {
		ctx.StatusCode(http.StatusNoContent)
	} else {
		ctx.StatusCode(http.StatusNotModified)
	}

	return true
}

// emptyJSONArray renders as [] whatever the element type of the list was.
var emptyJSONArray = []any{}

// List renders "items" as a 200 JSON array. A nil or empty slice renders [] and
// never null, so a client can always iterate the answer.
//
// An error that matches one of "tolerate" (errors.Is) also renders [] with 200:
// a repository's not-found for an empty set is not a failure of the list
// endpoint. Any other error goes through the error renderer and List returns
// false, so the caller stops the handler with a `return`.
//
// A non-slice, non-array "items" is a programmer error and panics naming the
// type.
//
// Usage:
//
//	func(ctx iris.Context) {
//		users, err := svc.ListUsers(ctx)
//		ctx.List(users, err, sql.ErrNoRows)
//	}
func (ctx *Context) List(items any, err error, tolerate ...error) bool {
	if err != nil && !isToleratedError(err, tolerate) {
		restErrorRenderer.HandleError(ctx, err)
		return false
	}

	if items == nil {
		return ctx.JSONOrError(emptyJSONArray)
	}

	value := reflect.ValueOf(items)
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		if err != nil || value.Len() == 0 {
			// A tolerated error means "no items", whatever came back with it.
			return ctx.JSONOrError(emptyJSONArray)
		}
	default:
		panic(fmt.Sprintf("iris: Context.List: items of type %T is not a slice or an array", items))
	}

	return ctx.JSONOrError(items)
}

// isToleratedError reports whether "err" matches any of the sentinels the
// caller of List accepts as an empty list.
func isToleratedError(err error, tolerate []error) bool {
	for _, target := range tolerate {
		if target != nil && errors.Is(err, target) {
			return true
		}
	}

	return false
}

// JSONOrError sends "resp" as a JSON response to the client and, when
// encoding or writing it fails, renders that failure through the error
// renderer as a server error (see rest/errors.HandleResponseJSONError).
// It returns a boolean value indicating whether the handle was successful or
// not. If it's not successful, an error is written to the client and the
// caller should stop the execution flow with a `return` statement.
//
// It is the response half of OK, Created, Count and Paginated; use it when the
// status code or the value needs preparing first.
//
// Usage:
//
//	func(ctx iris.Context) {
//		ctx.StatusCode(iris.StatusAccepted)
//		if !ctx.JSONOrError(job) {
//			return
//		}
//		// [... do other jobs here, e.g. run a go routine]
//	}
func (ctx *Context) JSONOrError(resp any) bool {
	return writeJSONChecked(ctx, resp)
}
