package identity

import (
	"fmt"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/oauth2"
)

// ClientCredentials is what a client application needs to talk to the identity server.
type ClientCredentials struct {
	Audience      string     `json:"audience"` // The client ID, consumers set Audiences: [audience].
	Kind          ClientKind `json:"kind"`
	Token         string     `json:"token"`
	EncryptionKey string     `json:"encryption_key,omitempty"` // Server and admin clients only.
}

// ClientTokenSigner signs the tokens of the client applications.
type ClientTokenSigner struct {
	auth          *auth.Auth[oauth2.ClientTokenClaims]
	encryptionKey string
}

// NewClientTokenSigner returns a signer that uses the given client authenticator
// and hands out the given encryption key to server and admin clients.
func NewClientTokenSigner(clientAuth *auth.Auth[oauth2.ClientTokenClaims], encryptionKey string) *ClientTokenSigner {
	return &ClientTokenSigner{auth: clientAuth, encryptionKey: encryptionKey}
}

// Sign signs the token pair of the given client.
func (s *ClientTokenSigner) Sign(client *oauth2.Client) (*oauth2.Token, error) {
	// We should not pass all the original app's scopes here, more than user scopes.
	// The client is fetched by the storage every time, and we check these scopes on every request instead of the claims themselves.
	// So the generated token passed on `Header`, which is used for signing a user in through oauth2 server (e.g. password grant),
	// is always having the minimal scopes required
	// for user-based actions and not the server-app actions.
	clientClaims := client.Claims(UserAppScopes)
	// We need the client secret in there so we can transmit it back to the server for app client authorization.
	clientClaims.ClientSecret = client.Secret

	// access & refresh max age are 0 because we use the refresh token to authenticate the client apps.
	signOptions := oauth2.ClientTokenSignOptions(client.ID, 0) // Although max age it's zero because we set the max age to 0 at initialization.
	token, err := s.auth.SignToken(clientClaims, signOptions...)
	if err != nil {
		return nil, fmt.Errorf("%w: client access token: %w", ErrSignToken, err)
	}
	return token, err
}

// Credentials signs the client's token and returns it with the other values the client needs.
// Server and admin clients get the encrypted refresh token and the encryption key,
// user clients get the plain access token.
func (s *ClientTokenSigner) Credentials(client *oauth2.Client) (*ClientCredentials, error) {
	token, err := s.Sign(client)
	if err != nil {
		return nil, err
	}

	kind := KindOfClient(client)
	creds := &ClientCredentials{Audience: client.ID, Kind: kind}
	if kind == ClientKindUser {
		creds.Token = token.AccessToken
	} else {
		creds.Token = token.RefreshToken // use the encrypted one.
		creds.EncryptionKey = s.encryptionKey
	}
	return creds, nil
}
