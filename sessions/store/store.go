// Package store the package is in diffent folder to reduce the import cycles from the ./context/context.go *
package store

import "time"

// IStore is the interface which all session stores should implement
type IStore interface {
	Get(interface{}) interface{}
	GetString(key interface{}) string
	GetInt(key interface{}) int
	Set(interface{}, interface{}) error
	Delete(interface{}) error
	Clear() error
	VisitAll(func(interface{}, interface{}))
	GetAll() map[interface{}]interface{}
	ID() string
	LastAccessedTime() time.Time
	SetLastAccessedTime(time.Time)
	Destroy()
}
