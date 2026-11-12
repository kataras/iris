package identity

import (
	"testing"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

// newTestClientSigner builds a signer the way New builds the client auth, without a database.
func newTestClientSigner(t *testing.T) (*ClientTokenSigner, *auth.Auth[oauth2.ClientTokenClaims]) {
	t.Helper()

	var config auth.Configuration
	require.NoError(t, config.BindRandom())
	config.Cookie = auth.CookieConfiguration{}
	config.Headers = []string{Header}
	config.HeadersNoBearer = true
	for i, keyConfig := range config.Keys {
		config.Keys[i].MaxAge = 0
		if keyConfig.ID == auth.KIDRefresh {
			config.Keys[i].EncryptionKey = testEncryptionKey
		} else {
			config.Keys[i].EncryptionKey = ""
		}
	}

	clientAuth, err := auth.New[oauth2.ClientTokenClaims](config)
	require.NoError(t, err)
	return NewClientTokenSigner(clientAuth, testEncryptionKey), clientAuth
}

func TestSpecForKind(t *testing.T) {
	server, err := specForKind(ClientKindServer, "")
	require.NoError(t, err)
	assert.Equal(t, []string{oauth2.GrantTypePassword, oauth2.GrantTypeRefreshToken}, server.GrantTypes)
	assert.Equal(t, oauth2.PolicyScopes(ServerAppPolicies...), server.Scopes)
	assert.Equal(t, 7200, server.CodeMaxAge)
	assert.Equal(t, 7200, server.AccessMaxAge)
	assert.Equal(t, 604800, server.RefreshMaxAge)
	assert.True(t, server.Encrypted)

	admin, err := specForKind(ClientKindAdmin, "")
	require.NoError(t, err)
	assert.Equal(t, server.GrantTypes, admin.GrantTypes)
	assert.Equal(t, oauth2.PolicyScopes(AdminAppPolicies...), admin.Scopes)
	assert.Equal(t, 7200, admin.CodeMaxAge)
	assert.Equal(t, 7200, admin.AccessMaxAge)
	assert.Equal(t, 604800, admin.RefreshMaxAge)
	assert.True(t, admin.Encrypted)

	user, err := specForKind(ClientKindUser, "")
	require.NoError(t, err)
	assert.Equal(t, []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeRefreshToken}, user.GrantTypes)
	assert.Equal(t, oauth2.PolicyScopes(UserAppPolicies...), user.Scopes)
	assert.Equal(t, 600, user.CodeMaxAge)
	assert.Equal(t, 3600, user.AccessMaxAge)
	assert.Equal(t, 604800, user.RefreshMaxAge)
	assert.False(t, user.Encrypted)

	userDevice, err := specForKind(ClientKindUser, "https://example.com/device")
	require.NoError(t, err)
	assert.Equal(t, []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeRefreshToken, oauth2.GrantTypeDeviceCode}, userDevice.GrantTypes)

	// device_code is a user-kind feature only.
	serverWithURI, err := specForKind(ClientKindServer, "https://example.com/device")
	require.NoError(t, err)
	assert.NotContains(t, serverWithURI.GrantTypes, oauth2.GrantTypeDeviceCode)

	_, err = specForKind(ClientKind("bogus"), "")
	assert.ErrorIs(t, err, entity.ErrInvalidArgument)
}

func TestKindOfClient(t *testing.T) {
	user := &oauth2.Client{GrantTypes: []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeRefreshToken}, Scopes: oauth2.PolicyScopes(UserAppPolicies...)}
	server := &oauth2.Client{GrantTypes: []string{oauth2.GrantTypePassword, oauth2.GrantTypeRefreshToken}, Scopes: oauth2.PolicyScopes(ServerAppPolicies...)}
	admin := &oauth2.Client{GrantTypes: []string{oauth2.GrantTypePassword, oauth2.GrantTypeRefreshToken}, Scopes: oauth2.PolicyScopes(AdminAppPolicies...)}

	assert.Equal(t, ClientKindUser, KindOfClient(user))
	assert.Equal(t, ClientKindServer, KindOfClient(server))
	assert.Equal(t, ClientKindAdmin, KindOfClient(admin))
}

func TestClientTokenSignerCredentials(t *testing.T) {
	signer, clientAuth := newTestClientSigner(t)
	ctx := t.Context()

	server := &oauth2.Client{ID: "server-id", Name: "server", Secret: "placeholder-secret", GrantTypes: []string{oauth2.GrantTypePassword, oauth2.GrantTypeRefreshToken}, Scopes: oauth2.PolicyScopes(ServerAppPolicies...)}
	creds, err := signer.Credentials(server)
	require.NoError(t, err)
	assert.Equal(t, server.ID, creds.Audience)
	assert.Equal(t, ClientKindServer, creds.Kind)
	assert.Equal(t, testEncryptionKey, creds.EncryptionKey)
	_, verified, err := clientAuth.Verify(ctx, []byte(creds.Token))
	require.NoError(t, err)
	assert.Equal(t, []string{server.ID}, []string(verified.StandardClaims.Audience))

	user := &oauth2.Client{ID: "user-id", Name: "user", Secret: "placeholder-secret", GrantTypes: []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeRefreshToken}, Scopes: oauth2.PolicyScopes(UserAppPolicies...)}
	creds, err = signer.Credentials(user)
	require.NoError(t, err)
	assert.Equal(t, user.ID, creds.Audience)
	assert.Equal(t, ClientKindUser, creds.Kind)
	assert.Empty(t, creds.EncryptionKey)
	_, verified, err = clientAuth.Verify(ctx, []byte(creds.Token))
	require.NoError(t, err)
	assert.Equal(t, []string{user.ID}, []string(verified.StandardClaims.Audience))
}
