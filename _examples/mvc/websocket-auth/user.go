//go:build go1.18
// +build go1.18

package main

type AccessRole uint16

func (r AccessRole) Is(v AccessRole) bool {
	return r&v != 0
}

func (r AccessRole) Allow(v AccessRole) bool {
	return r&v >= v
}

const (
	InvalidAccessRole AccessRole = 1 << iota
	Read
	Write
	Delete

	Owner  = Read | Write | Delete
	Member = Read | Write
)

type User struct {
	ID    string     `json:"id"`
	Email string     `json:"email"`
	Role  AccessRole `json:"role"`
}

func (u User) GetID() string {
	return u.ID
}
