package oauth2

import (
	"fmt"
	"slices"
	"time"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/core/pkg/uid"
	"github.com/kataras/iris/v14/rest/types"
	"github.com/kataras/jwt"
)

type (
	// Client represents a client application that is registered to the server.
	// The client is the application that wants to access the user's data.
	//
	// If the client is a
	// - web application then the client is the web application itself.
	// - mobile application then the client is the mobile application itself.
	// - server-side application then the client is the server-side application itself.
	//
	// Note that if custom ownership is required then end-developer can
	// create a separate store/table to store the owner id and the client id from this store/table.
	Client struct {
		// ID is the unique identifier for the client.
		ID string `pg:"type=uuid,primary" json:"id,omitempty"`
		// CreatedAt is the timestamp when the client was created.
		CreatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
		// UpdatedAt is the timestamp of the last update to the client.
		UpdatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"updated_at,omitempty"`

		// Name is the name of the client application.
		Name string `pg:"type=varchar(255),unique" json:"name"`
		// Website is the website URL of the client application.
		Website string `pg:"type=varchar(255)" json:"website"`
		// LogoURL is the URL of the client's logo.
		LogoURL string `pg:"type=varchar(255)" json:"logo_url"`
		// RedirectURI is the callback URL for the client.
		RedirectURI string `pg:"type=text" json:"redirect_uri"`
		// VerificationURI is the URL where the user should enter the user code.
		VerificationURI string `pg:"type=text,default=''" json:"verification_uri"`
		// Secret is the randomly generated secret of the client,
		// stored as-is (plaintext, it is neither encrypted nor hashed).
		// Client send it to the server to authenticate itself on every request,
		// server matches it with the stored secret to verify the client.
		Secret string `pg:"type=varchar(255)" json:"secret,omitempty"`
		// GrantTypes are the allowed grant types for the client.
		// If empty then no grant type is allowed;
		// NewClient fills it in automatically when it is not provided.
		GrantTypes []string `pg:"type=varchar[]" json:"grant_types"`

		// Scopes are the allowed scopes for the client.
		// It checks if the requested scopes are allowed by the client,
		// and passes only what is defined in the request to the actual generated token,
		// so the end-server knows what the client is allowed to do.
		Scopes []string `pg:"type=varchar[]" json:"scopes"`

		// CodeMaxAge is the maximum age for the codes, in seconds.
		// Defaults to 1 hour (3600 seconds).
		CodeMaxAge int `pg:"type=int,default=3600" json:"code_max_age"`
		// AccessMaxAge is the max age for the access tokens, in seconds. Defaults to 1 hour.
		AccessMaxAge int `pg:"type=int,default=3600" json:"access_max_age"`
		// RefreshMaxAge is the max age for the refresh tokens, in seconds. Defaults to 7 days.
		RefreshMaxAge int `pg:"type=int,default=604800" json:"refresh_max_age"`
		// Version is the version of the client. It is used as an extra claim.
		// In order to invalidate all tokens created by this client just upgrade the version.
		Version int `pg:"type=smallint,default=0" json:"version"`
		// Codes is a JSONB field that contains the issued codes.
		// The user data is the user's data that is associated with the code.
		// Codes struct { // decided to embed those here instead of separate stores. The only disadvantage is that we require the auth user data type in the client type too.
		// 	Auth   []AuthCode[UserData]   `json:"auth_codes"`
		// 	Device []DeviceCode `json:"device_codes"`
		// } `pg:"type=jsonb"`
		// No, let's keep them separated from the actual Client.
	}

	// AuthUserData is the user data that is associated with the auth code.
	// It can be any custom data type.
	AuthUserData auth.User
	// interface { GetID() string }
)

// ClientRegistration holds the values NewClient, Server.RegisterClient and
// Server.CreateClient build a Client from. The ID and the secret are generated.
//
// Name, Website and LogoURL are required: an empty one makes NewClient return
// "name, website, logo and (redirect_uri or verification_uri) are required".
type ClientRegistration struct {
	// Name is the unique name of the client application. Required.
	Name string
	// Website is the website URL of the client application. Required.
	Website string
	// LogoURL is the URL of the client's logo. Required.
	LogoURL string
	// RedirectURI is the callback URL. Required when GrantTypes contains the authorization code
	// or the device code grant type.
	RedirectURI string
	// VerificationURI is the URL where the user enters the user code. Required when GrantTypes
	// contains the device code grant type.
	VerificationURI string
	// GrantTypes are the allowed grant types. When empty they are resolved from the URIs:
	// password and refresh_token always, plus authorization_code when RedirectURI is set,
	// plus the device code grant type when VerificationURI is set too.
	GrantTypes []string
	// Scopes are the allowed scopes, stored as given.
	Scopes []string
	// CodeMaxAge is the max age of the authorization and device codes, in seconds.
	// Zero or negative means 3600 (one hour).
	CodeMaxAge int
	// AccessMaxAge is the max age of the access tokens, in seconds, stored as given.
	// Zero or negative means the token does not expire.
	AccessMaxAge int
	// RefreshMaxAge is the max age of the refresh tokens, in seconds, stored as given.
	// Zero or negative means the token does not expire. When both max ages are above zero,
	// RefreshMaxAge must be at least AccessMaxAge plus 60.
	RefreshMaxAge int
}

// NewClient creates and returns a new client from reg.
// The client's ID and secret are generated automatically.
// See ClientRegistration for the required fields, the defaults and the validation.
func NewClient(reg ClientRegistration) (*Client, error) {
	var (
		name, website, logo          = reg.Name, reg.Website, reg.LogoURL
		redirectURI, verificationURI = reg.RedirectURI, reg.VerificationURI
		grantTypes, scopes           = reg.GrantTypes, reg.Scopes
		codeMaxAge                   = reg.CodeMaxAge
		accessMaxAge, refreshMaxAge  = reg.AccessMaxAge, reg.RefreshMaxAge
	)

	if name == "" || website == "" || logo == "" {
		return nil, fmt.Errorf("new client: name, website, logo and (redirect_uri or verification_uri) are required")
	}

	if len(grantTypes) == 0 { // resolve the grant types automatically.
		// defaults to server-app if no redirect_uri & verification_uri given.
		grantTypes = []string{GrantTypePassword, GrantTypeRefreshToken}
		if redirectURI != "" { // add authorization code grant type if redirect_uri is provided.
			grantTypes = append(grantTypes, GrantTypeAuthorizationCode)

			if verificationURI != "" { // add device code grant type if verification_uri is provided.
				grantTypes = append(grantTypes, GrantTypeDeviceCode)
			}
		}
	} else {
		if matchGrantType(grantTypes, GrantTypeAuthorizationCode) && redirectURI == "" {
			return nil, fmt.Errorf("new client: redirect_uri is required for the authorization code grant type")
		}

		if matchGrantType(grantTypes, GrantTypeDeviceCode) && (redirectURI == "" || verificationURI == "") {
			return nil, fmt.Errorf("new client: redirect_uri and verification_uri is required for the device code grant type")
		}
	}

	if matchGrantType(grantTypes, GrantTypeDeviceCode) && verificationURI == "" {
		return nil, fmt.Errorf("new client: verification_uri is required for the device code grant type")
	}

	secret, err := generateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("new client: failed to generate client secret: %s: %w", name, err)
	}

	if codeMaxAge <= 0 {
		codeMaxAge = 3600 // defaults to 1 hour.
	}

	// No, as they can be zero for no expiration.
	// if accessMaxAge <= 0 {
	// 	accessMaxAge = 3600 // defaults to 1 hour.
	// }
	// if refreshMaxAge <= 0 {
	// 	refreshMaxAge = 604800 // defaults to 7 days.
	// }

	if accessMaxAge > 0 && refreshMaxAge > 0 {
		if refreshMaxAge < accessMaxAge+60 { // 60 seconds is the minimum difference between access and refresh max age.
			return nil, fmt.Errorf("new client: refresh_max_age must be greater than access_max_age + 60 seconds, got %d < %d", refreshMaxAge, accessMaxAge+60)
		}
	}

	client := &Client{
		ID:              uid.NewString(),
		Name:            name,
		Website:         website,
		LogoURL:         logo,
		RedirectURI:     redirectURI,
		VerificationURI: verificationURI,
		Secret:          secret,
		GrantTypes:      grantTypes,
		Scopes:          scopes,
		CodeMaxAge:      codeMaxAge,
		AccessMaxAge:    accessMaxAge,
		RefreshMaxAge:   refreshMaxAge,
	}
	return client, nil
}

// ClientTokenSignOptions returns the signing options for the client token.
func ClientTokenSignOptions(clientID string, maxAge int) []jwt.SignOption {
	return []jwt.SignOption{jwt.Audience{clientID}, jwt.MaxAge(time.Duration(maxAge) * time.Second)}
}

// AccessTokenOptions returns the signing option for the access token.
func (c *Client) AccessTokenOptions() jwt.SignOption {
	return auth.AccessTokenOption(ClientTokenSignOptions(c.ID, c.AccessMaxAge)...)
}

// RefreshTokenOptions returns the signing option for the refresh token.
func (c *Client) RefreshTokenOptions() jwt.SignOption {
	return auth.RefreshTokenOption(ClientTokenSignOptions(c.ID, c.RefreshMaxAge)...)
}

// Claims returns the client's token claims.
func (c *Client) Claims(scopes []string) ClientTokenClaims {
	return ClientTokenClaims{
		ClientID:      c.ID,
		ClientName:    c.Name,
		ClientWebsite: c.Website,
		ClientLogoURL: c.LogoURL,
		Version:       c.Version,
		Scopes:        scopes,
	}
}

// MatchScopes returns true if the client has the specific scopes.
// If "scopes" is empty then it returns true, as there is nothing to check.
func (c *Client) MatchScopes(scopes []string) bool {
	_, _, matched := matchScopes(c.Scopes, scopes...)
	return matched
}

// MatchGrantType returns true if the client has the specific grant type.
func (c *Client) MatchGrantType(grantType string) bool {
	return matchGrantType(c.GrantTypes, grantType)
}

// matchGrantType checks if the provided grant type exists in the list of grant types.
func matchGrantType(grantTypes []string, grantType string) bool {
	if len(grantTypes) == 0 {
		return false
	}

	if grantType == "" {
		grantType = GrantTypeAuthorizationCode
	}

	return slices.Contains(grantTypes, grantType)
}
