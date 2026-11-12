package handler

import (
	"slices"
	"strings"

	"github.com/kataras/iris/v14/context"
)

// readUpdateColumns collects the columns of a partial update from the URL
// query, in the order the client sent them, without empties and without
// duplicates.
//
// It mirrors context.BindUpdate's own reader exactly, down to the
// context.UpdateColumnsQueryKey it reads: the columns arrive either as a
// repeated key (?columns=a&columns=b) or as one comma-separated value
// (?columns=a,b). The parsing is duplicated rather than shared because the
// context reader is unexported and BindUpdate decodes the body too, which
// these handlers do through their own pluggable request decoder.
//
// The two spellings used to disagree between the handler package
// (GetSlice, so "a,b" was one bogus column name) and the repository routes
// (GetStringSlice), which meant the same request updated different columns
// depending on which entry point had generated the route.
func readUpdateColumns(ctx *context.Context) []string {
	rawValues := ctx.Query().GetSlice(context.UpdateColumnsQueryKey)
	if len(rawValues) == 0 {
		return nil
	}

	columns := make([]string, 0, len(rawValues))
	for _, rawValue := range rawValues {
		for _, column := range strings.Split(rawValue, ",") {
			column = strings.TrimSpace(column)
			if column == "" || slices.Contains(columns, column) {
				continue
			}

			columns = append(columns, column)
		}
	}

	if len(columns) == 0 {
		return nil
	}

	return columns
}
