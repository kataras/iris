package database

// Result is our imaginary result, it will never be used, it's
// here to show you a method of doing these things.
type Result struct {
	cur int
}

// Next moves the cursor to the next result.
func (r *Result) Next() interface{} {
	return nil
}

// Database is our imaginary database interface, it will never be used here.
type Database interface {
	Open(connstring string) error
	Close() error
	Query(q string) (result Result, err error)
	Exec(q string) (lastInsertedID int64, err error)
}
