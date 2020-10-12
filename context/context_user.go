package context

import (
	"errors"
	"time"
)

// ErrNotSupported is fired when a specific method is not implemented
// or not supported entirely.
// Can be used by User implementations when
// an authentication system does not implement a specific, but required,
// method of the User interface.
var ErrNotSupported = errors.New("not supported")

// User is a generic view of an authorized client.
// See `Context.User` and `SetUser` methods for more.
//
// The informational methods starts with a "Get" prefix
// in order to allow the implementation to contain exported
// fields such as `Username` so they can be JSON encoded when necessary.
//
// The caller is free to cast this with the implementation directly
// when special features are offered by the authorization system.
type User interface {
	// GetAuthorization should return the authorization method,
	// e.g. Basic Authentication.
	GetAuthorization() string
	// GetAuthorizedAt should return the exact time the
	// client has been authorized for the "first" time.
	GetAuthorizedAt() time.Time
	// GetUsername should return the name of the User.
	GetUsername() string
	// GetPassword should return the encoded or raw password
	// (depends on the implementation) of the User.
	GetPassword() string
	// GetEmail should return the e-mail of the User.
	GetEmail() string
}

// FeaturedUser optional interface that a User can implement.
type FeaturedUser interface {
	User
	// GetFeatures should optionally return a list of features
	// the User implementation offers.
	GetFeatures() []UserFeature
}

// UserFeature a type which represents a user's optional feature.
// See `HasUserFeature` function for more.
type UserFeature uint32

// The list of standard UserFeatures.
const (
	AuthorizedAtFeature UserFeature = iota
	UsernameFeature
	PasswordFeature
	EmailFeature
)

// HasUserFeature reports whether the "u" User
// implements a specific "feature" User Feature.
//
// It returns ErrNotSupported if a user does not implement
// the FeaturedUser interface.
func HasUserFeature(user User, feature UserFeature) (bool, error) {
	if u, ok := user.(FeaturedUser); ok {
		for _, f := range u.GetFeatures() {
			if f == feature {
				return true, nil
			}
		}

		return false, nil
	}

	return false, ErrNotSupported
}

// SimpleUser is a simple implementation of the User interface.
type SimpleUser struct {
	Authorization string        `json:"authorization"`
	AuthorizedAt  time.Time     `json:"authorized_at"`
	Username      string        `json:"username"`
	Password      string        `json:"-"`
	Email         string        `json:"email,omitempty"`
	Features      []UserFeature `json:"-"`
}

var _ User = (*SimpleUser)(nil)

// GetAuthorization returns the authorization method,
// e.g. Basic Authentication.
func (u *SimpleUser) GetAuthorization() string {
	return u.Authorization
}

// GetAuthorizedAt returns the exact time the
// client has been authorized for the "first" time.
func (u *SimpleUser) GetAuthorizedAt() time.Time {
	return u.AuthorizedAt
}

// GetUsername returns the name of the User.
func (u *SimpleUser) GetUsername() string {
	return u.Username
}

// GetPassword returns the raw password of the User.
func (u *SimpleUser) GetPassword() string {
	return u.Password
}

// GetEmail returns the e-mail of the User.
func (u *SimpleUser) GetEmail() string {
	return u.Email
}

// GetFeatures returns a list of features
// this User implementation offers.
func (u *SimpleUser) GetFeatures() []UserFeature {
	if u.Features != nil {
		return u.Features
	}

	var features []UserFeature

	if !u.AuthorizedAt.IsZero() {
		features = append(features, AuthorizedAtFeature)
	}

	if u.Username != "" {
		features = append(features, UsernameFeature)
	}

	if u.Password != "" {
		features = append(features, PasswordFeature)
	}

	if u.Email != "" {
		features = append(features, EmailFeature)
	}

	return features
}
