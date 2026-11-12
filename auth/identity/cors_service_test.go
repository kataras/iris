package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateOriginPatterns(t *testing.T) {
	assert.NoError(t, validateOriginPatterns(nil))
	assert.NoError(t, validateOriginPatterns([]string{"https://id.hellenic.dev"}))
	assert.NoError(t, validateOriginPatterns([]string{"*"}))
	assert.NoError(t, validateOriginPatterns([]string{"https://example.com", " * "}))
	assert.Error(t, validateOriginPatterns([]string{"  "}))

	assert.True(t, patternsAllowAny([]string{"https://example.com", "*"}))
	assert.True(t, patternsAllowAny([]string{" * "}))
	assert.False(t, patternsAllowAny([]string{"https://example.com"}))
}
