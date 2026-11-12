package oauth2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient(ClientRegistration{
		Name:          "test-client",
		Website:       "http://example.com",
		LogoURL:       "http://example.com/logo.png",
		RedirectURI:   "http://example.com/callback",
		CodeMaxAge:    3600,
		AccessMaxAge:  3600,
		RefreshMaxAge: 604800,
	})
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "test-client", client.Name)
	assert.Equal(t, "http://example.com", client.Website)
	assert.Equal(t, "http://example.com/logo.png", client.LogoURL)
	assert.Equal(t, "http://example.com/callback", client.RedirectURI)
	assert.NotEmpty(t, client.Secret)
	assert.Equal(t, 3600, client.CodeMaxAge)
	assert.Equal(t, 3600, client.AccessMaxAge)
	assert.Equal(t, 604800, client.RefreshMaxAge)
}

func TestNewClientExplicitGrantsRedirectURI(t *testing.T) {
	// A redirect URI alone is enough when the device code grant is not requested.
	client, err := NewClient(ClientRegistration{
		Name:          "web-client",
		Website:       "http://example.com",
		LogoURL:       "http://example.com/logo.png",
		RedirectURI:   "http://example.com/callback",
		GrantTypes:    []string{GrantTypeAuthorizationCode, GrantTypeRefreshToken},
		CodeMaxAge:    3600,
		AccessMaxAge:  3600,
		RefreshMaxAge: 604800,
	})
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// The device code grant still needs the verification URI.
	client, err = NewClient(ClientRegistration{
		Name:          "device-client",
		Website:       "http://example.com",
		LogoURL:       "http://example.com/logo.png",
		RedirectURI:   "http://example.com/callback",
		GrantTypes:    []string{GrantTypeAuthorizationCode, GrantTypeRefreshToken, GrantTypeDeviceCode},
		CodeMaxAge:    3600,
		AccessMaxAge:  3600,
		RefreshMaxAge: 604800,
	})
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestClient_MatchScopes(t *testing.T) {
	client := &Client{
		Scopes: []string{"read", "write"},
	}

	assert.True(t, client.MatchScopes([]string{"read"}))
	assert.False(t, client.MatchScopes([]string{"delete"}))
}

func TestClient_HasGrantType(t *testing.T) {
	client := &Client{
		GrantTypes: []string{GrantTypeAuthorizationCode, GrantTypePassword},
	}

	assert.True(t, client.MatchGrantType(GrantTypeAuthorizationCode))
	assert.False(t, client.MatchGrantType(GrantTypeRefreshToken))
}
