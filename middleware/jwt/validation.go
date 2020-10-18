package jwt

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"

	"github.com/square/go-jose/v3/json"
	// Use this package instead of the standard encoding/json
	// to marshal the NumericDate as expected by the implementation (see 'normalize`).
	"github.com/square/go-jose/v3/jwt"
)

const (
	claimsExpectedContextKey  = "iris.jwt.claims.expected"
	needsValidationContextKey = "iris.jwt.claims.unvalidated"
)

var (
	// ErrMissing when token cannot be extracted from the request (custm error).
	ErrMissing = errors.New("token is missing")
	// ErrMissingKey when token does not contain a required JSON field (custom error).
	ErrMissingKey = errors.New("token is missing a required field")
	// ErrExpired indicates that token is used after expiry time indicated in exp claim.
	ErrExpired = errors.New("token is expired (exp)")
	// ErrNotValidYet indicates that token is used before time indicated in nbf claim.
	ErrNotValidYet = errors.New("token not valid yet (nbf)")
	// ErrIssuedInTheFuture indicates that the iat field is in the future.
	ErrIssuedInTheFuture = errors.New("token issued in the future (iat)")
	// ErrBlocked indicates that the token was not yet expired
	// but was blocked by the server's Blocklist (custom error).
	ErrBlocked = errors.New("token is blocked")
	// ErrInvalidMaxAge indicates that the token is using a different
	// max age than the configurated one ( custom error).
	ErrInvalidMaxAge = errors.New("token contains invalid max age")
	// ErrExpectRefreshToken indicates that the retrieved token
	// was not a refresh token one when `ExpectRefreshToken` is set (custome rror).
	ErrExpectRefreshToken = errors.New("expect refresh token")
)

// Expectation option to provide
// an extra layer of token validation, a claims type protection.
// See `VerifyToken` method.
type Expectation func(e *Expected, claims interface{}) error

// Expect protects the claims with the expected values.
func Expect(expected Expected) Expectation {
	return func(e *Expected, _ interface{}) error {
		*e = expected
		return nil
	}
}

// ExpectID protects the claims with an ID validation.
func ExpectID(id string) Expectation {
	return func(e *Expected, _ interface{}) error {
		e.ID = id
		return nil
	}
}

// ExpectIssuer protects the claims with an issuer validation.
func ExpectIssuer(issuer string) Expectation {
	return func(e *Expected, _ interface{}) error {
		e.Issuer = issuer
		return nil
	}
}

// ExpectSubject protects the claims with a subject validation.
func ExpectSubject(sub string) Expectation {
	return func(e *Expected, _ interface{}) error {
		e.Subject = sub
		return nil
	}
}

// ExpectAudience protects the claims with an audience validation.
func ExpectAudience(audience ...string) Expectation {
	return func(e *Expected, _ interface{}) error {
		e.Audience = audience
		return nil
	}
}

// ExpectRefreshToken SHOULD be passed when a token should be verified
// based on the expiration set by `TokenPair` method instead of the JWT instance's MaxAge setting.
// Useful to validate Refresh Tokens and invalidate Access ones when refresh API is fired,
// if that option is missing then refresh tokens are invalidated when an access token was expected.
//
// Usage:
// var refreshClaims jwt.Claims
// _, err := j.VerifyTokenString(ctx, tokenPair.RefreshToken, &refreshClaims, jwt.ExpectRefreshToken)
func ExpectRefreshToken(e *Expected, _ interface{}) error { return ErrExpectRefreshToken }

// MeetRequirements protects the custom fields of JWT claims
// based on the json:required tag; `json:"name,required"`.
// It accepts the value type.
//
// Usage:
// Verify/VerifyToken(... MeetRequirements(MyUser{}))
func MeetRequirements(claimsType interface{}) Expectation {
	// pre-calculate if we need to use reflection at serve time to check for required fields,
	// this can work as an alternative of expections for custom non-standard JWT fields.
	requireFieldsIndexes := getRequiredFieldIndexes(claimsType)

	return func(e *Expected, claims interface{}) error {
		if len(requireFieldsIndexes) > 0 {
			val := reflect.Indirect(reflect.ValueOf(claims))
			for _, idx := range requireFieldsIndexes {
				field := val.Field(idx)
				if field.IsZero() {
					return ErrMissingKey
				}
			}
		}

		return nil
	}
}

// WithExpected is a middleware wrapper. It wraps a VerifyXXX middleware
// with expected claims fields protection.
// Usage:
// jwt.WithExpected(jwt.Expected{Issuer:"app"}, j.VerifyUser)
func WithExpected(e Expected, verifyHandler context.Handler) context.Handler {
	return func(ctx *context.Context) {
		ctx.Values().Set(claimsExpectedContextKey, e)
		verifyHandler(ctx)
	}
}

// ContextValidator validates the object based on the given
// claims and the expected once. The end-developer
// can use this method for advanced validations based on the request Context.
type ContextValidator interface {
	Validate(ctx *context.Context, claims Claims, e Expected) error
}

func validateClaims(ctx *context.Context, dest interface{}, claims Claims, expected Expected) (err error) {
	// Get any dynamic expectation set by prior middleware.
	// See `WithExpected` middleware.
	if v := ctx.Values().Get(claimsExpectedContextKey); v != nil {
		if e, ok := v.(Expected); ok {
			expected = e
		}
	}
	// Force-set the time, it's important for expiration.
	expected.Time = time.Now()
	switch c := dest.(type) {
	case Claims:
		err = c.ValidateWithLeeway(expected, 0)
	case ContextValidator:
		err = c.Validate(ctx, claims, expected)
	case *context.Map:
		// if the dest is a map then set automatically the expiration settings here,
		// so the caller can work further with it.
		err = claims.ValidateWithLeeway(expected, 0)
		if err == nil {
			(*c)["exp"] = claims.Expiry
			(*c)["iat"] = claims.IssuedAt
			if claims.NotBefore != nil {
				(*c)["nbf"] = claims.NotBefore
			}
		}
	default:
		err = claims.ValidateWithLeeway(expected, 0)
	}

	if err != nil {
		switch err {
		case jwt.ErrExpired:
			return ErrExpired
		case jwt.ErrNotValidYet:
			return ErrNotValidYet
		case jwt.ErrIssuedInTheFuture:
			return ErrIssuedInTheFuture
		}
	}

	return err
}

func normalize(i interface{}) (context.Map, error) {
	if m, ok := i.(context.Map); ok {
		return m, nil
	}

	m := make(context.Map)

	raw, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()

	if err := d.Decode(&m); err != nil {
		return nil, err
	}

	return m, nil
}

func getRequiredFieldIndexes(i interface{}) (v []int) {
	val := reflect.Indirect(reflect.ValueOf(i))
	typ := val.Type()
	if typ.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		// Note: for the sake of simplicity we don't lookup for nested objects (FieldByIndex),
		// we could do that as we do in dependency injection feature but unless requirested we don't.
		tag := field.Tag.Get("json")
		if strings.Contains(tag, ",required") {
			v = append(v, i)
		}
	}

	return
}

// getMaxAge returns the result of expiry-issued at.
// Note that if in JWT MaxAge's was set to a value like: 3.5 seconds
// this will return 3 on token retreival. Of course this is not a problem
// in real world apps as they don't invalidate tokens in seconds
// based on a division result like 2/7.
func getMaxAge(claims Claims) time.Duration {
	if issuedAt := claims.IssuedAt.Time(); !issuedAt.IsZero() {
		gotMaxAge := claims.Expiry.Time().Sub(issuedAt)
		return gotMaxAge
	}

	return 0
}

func compareMaxAge(expected, got time.Duration) bool {
	if expected == got {
		return true
	}

	// got is int64, maybe rounded, but the max age setting is precise, may be a float result
	// e.g. the result of a division 2/7=3.5,
	// try to validate by round of second so similar/or equal max age setting are considered valid.
	min, max := expected-time.Second, expected+time.Second
	if got < min || got > max {
		return false
	}

	return true
}
