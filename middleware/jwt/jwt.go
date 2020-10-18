package jwt

import (
	"crypto"
	"encoding/json"
	"fmt"
	"os"
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
//
// For an easy use look the `HMAC` package-level function
// and the its `NewUser` and `VerifyUser` methods.
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

	// Blocklist holds the invalidated-by-server tokens (that are not yet expired).
	// It is not initialized by default.
	// Initialization Usage:
	// j.InitDefaultBlocklist()
	// OR
	// j.Blocklist = jwt.NewBlocklist(gcEveryDuration)
	// Usage:
	// - ctx.Logout()
	// - j.Invalidate(ctx)
	Blocklist Blocklist
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
//
// Example at:
// https://github.com/kataras/iris/tree/master/_examples/auth/jwt/overview/main.go
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

// InitDefaultBlocklist initializes the Blocklist field with the default in-memory implementation.
// Should be called on jwt middleware creation-time,
// after this, the developer can use the Context.Logout method
// to invalidate a verified token by the server-side.
func (j *JWT) InitDefaultBlocklist() {
	gcEvery := 30 * time.Minute
	if j.MaxAge > 0 {
		gcEvery = j.MaxAge
	}
	j.Blocklist = NewBlocklist(gcEvery)
}

// ExpiryMap adds the expiration based on the "maxAge" to the "claims" map.
// It's called automatically on `Token` method.
func ExpiryMap(maxAge time.Duration, claims context.Map) {
	now := time.Now()
	if claims["exp"] == nil {
		claims["exp"] = NewNumericDate(now.Add(maxAge))
	}

	if claims["iat"] == nil {
		claims["iat"] = NewNumericDate(now)
	}
}

// Token generates and returns a new token string.
// See `VerifyToken` too.
func (j *JWT) Token(claims interface{}) (string, error) {
	return j.token(j.MaxAge, claims)
}

func (j *JWT) token(maxAge time.Duration, claims interface{}) (string, error) {
	if claims == nil {
		return "", ErrInvalidKey
	}

	c, nErr := normalize(claims)
	if nErr != nil {
		return "", nErr
	}

	ExpiryMap(maxAge, c)

	var (
		token string
		err   error
	)
	// jwt.Builder and jwt.NestedBuilder contain same methods but they are not the same.
	//
	// Note that the .Claims method there, converts a Struct to a map under the hoods.
	// That means that we will not have any performance cost
	// if we do it by ourselves and pass always a Map there.
	// That gives us the option to allow user to pass ANY go struct
	// and we can add the "exp", "nbf", "iat" map values by ourselves
	// based on the j.MaxAge.
	// (^ done, see normalize, all methods are
	// changed to accept totally custom types, no need to embed the standard Claims anymore).
	if j.DecriptionKey != nil {
		token, err = jwt.SignedAndEncrypted(j.Signer, j.Encrypter).Claims(c).CompactSerialize()
	} else {
		token, err = jwt.Signed(j.Signer).Claims(c).CompactSerialize()
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
func (j *JWT) WriteToken(ctx *context.Context, claims interface{}) error {
	token, err := j.Token(claims)
	if err != nil {
		ctx.StatusCode(500)
		return err
	}

	_, err = ctx.WriteString(token)
	return err
}

// VerifyToken verifies (and decrypts) the request token,
// it also validates and binds the parsed token's claims to the "claimsPtr" (destination).
//
// The last, variadic, input argument is optionally, if provided then the
// parsed claims must match the expectations;
// e.g. Audience, Issuer, ID, Subject.
// See `ExpectXXX` package-functions for details.
func (j *JWT) VerifyToken(ctx *context.Context, claimsPtr interface{}, expectations ...Expectation) (*TokenInfo, error) {
	token := j.RequestToken(ctx)
	return j.VerifyTokenString(ctx, token, claimsPtr, expectations...)
}

// VerifyRefreshToken like the `VerifyToken` but it verifies a refresh token one instead.
// If the implementation does not fill the application's requirements,
// you can ignore this method and still use the `VerifyToken` for refresh tokens too.
//
// This method adds the ExpectRefreshToken expectation and it
// tries to read the refresh token from raw body or,
// if content type was application/json, then it extracts the token
// from the JSON request body's {"refresh_token": "$token"} key.
func (j *JWT) VerifyRefreshToken(ctx *context.Context, claimsPtr interface{}, expectations ...Expectation) (*TokenInfo, error) {
	token := j.RequestToken(ctx)
	if token == "" {
		var tokenPair TokenPair // read "refresh_token" from JSON.
		if ctx.GetContentTypeRequested() == context.ContentJSONHeaderValue {
			ctx.ReadJSON(&tokenPair) // ignore error.
			token = tokenPair.RefreshToken
			if token == "" {
				return nil, ErrMissing
			}
		} else {
			ctx.ReadBody(&token)
		}
	}

	return j.VerifyTokenString(ctx, token, claimsPtr, append(expectations, ExpectRefreshToken)...)
}

// RequestToken extracts the token from the request.
func (j *JWT) RequestToken(ctx *context.Context) (token string) {
	for _, extract := range j.Extractors {
		if token = extract(ctx); token != "" {
			break // ok we found it.
		}
	}

	return
}

// TokenSetter is an interface which if implemented
// the extracted, verified, token is stored to the object.
type TokenSetter interface {
	SetToken(token string)
}

// TokenInfo holds the standard token information may required
// for further actions.
// This structure is mostly useful when the developer's go structure
// does not hold the standard jwt fields (e.g. "exp")
// but want access to the parsed token which contains those fields.
// Inside the middleware, it is used to invalidate tokens through server-side, see `Invalidate`.
type TokenInfo struct {
	RequestToken string      // The request token.
	Claims       Claims      // The standard JWT parsed fields from the request Token.
	Value        interface{} // The pointer to the end-developer's custom claims structure (see `Get`).
}

const tokenInfoContextKey = "iris.jwt.token"

// Get returns the verified developer token claims.
//
//
// Usage:
// j := jwt.New(...)
// app.Use(j.Verify(func() interface{} { return new(CustomClaims) }))
// app.Post("/restricted", func(ctx iris.Context){
//  claims := jwt.Get(ctx).(*CustomClaims)
//  [use claims...]
// })
//
// Note that there is one exception, if the value was a pointer
// to a map[string]interface{}, it returns the map itself so it can be
// accessible directly without the requirement of unwrapping it, e.g.
// j.Verify(func() interface{} {
// 	return &iris.Map{}
// }
// [...]
// 	claims := jwt.Get(ctx).(iris.Map)
func Get(ctx *context.Context) interface{} {
	if tok := GetTokenInfo(ctx); tok != nil {
		switch v := tok.Value.(type) {
		case *context.Map:
			return *v
		case *json.RawMessage:
			// This is useful when we can accept more than one
			// type of JWT token in the same request path,
			// but we also want to keep type safety.
			// Usage:
			// type myClaims struct { Roles []string `json:"roles"`}
			// v := jwt.Get(ctx)
			// var claims myClaims
			// jwt.Unmarshal(v, &claims)
			// [...claims.Roles]
			return *v
		default:
			return v
		}
	}

	return nil
}

// GetTokenInfo returns the verified token's information.
func GetTokenInfo(ctx *context.Context) *TokenInfo {
	if v := ctx.Values().Get(tokenInfoContextKey); v != nil {
		if t, ok := v.(*TokenInfo); ok {
			return t
		}
	}

	return nil
}

// Invalidate invalidates a verified JWT token.
// It adds the request token, retrieved by Verify methods, to the block list.
// Next request will be blocked, even if the token was not yet expired.
// This method can be used when the client-side does not clear the token
// on a user logout operation.
//
// Note: the Blocklist should be initialized before serve-time: j.InitDefaultBlocklist().
func (j *JWT) Invalidate(ctx *context.Context) {
	if j.Blocklist == nil {
		ctx.Application().Logger().Debug("jwt.Invalidate: Blocklist is nil")
		return
	}

	tokenInfo := GetTokenInfo(ctx)
	if tokenInfo == nil {
		return
	}

	j.Blocklist.Set(tokenInfo.RequestToken, tokenInfo.Claims.Expiry.Time())
}

// VerifyTokenString verifies and unmarshals an extracted request token to "dest" destination.
// The last variadic input indicates any further validations against the verified token claims.
// If the given "dest" is a valid context.User then ctx.User() will return it.
// If the token is missing an `ErrMissing` is returned.
// If the incoming token was expired an `ErrExpired` is returned.
// If the incoming token was blocked by the server an `ErrBlocked` is returned.
func (j *JWT) VerifyTokenString(ctx *context.Context, token string, dest interface{}, expectations ...Expectation) (*TokenInfo, error) {
	if token == "" {
		return nil, ErrMissing
	}

	var (
		parsedToken *jwt.JSONWebToken
		err         error
	)

	if j.DecriptionKey != nil {
		t, cerr := jwt.ParseSignedAndEncrypted(token)
		if cerr != nil {
			return nil, cerr
		}

		parsedToken, err = t.Decrypt(j.DecriptionKey)
	} else {
		parsedToken, err = jwt.ParseSigned(token)
	}
	if err != nil {
		return nil, err
	}

	var (
		claims       Claims
		tokenMaxAger tokenWithMaxAge
	)

	var (
		ignoreDest      = dest == nil
		ignoreVarClaims bool
	)
	if !ignoreDest { // if dest was not nil, check if the dest is already a standard claims pointer.
		_, ignoreVarClaims = dest.(*Claims)
	}

	// Ensure read the standard claims one  if dest was Claims or was nil.
	// (it wont break anything if we unmarshal them twice though, we just do it for performance reasons).
	var pointers = []interface{}{&tokenMaxAger}
	if !ignoreDest {
		pointers = append(pointers, dest)
	}
	if !ignoreVarClaims {
		pointers = append(pointers, &claims)
	}
	if err = parsedToken.Claims(j.VerificationKey, pointers...); err != nil {
		return nil, err
	}

	// Set the std claims, if missing from receiver so the expectations and validation still work.
	if ignoreVarClaims {
		claims = *dest.(*Claims)
	} else if ignoreDest {
		dest = &claims
	}

	expectMaxAge := j.MaxAge

	// Build the Expected value.
	expected := Expected{}
	for _, e := range expectations {
		if e != nil {
			// expection can be used as a field validation too (see MeetRequirements).
			if err = e(&expected, dest); err != nil {
				if err == ErrExpectRefreshToken {
					if tokenMaxAger.MaxAge > 0 {
						// If max age exists, grab it and compare it later.
						// Otherwise fire the ErrExpectRefreshToken.
						expectMaxAge = tokenMaxAger.MaxAge
						continue
					}
				}
				return nil, err
			}
		}
	}

	gotMaxAge := getMaxAge(claims)
	if !compareMaxAge(expectMaxAge, gotMaxAge) {
		// Additional check to automatically invalidate
		// any previous jwt maxAge setting change.
		// In-short, if the time.Now().Add j.MaxAge
		// does not match the "iat" (issued at) then we invalidate the token.
		return nil, ErrInvalidMaxAge
	}

	// For other standard JWT claims fields such as "exp"
	// The developer can just add a field of Expiry *NumericDate `json:"exp"`
	// and will be filled by the parsed token automatically.
	// No need for more interfaces.

	err = validateClaims(ctx, dest, claims, expected)
	if err != nil {
		if err == ErrExpired {
			// If token was expired remove it from the block list.
			if j.Blocklist != nil {
				j.Blocklist.Del(token)
			}
		}

		return nil, err
	}

	if j.Blocklist != nil {
		// If token exists in the block list, then stop here.
		if j.Blocklist.Has(token) {
			return nil, ErrBlocked
		}
	}

	if !ignoreDest {
		if ut, ok := dest.(TokenSetter); ok {
			// The u.Token is empty even if we set it and export it on JSON structure.
			// Set it manually.
			ut.SetToken(token)
		}
	}

	// Set the information.
	tokenInfo := &TokenInfo{
		RequestToken: token,
		Claims:       claims,
		Value:        dest,
	}

	return tokenInfo, nil
}

// TokenPair holds the access token and refresh token response.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type tokenWithMaxAge struct {
	// Useful to separate access from refresh tokens.
	// Can be used to by-pass the internal check of expected
	// MaxAge setting to match the token's received max age too.
	MaxAge time.Duration `json:"tokenMaxAge"`
}

// TokenPair generates a token pair of access and refresh tokens.
// The first two arguments required for the refresh token
// and the last one is the claims for the access token one.
func (j *JWT) TokenPair(refreshMaxAge time.Duration, refreshClaims interface{}, accessClaims interface{}) (TokenPair, error) {
	if refreshMaxAge <= j.MaxAge {
		return TokenPair{}, fmt.Errorf("refresh max age should be bigger than access token's one[%d - %d]", refreshMaxAge, j.MaxAge)
	}

	accessToken, err := j.Token(accessClaims)
	if err != nil {
		return TokenPair{}, err
	}

	c, err := normalize(refreshClaims)
	if err != nil {
		return TokenPair{}, err
	}
	if c == nil {
		c = make(context.Map)
	}
	// need to validate against its value instead of the setting's one (see `VerifyTokenString`).
	c["tokenMaxAge"] = refreshMaxAge

	refreshToken, err := j.token(refreshMaxAge, c)
	if err != nil {
		return TokenPair{}, nil
	}

	pair := TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return pair, nil
}

// Verify returns a middleware which
// decrypts an incoming request token to the result of the given "newPtr".
// It does write a 401 unauthorized status code if verification or decryption failed.
// It calls the `ctx.Next` on verified requests.
//
// Iit unmarshals the token to the specific type returned from the given "newPtr" function.
// It sets the Context User and User's Token too. So the next handler(s)
// of the same chain can access the User through a `Context.User()` call.
//
// Note unlike `VerifyToken`, this method automatically protects
// the claims with JSON required tags (see `MeetRequirements` Expection).
//
// On verified tokens:
// - The information can be retrieved through `Get` and `GetTokenInfo` functions.
// - User is set if the newPtr returns a valid Context User
// - The Context Logout method is set if Blocklist was initialized
// Any error is captured to the Context,
// which can be retrieved by a `ctx.GetErr()` call.
//
// See `VerifyJSON` too.
func (j *JWT) Verify(newPtr func() interface{}, expections ...Expectation) context.Handler {
	if newPtr == nil {
		newPtr = func() interface{} {
			// Return a map here as the default type one,
			// as it does allow .Get callers to access its fields with ease
			// (although, I always recommend using structs for type-safety and
			// also they can accept a required tag option too).
			return &context.Map{}
		}
	}

	expections = append(expections, MeetRequirements(newPtr()))

	return func(ctx *context.Context) {
		ptr := newPtr()

		tokenInfo, err := j.VerifyToken(ctx, ptr, expections...)
		if err != nil {
			ctx.Application().Logger().Debugf("iris.jwt.Verify: %v", err)
			ctx.StopWithError(401, context.PrivateError(err))
			return
		}

		if u, ok := ptr.(context.User); ok {
			ctx.SetUser(u)
		}

		if j.Blocklist != nil {
			ctx.SetLogoutFunc(j.Invalidate)
		}

		ctx.Values().Set(tokenInfoContextKey, tokenInfo)
		ctx.Next()
	}
}

// VerifyMap is a shortcut of Verify with a function which will bind
// the claims to a standard Go map[string]interface{}.
func (j *JWT) VerifyMap(exceptions ...Expectation) context.Handler {
	return j.Verify(func() interface{} {
		return &context.Map{}
	})
}

// VerifyJSON works like `Verify` but instead it
// binds its "newPtr" function to return a raw JSON message.
// It does NOT read the token from JSON by itself,
// to do that add the `FromJSON` to the Token Extractors.
// It's used to bind the claims in any value type on the next handler.
//
// This allows the caller to bind this JSON message to any Go structure (or map).
// This is useful when we can accept more than one
// type of JWT token in the same request path,
// but we also want to keep type safety.
// Usage:
// app.Use(jwt.VerifyJSON())
// Inside a route Handler:
// claims := struct { Roles []string `json:"roles"`}{}
// jwt.ReadJSON(ctx, &claims)
// ...access to claims.Roles as []string
func (j *JWT) VerifyJSON(expections ...Expectation) context.Handler {
	return j.Verify(func() interface{} {
		return new(json.RawMessage)
	})
}

// ReadJSON is a helper which binds "claimsPtr" to the
// raw JSON token claims.
// Use inside the handlers when `VerifyJSON()` middleware was registered.
func ReadJSON(ctx *context.Context, claimsPtr interface{}) error {
	v := Get(ctx)
	if v == nil {
		return ErrMissing
	}
	data, ok := v.(json.RawMessage)
	if !ok {
		return ErrMissing
	}
	return Unmarshal(data, claimsPtr)
}

// NewUser returns a new User based on the given "opts".
// The caller can modify the User until its `GetToken` is called.
func (j *JWT) NewUser(opts ...UserOption) *User {
	u := &User{
		j: j,
		SimpleUser: &context.SimpleUser{
			Authorization: "IRIS_JWT_USER", // Used to separate a refresh token with a user/access one too.
			Features: []context.UserFeature{
				context.TokenFeature,
			},
		},
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}

// VerifyUser works like the `Verify` method but instead
// it unmarshals the token to the specific User type.
// It sets the Context User too. So the next handler(s)
// of the same chain can access the User through a `Context.User()` call.
func (j *JWT) VerifyUser() context.Handler {
	return j.Verify(func() interface{} {
		return new(User)
	})
}
