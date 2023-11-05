package sqlx

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12/x/reflex"
)

// DefaultTag is the default struct field tag.
var DefaultTag = "db"

// ColumnNameFunc is the function which converts a struct field name to a database column name.
type ColumnNameFunc = func(string) string

func convertStructToColumns(typ reflect.Type, nameFunc ColumnNameFunc) (map[string]*Column, error) {
	if kind := typ.Kind(); kind != reflect.Struct {
		return nil, fmt.Errorf("convert struct: invalid type: expected a struct value but got: %q", kind.String())
	}

	// Retrieve only fields valid for database.
	fields := reflex.LookupFields(typ, "")

	columns := make(map[string]*Column, len(fields))
	for i, field := range fields {
		column, ok, err := convertStructFieldToColumn(field, DefaultTag, nameFunc)
		if !ok {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("convert struct: field name: %q: %w", field.Name, err)
		}

		column.Index = i
		columns[column.Name] = column
	}

	return columns, nil
}

func convertStructFieldToColumn(field reflect.StructField, optionalTag string, nameFunc ColumnNameFunc) (*Column, bool, error) {
	c := &Column{
		Name:       nameFunc(field.Name),
		FieldIndex: field.Index,
	}

	fieldTag, ok := field.Tag.Lookup(optionalTag)
	if ok {
		if fieldTag == "-" {
			return nil, false, nil
		}

		if err := parseOptions(fieldTag, c); err != nil {
			return nil, false, err
		}
	}

	return c, true, nil
}

func parseOptions(fieldTag string, c *Column) error {
	options := strings.Split(fieldTag, ",")
	for _, opt := range options {
		if opt == "" {
			continue // skip empty.
		}

		var key, value string

		kv := strings.Split(opt, "=") // When more options come to play.
		switch len(kv) {
		case 2:
			key = kv[0]
			value = kv[1]
		case 1:
			c.Name = kv[0]
			return nil
		default:
			return fmt.Errorf("option: %s: expected key value separated by '='", opt)
		}

		switch key {
		case "name":
			c.Name = value
		default:
			return fmt.Errorf("unexpected tag option: %s", key)
		}
	}

	return nil
}
