package sqlx

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/kataras/iris/v12/x/reflex"
)

type (
	// Schema holds the row definitions.
	Schema struct {
		Name           string
		Rows           map[reflect.Type]*Row
		ColumnNameFunc ColumnNameFunc
		AutoCloseRows  bool
	}

	// Row holds the column definitions and the struct type & name.
	Row struct {
		Schema     string // e.g. public
		Name       string // e.g. users. Must set to a custom one if the select query contains AS names.
		StructType reflect.Type
		Columns    map[string]*Column // e.g. "id":{"id", 0, [0]}
	}

	// Column holds the database column name and other properties extracted by a struct's field.
	Column struct {
		Name       string
		Index      int
		FieldIndex []int
	}
)

// NewSchema returns a new Schema. Use its Register() method to cache
// a structure value so Bind() can fill all struct's fields based on a query.
func NewSchema() *Schema {
	return &Schema{
		Name:           "public",
		Rows:           make(map[reflect.Type]*Row),
		ColumnNameFunc: snakeCase,
		AutoCloseRows:  true,
	}
}

// DefaultSchema initializes a common Schema.
var DefaultSchema = NewSchema()

// Register caches a struct value to the default schema.
func Register(tableName string, value interface{}) *Schema {
	return DefaultSchema.Register(tableName, value)
}

// Bind sets "dst" to the result of "src" and reports any errors.
func Bind(dst interface{}, src *sql.Rows) error {
	return DefaultSchema.Bind(dst, src)
}

// Register caches a struct value to the schema.
func (s *Schema) Register(tableName string, value interface{}) *Schema {
	typ := reflect.TypeOf(value)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if tableName == "" {
		// convert to a human name, e.g. sqlx.Food -> food.
		typeName := typ.String()
		if idx := strings.LastIndexByte(typeName, '.'); idx > 0 && len(typeName) > idx {
			typeName = typeName[idx+1:]
		}
		tableName = snakeCase(typeName)
	}

	columns, err := convertStructToColumns(typ, s.ColumnNameFunc)
	if err != nil {
		panic(fmt.Sprintf("sqlx: register: %q: %s", reflect.TypeOf(value).String(), err.Error()))
	}

	s.Rows[typ] = &Row{
		Schema:     s.Name,
		Name:       tableName,
		StructType: typ,
		Columns:    columns,
	}

	return s
}

// Bind sets "dst" to the result of "src" and reports any errors.
func (s *Schema) Bind(dst interface{}, src *sql.Rows) error {
	typ := reflect.TypeOf(dst)
	if typ.Kind() != reflect.Ptr {
		return fmt.Errorf("sqlx: bind: destination not a pointer")
	}

	typ = typ.Elem()

	originalKind := typ.Kind()
	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}

	r, ok := s.Rows[typ]
	if !ok {
		return fmt.Errorf("sqlx: bind: unregistered type: %q", typ.String())
	}

	columnTypes, err := src.ColumnTypes()
	if err != nil {
		return fmt.Errorf("sqlx: bind: table: %q: %w", r.Name, err)
	}

	if expected, got := len(r.Columns), len(columnTypes); expected != got {
		return fmt.Errorf("sqlx: bind: table: %q: unexpected number of result columns: %d: expected: %d", r.Name, got, expected)
	}

	val := reflex.IndirectValue(reflect.ValueOf(dst))
	if s.AutoCloseRows {
		defer src.Close()
	}

	switch originalKind {
	case reflect.Struct:
		if src.Next() {
			if err = r.bindSingle(typ, val, columnTypes, src); err != nil {
				return err
			}
		} else {
			return sql.ErrNoRows
		}

		return src.Err()
	case reflect.Slice:
		for src.Next() {
			elem := reflect.New(typ).Elem()
			if err = r.bindSingle(typ, elem, columnTypes, src); err != nil {
				return err
			}

			val = reflect.Append(val, elem)
		}

		if err = src.Err(); err != nil {
			return err
		}

		reflect.ValueOf(dst).Elem().Set(val)
		return nil
	default:
		return fmt.Errorf("sqlx: bind: table: %q: unexpected destination kind: %q", r.Name, typ.Kind().String())
	}
}

func (r *Row) bindSingle(typ reflect.Type, val reflect.Value, columnTypes []*sql.ColumnType, scanner interface{ Scan(...interface{}) error }) error {
	fieldPtrs, err := r.lookupStructFieldPtrs(typ, val, columnTypes)
	if err != nil {
		return fmt.Errorf("sqlx: bind: table: %q: %w", r.Name, err)
	}

	return scanner.Scan(fieldPtrs...)
}

func (r *Row) lookupStructFieldPtrs(typ reflect.Type, val reflect.Value, columnTypes []*sql.ColumnType) ([]interface{}, error) {
	fieldPtrs := make([]interface{}, 0, len(columnTypes))

	for _, columnType := range columnTypes {
		columnName := columnType.Name()
		tableColumn, ok := r.Columns[columnName]
		if !ok {
			continue
		}

		// TODO: when go 1.18 released, replace with that:
		/*
			tableColumnField, err := val.FieldByIndexErr(tableColumn.FieldIndex)
			if err != nil {
				return nil, fmt.Errorf("column: %q: %w", tableColumn.Name, err)
			}
		*/
		tableColumnField := val.FieldByIndex(tableColumn.FieldIndex)

		tableColumnFieldType := tableColumnField.Type()

		fieldPtr := reflect.NewAt(tableColumnFieldType, unsafe.Pointer(tableColumnField.UnsafeAddr())).Elem().Addr().Interface()
		fieldPtrs = append(fieldPtrs, fieldPtr)
	}

	return fieldPtrs, nil
}
