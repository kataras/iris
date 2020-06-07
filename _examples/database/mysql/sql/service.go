package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// Service holder for common queries.
// Note: each entity service keeps its own base Service instance.
type Service struct {
	db  Database
	rec Record // see `Count`, `List` and `DeleteByID` methods.
}

// NewService returns a new (SQL) base service for common operations.
func NewService(db Database, of Record) *Service {
	return &Service{db: db, rec: of}
}

// DB exposes the database instance.
func (s *Service) DB() Database {
	return s.db
}

// RecordInfo returns the record info provided through `NewService`.
func (s *Service) RecordInfo() Record {
	return s.rec
}

// ErrNoRows is returned when GET doesn't return a row.
// A shortcut of sql.ErrNoRows.
var ErrNoRows = sql.ErrNoRows

// GetByID binds a single record from the databases to the "dest".
func (s *Service) GetByID(ctx context.Context, dest interface{}, id int64) error {
	q := fmt.Sprintf("SELECT * FROM %s WHERE %s = ? LIMIT 1", s.rec.TableName(), s.rec.PrimaryKey())
	err := s.db.Get(ctx, dest, q, id)
	return err
	// if err != nil {
	// 	if err == sql.ErrNoRows {
	// 		return false, nil
	// 	}

	// 	return false, err
	// }

	// return true, nil
}

// Count returns the total records count in the table.
func (s *Service) Count(ctx context.Context) (total int64, err error) {
	q := fmt.Sprintf("SELECT COUNT(DISTINCT %s) FROM %s", s.rec.PrimaryKey(), s.rec.TableName())
	if err = s.db.Select(ctx, &total, q); err == sql.ErrNoRows {
		err = nil
	}
	return
}

// ListOptions holds the options to be passed on the `Service.List` method.
type ListOptions struct {
	Table         string // the table name.
	Offset        uint64 // inclusive.
	Limit         uint64
	OrderByColumn string
	Order         string // "ASC" or "DESC" (could be a bool type instead).
	WhereColumn   string
	WhereValue    interface{}
}

// Where accepts a column name and column value to set
// on the WHERE clause of the result query.
// It returns a new `ListOptions` value.
// Note that this is a basic implementation which just takes care our current needs.
func (opt ListOptions) Where(colName string, colValue interface{}) ListOptions {
	opt.WhereColumn = colName
	opt.WhereValue = colValue
	return opt
}

// BuildQuery returns the query and the arguments that
// should be form a SELECT command.
func (opt ListOptions) BuildQuery() (q string, args []interface{}) {
	q = fmt.Sprintf("SELECT * FROM %s", opt.Table)

	if opt.WhereColumn != "" && opt.WhereValue != nil {
		q += fmt.Sprintf(" WHERE %s = ?", opt.WhereColumn)
		args = append(args, opt.WhereValue)
	}

	if opt.OrderByColumn != "" {
		q += fmt.Sprintf(" ORDER BY %s %s", opt.OrderByColumn, ParseOrder(opt.Order))
	}

	if opt.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", opt.Limit) // offset below.
	}

	if opt.Offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", opt.Offset)
	}

	return
}

// const defaultLimit = 30 // default limit if not set.

// ParseListOptions returns a `ListOptions` from a map[string][]string.
func ParseListOptions(q url.Values) ListOptions {
	offset, _ := strconv.ParseUint(q.Get("offset"), 10, 64)
	limit, _ := strconv.ParseUint(q.Get("limit"), 10, 64)
	order := q.Get("order") // empty, asc(...) or desc(...).
	orderBy := q.Get("by")  // e.g. price

	return ListOptions{Offset: offset, Limit: limit, Order: order, OrderByColumn: orderBy}
}

// List binds one or more records from the database to the "dest".
// If the record supports ordering then it will sort by the `Sorted.OrderBy` column name(s).
// Use the "order" input parameter to set a descending order ("DESC").
func (s *Service) List(ctx context.Context, dest interface{}, opts ListOptions) error {
	// Set table and order by column from record info for `List` by options
	// so it can be more flexible to perform read-only calls of other table's too.
	if opts.Table == "" {
		// If missing then try to set it by record info.
		opts.Table = s.rec.TableName()
	}
	if opts.OrderByColumn == "" {
		if b, ok := s.rec.(Sorted); ok {
			opts.OrderByColumn = b.SortBy()
		}
	}

	q, args := opts.BuildQuery()
	return s.db.Select(ctx, dest, q, args...)
}

// DeleteByID removes a single record of "dest" from the database.
func (s *Service) DeleteByID(ctx context.Context, id int64) (int, error) {
	q := fmt.Sprintf("DELETE FROM %s WHERE %s = ? LIMIT 1", s.rec.TableName(), s.rec.PrimaryKey())
	res, err := s.db.Exec(ctx, q, id)
	if err != nil {
		return 0, err
	}

	return GetAffectedRows(res), nil
}

// ErrUnprocessable indicates error caused by invalid entity (entity's key-values).
// The syntax of the request entity is correct, but it was unable to process the contained instructions
// e.g. empty or unsupported value.
//
// See `../service/XService.Insert` and `../service/XService.Update`
// and `PartialUpdate`.
var ErrUnprocessable = errors.New("invalid entity")

// PartialUpdate accepts a columns schema and a key-value map to
// update the record based on the given "id".
// Note: Trivial string, int and boolean type validations are performed here.
func (s *Service) PartialUpdate(ctx context.Context, id int64, schema map[string]reflect.Kind, attrs map[string]interface{}) (int, error) {
	if len(schema) == 0 || len(attrs) == 0 {
		return 0, nil
	}

	var (
		keyLines []string
		values   []interface{}
	)

	for key, kind := range schema {
		v, ok := attrs[key]
		if !ok {
			continue
		}

		switch v.(type) {
		case string:
			if kind != reflect.String {
				return 0, ErrUnprocessable
			}
		case int:
			if kind != reflect.Int {
				return 0, ErrUnprocessable
			}
		case bool:
			if kind != reflect.Bool {
				return 0, ErrUnprocessable
			}
		}

		keyLines = append(keyLines, fmt.Sprintf("%s = ?", key))
		values = append(values, v)
	}

	if len(values) == 0 {
		return 0, nil
	}

	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?;",
		s.rec.TableName(), strings.Join(keyLines, ", "), s.rec.PrimaryKey())

	res, err := s.DB().Exec(ctx, q, append(values, id)...)
	if err != nil {
		return 0, err
	}

	n := GetAffectedRows(res)
	return n, nil
}

// GetAffectedRows returns the number of affected rows after
// a DELETE or UPDATE operation.
func GetAffectedRows(result sql.Result) int {
	if result == nil {
		return 0
	}

	n, _ := result.RowsAffected()
	return int(n)
}

const (
	ascending  = "ASC"
	descending = "DESC"
)

// ParseOrder accept an order string and returns a valid mysql ORDER clause.
// Defaults to "ASC". Two possible outputs: "ASC" and "DESC".
func ParseOrder(order string) string {
	order = strings.TrimSpace(order)
	if len(order) >= 4 {
		if strings.HasPrefix(strings.ToUpper(order), descending) {
			return descending
		}
	}

	return ascending
}
