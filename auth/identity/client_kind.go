package identity

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"
)

// ClientKind is the kind of an OAuth2 client application managed by the identity server.
type ClientKind string

const (
	// ClientKindServer is a server application: password and refresh token grants,
	// the user management scopes and an encrypted, non-expiring token.
	ClientKindServer ClientKind = "server"
	// ClientKindUser is a user application: authorization code flow (and device code
	// when it has a verification URI) with a plain, non-expiring token.
	ClientKindUser ClientKind = "user"
	// ClientKindAdmin is a server application that may also manage clients.
	ClientKindAdmin ClientKind = "admin"
)

// clientKindSpec is what a client kind implies when a client is created.
type clientKindSpec struct {
	GrantTypes    []string
	Scopes        []string
	CodeMaxAge    int
	AccessMaxAge  int
	RefreshMaxAge int
	// Encrypted reports whether the client token is the encrypted refresh variant,
	// delivered together with the shared encryption key.
	Encrypted bool
}

// specForKind returns the grant types, scopes, default max ages and token variant of a client kind.
// The verification URI adds the device code grant to user clients only.
// It does not validate the redirect URI, callers do.
func specForKind(kind ClientKind, verificationURI string) (clientKindSpec, error) {
	switch kind {
	case ClientKindServer, ClientKindAdmin:
		policies := ServerAppPolicies
		if kind == ClientKindAdmin {
			policies = AdminAppPolicies
		}
		return clientKindSpec{
			GrantTypes:    []string{oauth2.GrantTypePassword, oauth2.GrantTypeRefreshToken},
			Scopes:        oauth2.PolicyScopes(policies...),
			CodeMaxAge:    3600 * 2,
			AccessMaxAge:  3600 * 2,
			RefreshMaxAge: 604800,
			Encrypted:     true,
		}, nil
	case ClientKindUser:
		grantTypes := []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeRefreshToken}
		if verificationURI != "" {
			grantTypes = append(grantTypes, oauth2.GrantTypeDeviceCode)
		}
		return clientKindSpec{
			GrantTypes:    grantTypes,
			Scopes:        oauth2.PolicyScopes(UserAppPolicies...),
			CodeMaxAge:    600,
			AccessMaxAge:  3600,
			RefreshMaxAge: 604800,
		}, nil
	default:
		return clientKindSpec{}, fmt.Errorf("%w: unknown client kind %q", entity.ErrInvalidArgument, kind)
	}
}

// KindOfClient infers the kind of a stored client.
// A client without the password grant is a user client, a client with any "clients:" scope
// is an admin client and everything else is a server client.
func KindOfClient(c *oauth2.Client) ClientKind {
	if !slices.Contains(c.GrantTypes, oauth2.GrantTypePassword) {
		return ClientKindUser
	}

	for _, scope := range c.Scopes {
		if strings.HasPrefix(scope, "clients:") {
			return ClientKindAdmin
		}
	}
	return ClientKindServer
}
