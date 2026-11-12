package context

import (
	"reflect"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/core/pkg/pagination"
)

// ListFilter is an alias of pagination.ListFilter: the request half every
// list endpoint shares. Embed it in a service's own filter.
type ListFilter = pagination.ListFilter

// ErrSortNotAllowed is the error ListFilter.SortColumn wraps when the client
// asked to sort by a column the endpoint does not offer.
var ErrSortNotAllowed = pagination.ErrSortNotAllowed

// sortRequester is the interface the sort policy options check a decoded
// filter against. ListFilter implements it.
type sortRequester = pagination.SortRequester

// paginationOptions is the resolved option set of BindPagination and Paginated.
type paginationOptions struct {
	allowedSort []string
	denySort    bool
	denyMessage string
	denyRender  func(*Context, string)
}

// PaginationOption configures Context.BindPagination and Context.Paginated.
// See AllowSort and DenySort.
type PaginationOption func(*paginationOptions)

// AllowSort limits the sort column a client may ask for. A sort outside the
// list answers 400 Bad Request naming the column and the allowed ones, before
// the collector runs.
//
// It reads the decoded filter through its SortRequest method (ListFilter has
// one), so a filter without a sort request is unaffected.
func AllowSort(columns ...string) PaginationOption {
	return func(opts *paginationOptions) {
		opts.allowedSort = append(opts.allowedSort, columns...)
	}
}

// DenySort rejects any sort request with the given message. Use it on a list
// whose order is fixed by the query itself, where a client-chosen ORDER BY
// would either break the result or scan the table.
//
// The rejection is a 400 Bad Request by default. An optional "render" answers
// it instead, so a caller that needs another status can say so: package
// rest/errors builds one from an error code, as
// errors.PermissionDenied.DenySort("..."). Only the first one is used.
func DenySort(message string, render ...func(*Context, string)) PaginationOption {
	return func(opts *paginationOptions) {
		opts.denySort = true
		opts.denyMessage = message

		for _, r := range render {
			if r != nil {
				opts.denyRender = r
				break
			}
		}
	}
}

// resolvePaginationOptions applies the options in order and returns the
// resolved set.
func resolvePaginationOptions(opts []PaginationOption) paginationOptions {
	var options paginationOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return options
}

// checkSort renders the rejection of a sort request and reports whether the
// caller may continue. "filter" is the decoded value and "filterAddr" its
// address, so a SortRequest declared on a pointer receiver is policed too.
func (options paginationOptions) checkSort(ctx *Context, filter, filterAddr any) bool {
	if !options.denySort && len(options.allowedSort) == 0 {
		return true
	}

	requester, ok := filter.(sortRequester)
	if !ok {
		// A SortRequest with a pointer receiver is not in the value's method
		// set, so ask the addressable copy before giving up: a filter that
		// carries a sort request must not escape the policy over a receiver.
		if requester, ok = filterAddr.(sortRequester); !ok {
			return true
		}
	}

	if hasNilSortRequester(reflect.ValueOf(filter)) {
		// Nothing to answer the request: a nil Filter pointer, or a nil
		// embedded pointer promoting the method because the client left every
		// one of its members out of the body. That is "no sort requested",
		// never a panic in the handler.
		return true
	}

	column, _, requested := requester.SortRequest()
	if !requested {
		return true
	}

	if options.denySort {
		message := options.denyMessage
		if message == "" {
			message = "this list cannot be sorted"
		}

		if options.denyRender != nil {
			options.denyRender(ctx, message)
		} else {
			restErrorRenderer.HandleInvalidArgument(ctx, "%s", message)
		}

		return false
	}

	if !slices.Contains(options.allowedSort, column) {
		restErrorRenderer.HandleInvalidArgument(ctx,
			"invalid sort query value: %s: allowed: %s",
			column, strings.Join(options.allowedSort, ", "))
		return false
	}

	return true
}

// sortRequesterType is the interface the nil check looks for behind an
// embedded pointer.
var sortRequesterType = reflect.TypeFor[sortRequester]()

// maxSortRequesterDepth stops the embedded walk on a type that embeds itself.
const maxSortRequesterDepth = 10

// hasNilSortRequester reports whether the sort request method of "value" would
// be promoted from a nil pointer, so that calling it would panic.
func hasNilSortRequester(value reflect.Value) bool {
	return hasNilSortRequesterAt(value, 0)
}

func hasNilSortRequesterAt(value reflect.Value, depth int) bool {
	if depth > maxSortRequesterDepth {
		return false
	}

	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return true
		}

		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return false
	}

	typ := value.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.Anonymous {
			continue
		}

		switch field.Type.Kind() {
		case reflect.Pointer, reflect.Interface:
			if !field.Type.Implements(sortRequesterType) {
				continue
			}

			// Nil here, or nil one hop further down: an embedded pointer that
			// the body did allocate can still embed one that it did not.
			fieldValue := value.Field(i)
			if fieldValue.IsNil() || hasNilSortRequesterAt(fieldValue, depth+1) {
				return true
			}
		case reflect.Struct:
			// The method may be promoted from deeper in the embedding chain.
			if hasNilSortRequesterAt(value.Field(i), depth+1) {
				return true
			}
		}
	}

	return false
}
