package jwt

import (
	"github.com/square/go-jose/v3"
	"github.com/square/go-jose/v3/jwt"
)

type (
	// Claims represents public claim values (as specified in RFC 7519).
	Claims = jwt.Claims
	// Audience represents the recipients that the token is intended for.
	Audience = jwt.Audience
	// NumericDate represents date and time as the number of seconds since the
	// epoch, including leap seconds. Non-integer values can be represented
	// in the serialized format, but we round to the nearest second.
	NumericDate = jwt.NumericDate
)

var (
	// NewNumericDate constructs NumericDate from time.Time value.
	NewNumericDate = jwt.NewNumericDate
)

type (
	// KeyAlgorithm represents a key management algorithm.
	KeyAlgorithm = jose.KeyAlgorithm

	// SignatureAlgorithm represents a signature (or MAC) algorithm.
	SignatureAlgorithm = jose.SignatureAlgorithm

	// ContentEncryption represents a content encryption algorithm.
	ContentEncryption = jose.ContentEncryption
)

// Key management algorithms.
const (
	ED25519          = jose.ED25519
	RSA15            = jose.RSA1_5
	RSAOAEP          = jose.RSA_OAEP
	RSAOAEP256       = jose.RSA_OAEP_256
	A128KW           = jose.A128KW
	A192KW           = jose.A192KW
	A256KW           = jose.A256KW
	DIRECT           = jose.DIRECT
	ECDHES           = jose.ECDH_ES
	ECDHESA128KW     = jose.ECDH_ES_A128KW
	ECDHESA192KW     = jose.ECDH_ES_A192KW
	ECDHESA256KW     = jose.ECDH_ES_A256KW
	A128GCMKW        = jose.A128GCMKW
	A192GCMKW        = jose.A192GCMKW
	A256GCMKW        = jose.A256GCMKW
	PBES2HS256A128KW = jose.PBES2_HS256_A128KW
	PBES2HS384A192KW = jose.PBES2_HS384_A192KW
	PBES2HS512A256KW = jose.PBES2_HS512_A256KW
)

// Signature algorithms.
const (
	EdDSA = jose.EdDSA
	HS256 = jose.HS256
	HS384 = jose.HS384
	HS512 = jose.HS512
	RS256 = jose.RS256
	RS384 = jose.RS384
	RS512 = jose.RS512
	ES256 = jose.ES256
	ES384 = jose.ES384
	ES512 = jose.ES512
	PS256 = jose.PS256
	PS384 = jose.PS384
	PS512 = jose.PS512
)

// Content encryption algorithms.
const (
	A128CBCHS256 = jose.A128CBC_HS256
	A192CBCHS384 = jose.A192CBC_HS384
	A256CBCHS512 = jose.A256CBC_HS512
	A128GCM      = jose.A128GCM
	A192GCM      = jose.A192GCM
	A256GCM      = jose.A256GCM
)
