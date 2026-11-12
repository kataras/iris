package oauth2

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemAuthCodeStore(t *testing.T) {
	store := NewMemoryAuthCodeStore()
	ctx := context.Background()

	authCode := &AuthCode{
		Code: "test-code",
	}

	err := store.InsertAuthCode(ctx, authCode)
	assert.NoError(t, err)

	retrievedCode, ok, err := store.GetAndDeleteAuthCode(ctx, "test-code")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, authCode, retrievedCode)

	_, ok, err = store.GetAndDeleteAuthCode(ctx, "test-code")
	assert.NoError(t, err)
	assert.False(t, ok)
}
