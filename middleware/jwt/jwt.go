package jwt

import (
	"crypto"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/kataras/iris/context"

	"github.com/square/go-jose/v3"
	"github.com/square/go-jose/v3/jwt"
)

func init() {
	context.SetHandlerName("iris/middleware/jwt.*", "iris.jwt")
}

// TokenExtractor is a function that takes a context as input and returns
// a token. An empty string should be returned if no token found
// without additional information.
type TokenExtractor func(*context.Context) string

// FromHeader is a token extractor.
// It reads the token from the Authorization request header of form:
// Authorization: "Bearer {token}".
func FromHeader(ctx *context.Context) string {
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// pure check: authorization header format must be Bearer {token}
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return ""
	}

	return authHeaderParts[1]
}

// FromQuery is a token extractor.
// It reads the token from the "token" url query parameter.
func FromQuery(ctx *context.Context) string {
	return ctx.URLParam("token")
}

// FromJSON is a token extractor.
// Reads a json request body and extracts the json based on the given field.
// The request content-type should contain the: application/json header value, otherwise
// this method will not try to read and consume the body.
func FromJSON(jsonKey string) TokenExtractor {
	return func(ctx *context.Context) string {
		if ctx.GetContentTypeRequested() != context.ContentJSONHeaderValue {
			return ""
		}

		var m context.Map
		if err := ctx.ReadJSON(&m); err != nil {
			return ""
		}

		if m == nil {
			return ""
		}

		v, ok := m[jsonKey]
		if !ok {
			return ""
		}

		tok, ok := v.(string)
		if !ok {
			return ""
		}

		return tok
	}
}

// JWT holds the necessary information the middleware need
// to sign and verify tokens.
//
// The `RSA(privateFile, publicFile, password)` package-level helper function
// can be used to decode the SignKey and VerifyKey.
type JWT struct {
	// MaxAge is the expiration duration of the generated tokens.
	MaxAge time.Duration

	// Extractors are used to extract a raw token string value
	// from the request.
	// Builtin extractors:
	// * FromHeader
	// * FromQuery
	// * FromJSON
	// Defaults to a slice of `FromHeader` and `FromQuery`.
	Extractors []TokenExtractor

	// Signer is used to sign the token.
	// It is set on `New` and `Default` package-level functions.
	Signer jose.Signer
	// VerificationKey is used to verify the token (public key).
	VerificationKey interface{}

	// Encrypter is used to, optionally, encrypt the token.
	// It is set on `WithEncryption` method.
	Encrypter jose.Encrypter
	// DecriptionKey is used to decrypt the token (private key)
	DecriptionKey interface{}
}

type privateKey interface{ Public() crypto.PublicKey }

// New returns a new JWT instance.
// It accepts a maximum time duration for token expiration
// and the algorithm among with its key for signing and verification.
//
// See `WithEncryption` method to add token encryption too.
// Use `Token` method to generate a new token string
// and `VerifyToken` method to decrypt, verify and bind claims of an incoming request token.
// Token, by default, is extracted by "Authorization: Bearer {token}" request header and
// url query parameter of "token". Token extractors can be modified through the `Extractors` field.
//
// For example, if you want to sign and verify using RSA-256 key:
// 1. Generate key file, e.g:
// 		$ openssl genrsa -des3 -out private.pem 2048
// 2. Read file contents with io.ReadFile("./private.pem")
// 3. Pass the []byte result to the `ParseRSAPrivateKey(contents, password)` package-level helper
// 4. Use the result *rsa.PrivateKey as "key" input parameter of this `New` function.
//
// See aliases.go file for available algorithms.
func New(maxAge time.Duration, alg SignatureAlgorithm, key interface{}) (*JWT, error) {
	sig, err := jose.NewSigner(jose.SigningKey{
		Algorithm: alg,
		Key:       key,
	}, (&jose.SignerOptions{}).WithType("JWT"))

	if err != nil {
		return nil, err
	}

	j := &JWT{
		Signer:          sig,
		VerificationKey: key,
		MaxAge:          maxAge,
		Extractors:      []TokenExtractor{FromHeader, FromQuery},
	}

	if s, ok := key.(privateKey); ok {
		j.VerificationKey = s.Public()
	}

	return j, nil
}

// Default key filenames for `RSA`.
const (
	DefaultSignFilename = "jwt_sign.key"
	DefaultEncFilename  = "jwt_enc.key"
)

// RSA returns a new `JWT` instance.
// It tries to parse RSA256 keys from "filenames[0]" (defaults to  "jwt_sign.key") and
// "filenames[1]" (defaults to "jwt_enc.key") files or generates and exports new random keys.
//
// It panics on errors.
// Use the `New` package-level function instead for more options.
func RSA(maxAge time.Duration, filenames ...string) *JWT {
	var (
		signFilename = DefaultSignFilename
		encFilename  = DefaultEncFilename
	)

	switch len(filenames) {
	case 1:
		signFilename = filenames[0]
	case 2:
		encFilename = filenames[1]
	}

	// Do not try to create or load enc key if only sign key already exists.
	withEncryption := true
	if fileExists(signFilename) {
		withEncryption = fileExists(encFilename)
	}

	sigKey, err := LoadRSA(signFilename, 2048)
	if err != nil {
		panic(err)
	}

	j, err := New(maxAge, RS256, sigKey)
	if err != nil {
		panic(err)
	}

	if withEncryption {
		encKey, err := LoadRSA(encFilename, 2048)
		if err != nil {
			panic(err)
		}
		err = j.WithEncryption(A128CBCHS256, RSA15, encKey)
		if err != nil {
			panic(err)
		}
	}

	return j
}

const (
	signEnv = "JWT_SECRET"
	encEnv  = "JWT_SECRET_ENC"
)

func getenv(key string, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	return v
}

// HMAC returns a new `JWT` instance.
// It tries to read hmac256 secret keys from system environment variables:
// * JWT_SECRET for signing and verification key and
// * JWT_SECRET_ENC for encryption and decryption key
// and defaults them to the given "keys" respectfully.
//
// It panics on errors.
// Use the `New` package-level function instead for more options.
func HMAC(maxAge time.Duration, keys ...string) *JWT {
	var defaultSignSecret, defaultEncSecret string

	switch len(keys) {
	case 1:
		defaultSignSecret = keys[0]
	case 2:
		defaultEncSecret = keys[1]
	}

	signSecret := getenv(signEnv, defaultSignSecret)
	encSecret := getenv(encEnv, defaultEncSecret)

	j, err := New(maxAge, HS256, []byte(signSecret))
	if err != nil {
		panic(err)
	}

	if encSecret != "" {
		err = j.WithEncryption(A128GCM, DIRECT, []byte(encSecret))
		if err != nil {
			panic(err)
		}
	}

	return j
}

// WithEncryption method enables encryption and decryption of the token.
// It sets an appropriate encrypter(`Encrypter` and the `DecriptionKey` fields) based on the key type.
func (j *JWT) WithEncryption(contentEncryption ContentEncryption, alg KeyAlgorithm, key interface{}) error {
	var publicKey interface{} = key
	if s, ok := key.(privateKey); ok {
		publicKey = s.Public()
	}

	enc, err := jose.NewEncrypter(contentEncryption, jose.Recipient{
		Algorithm: alg,
		Key:       publicKey,
	},
		(&jose.EncrypterOptions{}).WithType("JWT").WithContentType("JWT"),
	)

	if err != nil {
		return err
	}

	j.Encrypter = enc
	j.DecriptionKey = key
	return nil
}

// Expiry returns a new standard Claims with
// the `Expiry` and `IssuedAt` fields of the "claims" filled
// based on the given "maxAge" duration.
//
// See the `JWT.Expiry` method too.
func Expiry(maxAge time.Duration, claims Claims) Claims {
	now := time.Now()
	claims.Expiry = NewNumericDate(now.Add(maxAge))
	claims.IssuedAt = NewNumericDate(now)
	return claims
}

// Expiry method same as `Expiry` package-level function,
// it returns a Claims with the expiration fields of the "claims"
// filled based on the JWT's `MaxAge` field.
// Only use it when this standard "claims"
// is embedded on a custom claims structure.
// Usage:
// type UserClaims struct {
// 	jwt.Claims
// 	Username string
// }
// [...]
// standardClaims := j.Expiry(jwt.Claims{...})
// customClaims := UserClaims{
// 	Claims:   standardClaims,
// 	Username: "kataras",
// }
// j.WriteToken(ctx, customClaims)
func (j *JWT) Expiry(claims Claims) Claims {
	return Expiry(j.MaxAge, claims)
}

// Token generates and returns a new token string.
// See `VerifyToken` too.
func (j *JWT) Token(claims interface{}) (string, error) {
	// switch c := claims.(type) {
	// case Claims:
	// 	claims = Expiry(j.MaxAge, c)
	// case map[string]interface{}: let's not support map.
	// 	now := time.Now()
	// 	c["iat"] = now.Unix()
	// 	c["exp"] = now.Add(j.MaxAge).Unix()
	// }
	if c, ok := claims.(Claims); ok {
		claims = Expiry(j.MaxAge, c)
	}

	var (
		token string
		err   error
	)

	// jwt.Builder and jwt.NestedBuilder contain same methods but they are not the same.
	if j.DecriptionKey != nil {
		token, err = jwt.SignedAndEncrypted(j.Signer, j.Encrypter).Claims(claims).CompactSerialize()
	} else {
		token, err = jwt.Signed(j.Signer).Claims(claims).CompactSerialize()
	}

	if err != nil {
		return "", err
	}

	return token, nil
}

/* Let's no support maps, typed claim is the way to go.
// validateMapClaims validates claims of map type.
func validateMapClaims(m map[string]interface{}, e jwt.Expected, leeway time.Duration) error {
	if !e.Time.IsZero() {
		if v, ok := m["nbf"]; ok {
			if notBefore, ok := v.(NumericDate); ok {
				if e.Time.Add(leeway).Before(notBefore.Time()) {
					return ErrNotValidYet
				}
			}
		}

		if v, ok := m["exp"]; ok {
			if exp, ok := v.(int64); ok {
				if e.Time.Add(-leeway).Before(time.Unix(exp, 0)) {
					return ErrExpired
				}
			}
		}

		if v, ok := m["iat"]; ok {
			if issuedAt, ok := v.(int64); ok {
				if e.Time.Add(leeway).Before(time.Unix(issuedAt, 0)) {
					return ErrIssuedInTheFuture
				}
			}
		}
	}

	return nil
}
*/

// WriteToken is a helper which just generates(calls the `Token` method) and writes
// a new token to the client in plain text format.
//
// Use the `Token` method to get a new generated token raw string value.
func (j *JWT) WriteToken(ctx *context.Context, claims interface{}) error {
	token, err := j.Token(claims)
	if err != nil {
		ctx.StatusCode(500)
		return err
	}

	_, err = ctx.WriteString(token)
	return err
}

var (
	// ErrMissing when token cannot be extracted from the request.
	ErrMissing = errors.New("token is missing")
	// ErrExpired indicates that token is used after expiry time indicated in exp claim.
	ErrExpired = errors.New("token is expired (exp)")
	// ErrNotValidYet indicates that token is used before time indicated in nbf claim.
	ErrNotValidYet = errors.New("token not valid yet (nbf)")
	// ErrIssuedInTheFuture indicates that the iat field is in the future.
	ErrIssuedInTheFuture = errors.New("token issued in the future (iat)")
)

type (
	claimsValidator interface {
		ValidateWithLeeway(e jwt.Expected, leeway time.Duration) error
	}
	claimsAlternativeValidator interface { // to keep iris-contrib/jwt MapClaims compatible.
		Validate() error
	}
	claimsContextValidator interface {
		Validate(ctx *context.Context) error
	}
)

// IsValidated reports whether a token is already validated through
// `VerifyToken`. It returns true when the claims are compatible
// validators: a `Claims` value or a value that implements the `Validate() error` method.
func IsValidated(ctx *context.Context) bool { // see the `ReadClaims`.
	return ctx.Values().Get(needsValidationContextKey) == nil
}

func validateClaims(ctx *context.Context, claims interface{}) (err error) {
	switch c := claims.(type) {
	case claimsValidator:
		err = c.ValidateWithLeeway(jwt.Expected{Time: time.Now()}, 0)
	case claimsAlternativeValidator:
		err = c.Validate()
	case claimsContextValidator:
		err = c.Validate(ctx)
	case *json.RawMessage:
		// if the data type is raw message (json []byte)
		// then it should contain exp (and iat and nbf) keys.
		// Unmarshal raw message to validate against.
		v := new(Claims)
		err = json.Unmarshal(*c, v)
		if err == nil {
			return validateClaims(ctx, v)
		}
	default:
		ctx.Values().Set(needsValidationContextKey, struct{}{})
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

// VerifyToken verifies (and decrypts) the request token,
// it also validates and binds the parsed token's claims to the "claimsPtr" (destination).
// It does return a nil error on success.
func (j *JWT) VerifyToken(ctx *context.Context, claimsPtr interface{}) error {
	var token string

	for _, extract := range j.Extractors {
		if token = extract(ctx); token != "" {
			break // ok we found it.
		}
	}

	if token == "" {
		return ErrMissing
	}

	var (
		parsedToken *jwt.JSONWebToken
		err         error
	)

	if j.DecriptionKey != nil {
		t, cerr := jwt.ParseSignedAndEncrypted(token)
		if cerr != nil {
			return cerr
		}

		parsedToken, err = t.Decrypt(j.DecriptionKey)
	} else {
		parsedToken, err = jwt.ParseSigned(token)
	}
	if err != nil {
		return err
	}

	if err = parsedToken.Claims(j.VerificationKey, claimsPtr); err != nil {
		return err
	}

	return validateClaims(ctx, claimsPtr)
}

const (
	// ClaimsContextKey is the context key which the jwt claims are stored from the `Verify` method.
	ClaimsContextKey          = "iris.jwt.claims"
	needsValidationContextKey = "iris.jwt.claims.unvalidated"
)

// Verify is a middleware. It verifies and optionally decrypts an incoming request token.
// It does write a 401 unauthorized status code if verification or decryption failed.
// It calls the `ctx.Next` on verified requests.
//
// See `VerifyToken` instead to verify, decrypt, validate and acquire the claims at once.
//
// A call of `ReadClaims` is required to validate and acquire the jwt claims
// on the next request.
func (j *JWT) Verify(ctx *context.Context) {
	var raw json.RawMessage
	if err := j.VerifyToken(ctx, &raw); err != nil {
		ctx.StopWithStatus(401)
		return
	}

	ctx.Values().Set(ClaimsContextKey, raw)
	ctx.Next()
}

// ReadClaims binds the "claimsPtr" (destination)
// to the verified (and decrypted) claims.
// The `Verify` method should be called  first (registered as middleware).
func ReadClaims(ctx *context.Context, claimsPtr interface{}) error {
	v := ctx.Values().Get(ClaimsContextKey)
	if v == nil {
		return ErrMissing
	}

	raw, ok := v.(json.RawMessage)
	if !ok {
		return ErrMissing
	}

	err := json.Unmarshal(raw, claimsPtr)
	if err != nil {
		return err
	}

	if !IsValidated(ctx) {
		// If already validated on `Verify/VerifyToken`
		// then no need to perform the check again.
		ctx.Values().Remove(needsValidationContextKey)
		return validateClaims(ctx, claimsPtr)
	}

	return nil
}

// Get returns and validates (if not already) the claims
// stored on request context's values storage.
//
// Should be used instead of the `ReadClaims` method when
// a custom verification middleware was registered (see the `Verify` method for an example).
//
// Usage:
// j := jwt.New(...)
// [...]
// app.Use(func(ctx iris.Context) {
//	var claims CustomClaims_or_jwt.Claims
// 	if err := j.VerifyToken(ctx, &claims); err != nil {
// 		ctx.StopWithStatus(iris.StatusUnauthorized)
// 		return
// 	}
//
// 	ctx.Values().Set(jwt.ClaimsContextKey, claims)
// 	ctx.Next()
// })
// [...]
// app.Post("/restricted", func(ctx iris.Context){
//	v, err := jwt.Get(ctx)
//  [handle error...]
//  claims,ok := v.(CustomClaims_or_jwt.Claims)
//  if !ok {
// 	  [do you support more than one type of claims? Handle here]
// 	}
//  [use claims...]
// })
func Get(ctx *context.Context) (interface{}, error) {
	claims := ctx.Values().Get(ClaimsContextKey)
	if claims == nil {
		return nil, ErrMissing
	}

	if !IsValidated(ctx) {
		ctx.Values().Remove(needsValidationContextKey)
		err := validateClaims(ctx, claims)
		if err != nil {
			return nil, err
		}
	}

	return claims, nil
}
