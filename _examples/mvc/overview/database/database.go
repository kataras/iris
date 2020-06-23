package database

import "app/environment"

// DB example database interface.
type DB interface {
	Exec(q string) error
}

// NewDB returns a database based on "env".
func NewDB(env environment.Env) DB {
	switch env {
	case environment.PROD:
		return &mysql{}
	case environment.DEV:
		return &sqlite{}
	default:
		panic("unknown environment")
	}
}
