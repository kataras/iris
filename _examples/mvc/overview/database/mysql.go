package database

import "fmt"

type mysql struct{}

func (db *mysql) Exec(q string) error {
	// simulate an error response.
	return fmt.Errorf("mysql: not implemented <%s>", q)
}
