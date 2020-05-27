package jwt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"

	"github.com/square/go-jose/v3"
	"github.com/square/go-jose/v3/jwt"
)

func init() {
	context.SetHandlerName("iris/middleware/jwt.*", "iris.jwt")
}

// TokenExtractor is a function that takes a context as input and returns
// a token. An empty string should be returned if no token found
// without additional information.
type TokenExtractor func(context.Context) string

// FromHeader is a token extractor.
// It reads the token from the Authorization request header of form:
// Authorization: "Bearer {token}".
func FromHeader(ctx context.Context) string {
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
func FromQuery(ctx context.Context) string {
	return ctx.URLParam("token")
}

// FromJSON is a token extractor.
// Reads a json request body and extracts the json based on the given field.
// The request content-type should contain the: application/json header value, otherwise
// this method will not try to read and consume the body.
func FromJSON(jsonKey string) TokenExtractor {
	return func(ctx context.Context) string {
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
	// It is set on `WithExpiration` method.
	Encrypter jose.Encrypter
	// DecriptionKey is used to decrypt the token (private key)
	DecriptionKey interface{}
}

// Random returns a new `JWT` instance
// with in-memory generated rsa256 signing and encryption keys (development).
// It panics on errors. Next server ran will invalidate all request tokens.
//
// Use the `New` package-level function for production use.
func Random(maxAge time.Duration) *JWT {
	sigKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	j, err := New(maxAge, RS256, sigKey)
	if err != nil {
		panic(err)
	}

	encKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	err = j.WithEncryption(A128CBCHS256, RSA15, encKey)
	if err != nil {
		panic(err)
	}

	return j
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
// 3. Pass the []byte result to the `MustParseRSAPrivateKey(contents, password)` package-level helper
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
	claims.Expiry = jwt.NewNumericDate(now.Add(maxAge))
	claims.IssuedAt = jwt.NewNumericDate(now)
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

// WriteToken is a helper which just generates(calls the `Token` method) and writes
// a new token to the client in plain text format.
//
// Use the `Token` method to get a new generated token raw string value.
func (j *JWT) WriteToken(ctx context.Context, claims interface{}) error {
	token, err := j.Token(claims)
	if err != nil {
		ctx.StatusCode(500)
		return err
	}

	_, err = ctx.WriteString(token)
	return err
}

var (
	// ErrTokenMissing when token cannot be extracted from the request.
	ErrTokenMissing = errors.New("token is missing")
	// ErrTokenInvalid when incoming token is invalid.
	ErrTokenInvalid = errors.New("token is invalid")
	// ErrTokenExpired when incoming token has expired.
	ErrTokenExpired = errors.New("token has expired")
)

type (
	claimsValidator interface {
		ValidateWithLeeway(e jwt.Expected, leeway time.Duration) error
	}
	claimsAlternativeValidator interface {
		Validate() error
	}
)

// IsValidated reports whether a token is already validated through
// `VerifyToken`. It returns true when the claims are compatible
// validators: a `Claims` value or a value that implements the `Validate() error` method.
func IsValidated(ctx context.Context) bool { // see the `ReadClaims`.
	return ctx.Values().Get(needsValidationContextKey) == nil
}

func validateClaims(ctx context.Context, claimsPtr interface{}) (err error) {
	switch claims := claimsPtr.(type) {
	case claimsValidator:
		err = claims.ValidateWithLeeway(jwt.Expected{Time: time.Now()}, 0)
	case claimsAlternativeValidator:
		err = claims.Validate()
	default:
		ctx.Values().Set(needsValidationContextKey, struct{}{})
	}

	if err != nil {
		if err == jwt.ErrExpired {
			return ErrTokenExpired
		}
	}

	return err
}

// VerifyToken verifies (and decrypts) the request token,
// it also validates and binds the parsed token's claims to the "claimsPtr" (destination).
// It does return a nil error on success.
func (j *JWT) VerifyToken(ctx context.Context, claimsPtr interface{}) error {
	var token string

	for _, extract := range j.Extractors {
		if token = extract(ctx); token != "" {
			break // ok we found it.
		}
	}

	if token == "" {
		return ErrTokenMissing
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
		return ErrTokenInvalid
	}

	if err = parsedToken.Claims(j.VerificationKey, claimsPtr); err != nil {
		return ErrTokenInvalid
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
func (j *JWT) Verify(ctx context.Context) {
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
func ReadClaims(ctx context.Context, claimsPtr interface{}) error {
	v := ctx.Values().Get(ClaimsContextKey)
	if v == nil {
		return ErrTokenMissing
	}

	raw, ok := v.(json.RawMessage)
	if !ok {
		return ErrTokenMissing
	}

	err := json.Unmarshal(raw, claimsPtr)
	if err != nil {
		return err
	}

	// If already validated on VerifyToken (a claimsValidator/claimsAlternativeValidator)
	// then no need to perform the check again.
	if !IsValidated(ctx) {
		ctx.Values().Remove(needsValidationContextKey)
		return validateClaims(ctx, claimsPtr)
	}

	return nil
}
