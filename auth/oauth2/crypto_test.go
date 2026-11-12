package oauth2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSecret(t *testing.T) {
	secret, err := generateSecret(32)
	assert.NoError(t, err)
	assert.Len(t, secret, 64) // 32 bytes in hex representation
}

func TestVerifyPKCES256(t *testing.T) {
	// RFC 7636 appendix B.
	const (
		verifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	)
	assert.True(t, verifyPKCES256(challenge, verifier))
	assert.False(t, verifyPKCES256(challenge, verifier+"x"))
	assert.False(t, verifyPKCES256("", verifier))
	assert.False(t, verifyPKCES256(challenge, "short"))
}

func TestApplyPKCE(t *testing.T) {
	code := &AuthCode{}
	assert.NoError(t, applyPKCE(code, "", ""))
	assert.Empty(t, code.CodeChallenge)

	assert.Error(t, applyPKCE(code, "abc", "plain"))
	assert.NoError(t, applyPKCE(code, "abc", "S256"))
	assert.Equal(t, "S256", code.CodeChallengeMethod)
	assert.Error(t, checkPKCE(code, "not-the-verifier-not-the-verifier-not-the-verifier"))

	const (
		verifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	)
	code = &AuthCode{}
	assert.NoError(t, applyPKCE(code, challenge, "S256"))
	assert.NoError(t, checkPKCE(code, verifier))
	assert.NoError(t, checkPKCE(&AuthCode{}, ""))
}

func TestGenerateCode(t *testing.T) {
	code, err := generateCode(6)
	assert.NoError(t, err)
	assert.Len(t, code, 6)
}
