package oauth2

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14/rest/types"
	"github.com/stretchr/testify/assert"
)

type MockUserData struct {
	ID string `json:"id"`
}

func TestNewAuthCode(t *testing.T) {
	clientID := "test-client-id"
	userData := MockUserData{ID: "test-user-id"}
	state := "test-state"
	scopes := []string{"read", "write"}
	maxAge := 3600

	authCode, err := NewAuthCode(clientID, userData, state, scopes, maxAge)
	assert.NoError(t, err)
	assert.NotNil(t, authCode)
	assert.Equal(t, clientID, authCode.ClientID)
	assert.Equal(t, userData, authCode.UserData)
	assert.Equal(t, state, authCode.State)
	assert.Equal(t, scopes, authCode.Scopes)
	assert.WithinDuration(t, time.Now(), authCode.ExpiresAt.ToTime(), time.Hour)
}

func TestAuthCode_HasExpired(t *testing.T) {
	authCode := &AuthCode{
		ExpiresAt: types.ISO8601(time.Now().Add(-time.Hour)),
	}
	assert.True(t, authCode.HasExpired())

	authCode.ExpiresAt = types.ISO8601(time.Now().Add(time.Hour))
	assert.False(t, authCode.HasExpired())
}

func TestAuthCode_URL(t *testing.T) {
	authCode := &AuthCode{
		Code:   "test-code",
		State:  "test-state",
		Scopes: []string{"read", "write"},
	}
	redirectURI := "http://localhost/callback"
	expectedURL := "http://localhost/callback?code=test-code&scope=read+write&state=test-state"
	assert.Equal(t, expectedURL, authCode.URL(redirectURI))
}
