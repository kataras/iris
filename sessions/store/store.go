// Package store the package is in diffent folder to reduce the import cycles from the ./context/context.go *
package store

import "time"

// IStore is the interface which all session stores should implement
type IStore interface {
	Get(string) interface{}
	GetString(string) string
	GetInt(string) int
	Set(string, interface{}) error
	Delete(string) error
	Clear() error
	VisitAll(func(string, interface{}))
	GetAll() map[string]interface{}
	ID() string
	LastAccessedTime() time.Time
	SetLastAccessedTime(time.Time)
	Destroy()
}
