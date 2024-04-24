package model

// User represents our User model.
type User struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	HashedPassword []byte `json:"-"`
	Roles          []Role `json:"roles"`
}
