package pagination

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ListFilter is the request half every list endpoint shares: the free-text
// term, an explicit set of names, the substring switch and the sort request.
// Embed it in a service's own filter and keep only the fields that service
// adds.
//
// It carries both tag sets on purpose: the JSON one for the POST or QUERY /list body
// and the url one for a GET list route read through Context.ReadQuery.
type ListFilter struct {
	Term           string   `json:"term,omitzero" url:"term"`
	Names          []string `json:"names,omitzero" url:"names"`
	Substring      bool     `json:"substring,omitzero" url:"substring"`
	Sort           string   `json:"sort,omitzero" url:"sort"`
	SortDescending bool     `json:"sort_descending,omitzero" url:"sort_descending"`
}

// ErrSortNotAllowed is the error ListFilter.SortColumn wraps when the client
// asked to sort by a column the endpoint does not offer. A sort column reaches
// an ORDER BY clause, so the allowed set is stated by the endpoint and never
// taken from the request.
var ErrSortNotAllowed = errors.New("sort column not allowed")

// SortColumn returns the requested sort column when it is one of "allowed".
// An empty sort request answers ("", nil), so the caller keeps its own default
// ordering. A column outside the list answers an error wrapping
// ErrSortNotAllowed and naming the column.
func (f ListFilter) SortColumn(allowed ...string) (string, error) {
	column := strings.TrimSpace(f.Sort)
	if column == "" {
		return "", nil
	}

	if !slices.Contains(allowed, column) {
		return "", fmt.Errorf("%w: %s", ErrSortNotAllowed, column)
	}

	return column, nil
}

// SortOrder returns the SQL sort direction of the request: "DESC" when
// SortDescending is set, "ASC" otherwise.
func (f ListFilter) SortOrder() string {
	if f.SortDescending {
		return "DESC"
	}

	return "ASC"
}

// SortRequest returns the sort request as the sort policy options read it:
// the trimmed column, the direction and whether a sort was asked for at all.
//
// It is the method AllowSort and DenySort look for, so a service's own filter
// type takes the policy by embedding ListFilter and nothing else.
func (f ListFilter) SortRequest() (column string, descending bool, requested bool) {
	column = strings.TrimSpace(f.Sort)
	return column, f.SortDescending, column != ""
}

// SortRequester is the interface a decoded filter is checked against by the
// sort policy options (context.AllowSort and context.DenySort). ListFilter implements it, so any filter embedding it
// does too; a filter without it is not subject to a policy.
type SortRequester interface {
	SortRequest() (column string, descending bool, requested bool)
}
