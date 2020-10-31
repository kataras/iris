package jwt

import (
	"fmt"
	"time"

	"github.com/kataras/jwt"
)

type Signer struct {
	Alg Alg
	Key interface{}

	MaxAge time.Duration

	Encrypt func([]byte) ([]byte, error)
}

func NewSigner(signatureAlg Alg, signatureKey interface{}, maxAge time.Duration) *Signer {
	return &Signer{
		Alg:    signatureAlg,
		Key:    signatureKey,
		MaxAge: maxAge,
	}
}

// WithEncryption enables AES-GCM payload decryption.
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
	return SignEncrypted(s.Alg, s.Key, s.Encrypt, claims, append([]SignOption{MaxAge(s.MaxAge)}, opts...)...)
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
