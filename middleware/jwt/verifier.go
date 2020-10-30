package jwt

import (
	"reflect"
	"time"

	"github.com/kataras/iris/v12/context"

	"github.com/kataras/jwt"
)

const (
	claimsContextKey        = "iris.jwt.claims"
	verifiedTokenContextKey = "iris.jwt.token"
)

// Get returns the claims decoded by a verifier.
func Get(ctx *context.Context) interface{} {
	if v := ctx.Values().Get(claimsContextKey); v != nil {
		return v
	}

	return nil
}

// GetVerifiedToken returns the verified token structure
// which holds information about the decoded token
// and its standard claims.
func GetVerifiedToken(ctx *context.Context) *VerifiedToken {
	if v := ctx.Values().Get(verifiedTokenContextKey); v != nil {
		if tok, ok := v.(*VerifiedToken); ok {
			return tok
		}
	}

	return nil
}

// Verifier holds common options to verify an incoming token.
// Its Verify method can be used as a middleware to allow authorized clients to access an API.
type Verifier struct {
	Alg Alg
	Key interface{}

	Decrypt func([]byte) ([]byte, error)

	Extractors []TokenExtractor
	Blocklist  Blocklist
	Validators []TokenValidator

	ErrorHandler func(ctx *context.Context, err error)
}

// NewVerifier accepts the algorithm for the token's signature among with its (private) key
// and optionally some token validators for all verify middlewares that may initialized under this Verifier.
//
// See its Verify method.
func NewVerifier(signatureAlg Alg, signatureKey interface{}, validators ...TokenValidator) *Verifier {
	return &Verifier{
		Alg:        signatureAlg,
		Key:        signatureKey,
		Extractors: []TokenExtractor{FromHeader, FromQuery},
		ErrorHandler: func(ctx *context.Context, err error) {
			ctx.StopWithError(401, context.PrivateError(err))
		},
		Validators: validators,
	}
}

// WithGCM enables AES-GCM payload encryption.
func (v *Verifier) WithGCM(key, additionalData []byte) *Verifier {
	_, decrypt, err := jwt.GCM(key, additionalData)
	if err != nil {
		panic(err) // important error before serve, stop everything.
	}

	v.Decrypt = decrypt
	return v
}

// WithDefaultBlocklist attaches an in-memory blocklist storage
// to invalidate tokens through server-side.
// To invalidate a token simply call the Context.Logout method.
func (v *Verifier) WithDefaultBlocklist() *Verifier {
	v.Blocklist = jwt.NewBlocklist(30 * time.Minute)
	return v
}

func (v *Verifier) invalidate(ctx *context.Context) {
	if verifiedToken := GetVerifiedToken(ctx); verifiedToken != nil {
		v.Blocklist.InvalidateToken(verifiedToken.Token, verifiedToken.StandardClaims.Expiry)
		ctx.Values().Remove(claimsContextKey)
		ctx.Values().Remove(verifiedTokenContextKey)
		ctx.SetUser(nil)
		ctx.SetLogoutFunc(nil)
	}
}

// RequestToken extracts the token from the request.
func (v *Verifier) RequestToken(ctx *context.Context) (token string) {
	for _, extract := range v.Extractors {
		if token = extract(ctx); token != "" {
			break // ok we found it.
		}
	}

	return
}

type (
	// ClaimsValidator is a special interface which, if the destination claims
	// implements it then the verifier runs its Validate method before return.
	ClaimsValidator interface {
		Validate() error
	}

	// ClaimsContextValidator same as ClaimsValidator but it accepts
	// a request context which can be used for further checks before
	// validating the incoming token's claims.
	ClaimsContextValidator interface {
		Validate(*context.Context) error
	}
)

// VerifyToken simply verifies the given "token" and validates its standard claims (such as expiration).
// Returns a structure which holds the token's information. See the Verify method instead.
func (v *Verifier) VerifyToken(token []byte, validators ...TokenValidator) (*VerifiedToken, error) {
	return jwt.VerifyEncrypted(v.Alg, v.Key, v.Decrypt, token, validators...)
}

// Verify is the most important piece of code inside the Verifier.
// It accepts the "claimsType" function which should return a pointer to a custom structure
// which the token's decode claims valuee will be binded and validated to.
// Returns a common Iris handler which can be used as a middleware to protect an API
// from unauthorized client requests. After this, the route handlers can access the claims
// through the jwt.Get package-level function.
//
// Example Code:
func (v *Verifier) Verify(claimsType func() interface{}, validators ...TokenValidator) context.Handler {
	unmarshal := jwt.Unmarshal
	if claimsType != nil {
		c := claimsType()
		if hasRequired(c) {
			unmarshal = jwt.UnmarshalWithRequired
		}
	}

	if v.Blocklist != nil {
		validators = append([]TokenValidator{v.Blocklist}, append(v.Validators, validators...)...)
	}

	return func(ctx *context.Context) {
		token := []byte(v.RequestToken(ctx))
		verifiedToken, err := v.VerifyToken(token, validators...)
		if err != nil {
			v.ErrorHandler(ctx, err)
			return
		}

		if claimsType != nil {
			dest := claimsType()
			if err = unmarshal(verifiedToken.Payload, dest); err != nil {
				v.ErrorHandler(ctx, err)
				return
			}

			if validator, ok := dest.(ClaimsValidator); ok {
				if err = validator.Validate(); err != nil {
					v.ErrorHandler(ctx, err)
					return
				}
			} else if contextValidator, ok := dest.(ClaimsContextValidator); ok {
				if err = contextValidator.Validate(ctx); err != nil {
					v.ErrorHandler(ctx, err)
					return
				}
			}

			if u, ok := dest.(context.User); ok {
				ctx.SetUser(u)
			}

			ctx.Values().Set(claimsContextKey, dest)
		}

		if v.Blocklist != nil {
			ctx.SetLogoutFunc(v.invalidate)
		}

		ctx.Values().Set(verifiedTokenContextKey, verifiedToken)
		ctx.Next()
	}
}

func hasRequired(i interface{}) bool {
	val := reflect.Indirect(reflect.ValueOf(i))
	typ := val.Type()
	if typ.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if jwt.HasRequiredJSONTag(field) {
			return true
		}
	}

	return false
}
