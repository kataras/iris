package oauth2

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kataras/iris/v14/rest/types"
)

// AuthCode holds the data for an OAuth2 authorization code.
type AuthCode struct {
	// Code is the short-lived authorization code string.
	Code string `pg:"type=varchar(128),primary" json:"code"`
	// CreatedAt is the timestamp of code creation.
	CreatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
	// UpdatedAt is the timestamp of last code update.
	UpdatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"updated_at,omitempty"`
	// ExpiresAt is the time when the code becomes invalid.
	ExpiresAt types.ISO8601 `pg:"type=timestamp" json:"expires_at"`
	// ClientID refers to the client that requested the code.
	ClientID string `pg:"type=uuid,ref=oauth2_clients(id cascade),index" json:"client_id"`
	// UserData contains data about the authenticated user.
	UserData AuthUserData `pg:"type=jsonb" json:"user_data"`
	// State is an optional value for client tracking.
	State string `pg:"type=varchar(128)" json:"state"`
	// Scopes lists the permissions granted by the user.
	Scopes []string `pg:"type=varchar[]" json:"scopes"`
	// CodeChallenge is the PKCE S256 challenge, empty when the client did not send one.
	CodeChallenge string `json:"code_challenge,omitempty"`
	// CodeChallengeMethod is "S256" when CodeChallenge is set.
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

// NewAuthCode creates a new authorization code instance.
func NewAuthCode(clientID string, userData AuthUserData, state string, scopes []string, maxAge int) (*AuthCode, error) {
	// Authorization codes are machine-handled and single-use; use a high-entropy
	// opaque token (was a 6-digit numeric code before v14).
	code, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth code: %s: %w", clientID, err)
	}

	now := Clock()

	authCode := &AuthCode{
		Code:      code,
		CreatedAt: types.ISO8601(now),
		UpdatedAt: types.ISO8601(now),
		ExpiresAt: types.ISO8601(now.Add(time.Duration(maxAge) * time.Second)),
		ClientID:  clientID,
		UserData:  userData,
		State:     state,
		Scopes:    scopes,
	}

	return authCode, nil
}

// HasExpired checks if this authorization code is expired.
func (a *AuthCode) HasExpired() bool {
	return Clock().After(a.ExpiresAt.ToTime())
}

// URL builds a redirect URL with the code, state, and scopes attached.
func (a *AuthCode) URL(redirectURI string) string {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("code", a.Code)
	q.Set("state", a.State) // pass it back to the client.
	q.Set("scope", JoinScopes(a.Scopes...))
	u.RawQuery = q.Encode()
	return u.String()
}

// AuthorizeRequest holds the data submitted during authorization.
type AuthorizeRequest struct {
	// ClientID is the unique client identifier.
	ClientID string `json:"client_id" url:"client_id" form:"client_id"`
	// State is an optional opaque value for client tracking.
	State string `json:"state" url:"state" form:"state"`
	// Scope is the set of requested scopes, space-separated.
	Scope string `json:"scope" url:"scope" form:"scope"`
	// CodeChallenge is the optional PKCE challenge. When set, the method must be S256.
	CodeChallenge string `json:"code_challenge,omitempty" url:"code_challenge,omitempty" form:"code_challenge,omitempty"`
	// CodeChallengeMethod is the PKCE method. Only S256 is accepted.
	CodeChallengeMethod string `json:"code_challenge_method,omitempty" url:"code_challenge_method,omitempty" form:"code_challenge_method,omitempty"`
}

// ConsentCookieName returns a cookie name specific to the client ID.
func (req *AuthorizeRequest) ConsentCookieName() string {
	return "consent_" + req.ClientID
}

// Scopes parses and returns the requested scopes.
func (req *AuthorizeRequest) Scopes() []string {
	return SplitScopes(req.Scope)
}

// SplitScopes splits a space-separated scope string into a slice.
func SplitScopes(scope string) []string {
	if strings.TrimSpace(scope) == "" { // important.
		return nil
	}

	return strings.Split(scope, " ")
}

// JoinScopes joins multiple scopes into a space-separated string.
func JoinScopes(scopes ...string) string {
	return strings.Join(scopes, " ")
}

// applyPKCE stores an S256 challenge on the authorization code.
// An empty challenge means the client did not use PKCE, which stays allowed.
func applyPKCE(authCode *AuthCode, challenge, method string) error {
	if challenge == "" {
		return nil
	}
	if method != "S256" {
		return fmt.Errorf("code_challenge_method must be S256")
	}
	authCode.CodeChallenge = challenge
	authCode.CodeChallengeMethod = method
	return nil
}

// checkPKCE verifies the code verifier when the authorization code carried a challenge.
func checkPKCE(authCode *AuthCode, verifier string) error {
	if authCode == nil || authCode.CodeChallenge == "" {
		return nil
	}
	if !verifyPKCES256(authCode.CodeChallenge, verifier) {
		return fmt.Errorf("invalid code_verifier")
	}
	return nil
}
