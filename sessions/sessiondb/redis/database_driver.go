package redis

import (
	"github.com/kataras/iris/v12/sessions"
)

type DatabaseDriver interface {
	sessions.Database

	Config() *Config

	Connect(c Config) error

	Close() error
}

var (
	_ DatabaseDriver = (*DatabaseString)(nil)
	_ DatabaseDriver = (*DatabaseHashed)(nil)
)

func DatabaseDriverString(conf Config) *DatabaseString {
	return &DatabaseString{c: conf}
}

func DatabaseDriverHashed(conf Config) *DatabaseHashed {
	return &DatabaseHashed{c: conf}
}
