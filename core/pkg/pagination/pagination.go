// Package pagination holds the data types of a paginated list: the page
// options a client sends in the URL, the page envelope a server answers with
// and the list filter both share. It imports only the standard library, so a
// client can use the same types as the server without the framework.
//
// The server side (Context.BindPagination, Context.Paginated, AllowSort,
// DenySort) lives in package context, which aliases these types.
package pagination

import (
	"math"
	"net/http"
	"strconv"

	"github.com/kataras/iris/v14/core/pkg/constraints"
)

// The page size bounds of PageOptions.GetLimit. Set them once, at start-up;
// iris.Patches().Context().SetMaxPageSize and SetDefaultPageSize do the same.
var (
	// MaxPageSize defines the max size of items to display.
	MaxPageSize = 100000
	// DefaultPageSize defines the default size when PageOptions.Size is zero.
	DefaultPageSize = 100
)

// PageOptions is the list request object which should be provided by the client through
// URL Query. Then the server passes that options to a database query,
// including any custom filters may be given from the request body and,
// then the server responds back with a `Context.JSON(NewPage(...))` response based
// on the database query's results.
type PageOptions struct {
	// Current page number.
	// If Page > 0 then:
	// Limit = Size
	// Offset = Limit * (Page - 1), see the GetOffset method.
	// If Page == 0 then no actual data is return,
	// internally we must check for this value
	// because in postgres LIMIT 0 returns the columns but with an empty set.
	Page int `json:"page" url:"page"`
	// The elements to get, this modifies the LIMIT clause,
	// this Size can't be higher than the MaxPageSize.
	// If Size is zero then size is set to DefaultPageSize.
	Size int `json:"size" url:"size"`
	// Details determinates if the server should return nested objects or not.
	Details bool `json:"details" url:"details"`
}

// GetLimit returns the LIMIT value of a query: Size, or DefaultPageSize when
// Size is not positive, clamped to MaxPageSize when that is positive. The
// result is always positive, so it is safe as a divisor and as a LIMIT even
// when the package variables are misconfigured to zero.
func (opts *PageOptions) GetLimit() int {
	size := opts.Size
	if size <= 0 {
		size = DefaultPageSize
	}

	if MaxPageSize > 0 && size > MaxPageSize {
		size = MaxPageSize
	}

	if size <= 0 { // DefaultPageSize set to zero or negative.
		size = 1
	}

	return size
}

// GetOffset returns the OFFSET value of a query.
func (opts *PageOptions) GetOffset() int {
	if opts.Page > 1 {
		return (opts.Page - 1) * opts.GetLimit()
	}

	return 0
}

// GetCurrentPage returns the Page or 1.
func (opts *PageOptions) GetCurrentPage() int {
	current := opts.Page
	if current == 0 {
		current = 1
	}

	return current
}

// GetNextPage returns the next page, current page + 1.
func (opts *PageOptions) GetNextPage() int {
	return opts.GetCurrentPage() + 1
}

// Bind binds the PageOptions values to a request value.
// It should be used as an client.RequestOption to fire requests
// on a server that supports, e.g.
//
//	resp, err := c.Do(ctx, "QUERY", "/u/list", filter, page.Bind)
func (opts *PageOptions) Bind(r *http.Request) error {
	page := strconv.Itoa(opts.GetCurrentPage())
	size := strconv.Itoa(opts.GetLimit())
	details := strconv.FormatBool(opts.Details)

	q := r.URL.Query()
	q.Set("page", page)
	q.Set("size", size)
	q.Set("details", details)
	r.URL.RawQuery = q.Encode() // Query returns a copy, write it back.
	return nil
}

// Page is the http response of a server handler which should render
// items with pagination support.
type Page[T any] struct {
	CurrentPage int   `json:"current_page"`  // the current page.
	PageSize    int   `json:"page_size"`     // the total amount of the entities return.
	TotalPages  int   `json:"total_pages"`   // the total number of pages based on page, size and total count.
	TotalItems  int64 `json:"total_items"`   // the total number of rows.
	HasNextPage bool  `json:"has_next_page"` // true if more data can be fetched, depending on the current page * page size and total pages.
	Filter      any   `json:"filter"`        // if any filter data.
	Items       []T   `json:"items"`         // Items is empty array if no objects returned. Do NOT modify from outside.
}

// NewPage returns a new Page response which holds
// the current page, page size, total pages, total items count, any custom filter
// and the items array.
//
// Example Code:
//
//	import "github.com/kataras/iris/v14/core/pkg/pagination"
//	...more code
//
//	type User struct {
//		Firstname string `json:"firstname"`
//		Lastname  string `json:"lastname"`
//	}
//
//	type ExtraUser struct {
//		User
//		ExtraData string
//	}
//
//	func main() {
//		users := []User{
//			{"Gerasimos", "Maropoulos"},
//			{"Efi", "Kwfidou"},
//		}
//
//		t := pagination.NewPage(pagination.PageOptions{Page: 1, Size: 50}, nil, users, 100)
//
//		// Optionally, transform a T list of objects to a V list of objects.
//		v, err := t.Transform[ExtraUser](func(u User) (ExtraUser, error) {
//			return ExtraUser{
//				User:      u,
//				ExtraData: "test extra data",
//			}, nil
//		})
//		if err != nil { panic(err) }
//
//		paginationJSON, err := json.MarshalIndent(v, "", "    ")
//		if err!=nil { panic(err) }
//		fmt.Println(string(paginationJSON))
//	}
func NewPage[T any](opts PageOptions, filter any, items []T, totalCount int64) *Page[T] {
	n := len(items)
	if n == 0 {
		// Items is documented as an empty array rather than null.
		items = make([]T, 0)
	}

	if totalCount < 0 {
		totalCount = 0
	}

	// An empty page of a non-empty result set is still a page of that result
	// set. The count and the page the client asked for are kept: a request for
	// page 9 of 4 answers "9 of 4, 10 rows, none here", so a pager can say so
	// and offer the way back. Reporting "1 of 0, 0 rows" instead loses both.

	// Integer ceiling: a float64 division loses the low bits of a total
	// above 2^53. GetLimit is always positive; the guard keeps a divide by
	// zero out of a request handler should that ever change.
	pageSize := int64(opts.GetLimit())
	if pageSize <= 0 {
		pageSize = 1
	}
	numberOfPages := int((totalCount + pageSize - 1) / pageSize)
	currentPage := opts.GetCurrentPage()

	return &Page[T]{
		CurrentPage: currentPage,
		PageSize:    n,
		TotalPages:  numberOfPages,
		TotalItems:  totalCount,
		// Nothing came back, so there is nothing after it either, whatever the
		// count says: a page that hands out no items must not promise a next.
		HasNextPage: n > 0 && currentPage < numberOfPages,
		Filter:      filter,
		Items:       items,
	}
}

// Transform accepts a Page response and converts to a list of V items.
// T => from
// V => to
//
// Example Code:
//
//	listOfUsers := NewPage(...)
//	newPageOfExtraUsers, err := listOfUsers.Transform[ExtraUser](func(u User) (ExtraUser, error) {
//		return ExtraUser{
//			User:      u,
//			ExtraData: "test extra data",
//		}, nil
//	})
func (list *Page[T]) Transform[V any](transform func(T) (V, error)) (*Page[V], error) {
	if list == nil {
		return &Page[V]{
			CurrentPage: 1,
			PageSize:    0,
			TotalItems:  0,
			TotalPages:  0,
			Filter:      nil,
			Items:       make([]V, 0),
		}, nil
	}

	items := list.Items

	toItems := make([]V, 0, len(items))
	if transform != nil { // If it's nil, it's ok - we just transform an empty page to other.
		for _, fromItem := range items {
			toItem, err := transform(fromItem)
			if err != nil {
				return nil, err
			}

			toItems = append(toItems, toItem)
		}
	}

	newList := &Page[V]{
		CurrentPage: list.CurrentPage,
		PageSize:    list.PageSize,
		TotalItems:  list.TotalItems,
		TotalPages:  list.TotalPages,
		HasNextPage: list.HasNextPage,
		Filter:      list.Filter,
		Items:       toItems,
	}
	return newList, nil
}

// CountType is a type that can be used as a total count of the items in a list.
type CountType interface {
	constraints.Number
}

// CountResponse is a generic response for a count operation.
type CountResponse[N CountType] struct {
	Count N `json:"count"`
}

// CountToInt64 converts a total count to the Page's int64. Integer counts
// convert exactly; a float count rounds half up (2.5 becomes 3), because a
// fractional total can only come from an aggregate and truncating it could
// hide the last page.
//
// It is exported so that every door into NewPage answers the same: the MVC list
// handlers take their own total from the service and must not truncate what
// Paginated rounds.
func CountToInt64[N CountType](n N) int64 {
	if N(int64(n)) == n {
		return int64(n)
	}

	return int64(math.Floor(float64(n) + 0.5))
}
