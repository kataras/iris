package oauth2

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14/rest/types"
	"github.com/stretchr/testify/assert"
)

func TestNewDeviceCode(t *testing.T) {
	clientID := "test-client-id"
	userData := MockUserData{ID: "test-user-id"}
	interval := 5
	scopes := []string{"read", "write"}
	maxAge := 3600

	deviceCode, err := NewDeviceCode(clientID, userData, interval, scopes, maxAge)
	assert.NoError(t, err)
	assert.NotNil(t, deviceCode)
	assert.Equal(t, clientID, deviceCode.ClientID)
	assert.Equal(t, userData, deviceCode.UserData)
	assert.Equal(t, interval, deviceCode.Interval)
	assert.Equal(t, scopes, deviceCode.Scopes)
	assert.WithinDuration(t, time.Now(), deviceCode.ExpiresAt.ToTime(), time.Hour)
}

func TestDeviceCode_HasExpired(t *testing.T) {
	deviceCode := &DeviceCode{
		ExpiresAt: types.ISO8601(time.Now().Add(-time.Hour)),
	}
	assert.True(t, deviceCode.HasExpired())

	deviceCode.ExpiresAt = types.ISO8601(time.Now().Add(time.Hour))
	assert.False(t, deviceCode.HasExpired())
}

func TestDeviceCode_AuthResponse(t *testing.T) {
	deviceCode := &DeviceCode{
		Code:      "test-code",
		UserCode:  "test-user-code",
		ExpiresAt: types.ISO8601(time.Now().Add(time.Hour)),
		Interval:  5,
	}
	verificationURI := "http://example.com/verify"
	authResponse := deviceCode.AuthResponse(verificationURI)

	assert.Equal(t, deviceCode.Code, authResponse.DeviceCode)
	assert.Equal(t, deviceCode.UserCode, authResponse.UserCode)
	assert.Equal(t, verificationURI, authResponse.VerificationURI)
	assert.Equal(t, verificationURI+"?user_code="+deviceCode.UserCode, authResponse.VerificationURIComplete)
	assert.WithinDuration(t, deviceCode.ExpiresAt.ToTime(), authResponse.Expiry, time.Hour)
	assert.Equal(t, int64(deviceCode.Interval), authResponse.Interval)
}
