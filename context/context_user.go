package context

import (
	"errors"
	"strings"
	"time"
	"unicode"
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
//
// To make optional some of the fields you can just embed the User interface
// and implement whatever methods you want to support.
//
// There are two builtin implementations of the User interface:
// - SimpleUser (type-safe)
// - UserMap (wraps a map[string]interface{})
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
	// GetRoles should optionally return the specific user's roles.
	// Returns `ErrNotSupported` if this method is not
	// implemented by the User implementation.
	GetRoles() ([]string, error)
	// GetToken should optionally return a token used
	// to authorize this User.
	GetToken() (string, error)
	// GetField should optionally return a dynamic field
	// based on its key. Useful for custom user fields.
	// Keep in mind that these fields are encoded as a separate JSON key.
	GetField(key string) (interface{}, error)
} /* Notes:
We could use a structure of User wrapper and separate interfaces for each of the methods
so they return ErrNotSupported if the implementation is missing it, so the `Features`
field and HasUserFeature can be omitted and
add a Raw() interface{} to return the underline User implementation too.
The advandages of the above idea is that we don't have to add new methods
for each of the builtin features and we can keep the (assumed) struct small.
But we dont as it has many disadvantages, unless is requested.

The disadvantage of the current implementation is that the developer MUST
complete the whole interface in order to be a valid User and if we add
new methods in the future their implementation will break
(unless they have a static interface implementation check as we have on SimpleUser).
We kind of by-pass this disadvantage by providing a SimpleUser which can be embedded (as pointer)
to the end-developer's custom implementations.
*/

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
	RolesFeature
	TokenFeature
	FieldsFeature
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
	Username      string        `json:"username,omitempty"`
	Password      string        `json:"-"`
	Email         string        `json:"email,omitempty"`
	Roles         []string      `json:"roles,omitempty"`
	Features      []UserFeature `json:"features,omitempty"`
	Token         string        `json:"token,omitempty"`
	Fields        Map           `json:"fields,omitempty"`
}

var _ FeaturedUser = (*SimpleUser)(nil)

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

// GetRoles returns the specific user's roles.
// Returns with `ErrNotSupported` if the Roles field is not initialized.
func (u *SimpleUser) GetRoles() ([]string, error) {
	if u.Roles == nil {
		return nil, ErrNotSupported
	}

	return u.Roles, nil
}

// GetToken returns the token associated with this User.
// It may return empty if the User is not featured with a Token.
//
// The implementation can change that behavior.
// Returns with `ErrNotSupported` if the Token field is empty.
func (u *SimpleUser) GetToken() (string, error) {
	if u.Token == "" {
		return "", ErrNotSupported
	}

	return u.Token, nil
}

// GetField optionally returns a dynamic field from the `Fields` field
// based on its key.
func (u *SimpleUser) GetField(key string) (interface{}, error) {
	if u.Fields == nil {
		return nil, ErrNotSupported
	}

	return u.Fields[key], nil
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

	if u.Roles != nil {
		features = append(features, RolesFeature)
	}

	if u.Fields != nil {
		features = append(features, FieldsFeature)
	}

	return features
}

// UserMap can be used to convert a common map[string]interface{} to a User.
// Usage:
// user := map[string]interface{}{
//   "username": "kataras",
//   "age"     : 27,
// }
// ctx.SetUser(UserMap(user))
// OR
// user := UserMap{"key": "value",...}
// ctx.SetUser(user)
// [...]
// username := ctx.User().GetUsername()
// age := ctx.User().GetField("age").(int)
// OR cast it:
// user := ctx.User().(UserMap)
// username := user["username"].(string)
// age := user["age"].(int)
type UserMap Map

var _ FeaturedUser = UserMap{}

// GetAuthorization returns the authorization or Authorization value of the map.
func (u UserMap) GetAuthorization() string {
	return u.str("authorization")
}

// GetAuthorizedAt returns the authorized_at or Authorized_At value of the map.
func (u UserMap) GetAuthorizedAt() time.Time {
	return u.time("authorized_at")
}

// GetUsername returns the username or Username value of the map.
func (u UserMap) GetUsername() string {
	return u.str("username")
}

// GetPassword returns the password or Password value of the map.
func (u UserMap) GetPassword() string {
	return u.str("password")
}

// GetEmail returns the email or Email value of the map.
func (u UserMap) GetEmail() string {
	return u.str("email")
}

// GetRoles returns the roles or Roles value of the map.
func (u UserMap) GetRoles() ([]string, error) {
	if s := u.strSlice("roles"); s != nil {
		return s, nil
	}

	return nil, ErrNotSupported
}

// GetToken returns the roles or Roles value of the map.
func (u UserMap) GetToken() (string, error) {
	if s := u.str("token"); s != "" {
		return s, nil
	}

	return "", ErrNotSupported
}

// GetField returns the raw map's value based on its "key".
// It's not kind of useful here as you can just use the map.
func (u UserMap) GetField(key string) (interface{}, error) {
	return u[key], nil
}

// GetFeatures returns a list of features
// this map offers.
func (u UserMap) GetFeatures() []UserFeature {
	if v := u.val("features"); v != nil { // if already contain features.
		if features, ok := v.([]UserFeature); ok {
			return features
		}
	}

	// else try to resolve from map values.
	features := []UserFeature{FieldsFeature}

	if !u.GetAuthorizedAt().IsZero() {
		features = append(features, AuthorizedAtFeature)
	}

	if u.GetUsername() != "" {
		features = append(features, UsernameFeature)
	}

	if u.GetPassword() != "" {
		features = append(features, PasswordFeature)
	}

	if u.GetEmail() != "" {
		features = append(features, EmailFeature)
	}

	if roles, err := u.GetRoles(); err == nil && roles != nil {
		features = append(features, RolesFeature)
	}

	return features
}

func (u UserMap) val(key string) interface{} {
	isTitle := unicode.IsTitle(rune(key[0])) // if starts with uppercase.
	if isTitle {
		key = strings.ToLower(key)
	}

	return u[key]
}

func (u UserMap) str(key string) string {
	if v := u.val(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}

		// exists or not we don't care, if it's invalid type we don't fill it.
	}

	return ""
}

func (u UserMap) strSlice(key string) []string {
	if v := u.val(key); v != nil {
		if s, ok := v.([]string); ok {
			return s
		}
	}

	return nil
}

func (u UserMap) time(key string) time.Time {
	if v := u.val(key); v != nil {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}

	return time.Time{}
}
