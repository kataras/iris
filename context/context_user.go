package context

import (
	"encoding/json"
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
// There are three builtin implementations of the User interface:
// - SimpleUser
// - UserMap (a wrapper by SetUser)
// - UserPartial (a wrapper by SetUser)
type User interface {
	// GetRaw should return the raw instance of the user, if supported.
	GetRaw() (interface{}, error)
	// GetAuthorization should return the authorization method,
	// e.g. Basic Authentication.
	GetAuthorization() (string, error)
	// GetAuthorizedAt should return the exact time the
	// client has been authorized for the "first" time.
	GetAuthorizedAt() (time.Time, error)
	// GetID should return the ID of the User.
	GetID() (string, error)
	// GetUsername should return the name of the User.
	GetUsername() (string, error)
	// GetPassword should return the encoded or raw password
	// (depends on the implementation) of the User.
	GetPassword() (string, error)
	// GetEmail should return the e-mail of the User.
	GetEmail() (string, error)
	// GetRoles should optionally return the specific user's roles.
	// Returns `ErrNotSupported` if this method is not
	// implemented by the User implementation.
	GetRoles() ([]string, error)
	// GetToken should optionally return a token used
	// to authorize this User.
	GetToken() ([]byte, error)
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
^ UPDATE: this is done through UserPartial.

The disadvantage of the current implementation is that the developer MUST
complete the whole interface in order to be a valid User and if we add
new methods in the future their implementation will break
(unless they have a static interface implementation check as we have on SimpleUser).
We kind of by-pass this disadvantage by providing a SimpleUser which can be embedded (as pointer)
to the end-developer's custom implementations.
*/

// SimpleUser is a simple implementation of the User interface.
type SimpleUser struct {
	Authorization string          `json:"authorization,omitempty" db:"authorization"`
	AuthorizedAt  time.Time       `json:"authorized_at,omitempty" db:"authorized_at"`
	ID            string          `json:"id,omitempty" db:"id"`
	Username      string          `json:"username,omitempty" db:"username"`
	Password      string          `json:"password,omitempty" db:"password"`
	Email         string          `json:"email,omitempty" db:"email"`
	Roles         []string        `json:"roles,omitempty" db:"roles"`
	Token         json.RawMessage `json:"token,omitempty" db:"token"`
	Fields        Map             `json:"fields,omitempty" db:"fields"`
}

var _ User = (*SimpleUser)(nil)

// GetRaw returns itself.
func (u *SimpleUser) GetRaw() (interface{}, error) {
	return u, nil
}

// GetAuthorization returns the authorization method,
// e.g. Basic Authentication.
func (u *SimpleUser) GetAuthorization() (string, error) {
	return u.Authorization, nil
}

// GetAuthorizedAt returns the exact time the
// client has been authorized for the "first" time.
func (u *SimpleUser) GetAuthorizedAt() (time.Time, error) {
	return u.AuthorizedAt, nil
}

// GetID returns the ID of the User.
func (u *SimpleUser) GetID() (string, error) {
	return u.ID, nil
}

// GetUsername returns the name of the User.
func (u *SimpleUser) GetUsername() (string, error) {
	return u.Username, nil
}

// GetPassword returns the raw password of the User.
func (u *SimpleUser) GetPassword() (string, error) {
	return u.Password, nil
}

// GetEmail returns the e-mail of (string,error) User.
func (u *SimpleUser) GetEmail() (string, error) {
	return u.Email, nil
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
func (u *SimpleUser) GetToken() ([]byte, error) {
	if len(u.Token) == 0 {
		return nil, ErrNotSupported
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

// UserMap can be used to convert a common map[string]interface{} to a User.
// Usage:
//
//	user := map[string]interface{}{
//	  "username": "kataras",
//	  "age"     : 27,
//	}
//
// ctx.SetUser(user)
// OR
// user := UserStruct{....}
// ctx.SetUser(user)
// [...]
// username, err := ctx.User().GetUsername()
// field,err := ctx.User().GetField("age")
// age := field.(int)
// OR cast it:
// user := ctx.User().(UserMap)
// username := user["username"].(string)
// age := user["age"].(int)
type UserMap Map

var _ User = UserMap{}

// GetRaw returns the underline map.
func (u UserMap) GetRaw() (interface{}, error) {
	return Map(u), nil
}

// GetAuthorization returns the authorization or Authorization value of the map.
func (u UserMap) GetAuthorization() (string, error) {
	return u.str("authorization")
}

// GetAuthorizedAt returns the authorized_at or Authorized_At value of the map.
func (u UserMap) GetAuthorizedAt() (time.Time, error) {
	return u.time("authorized_at")
}

// GetID returns the id or Id or ID value of the map.
func (u UserMap) GetID() (string, error) {
	return u.str("id")
}

// GetUsername returns the username or Username value of the map.
func (u UserMap) GetUsername() (string, error) {
	return u.str("username")
}

// GetPassword returns the password or Password value of the map.
func (u UserMap) GetPassword() (string, error) {
	return u.str("password")
}

// GetEmail returns the email or Email value of the map.
func (u UserMap) GetEmail() (string, error) {
	return u.str("email")
}

// GetRoles returns the roles or Roles value of the map.
func (u UserMap) GetRoles() ([]string, error) {
	return u.strSlice("roles")
}

// GetToken returns the roles or Roles value of the map.
func (u UserMap) GetToken() ([]byte, error) {
	return u.bytes("token")
}

// GetField returns the raw map's value based on its "key".
// It's not kind of useful here as you can just use the map.
func (u UserMap) GetField(key string) (interface{}, error) {
	return u[key], nil
}

func (u UserMap) val(key string) interface{} {
	isTitle := unicode.IsTitle(rune(key[0])) // if starts with uppercase.
	if isTitle {
		key = strings.ToLower(key)
	}

	return u[key]
}

func (u UserMap) bytes(key string) ([]byte, error) {
	if v := u.val(key); v != nil {
		switch s := v.(type) {
		case []byte:
			return s, nil
		case string:
			return []byte(s), nil
		}
	}

	return nil, ErrNotSupported
}

func (u UserMap) str(key string) (string, error) {
	if v := u.val(key); v != nil {
		if s, ok := v.(string); ok {
			return s, nil
		}

		// exists or not we don't care, if it's invalid type we don't fill it.
	}

	return "", ErrNotSupported
}

func (u UserMap) strSlice(key string) ([]string, error) {
	if v := u.val(key); v != nil {
		if s, ok := v.([]string); ok {
			return s, nil
		}
	}

	return nil, ErrNotSupported
}

func (u UserMap) time(key string) (time.Time, error) {
	if v := u.val(key); v != nil {
		if t, ok := v.(time.Time); ok {
			return t, nil
		}
	}

	return time.Time{}, ErrNotSupported
}

type (
	userGetAuthorization interface {
		GetAuthorization() string
	}

	userGetAuthorizedAt interface {
		GetAuthorizedAt() time.Time
	}

	userGetID interface {
		GetID() string
	}

	// UserGetUsername interface which
	// requires a single method to complete
	// a User on Context.SetUser.
	UserGetUsername interface {
		GetUsername() string
	}

	// UserGetPassword interface which
	// requires a single method to complete
	// a User on Context.SetUser.
	UserGetPassword interface {
		GetPassword() string
	}

	userGetEmail interface {
		GetEmail() string
	}

	userGetRoles interface {
		GetRoles() []string
	}

	userGetToken interface {
		GetToken() []byte
	}

	userGetField interface {
		GetField(string) interface{}
	}

	// UserPartial is a User.
	// It's a helper which wraps a struct value that
	// may or may not complete the whole User interface.
	// See Context.SetUser.
	UserPartial struct {
		Raw                  interface{} `json:"raw"`
		userGetAuthorization `json:",omitempty"`
		userGetAuthorizedAt  `json:",omitempty"`
		userGetID            `json:",omitempty"`
		UserGetUsername      `json:",omitempty"`
		UserGetPassword      `json:",omitempty"`
		userGetEmail         `json:",omitempty"`
		userGetRoles         `json:",omitempty"`
		userGetToken         `json:",omitempty"`
		userGetField         `json:",omitempty"`
	}
)

var _ User = (*UserPartial)(nil)

func newUserPartial(i interface{}) *UserPartial {
	if i == nil {
		return nil
	}

	p := &UserPartial{Raw: i}

	if u, ok := i.(userGetAuthorization); ok {
		p.userGetAuthorization = u
	}

	if u, ok := i.(userGetAuthorizedAt); ok {
		p.userGetAuthorizedAt = u
	}

	if u, ok := i.(userGetID); ok {
		p.userGetID = u
	}

	if u, ok := i.(UserGetUsername); ok {
		p.UserGetUsername = u
	}

	if u, ok := i.(UserGetPassword); ok {
		p.UserGetPassword = u
	}

	if u, ok := i.(userGetEmail); ok {
		p.userGetEmail = u
	}

	if u, ok := i.(userGetRoles); ok {
		p.userGetRoles = u
	}

	if u, ok := i.(userGetToken); ok {
		p.userGetToken = u
	}

	if u, ok := i.(userGetField); ok {
		p.userGetField = u
	}

	// if !containsAtLeastOneMethod {
	// 	return nil
	// }

	return p
}

// GetRaw returns the original raw instance of the user.
func (u *UserPartial) GetRaw() (interface{}, error) {
	if u == nil {
		return nil, ErrNotSupported
	}

	return u.Raw, nil
}

// GetAuthorization should return the authorization method,
// e.g. Basic Authentication.
func (u *UserPartial) GetAuthorization() (string, error) {
	if v := u.userGetAuthorization; v != nil {
		return v.GetAuthorization(), nil
	}

	return "", ErrNotSupported
}

// GetAuthorizedAt should return the exact time the
// client has been authorized for the "first" time.
func (u *UserPartial) GetAuthorizedAt() (time.Time, error) {
	if v := u.userGetAuthorizedAt; v != nil {
		return v.GetAuthorizedAt(), nil
	}

	return time.Time{}, ErrNotSupported
}

// GetID should return the ID of the User.
func (u *UserPartial) GetID() (string, error) {
	if v := u.userGetID; v != nil {
		return v.GetID(), nil
	}

	return "", ErrNotSupported
}

// GetUsername should return the name of the User.
func (u *UserPartial) GetUsername() (string, error) {
	if v := u.UserGetUsername; v != nil {
		return v.GetUsername(), nil
	}

	return "", ErrNotSupported
}

// GetPassword should return the encoded or raw password
// (depends on the implementation) of the User.
func (u *UserPartial) GetPassword() (string, error) {
	if v := u.UserGetPassword; v != nil {
		return v.GetPassword(), nil
	}

	return "", ErrNotSupported
}

// GetEmail should return the e-mail of the User.
func (u *UserPartial) GetEmail() (string, error) {
	if v := u.userGetEmail; v != nil {
		return v.GetEmail(), nil
	}

	return "", ErrNotSupported
}

// GetRoles should optionally return the specific user's roles.
// Returns `ErrNotSupported` if this method is not
// implemented by the User implementation.
func (u *UserPartial) GetRoles() ([]string, error) {
	if v := u.userGetRoles; v != nil {
		return v.GetRoles(), nil
	}

	return nil, ErrNotSupported
}

// GetToken should optionally return a token used
// to authorize this User.
func (u *UserPartial) GetToken() ([]byte, error) {
	if v := u.userGetToken; v != nil {
		return v.GetToken(), nil
	}

	return nil, ErrNotSupported
}

// GetField should optionally return a dynamic field
// based on its key. Useful for custom user fields.
// Keep in mind that these fields are encoded as a separate JSON key.
func (u *UserPartial) GetField(key string) (interface{}, error) {
	if v := u.userGetField; v != nil {
		return v.GetField(key), nil
	}

	return nil, ErrNotSupported
}
