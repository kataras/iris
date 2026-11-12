package jwt

import (
	"reflect"
	"slices"
	"time"

	"github.com/kataras/iris/v14/context"

	"github.com/kataras/jwt"
)

const (
	claimsContextKey        = "iris.jwt.claims"
	verifiedTokenContextKey = "iris.jwt.token"
)

// Get returns the claims decoded by a verifier, as type T. It reports false
// when the request carries no claims or when they are not a T; it never panics.
//
// The stored value is what the claims constructor passed to Verify returned,
// usually a pointer, so T is that pointer type:
//
//	claims, ok := jwt.Get[*UserClaims](ctx)
func Get[T any](ctx *context.Context) (T, bool) {
	return ctx.ValueAs[T](claimsContextKey)
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
//
// It does not support JWE, JWK.
type Verifier struct {
	Alg Alg
	Key any

	Decrypt func([]byte) ([]byte, error)

	Extractors []TokenExtractor
	Blocklist  Blocklist
	Validators []TokenValidator

	ErrorHandler func(ctx *context.Context, err error)
	// DisableContextUser disables the registration of the claims as context User.
	DisableContextUser bool
}

// NewVerifier accepts the algorithm for the token's signature among with its (public) key
// and optionally some token validators for all verify middlewares that may initialized under this Verifier.
// See its Verify method.
//
// Usage:
//
//	verifier := NewVerifier(HS256, secret)
//
// OR
//
//	verifier := NewVerifier(HS256, secret, Expected{Issuer: "my-app"})
//
//	claimsGetter := func() any { return new(userClaims) }
//	middleware := verifier.Verify(claimsGetter)
//
// OR
//
//	middleware := verifier.Verify(claimsGetter, Expected{Issuer: "my-app"})
//
// Register the middleware, e.g.
//
//	app.Use(middleware)
//
// Get the claims:
//
//	claims, ok := jwt.Get[*userClaims](ctx)
//	if !ok {
//		return // the request went through no verifier.
//	}
//	ctx.Writef("hello %s", claims.Username)
//
// Get the context user:
//
//	username, err := ctx.User().GetUsername()
func NewVerifier(signatureAlg Alg, signatureKey any, validators ...TokenValidator) *Verifier {
	if signatureAlg == HS256 {
		// A tiny helper if the end-developer uses string instead of []byte for hmac keys.
		if k, ok := signatureKey.(string); ok {
			signatureKey = []byte(k)
		}
	}

	return &Verifier{
		Alg:        signatureAlg,
		Key:        signatureKey,
		Extractors: []TokenExtractor{FromHeader},
		ErrorHandler: func(ctx *context.Context, err error) {
			ctx.StopWithError(401, context.PrivateError(err))
		},
		Validators: validators,
	}
}

// WithQueryToken adds the query-parameter (?token=) extractor to the verifier.
//
// As of v14 the default extractors read the Authorization header only, because
// tokens placed in the URL query string tend to leak into access logs, proxy
// logs, browser history and Referer headers. Opt back in only when you must
// (e.g. browser downloads that cannot set an Authorization header):
//
//	verifier := jwt.NewVerifier(jwt.HS256, key).WithQueryToken()
func (v *Verifier) WithQueryToken() *Verifier {
	v.Extractors = append(v.Extractors, FromQuery)
	return v
}

// WithDecryption enables AES-GCM payload-only decryption of the incoming token.
func (v *Verifier) WithDecryption(key, additionalData []byte) *Verifier {
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
		v.Blocklist.InvalidateToken(verifiedToken.Token, verifiedToken.StandardClaims)
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
// By default it extracts the token from the Authorization: Bearer $token header only.
// Query-parameter (?token=) extraction is opt-in through the Verifier.WithQueryToken
// method, or by modifying its Extractors field.
//
// By default a 401 status code with a generic message will be sent to the client on
// a token verification or claims validation failure, to change that behavior
// modify its ErrorHandler field and retrieve the error through Context.GetErr method.
//
// If the "claimsType" is nil then only the jwt.GetVerifiedToken is available
// and the handler should unmarshal the payload to extract the claims by itself.
func (v *Verifier) Verify(claimsType func() any, validators ...TokenValidator) context.Handler {
	unmarshal := jwt.Unmarshal
	if claimsType != nil {
		c := claimsType()
		if hasRequired(c) {
			unmarshal = jwt.UnmarshalWithRequired
		}
	}

	// Note(@kataras): the Verifier's own validators run for EVERY Verify call,
	// with or without a Blocklist.
	//
	// They used to be spliced in only inside the "if v.Blocklist != nil" branch
	// below, so the documented
	//
	//	jwt.NewVerifier(jwt.HS256, secret, jwt.Expected{Issuer: "my-app"})
	//
	// stored the validator and then never ran it unless a blocklist happened to
	// be configured. A token minted by a different service that shares the
	// signing key - or any token with a foreign "iss"/"aud" - was accepted.
	if len(v.Validators) > 0 {
		validators = append(slices.Clone(v.Validators), validators...)
	}

	if v.Blocklist != nil {
		// If blocklist implements the connect interface,
		// try to connect if it's not already connected manually by developer,
		// if errored then just return a handler which will fire this error every single time.
		if bc, ok := v.Blocklist.(blocklistConnect); ok {
			if !bc.IsConnected() {
				if err := bc.Connect(); err != nil {
					return func(ctx *context.Context) {
						v.ErrorHandler(ctx, err)
					}
				}
			}
		}

		// The blocklist runs first: a revoked token should be rejected before any
		// other claim is considered.
		validators = append([]TokenValidator{v.Blocklist}, validators...)
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

			if !v.DisableContextUser {
				ctx.SetUser(dest)
			}

			ctx.Values().Set(claimsContextKey, dest)
		}

		if v.Blocklist != nil {
			ctx.SetLogoutFunc(func(ctx *context.Context) error { v.invalidate(ctx); return nil })
		}

		ctx.Values().Set(verifiedTokenContextKey, verifiedToken)
		ctx.Next()
	}
}

func hasRequired(i any) bool {
	val := reflect.Indirect(reflect.ValueOf(i))
	typ := val.Type()
	if typ.Kind() != reflect.Struct {
		return false
	}

	for field := range typ.Fields() {
		if jwt.HasRequiredJSONTag(field) {
			return true
		}
	}

	return false
}
