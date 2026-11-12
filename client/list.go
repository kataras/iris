package client

import (
	"context"
	"iter"
	"net/http"
	"strconv"

	"github.com/kataras/iris/v14/core/pkg/pagination"
)

// MethodQuery is the RFC 10008 QUERY method: a safe, idempotent request that
// carries a body, the shape of a search or a filtered list. The client retries
// it like GET. It becomes http.MethodQuery once go.mod requires Go 1.28.
const MethodQuery = "QUERY"

// The paginated list types an Iris server answers with. They live in
// core/pkg/pagination, which imports only the standard library, and package
// iris aliases the same types, so a value decoded here is the server's type.
type (
	// PageOptions is the page, size and details a list request carries in the
	// URL query. Its Bind method is a RequestOption that writes them.
	PageOptions = pagination.PageOptions
	// Page is the response envelope of a paginated list.
	Page[T any] = pagination.Page[T]
	// CountResponse is the response of a count endpoint.
	CountResponse[N pagination.CountType] = pagination.CountResponse[N]
	// ListFilter is the filter half every Iris list endpoint shares: term,
	// names, substring and sort. Embed it in the filter sent as the body.
	ListFilter = pagination.ListFilter
)

// List fires one QUERY request to urlpath, with the page options in the URL
// query and "filter" as the JSON body, and decodes the Page of T it answers.
//
// Pass ListMethod(http.MethodPost) for a server that lists over POST, and
// PageSize or any other RequestOption through "opts"; they run after the page
// options, so they win.
//
//	page, err := client.List[Book](ctx, c, "/books", client.PageOptions{Page: 1, Size: 20}, filter)
func List[T any](ctx context.Context, c *Client, urlpath string, page PageOptions, filter any, opts ...RequestOption) (*Page[T], error) {
	opts = append([]RequestOption{page.Bind}, opts...)

	resp, err := c.BindJSON[Page[T]](ctx, MethodQuery, urlpath, filter, opts...)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Items walks a paginated list item by item, requesting page after page
// through List until the server reports no next page, a page comes back
// empty, or the loop breaks. A failed request yields its error once, with the
// zero T, and ends the walk.
//
//	for book, err := range client.Items[Book](ctx, c, "/books", filter, client.PageSize(50)) {
//		if err != nil {
//			return err
//		}
//		// use book.
//	}
func Items[T any](ctx context.Context, c *Client, urlpath string, filter any, opts ...RequestOption) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for number := 1; ; number++ {
			page, err := List[T](ctx, c, urlpath, PageOptions{Page: number}, filter, opts...)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}

			for _, item := range page.Items {
				if !yield(item, nil) {
					return
				}
			}

			// An empty page ends the walk even if the server claims a next one,
			// so a misbehaving server cannot loop it forever.
			if !page.HasNextPage || len(page.Items) == 0 {
				return
			}
		}
	}
}

// PageSize sets the "size" URL query parameter of a list request, the number
// of items per page. The server bounds it by its own maximum.
func PageSize(size int) RequestOption {
	return RequestParam("size", strconv.Itoa(size))
}

// ListMethod replaces the method of a list request, e.g. ListMethod(http.MethodPost)
// for a server that does not answer QUERY yet.
func ListMethod(method string) RequestOption {
	return func(req *http.Request) error {
		req.Method = method
		return nil
	}
}
