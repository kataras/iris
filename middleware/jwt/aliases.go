package jwt

import "github.com/kataras/jwt"

// Error values.
var (
	ErrBlocked           = jwt.ErrBlocked
	ErrDecrypt           = jwt.ErrDecrypt
	ErrExpected          = jwt.ErrExpected
	ErrExpired           = jwt.ErrExpired
	ErrInvalidKey        = jwt.ErrInvalidKey
	ErrIssuedInTheFuture = jwt.ErrIssuedInTheFuture
	ErrMissing           = jwt.ErrMissing
	ErrMissingKey        = jwt.ErrMissingKey
	ErrNotValidYet       = jwt.ErrNotValidYet
	ErrTokenAlg          = jwt.ErrTokenAlg
	ErrTokenForm         = jwt.ErrTokenForm
	ErrTokenSignature    = jwt.ErrTokenSignature
)

// Signature algorithms.
var (
	EdDSA = jwt.EdDSA
	HS256 = jwt.HS256
	HS384 = jwt.HS384
	HS512 = jwt.HS512
	RS256 = jwt.RS256
	RS384 = jwt.RS384
	RS512 = jwt.RS512
	ES256 = jwt.ES256
	ES384 = jwt.ES384
	ES512 = jwt.ES512
	PS256 = jwt.PS256
	PS384 = jwt.PS384
	PS512 = jwt.PS512
)

// Signature algorithm helpers.
var (
	MustLoadHMAC         = jwt.MustLoadHMAC
	LoadHMAC             = jwt.LoadHMAC
	MustLoadRSA          = jwt.MustLoadRSA
	LoadPrivateKeyRSA    = jwt.LoadPrivateKeyRSA
	LoadPublicKeyRSA     = jwt.LoadPublicKeyRSA
	ParsePrivateKeyRSA   = jwt.ParsePrivateKeyRSA
	ParsePublicKeyRSA    = jwt.ParsePublicKeyRSA
	MustLoadECDSA        = jwt.MustLoadECDSA
	LoadPrivateKeyECDSA  = jwt.LoadPrivateKeyECDSA
	LoadPublicKeyECDSA   = jwt.LoadPublicKeyECDSA
	ParsePrivateKeyECDSA = jwt.ParsePrivateKeyECDSA
	ParsePublicKeyECDSA  = jwt.ParsePublicKeyECDSA
	MustLoadEdDSA        = jwt.MustLoadEdDSA
	LoadPrivateKeyEdDSA  = jwt.LoadPrivateKeyEdDSA
	LoadPublicKeyEdDSA   = jwt.LoadPublicKeyEdDSA
	ParsePrivateKeyEdDSA = jwt.ParsePrivateKeyEdDSA
	ParsePublicKeyEdDSA  = jwt.ParsePublicKeyEdDSA
)

// Type alises for the underline jwt package.
type (
	// Alg is the signature algorithm interface alias.
	Alg = jwt.Alg
	// Claims represents the standard claim values (as specified in RFC 7519).
	Claims = jwt.Claims
	// Expected is a TokenValidator which performs simple checks
	// between standard claims values.
	//
	// Usage:
	//  expecteed := jwt.Expected{
	//	  Issuer: "my-app",
	//  }
	//  verifiedToken, err := verifier.Verify(..., expected)
	Expected = jwt.Expected

	// TokenValidator is the token validator interface alias.
	TokenValidator = jwt.TokenValidator
	// VerifiedToken is the type alias for the verfieid token type,
	// the result of the VerifyToken function.
	VerifiedToken = jwt.VerifiedToken
	// SignOption used to set signing options at Sign function.
	SignOption = jwt.SignOption
	// TokenPair is just a helper structure which holds both access and refresh tokens.
	TokenPair = jwt.TokenPair
)

// Encryption algorithms.
var (
	GCM = jwt.GCM
	// Helper to generate random key,
	// can be used to generate hmac signature key and GCM+AES for testing.
	MustGenerateRandom = jwt.MustGenerateRandom
)

var (
	// Leeway adds validation for a leeway expiration time.
	// If the token was not expired then a comparison between
	// this "leeway" and the token's "exp" one is expected to pass instead (now+leeway > exp).
	// Example of use case: disallow tokens that are going to be expired in 3 seconds from now,
	// this is useful to make sure that the token is valid when the when the user fires a database call for example.
	// Usage:
	//  verifiedToken, err := verifier.Verify(..., jwt.Leeway(5*time.Second))
	Leeway = jwt.Leeway
	// MaxAge is a SignOption to set the expiration "exp", "iat" JWT standard claims.
	// Can be passed as last input argument of the `Sign` function.
	//
	// If maxAge > second then sets expiration to the token.
	// It's a helper field to set the "exp" and "iat" claim values.
	// Usage:
	//  signer.Sign(..., jwt.MaxAge(15*time.Minute))
	MaxAge = jwt.MaxAge

	// ID is a shurtcut to set jwt ID on Sign.
	ID = func(id string) jwt.SignOptionFunc {
		return func(c *Claims) {
			c.ID = id
		}
	}
)

// Shortcuts for Signing and Verifying.
var (
	Verify               = jwt.Verify
	VerifyEncryptedToken = jwt.VerifyEncrypted
	Sign                 = jwt.Sign
	SignEncrypted        = jwt.SignEncrypted
)
