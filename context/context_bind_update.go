package context

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
)

// UpdateColumnsQueryKey is the URL query key BindUpdate reads the list of
// columns to update from: PUT/PATCH /resource/{id}?columns=a&columns=b.
const UpdateColumnsQueryKey = "columns"

// BindUpdate decodes the JSON body into T and reads the "columns" query key,
// the list of columns the client wants updated.
//
// The columns arrive either as a repeated key (?columns=a&columns=b) or as one
// comma-separated value (?columns=a,b); values are trimmed, empties dropped and
// duplicates removed, keeping the client's order. No columns at all means a
// full update and returns a nil slice.
//
// A full update decodes exactly as BindJSON does: the app-level request
// validator runs and so does T's own Validate method. A partial update runs
// the request validator but skips T's Validate: the body is incomplete by
// design, every member the client did not name stays at its zero value, and a
// whole-value check would reject the request for members that are not even
// being written (a one-column body failing on "measurement unit is required"
// is the shape of that bug). Whatever a partial write still has to enforce
// belongs to the caller, which knows the columns.
//
// Every column is checked against the JSON member names of T (the json tag
// name, or the Go field name when there is no tag) and an unknown one answers
// 400 naming the column and the allowed names. Without that check a typo
// updates nothing and the client is told the request succeeded. The comparison
// is exact where the body decoder is case-insensitive: a column reaches an
// UPDATE statement, so the endpoint must receive the spelling it declared and
// not a normalised guess. A T that is not a struct (a map, say) has no member
// names to check against and every column is accepted.
//
// It returns false when a response was already written, exactly like BindJSON.
//
// Usage:
//
//	func(ctx iris.Context) {
//		input, columns, ok := ctx.BindUpdate[UpdateUser]()
//		if !ok {
//			return
//		}
//		updated, err := svc.UpdateUser(ctx, id, input, columns...)
//		ctx.NoContent(updated, err)
//	}
func (ctx *Context) BindUpdate[T any]() (T, []string, bool) {
	columns := readUpdateColumns(ctx)

	value, ok := bindJSON[T](ctx, true, len(columns) == 0)
	if !ok {
		return value, nil, false
	}

	if len(columns) == 0 {
		return value, nil, true
	}

	if allowed := updateColumnNamesOf(reflect.TypeFor[T]()); allowed != nil {
		for _, column := range columns {
			if _, found := allowed.names[column]; !found {
				restErrorRenderer.HandleInvalidArgument(ctx,
					"unknown column: %s: allowed columns: %s",
					column, strings.Join(allowed.list, ", "))
				return value, nil, false
			}
		}
	}

	return value, columns, true
}

// readUpdateColumns collects the requested columns from the URL query, in the
// order the client sent them, without empties and without duplicates.
func readUpdateColumns(ctx *Context) []string {
	rawValues := ctx.Query().GetSlice(UpdateColumnsQueryKey)
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

// updateColumnNames is the set of JSON member names of a request type, with a
// sorted copy for the error message.
type updateColumnNames struct {
	names map[string]struct{}
	list  []string
}

// updateColumnNamesCache maps a reflect.Type to its updateColumnNames, so the
// field walk happens once per request type and not once per request.
var updateColumnNamesCache sync.Map // reflect.Type -> *updateColumnNames.

// updateColumnNamesOf returns the JSON member names of "typ", or nil when the
// type is not a struct and there is nothing to validate against.
func updateColumnNamesOf(typ reflect.Type) *updateColumnNames {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ == nil || typ.Kind() != reflect.Struct {
		return nil
	}

	if cached, ok := updateColumnNamesCache.Load(typ); ok {
		return cached.(*updateColumnNames)
	}

	names := make(map[string]struct{})
	collectUpdateColumnNames(typ, names, 0)

	value := &updateColumnNames{names: names, list: slices.Sorted(maps.Keys(names))}
	actual, _ := updateColumnNamesCache.LoadOrStore(typ, value)
	return actual.(*updateColumnNames)
}

// maxUpdateColumnDepth stops the embedded walk on a type that embeds itself
// through a pointer.
const maxUpdateColumnDepth = 10

// collectUpdateColumnNames adds the JSON member names of "typ" to "names",
// flattening embedded structs the way the JSON decoder sees them.
func collectUpdateColumnNames(typ reflect.Type, names map[string]struct{}, depth int) {
	if depth > maxUpdateColumnDepth {
		return
	}

	for i := range typ.NumField() {
		field := typ.Field(i)

		tag, tagged := field.Tag.Lookup("json")
		name, _, _ := strings.Cut(tag, ",")

		if tagged && name == "-" && !strings.Contains(tag, ",") {
			continue // json:"-" keeps the field out of the payload.
		}

		if field.Anonymous && name == "" {
			embedded := field.Type
			for embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}

			if embedded.Kind() == reflect.Struct {
				collectUpdateColumnNames(embedded, names, depth+1)
				continue
			}
		}

		if !field.IsExported() {
			continue
		}

		if name == "" {
			name = field.Name
		}

		names[name] = struct{}{}
	}
}
