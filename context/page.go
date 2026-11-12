package context

import (
	stdContext "context"
	"reflect"

	"github.com/kataras/iris/v14/core/pkg/pagination"
)

// The data types of a paginated list live in core/pkg/pagination, which
// imports only the standard library so the client package can share them.
// The page size bounds are pagination.MaxPageSize and pagination.DefaultPageSize.
type (
	// PageOptions is an alias of pagination.PageOptions: the page, size and
	// details a client sends in the URL query.
	PageOptions = pagination.PageOptions
	// Page is an alias of pagination.Page: the response envelope of a
	// paginated list.
	Page[T any] = pagination.Page[T]
	// CountType is an alias of pagination.CountType.
	CountType = pagination.CountType
	// CountResponse is an alias of pagination.CountResponse.
	CountResponse[N CountType] = pagination.CountResponse[N]
)

// NewPage forwards to pagination.NewPage.
func NewPage[T any](opts PageOptions, filter any, items []T, totalCount int64) *Page[T] {
	return pagination.NewPage(opts, filter, items, totalCount)
}

// CountToInt64 forwards to pagination.CountToInt64.
func CountToInt64[N CountType](n N) int64 {
	return pagination.CountToInt64(n)
}

// BindPagination reads the PageOptions from the URL Query and
// any filter options of generic Filter type from the request body.
//
// The options police the sort request the filter carries, see AllowSort and
// DenySort. A rejected sort answers 400 Bad Request and BindPagination
// reports false.
func (ctx *Context) BindPagination[Filter any](opts ...PaginationOption) (PageOptions, Filter, bool) {
	var page PageOptions
	if err := ctx.ReadQuery(&page); err != nil {
		restErrorRenderer.HandleRequestError(ctx, err)

		var t Filter
		return page, t, false
	}

	filter, ok := ctx.RequestValue[Filter]() // try to read the filter from the options context decoder, if any.
	if !ok {
		filter, ok = bindJSON[Filter](ctx, false, true)
	}

	if !ok {
		var t Filter
		return page, t, false
	}

	filter = allocateFilter(filter)

	if !resolvePaginationOptions(opts).checkSort(ctx, filter, &filter) {
		var t Filter
		return page, t, false
	}

	return page, filter, true
}

// allocateFilter replaces a nil pointer filter with a pointer to its zero
// value.
//
// A repository method takes its filter by pointer, so Paginated infers Filter
// as *T and an absent body decodes to a nil *T. Handing that to the collector
// is a nil dereference the first time the repository reads a field, and it
// hides the sort policy too: a nil filter has nothing to answer SortRequest
// with, so AllowSort and DenySort would silently pass. An empty filter is the
// honest reading of an absent body, and it requests no sort.
func allocateFilter[Filter any](filter Filter) Filter {
	filterType := reflect.TypeFor[Filter]()
	if filterType.Kind() != reflect.Pointer {
		return filter
	}

	if value := reflect.ValueOf(filter); value.IsValid() && !value.IsNil() {
		return filter
	}

	allocated, ok := reflect.New(filterType.Elem()).Interface().(Filter)
	if !ok { // a pointer to Filter's element is a Filter; keep the nil rather than panic.
		return filter
	}

	return allocated
}

type (
	// ItemsCollectorFunc is a function that collects items and returns them along with the total count.
	// It's used by the Context.Paginated method to collect items and return them in a paginated response.
	ItemsCollectorFunc[Filter any, Item any, TotalCount CountType] func(stdContext.Context, PageOptions, Filter) ([]Item, TotalCount, error)
)

// Count handles a count response and error from a service call and sends a JSON `CountResponse` to the client.
// If the error is not nil, it sends an appropriate error response to the client.
// A zero count is a valid 200 answer, `{"count":0}`; a negative count renders
// as zero. (304 Not Modified is a conditional-request status and net/http
// refuses to send a body with it.)
// It returns a boolean value indicating whether the handle was successful or not.
func (ctx *Context) Count[N CountType](rowsAffected N, err error) bool {
	if restErrorRenderer.HandleError(ctx, err) {
		return false
	}

	if rowsAffected < 0 {
		rowsAffected = 0
	}

	return writeJSONChecked(ctx, CountResponse[N]{Count: rowsAffected})
}

// writeJSONChecked renders v as the success response, routing a marshal or
// write failure through the renderer as a server-side JSON error.
func writeJSONChecked(ctx *Context, v any) bool {
	return !restErrorRenderer.HandleResponseJSONError(ctx, ctx.JSON(v))
}

// Paginated is a helper method that reads pagination options from the request context,
// calls the given collect function to collect items and returns them in a paginated response.
// It returns a boolean value indicating whether the handle was successful or not.
// If it's not successful, an error is written to the client and
// the caller should stop the handler's execution flow with a `return` statement.
//
// Usage:
//
//	ctx.Paginated(func(_ context.Context, pageOpts iris.PageOptions, filter food.Filter) ([]food.Food, int64, error) {
//		items, totalCount, err := api.foodService.ListFoodsPaginated(ctx, pageOpts, filter, customerID)
//		return items, totalCount, err
//	}, iris.AllowSort("name", "created_at"))
func (ctx *Context) Paginated[Filter any, Item any, TotalCount CountType](collect ItemsCollectorFunc[Filter, Item, TotalCount], opts ...PaginationOption) bool {
	pageOpts, filter, ok := ctx.BindPagination[Filter](opts...)
	if !ok {
		return false
	}

	items, totalCount, err := collect(ctx, pageOpts, filter)
	if restErrorRenderer.HandleError(ctx, err) {
		return false
	}

	page := NewPage(pageOpts, &filter, items, CountToInt64(totalCount))
	return writeJSONChecked(ctx, page)
}
