package jwt

import (
	"fmt"
	"time"

	"github.com/kataras/jwt"
)

// Signer holds common options to sign and generate a token.
// Its Sign method can be used to generate a token which can be sent to the client.
// Its NewTokenPair can be used to construct a token pair (access_token, refresh_token).
//
// It does not support JWE, JWK.
type Signer struct {
	Alg Alg
	Key interface{}

	// MaxAge to set "exp" and "iat".
	// Recommended value for access tokens: 15 minutes.
	// Defaults to 0, no limit.
	MaxAge  time.Duration
	Options []SignOption

	Encrypt func([]byte) ([]byte, error)
}

// NewSigner accepts the signature algorithm among with its (private or shared) key
// and the max life time duration of generated tokens and returns a JWT signer.
// See its Sign method.
//
// Usage:
//
//  signer := NewSigner(HS256, secret, 15*time.Minute)
//  token, err := signer.Sign(userClaims{Username: "kataras"})
func NewSigner(signatureAlg Alg, signatureKey interface{}, maxAge time.Duration) *Signer {
	if signatureAlg == HS256 {
		// A tiny helper if the end-developer uses string instead of []byte for hmac keys.
		if k, ok := signatureKey.(string); ok {
			signatureKey = []byte(k)
		}
	}

	s := &Signer{
		Alg:    signatureAlg,
		Key:    signatureKey,
		MaxAge: maxAge,
	}

	if maxAge > 0 {
		s.Options = []SignOption{MaxAge(maxAge)}
	}

	return s
}

// WithEncryption enables AES-GCM payload-only decryption.
func (s *Signer) WithEncryption(key, additionalData []byte) *Signer {
	encrypt, _, err := jwt.GCM(key, additionalData)
	if err != nil {
		panic(err) // important error before serve, stop everything.
	}

	s.Encrypt = encrypt
	return s
}

// Sign generates a new token based on the given "claims" which is valid up to "s.MaxAge".
func (s *Signer) Sign(claims interface{}, opts ...SignOption) ([]byte, error) {
	if len(opts) > 0 {
		opts = append(opts, s.Options...)
	} else {
		opts = s.Options
	}

	return SignEncrypted(s.Alg, s.Key, s.Encrypt, claims, opts...)
}

// NewTokenPair accepts the access and refresh claims plus the life time duration for the refresh token
// and generates a new token pair which can be sent to the client.
// The same token pair can be json-decoded.
func (s *Signer) NewTokenPair(accessClaims interface{}, refreshClaims interface{}, refreshMaxAge time.Duration, accessOpts ...SignOption) (TokenPair, error) {
	if refreshMaxAge <= s.MaxAge {
		return TokenPair{}, fmt.Errorf("refresh max age should be bigger than access token's one[%d - %d]", refreshMaxAge, s.MaxAge)
	}

	accessToken, err := s.Sign(accessClaims, accessOpts...)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := Sign(s.Alg, s.Key, refreshClaims, MaxAge(refreshMaxAge))
	if err != nil {
		return TokenPair{}, err
	}

	tokenPair := jwt.NewTokenPair(accessToken, refreshToken)
	return tokenPair, nil
}
