package oauth2

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemDeviceCodeStore(t *testing.T) {
	store := NewMemoryDeviceCodeStore()
	ctx := stdContext.Background()

	deviceCode := &DeviceCode{
		Code:     "test-code",
		UserCode: "test-user-code",
	}

	err := store.InsertDeviceCode(ctx, deviceCode)
	assert.NoError(t, err)

	retrievedCode, ok, err := store.GetAndDeleteDeviceCode(ctx, "test-code")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, deviceCode, retrievedCode)

	_, ok, err = store.GetAndDeleteDeviceCode(ctx, "test-code")
	assert.NoError(t, err)
	assert.False(t, ok)

	// Already deleted by code. Ok should be false.
	_, ok, err = store.GetDeviceCodeByUserCode(ctx, "test-user-code")
	assert.NoError(t, err)
	assert.False(t, ok)
}
