package sql

import (
	"context"
	"database/sql"
)

// Database is an interface which a database(sql) should implement.
type Database interface {
	Get(ctx context.Context, dest interface{}, q string, args ...interface{}) error
	Select(ctx context.Context, dest interface{}, q string, args ...interface{}) error
	Exec(ctx context.Context, q string, args ...interface{}) (sql.Result, error)
}

// Record should represent a database record.
// It holds the table name and the primary key.
// Entities should implement that
// in order to use the BaseService's methods.
type Record interface {
	TableName() string  // the table name which record belongs to.
	PrimaryKey() string // the primary key of the record.
}

// Sorted should represent a set of database records
// that should be rendered with order.
//
// It does NOT support the form of
// column1 ASC,
// column2 DESC
// The OrderBy method should return text in form of:
// column1
// or column1, column2
type Sorted interface {
	SortBy() string // column names separated by comma.
}

// Scannable for go structs to bind their fields.
type Scannable interface {
	Scan(*sql.Rows) error
}
