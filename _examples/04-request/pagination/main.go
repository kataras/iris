// A complete list endpoint in one call: ctx.Paginated.
//
// Paginated reads ?page=&size=&details= into an iris.PageOptions, reads the
// OPTIONAL JSON filter body next to it, runs your collector and renders the
// iris.Page[T] envelope. The collector receives the parsed options and
// returns the window plus the total count of matches.
//
// opts.GetLimit() is Size bounded by pagination.MaxPageSize, falling back to
// pagination.DefaultPageSize (100) when Size is zero, both settable through
// iris.Patches().Context(); opts.GetOffset() derives
// the offset from Page. Feed both to a database query - here to a slice.
//
// QUERY (RFC 10008) is the method for the filtered form: a safe, cacheable
// request that carries a body.
//
// Learn: serve a paged list with an optional filter body and the Page[T]
// envelope from one collector function.
// Before: ../strict-query
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl 'http://localhost:8080/books?page=1&size=2'   # 200, items 1-2 of 5, has_next_page true
//	curl 'http://localhost:8080/books?page=2&size=2'   # 200, items 3-4 of 5
//	curl -X QUERY 'http://localhost:8080/books?page=1&size=2' \
//	  -H 'Content-Type: application/json' -d '{"author":"Kurt Vonnegut"}'   # 200, the two Vonnegut books, total_items 2
//
// The envelope ("page_size" is how many items THIS page carries):
//
//	{
//	  "current_page": 1, "page_size": 2, "total_pages": 3, "total_items": 5,
//	  "has_next_page": true,
//	  "filter": {"author": ""},
//	  "items": [{"title":"...","author":"..."}, {...}]
//	}
//
// Next: ../uploads
package main

import (
	stdContext "context"
	"strings"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	app.Get("/books", listBooks)   // ?page=&size= only.
	app.Query("/books", listBooks) // ?page=&size= plus a filter body.

	app.Listen(":8080")
}

// Book is the listed resource.
type Book struct {
	Title  string `json:"title"`
	Author string `json:"author"`
}

// BookFilter is the optional JSON body; its zero value means "no filter".
type BookFilter struct {
	Author string `json:"author"`
}

var books = []Book{
	{Title: "Slaughterhouse-Five", Author: "Kurt Vonnegut"},
	{Title: "Cat's Cradle", Author: "Kurt Vonnegut"},
	{Title: "The Left Hand of Darkness", Author: "Ursula K. Le Guin"},
	{Title: "A Wizard of Earthsea", Author: "Ursula K. Le Guin"},
	{Title: "Solaris", Author: "Stanisław Lem"},
}

func listBooks(ctx iris.Context) {
	ctx.Paginated(collectBooks)
}

// collectBooks has the collector shape ctx.Paginated expects: it returns one page of items
// and the total number of matches, which the envelope needs for total_pages.
func collectBooks(_ stdContext.Context, opts iris.PageOptions,
	filter BookFilter) ([]Book, int64, error) {
	matched := make([]Book, 0, len(books))
	for _, b := range books {
		if filter.Author == "" || strings.EqualFold(b.Author, filter.Author) {
			matched = append(matched, b)
		}
	}

	total := int64(len(matched))

	offset := opts.GetOffset()
	if offset >= len(matched) {
		return nil, total, nil // past the last page.
	}

	end := min(offset+opts.GetLimit(), len(matched))
	return matched[offset:end], total, nil
}
