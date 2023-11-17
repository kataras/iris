//go:build go1.18
// +build go1.18

/*
Until go version 2, we can't really apply the type alias feature on a generic type or function,
so keep it separated on x/pagination.

import "github.com/kataras/iris/v12/context"

type ListResponse[T any] = context.ListResponse[T]
OR
type ListResponse = context.ListResponse doesn't work.

The only workable thing for generic aliases is when you know the type e.g.
type ListResponse = context.ListResponse[any] but that doesn't fit us.
*/

package pagination

import (
	"math"
	"net/http"
	"strconv"
)

var (
	// MaxSize defines the max size of items to display.
	MaxSize = 100000
	// DefaultSize defines the default size when ListOptions.Size is zero.
	DefaultSize = MaxSize
)

// ListOptions is the list request object which should be provided by the client through
// URL Query. Then the server passes that options to a database query,
// including any custom filters may be given from the request body and,
// then the server responds back with a `Context.JSON(NewList(...))` response based
// on the database query's results.
type ListOptions struct {
	// Current page number.
	// If Page > 0 then:
	// Limit = DefaultLimit
	// Offset = DefaultLimit * Page
	// If Page == 0 then no actual data is return,
	// internally we must check for this value
	// because in postgres LIMIT 0 returns the columns but with an empty set.
	Page int `json:"page" url:"page"`
	// The elements to get, this modifies the LIMIT clause,
	// this Size can't be higher than the MaxSize.
	// If Size is zero then size is set to DefaultSize.
	Size int `json:"size" url:"size"`
}

// GetLimit returns the LIMIT value of a query.
func (opts ListOptions) GetLimit() int {
	if opts.Size > 0 && opts.Size < MaxSize {
		return opts.Size
	}

	return DefaultSize
}

// GetLimit returns the OFFSET value of a query.
func (opts ListOptions) GetOffset() int {
	if opts.Page > 1 {
		return (opts.Page - 1) * opts.GetLimit()
	}

	return 0
}

// GetCurrentPage returns the Page or 1.
func (opts ListOptions) GetCurrentPage() int {
	current := opts.Page
	if current == 0 {
		current = 1
	}

	return current
}

// GetNextPage returns the next page, current page + 1.
func (opts ListOptions) GetNextPage() int {
	return opts.GetCurrentPage() + 1
}

// Bind binds the ListOptions values to a request value.
// It should be used as an x/client.RequestOption to fire requests
// on a server that supports pagination.
func (opts ListOptions) Bind(r *http.Request) error {
	page := strconv.Itoa(opts.GetCurrentPage())
	size := strconv.Itoa(opts.GetLimit())

	q := r.URL.Query()
	q.Set("page", page)
	q.Set("size", size)
	return nil
}

// List is the http response of a server handler which should render
// items with pagination support.
type List[T any] struct {
	CurrentPage int   `json:"current_page"`  // the current page.
	PageSize    int   `json:"page_size"`     // the total amount of the entities return.
	TotalPages  int   `json:"total_pages"`   // the total number of pages based on page, size and total count.
	TotalItems  int64 `json:"total_items"`   // the total number of rows.
	HasNextPage bool  `json:"has_next_page"` // true if more data can be fetched, depending on the current page * page size and total pages.
	Filter      any   `json:"filter"`        // if any filter data.
	Items       []T   `json:"items"`         // Items is empty array if no objects returned. Do NOT modify from outside.
}

// NewList returns a new List response which holds
// the current page, page size, total pages, total items count, any custom filter
// and the items array.
//
// Example Code:
//
//	import "github.com/kataras/iris/v12/x/pagination"
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
//		t := pagination.NewList(users, 100, nil, pagination.ListOptions{
//			Page: 1,
//			Size: 50,
//		})
//
//		// Optionally, transform a T list of objects to a V list of objects.
//		v, err := pagination.TransformList(t, func(u User) (ExtraUser, error) {
//			return ExtraUser{
//				User:      u,
//				ExtraData: "test extra data",
//			}, nil
//		})
//		if err != nil { panic(err) }
//
//		paginationJSON, err := json.MarshalIndent(v, "", "    ")
//		if err!=nil { panic(err) }
//		fmt.Println(paginationJSON)
//	}
func NewList[T any](items []T, totalCount int64, filter any, opts ListOptions) *List[T] {
	pageSize := opts.GetLimit()

	n := len(items)
	if n == 0 || pageSize <= 0 {
		return &List[T]{
			CurrentPage: 1,
			PageSize:    0,
			TotalItems:  0,
			TotalPages:  0,
			Filter:      filter,
			Items:       make([]T, 0),
		}
	}

	numberOfPages := int(roundUp(float64(totalCount)/float64(pageSize), 0))
	if numberOfPages <= 0 {
		numberOfPages = 1
	}

	var hasNextPage bool

	currentPage := opts.GetCurrentPage()
	if totalCount == 0 {
		currentPage = 1
	}

	if n > 0 {
		hasNextPage = currentPage < numberOfPages
	}

	return &List[T]{
		CurrentPage: currentPage,
		PageSize:    n,
		TotalPages:  numberOfPages,
		TotalItems:  totalCount,
		HasNextPage: hasNextPage,
		Filter:      filter,
		Items:       items,
	}
}

// TransformList accepts a List response and converts to a list of V items.
// T => from
// V => to
//
// Example Code:
//
//	listOfUsers := pagination.NewList(...)
//	newListOfExtraUsers, err := pagination.TransformList(listOfUsers, func(u User) (ExtraUser, error) {
//		return ExtraUser{
//			User:      u,
//			ExtraData: "test extra data",
//		}, nil
//	})
func TransformList[T any, V any](list *List[T], transform func(T) (V, error)) (*List[V], error) {
	if list == nil {
		return &List[V]{
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
	for _, fromItem := range items {
		toItem, err := transform(fromItem)
		if err != nil {
			return nil, err
		}

		toItems = append(toItems, toItem)
	}

	newList := &List[V]{
		CurrentPage: list.CurrentPage,
		PageSize:    list.PageSize,
		TotalItems:  list.TotalItems,
		TotalPages:  list.TotalPages,
		Filter:      list.Filter,
		Items:       toItems,
	}
	return newList, nil
}

func roundUp(input float64, places float64) float64 {
	pow := math.Pow(10, places)
	return math.Ceil(pow*input) / pow
}
