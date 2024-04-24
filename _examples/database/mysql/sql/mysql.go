package sql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql" // lint: mysql driver.
)

// MySQL holds the underline connection of a MySQL (or MariaDB) database.
// See the `ConnectMySQL` package-level function.
type MySQL struct {
	Conn *sql.DB
}

var _ Database = (*MySQL)(nil)

var (
	// DefaultCharset default charset parameter for new databases.
	DefaultCharset = "utf8mb4"
	// DefaultCollation default collation parameter for new databases.
	DefaultCollation = "utf8mb4_unicode_ci"
)

// ConnectMySQL returns a new ready to use MySQL Database instance.
// Accepts a single argument of "dsn", i.e:
// username:password@tcp(localhost:3306)/myapp?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci
func ConnectMySQL(dsn string) (*MySQL, error) {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	err = conn.Ping()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &MySQL{
		Conn: conn,
	}, nil
}

// CreateDatabase executes the CREATE DATABASE query.
func (db *MySQL) CreateDatabase(database string) error {
	q := fmt.Sprintf("CREATE DATABASE %s DEFAULT CHARSET = %s COLLATE = %s;", database, DefaultCharset, DefaultCollation)
	_, err := db.Conn.Exec(q)
	return err
}

// Drop executes the DROP DATABASE query.
func (db *MySQL) Drop(database string) error {
	q := fmt.Sprintf("DROP DATABASE %s;", database)
	_, err := db.Conn.Exec(q)
	return err
}

// Select performs the SELECT query for this database (dsn database name is required).
func (db *MySQL) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	rows, err := db.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if scannable, ok := dest.(Scannable); ok {
		return scannable.Scan(rows)
	}

	if !rows.Next() {
		return ErrNoRows
	}
	return rows.Scan(dest)

	/* Uncomment this and pass a slice if u want to see reflection powers <3
	v, ok := dest.(reflect.Value)
	if !ok {
		v = reflect.Indirect(reflect.ValueOf(dest))
	}

	sliceTyp := v.Type()

	if sliceTyp.Kind() != reflect.Slice {
		sliceTyp = reflect.SliceOf(sliceTyp)
	}

	sliceElementTyp := deref(sliceTyp.Elem())
	for rows.Next() {
		obj := reflect.New(sliceElementTyp)
		obj.Interface().(Scannable).Scan(rows)
		if err != nil {
			return err
		}

		v.Set(reflect.Append(v, reflect.Indirect(obj)))
	}
	*/
}

// Get same as `Select` but it moves the cursor to the first result.
func (db *MySQL) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	rows, err := db.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return ErrNoRows
	}

	if scannable, ok := dest.(Scannable); ok {
		return scannable.Scan(rows)
	}

	return rows.Scan(dest)
}

// Exec executes a query. It does not return any rows.
// Use the first output parameter to count the affected rows on UPDATE, INSERT, or DELETE.
func (db *MySQL) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query, args...)
}
