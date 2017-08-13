package models

import (
	"time"
)

// User is an example model.
type User struct {
	ID        int64
	Username  string
	Firstname string
	Lastname  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
