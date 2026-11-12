package oauth2

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
)

// secureCompare reports whether a and b are equal, using a constant-time
// comparison to avoid leaking their length-relationship or content through
// response timing. Use it for any secret comparison (client secrets, tokens).
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// verifyPKCES256 reports whether verifier satisfies an S256 code_challenge
// (RFC 7636). The verifier must be 43 to 128 characters.
func verifyPKCES256(challenge, verifier string) bool {
	if len(verifier) < 43 || len(verifier) > 128 || challenge == "" {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	got := base64.RawURLEncoding.EncodeToString(sum[:])
	return secureCompare(got, challenge)
}

// generateSecret generates a random and secure secret key.
func generateSecret(size int) (string, error) {
	b, err := generateRandomBytes(size)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generateToken returns a cryptographically random, URL-safe token with nBytes
// of entropy, encoded as base64url without padding. Use it for opaque codes and
// tokens (authorization codes, device codes) that are only read by programs.
func generateToken(nBytes int) (string, error) {
	b, err := generateRandomBytes(nBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCode generates a secure random numeric code.
func generateCode(length int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", length, n), nil
}

// GenerateSharedKey returns a base64-encoded (standard encoding) random 16-byte key
// (128-bit AES key).
// This is convenient for returning as JSON in an HTTP response.
func GenerateSharedKey() (string, error) {
	key, err := generateRandomBytes(16)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func generateRandomBytes(size int) ([]byte, error) {
	salt := make([]byte, size) // 16 byte size generates 128-bit AES key.
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// skip the base64 step.
	return salt, nil
}
