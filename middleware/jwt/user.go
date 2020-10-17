package jwt

import (
	"time"

	"github.com/kataras/iris/v12/context"
)

// User a common User structure for JWT.
// However, we're not limited to that one;
// any Go structure can be generated as a JWT token.
//
// Look `NewUser` and `VerifyUser` JWT middleware's methods.
// Use its `GetToken` method to generate the token when
// the User structure is set.
type User struct {
	Claims
	// Note: we could use a map too as the Token is generated when GetToken is called.
	*context.SimpleUser

	j *JWT
}

var (
	_ context.FeaturedUser = (*User)(nil)
	_ TokenSetter          = (*User)(nil)
	_ ContextValidator     = (*User)(nil)
)

// UserOption sets optional fields for a new User
// See `NewUser` instance function.
type UserOption func(*User)

// Username sets the Username and the JWT Claim's Subject
// to the given "username".
func Username(username string) UserOption {
	return func(u *User) {
		u.Username = username
		u.Claims.Subject = username
		u.Features = append(u.Features, context.UsernameFeature)
	}
}

// Email sets the Email field for the User field.
func Email(email string) UserOption {
	return func(u *User) {
		u.Email = email
		u.Features = append(u.Features, context.EmailFeature)
	}
}

// Roles upserts to the User's Roles field.
func Roles(roles ...string) UserOption {
	return func(u *User) {
		u.Roles = roles
		u.Features = append(u.Features, context.RolesFeature)
	}
}

// MaxAge sets claims expiration and the AuthorizedAt User field.
func MaxAge(maxAge time.Duration) UserOption {
	return func(u *User) {
		now := time.Now()
		u.Claims.Expiry = NewNumericDate(now.Add(maxAge))
		u.Claims.IssuedAt = NewNumericDate(now)
		u.AuthorizedAt = now

		u.Features = append(u.Features, context.AuthorizedAtFeature)
	}
}

// Fields copies the "fields" to the user's Fields field.
// This can be used to set custom fields to the User instance.
func Fields(fields context.Map) UserOption {
	return func(u *User) {
		if len(fields) == 0 {
			return
		}

		if u.Fields == nil {
			u.Fields = make(context.Map, len(fields))
		}

		for k, v := range fields {
			u.Fields[k] = v
		}

		u.Features = append(u.Features, context.FieldsFeature)
	}
}

// SetToken is called automaticaly on VerifyUser/VerifyObject.
// It sets the extracted from request, and verified from server raw token.
func (u *User) SetToken(token string) {
	u.Token = token
}

// GetToken overrides the SimpleUser's Token
// and returns the jwt generated token, among with
// a generator error, if any.
func (u *User) GetToken() (string, error) {
	if u.Token != "" {
		return u.Token, nil
	}

	if u.j != nil { // it's always not nil.
		if u.j.MaxAge > 0 {
			// if the MaxAge option was not manually set, resolve it from the JWT instance.
			MaxAge(u.j.MaxAge)(u)
		}

		// we could generate a token here
		// but let's do it on GetToken
		// as the user fields may change
		// by the caller manually until the token
		// sent to the client.
		tok, err := u.j.Token(u)
		if err != nil {
			return "", err
		}

		u.Token = tok
	}

	if u.Token == "" {
		return "", ErrMissing
	}

	return u.Token, nil
}

// Validate validates the current user's claims against
// the request. It's called automatically by the JWT instance.
func (u *User) Validate(ctx *context.Context, claims Claims, e Expected) error {
	err := u.Claims.ValidateWithLeeway(e, 0)
	if err != nil {
		return err
	}

	if u.SimpleUser.Authorization != "IRIS_JWT_USER" {
		return ErrInvalidKey
	}

	// We could add specific User Expectations (new struct and accept an interface{}),
	// but for the sake of code simplicity we don't, unless is requested, as the caller
	// can validate specific fields by its own at the next step.
	return nil
}

// UnmarshalJSON implements the json unmarshaler interface.
func (u *User) UnmarshalJSON(data []byte) error {
	err := Unmarshal(data, &u.Claims)
	if err != nil {
		return err
	}
	simpleUser := new(context.SimpleUser)
	err = Unmarshal(data, simpleUser)
	if err != nil {
		return err
	}
	u.SimpleUser = simpleUser
	return nil
}

// MarshalJSON implements the json marshaler interface.
func (u *User) MarshalJSON() ([]byte, error) {
	claimsB, err := Marshal(u.Claims)
	if err != nil {
		return nil, err
	}

	userB, err := Marshal(u.SimpleUser)
	if err != nil {
		return nil, err
	}

	if len(userB) == 0 {
		return claimsB, nil
	}

	claimsB = claimsB[0 : len(claimsB)-1] // remove last '}'
	userB = userB[1:]                     // remove first '{'

	raw := append(claimsB, ',')
	raw = append(raw, userB...)
	return raw, nil
}
