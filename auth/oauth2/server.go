// Package oauth2 provides an implementation of OAuth2 server-related logic, including
// the registration and retrieval of clients, authorization codes, device codes, tokens,
// and consent handling.
package oauth2

import (
	stdContext "context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/rate"
	"github.com/kataras/iris/v14/rest/types"

	"golang.org/x/oauth2"
)

// init registers the ClientTokenClaims type to the gob package
// so it can be encoded/decoded by the standard library.
func init() {
	gob.Register(ClientTokenClaims{})
}

type serverOptions struct {
	// ClientStore manages creation and retrieval of OAuth2 clients.
	ClientStore ClientStore
	// AuthCodeStore manages saving and retrieving OAuth2 authorization codes.
	AuthCodeStore AuthCodeStore
	// DeviceCodeStore manages saving and retrieving OAuth2 device codes.
	DeviceCodeStore DeviceCodeStore

	// LoginPageURL is the route URL to redirect unauthenticated users to.
	// The URL should accept a "redirect_to" or a "redirect_uri" query parameter
	// which the server will use to redirect the user back to oauth2 authorization after authentication.
	LoginPageURL string
	// ConsentPageURL is the route URL to redirect users for consent confirmation before the authorization.
	// The URL should accept the "client_id", "client_name", "state" and "scope" query parameters
	// and pass them along with the user's "decision" ("approve" or "deny")
	// and, optionally, "remember" ("1" for true) form values to the POST /consent endpoint.
	ConsentPageURL string

	// DeviceInterval is the default polling interval in seconds for device code confirmation if not explicitly set.
	DeviceInterval int
}

// Option is a function that configures the server.
type Option func(*serverOptions)

// WithClientStore sets the client store for the server.
func WithClientStore(clientStore ClientStore) Option {
	return func(o *serverOptions) {
		o.ClientStore = clientStore
	}
}

// WithAuthCodeStore sets the authorization code store for the server.
func WithAuthCodeStore(authCodeStore AuthCodeStore) Option {
	return func(o *serverOptions) {
		o.AuthCodeStore = authCodeStore
	}
}

// WithDeviceCodeStore sets the device code store for the server.
func WithDeviceCodeStore(deviceCodeStore DeviceCodeStore) Option {
	return func(o *serverOptions) {
		o.DeviceCodeStore = deviceCodeStore
	}
}

// WithLoginPageURL sets the login page URL for the server.
func WithLoginPageURL(loginPageURL string) Option {
	return func(o *serverOptions) {
		o.LoginPageURL = loginPageURL
	}
}

// WithConsentPageURL sets the consent page URL for the server.
func WithConsentPageURL(consentPageURL string) Option {
	return func(o *serverOptions) {
		o.ConsentPageURL = consentPageURL
	}
}

// WithDeviceInterval sets the default polling interval for device code confirmation.
func WithDeviceInterval(intervalSeconds int) Option {
	return func(o *serverOptions) {
		o.DeviceInterval = intervalSeconds
	}
}

// Server defines an OAuth2 server capable of handling various grant types and flows.
//
// Usage:
//
//	type User struct {
//		ID        string `json:"id"`
//		// Add any other fields you need.
//	}
//
//	oauth2Server := oauth2.New[User](s,
//		oauth2.WithClientStore(oauth2.NewMemoryClientStore()),
//		oauth2.WithAuthCodeStore(oauth2.NewMemoryAuthCodeStore()),
//		oauth2.WithDeviceCodeStore(oauth2.NewMemoryDeviceCodeStore()),
//		oauth2.WithLoginPageURL("/signin"),
//		oauth2.WithConsentPageURL("/oauth2/consent"),
//	)
//
//	app.Router("/oauth2", oauth2Server)
//
// - GET  /authorize: the authorization endpoint.
// - GET  /consent: the built-in consent page, if the ConsentPageURL is "default" or matches the *relpath*/consent endpoint.
// - POST /consent: the confirmation endpoint, if the ConsentPageURL is not empty.
// - GET  /consent/revoke: the revoke consent endpoint, if the ConsentPageURL is not empty.
// - POST /device/code: the device code endpoint.
// - POST /token: the token endpoint.
// - POST /token/introspect: the token introspection endpoint.
type Server[UserData AuthUserData] struct {
	// Auth is the Iris Auth instance used for user authentication and verification.
	Auth *auth.Auth[UserData]

	options *serverOptions
}

// New creates a new OAuth2 server with the given Auth instance and options.
// If the Auth instance is not provided, it will try to resolve it using the auth.Resolve function.
// The server will be built with the provided options.
//
// It can be registered as a router, as it implements the router.Router's Routing method, e.g. app.Router("/oauth2", server).
func New[UserData AuthUserData](a *auth.Auth[UserData], opts ...Option) *Server[UserData] {
	if a == nil {
		authInstance, _ := auth.Resolve[UserData]() // try to auto-resolve it if the end-developer didn't pass it.
		a = authInstance
	}

	options := &serverOptions{
		ClientStore:     NewMemoryClientStore(),
		AuthCodeStore:   NewMemoryAuthCodeStore(),
		DeviceCodeStore: NewMemoryDeviceCodeStore(),
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(options)
	}

	if a == nil || options.ClientStore == nil || options.AuthCodeStore == nil || options.DeviceCodeStore == nil {
		panic("oauth2: Auth, ClientStore, AuthCodeStore and DeviceCodeStore are required")
	}

	if options.DeviceInterval <= 0 {
		options.DeviceInterval = 5
	}

	server := &Server[UserData]{
		Auth:    a,
		options: options,
	}
	return server
}

// RegisterClient creates a new client from reg and inserts it into the configured client store,
// unless a client with the same name already exists, in which case it returns that one unchanged.
// See ClientRegistration for the required fields and the defaults.
func (s *Server[UserData]) RegisterClient(ctx stdContext.Context, reg ClientRegistration) (*Client, error) {
	client, err := s.getOrCreateClient(ctx, reg)
	if err != nil {
		return nil, err
	}

	/*
		// Generate a token for the client so it can pass the AllowPolicies checks,
		// as a server-app (without the user permissions and user access data token).
		// Add a NoMaxAge so the token will be valid, although we don't verify it at AllowPolicies check.
		// This token should not be transmitted to the server or the client by HTTP, it's only for libraries uses,
		// see sdk package.
		//
		// Note that even if the client is removed, the AllowPolicies will allow it to pass,
		// however it will fail on actual actions afterwards as we retrieve it from the store.
		signOptions := []jwt.SignOption{jwt.Audience{client.ID}, jwt.NoMaxAge}
		// clientFullInfoToken, _, err := s.Auth.SignToken(client , signOptions...)
		// if err != nil {
		// 	return nil, "", fmt.Errorf("oauth2: failed to sign client admin token: %w", err)
		// }

		token, err := s.Auth.SignToken(client.ServerClaims(), signOptions...)
		if err != nil {
			return nil, "", fmt.Errorf("oauth2: failed to sign client access token: %w", err)
		}

		return client, token.AccessToken, nil
	*/

	return client, nil
}

// ClientStore returns the client store the server was configured with.
func (s *Server[UserData]) ClientStore() ClientStore {
	return s.options.ClientStore
}

// CreateClient creates a new client from reg and inserts it into the client store.
// Unlike RegisterClient it never returns an existing client:
// it returns ErrClientAlreadyExists when the name is taken.
func (s *Server[UserData]) CreateClient(ctx stdContext.Context, reg ClientRegistration) (*Client, error) {
	_, found, err := s.options.ClientStore.GetClientByName(ctx, reg.Name)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, fmt.Errorf("%w: %s", ErrClientAlreadyExists, reg.Name)
	}

	return s.insertNewClient(ctx, reg)
}

// EnsureClientScopes adds to the stored client the scopes it lacks (a union that keeps the existing order
// and removes nothing) and updates client.Scopes to match. The secret and the version are untouched,
// so every token already issued for the client stays valid.
// It reports false, with a nil error, when nothing was missing.
func (s *Server[UserData]) EnsureClientScopes(ctx stdContext.Context, client *Client, scopes []string) (bool, error) {
	stored, found, err := s.options.ClientStore.GetClient(ctx, client.ID)
	if err != nil {
		return false, err
	}
	if !found {
		return false, ErrClientNotFound
	}

	merged := slices.Clone(stored.Scopes)
	for _, scope := range scopes {
		if !slices.Contains(merged, scope) {
			merged = append(merged, scope)
		}
	}
	if len(merged) == len(stored.Scopes) {
		return false, nil
	}

	changed, err := s.options.ClientStore.UpdateClientScopes(ctx, client.ID, merged)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, ErrClientNotFound
	}

	client.Scopes = merged
	return true, nil
}

func (s *Server[UserData]) getOrCreateClient(ctx stdContext.Context, reg ClientRegistration) (*Client, error) {
	client, found, err := s.options.ClientStore.GetClientByName(ctx, reg.Name)
	if err != nil {
		return nil, err
	}
	if found {
		return client, nil
	}

	return s.insertNewClient(ctx, reg)
}

// insertNewClient builds a client and inserts it into the store. It is the insert path of
// both CreateClient and getOrCreateClient.
func (s *Server[UserData]) insertNewClient(ctx stdContext.Context, reg ClientRegistration) (*Client, error) {
	client, err := NewClient(reg)
	if err != nil {
		return nil, err
	}

	if err = s.options.ClientStore.InsertClient(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}

// Routing registers the necessary routes for the OAuth2 server.
// - GET  /authorize: the authorization endpoint.
// - GET  /consent: the built-in consent page, if the ConsentPageURL is "default" or matches the *relpath*/consent endpoint.
// - POST /consent: the confirmation endpoint, if the ConsentPageURL is not empty.
// - GET  /consent/revoke: the revoke consent endpoint, if the ConsentPageURL is not empty.
// - POST /device/code: the device code endpoint.
// - POST /token: the token endpoint.
// - POST /token/introspect: the token introspection endpoint.
func (s *Server[UserData]) Routing(r router.Router) {
	r.Get("/authorize", s.AuthorizeHandler())

	if s.options.ConsentPageURL != "" {
		// If the consent page url matches the internal, default consent url then register the necessary rotues.
		if s.options.ConsentPageURL == "default" || strings.HasSuffix(s.options.ConsentPageURL, router.JoinPath(r.GetRelPath(), "/consent")) {
			r.Get("/consent", s.DefaultConsentPageHandler())
		}

		// This remains as it is as the custom consent pages should hit this one in order to pass the confirmation
		// and authorize a client.
		r.Post("/consent", s.ConfirmConsentHandler())

		// A simple endpoint helper to forget the consent decision.
		r.Get("/consent/revoke", s.RevokeConsentHandler())
	}

	// Register the device code handler.
	r.Post("/device/code", s.DeviceCodeHandler())

	// Register the token generation handler.
	// 10 requests/second, burst 20, keyed by the client address.
	r.Post("/token", rate.Limit(10, 20, rate.Options{}), s.TokenHandler())

	// Register the access token introspection handler.
	// https://www.oauth.com/oauth2-servers/token-introspection-endpoint.
	r.Post("/token/introspect", s.TokenIntrospectHandler())
}

// AuthorizeHandler returns the handler for the authorization endpoint.
// If ConsentPageURL is set, it returns the consent handler; otherwise, it returns the auth code handler.
func (s *Server[UserData]) AuthorizeHandler() context.Handler {
	if s.options.ConsentPageURL == "" { // Return the handler which will generate the authorization code directly.
		return s.handleAuthCode
	}

	// Else change the authorizeHandler to ask for user permissions first.
	return s.askConsent
}

// DefaultConsentPageHandler returns the default consent page handler.
func (s *Server[UserData]) DefaultConsentPageHandler() context.Handler {
	return defaultConsentHandler(s.Auth.ErrorHandler)
}

// ConfirmConsentHandler returns the handler for confirming user consent.
func (s *Server[UserData]) ConfirmConsentHandler() context.Handler {
	return s.confirmConsent
}

// RevokeConsentHandler returns the handler for revoking user consent.
func (s *Server[UserData]) RevokeConsentHandler() context.Handler {
	return s.revokeConsent
}

// DeviceCodeHandler returns the handler for the device code endpoint.
func (s *Server[UserData]) DeviceCodeHandler() context.Handler {
	return s.handleDeviceCode
}

// TokenHandler returns the handler for the token endpoint.
func (s *Server[UserData]) TokenHandler() context.Handler {
	return s.handleTokenRequest
}

// TokenIntrospectHandler returns the handler for the token introspection endpoint.
func (s *Server[UserData]) TokenIntrospectHandler() context.Handler {
	return s.introspectToken
}

const MaxAgeAccessToken = -1

// SignToken signs a token with the given client ID, max age (seconds) and one or more claims.
func (s *Server[UserData]) SignToken(clientID string, accessMaxAge, refreshMaxAge int, claims ...any) (*oauth2.Token, error) {
	if len(claims) == 0 {
		return nil, fmt.Errorf("oauth2: no claims provided")
	}

	// Set the standard "aud" claim field to the client ID for both access and refresh tokens.
	// And the max age for both tokens.
	accessOptions := ClientTokenSignOptions(clientID, accessMaxAge)
	refreshOptions := ClientTokenSignOptions(clientID, refreshMaxAge)

	mergedClaims, err := types.MergeJSON(claims...)
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to merge claims: %w", err)
	}

	token, err := s.Auth.SignToken(mergedClaims, auth.AccessTokenOption(accessOptions...), auth.RefreshTokenOption(refreshOptions...))
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to sign token: %w", err)
	}
	return token, nil
}

var clientContextKey = "iris.auth.oauth2.client"

func GetClient(ctx *context.Context) (*Client, bool) {
	return ctx.ValueAs[*Client](clientContextKey)
}

var (
	// ErrClientNotFound is returned when the client is not found in the client store.
	ErrClientNotFound = fmt.Errorf("oauth2: client not found")
	// ErrClientAlreadyExists is returned by CreateClient when a client with the same name exists.
	ErrClientAlreadyExists = errors.New("oauth2: client already exists")
	// ErrClientMissing is returned when the client id and client secret are missing from the token.
	ErrClientMissing = errors.New("client id and client secret are required")
	// ErrClientMismatch is returned when the client's secret or version does not match
	// the stored client, or when the client_id claim does not match the token's audience.
	ErrClientMismatch = fmt.Errorf("client_id or secret or version mismatch")
)

// ClientVerifier returns a VerifyUserFunc that verifies the full client (auth) claims.
func (s *Server[UserData]) ClientVerifier() auth.VerifyUserFunc[ClientTokenClaims] { // see auth#VerifyHandler.
	return func(ctx stdContext.Context, claims ClientTokenClaims) error {
		if claims.ClientID == "" {
			return fmt.Errorf("%w: client id is empty", ErrClientMissing)
		}

		client, ok, err := s.options.ClientStore.GetClient(ctx, claims.ClientID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrClientNotFound
		}

		if !secureCompare(client.Secret, claims.ClientSecret) {
			return ErrClientMismatch
		}

		if client.Version != claims.Version {
			return ErrClientMismatch
		}

		if irisCtx, ok := ctx.(*context.Context); ok {
			// Set the claims context value here (see oauth2.GetClientClaims).
			irisCtx.Values().Set(clientClaimsContextKey, claims)
			// And the Client instance itself, so we don't have to re-fetch it from the store (see oauth2.GetClient).
			irisCtx.Values().Set(clientContextKey, client)
		}

		// reachable on auth.GetUser[ClientTokenClaims].
		return nil
	}
}

func (s *Server[UserData]) buildCallbackURL(ctx *context.Context, req AuthorizeRequest) (string, bool, error) {
	if req.ClientID == "" {
		return "", false, fmt.Errorf("missing client_id")
	}

	// Check client existence.
	client, ok, err := s.options.ClientStore.GetClient(ctx, req.ClientID)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, fmt.Errorf("authorize: %w", ErrClientMissing)
	}
	if !client.MatchScopes(req.Scopes()) {
		return "", false, fmt.Errorf("authorize: %w", ErrClientInsufficientScopes)
	}

	userData, _, err := s.Auth.Verify(ctx, []byte(s.Auth.ExtractAccessToken(ctx)))
	if err != nil { // If the user is not authenticated, redirect the user to the login page.
		redirectBackURL := ctx.Request().RequestURI
		redirectURI := s.options.LoginPageURL + "?redirect_to=" + url.QueryEscape(redirectBackURL)
		return redirectURI, false, nil
	}

	// We don't allow setting the redirectURI from the client, the client should have a specific URL set.
	// redirectURI := form.Get("redirect_uri")
	// if client.RedirectURI != "" { // if client has a specific URL set it.
	// 	redirectURI = client.RedirectURI
	// }
	redirectURI := client.RedirectURI

	authCode, err := NewAuthCode(req.ClientID, userData, req.State, req.Scopes(), client.CodeMaxAge)
	if err != nil {
		return "", false, err
	}
	if err = applyPKCE(authCode, req.CodeChallenge, req.CodeChallengeMethod); err != nil {
		return "", false, fmt.Errorf("authorize: %w", err)
	}

	// Store the auth code which maps the temp code with the client id and the user data (we need this because go client will no have access to the cookie or the header information, maybe we need a cooie store for these storages).
	if err = s.options.AuthCodeStore.InsertAuthCode(ctx, authCode); err != nil {
		return "", false, err
	}

	// Redirect to the client callback URL with the auth temp code.
	// The callback URL should be able to handle the auth code and exchange it with an access token at the /token endpoint.
	callbackURI := authCode.URL(redirectURI)
	return callbackURI, true, nil
}

// processes the "authorization_code" flow requests.
func (s *Server[UserData]) handleAuthCode(ctx *context.Context) {
	var req AuthorizeRequest
	if err := ctx.ReadQuery(&req); err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, err)
		return
	}

	redirectURI, _, err := s.buildCallbackURL(ctx, req)
	if err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, err)
		return
	}
	ctx.Redirect(redirectURI)
}

// processes the request to show a consent screen.
func (s *Server[UserData]) askConsent(ctx *context.Context) {
	var req AuthorizeRequest
	if err := ctx.ReadQuery(&req); err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, err)
		return
	}

	if req.ClientID == "" {
		s.Auth.ErrorHandler.InvalidArgument(ctx, fmt.Errorf("missing client_id"))
		return
	}

	client, ok, err := s.options.ClientStore.GetClient(ctx, req.ClientID)
	if err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, err)
		return
	}
	if !ok {
		// remove the cookie as well, as the client removed.
		ctx.RemoveCookie(req.ConsentCookieName())

		s.Auth.ErrorHandler.InvalidArgument(ctx, fmt.Errorf("client not found"))
		return
	}
	if !client.MatchScopes(req.Scopes()) {
		// remove the cookie as well, as the client changed its scopes, so we must ask the user again.
		ctx.RemoveCookie(req.ConsentCookieName())

		s.Auth.ErrorHandler.InvalidArgument(ctx, fmt.Errorf("client does not have requested scopes"))
		return
	}

	// accessToken := opts.TokenExtractor.ExtractAccessToken(ctx)
	// if accessToken == "" { // redirect to login page, and give a redirect_to query parameter to redirect back here and continue the flow.
	// 	redirectBackURL := ctx.Request().RequestURI
	// 	ctx.Redirect(opts.LoginPageURL + "?redirect_to=" + url.QueryEscape(redirectBackURL))
	// 	return
	// }

	_, _, err = s.Auth.Verify(ctx, []byte(s.Auth.ExtractAccessToken(ctx)))
	if err != nil { // redirect to login page, and give a redirect_to query parameter to redirect back here and continue the flow.
		redirectBackURL := ctx.Request().RequestURI
		ctx.Redirect(s.options.LoginPageURL + "?redirect_to=" + url.QueryEscape(redirectBackURL))
		return
	}

	// Check if consent has already been granted using a cookie.
	cookie, err := ctx.Cookie(req.ConsentCookieName())
	if err == nil {
		if cookie.Value == req.Scope { // don't check for cookie's expiration, as for retrieved cookies these fields are empty.
			redirectURI, _, err := s.buildCallbackURL(ctx, req)
			if err != nil {
				s.Auth.ErrorHandler.InvalidArgument(ctx, err)
				return
			}

			ctx.Redirect(redirectURI)
			return
		} else {
			ctx.RemoveCookie(cookie.Name)
		}
	}

	// Redirect to confirm access screen with the nessesary data.
	redirectURI := fmt.Sprintf("%s?client_id=%s&client_name=%s&client_website=%s&client_logo_url=%s&state=%s&scope=%s",
		s.options.ConsentPageURL, req.ClientID, client.Name, url.QueryEscape(client.Website), url.QueryEscape(client.LogoURL), req.State, url.QueryEscape(req.Scope))
	ctx.Redirect(redirectURI)
}

// ConfirmConsentRequest is the form data submitted by the user on the consent page.
type ConfirmConsentRequest struct {
	// Decision stores the user's response: "approve" or "deny."
	Decision string `url:"decision" form:"decision"`
	// Remember indicates whether the user wants to remember this decision for future requests.
	Remember string `url:"remember" form:"remember"`

	AuthorizeRequest
}

// Approved returns true if the user decided to approve the requested scopes.
func (req *ConfirmConsentRequest) Approved() bool {
	return req.Decision == DecisionApprove
}

// ShouldRemember returns true if the user wants to remember the consent decision.
func (req *ConfirmConsentRequest) ShouldRemember() bool {
	return req.Remember == "1"
}

// processes the request to confirm or deny consent.
func (s *Server[UserData]) confirmConsent(ctx *context.Context) {
	var req ConfirmConsentRequest
	if err := ctx.ReadForm(&req); err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, err)
		return
	}

	if !req.Approved() {
		s.Auth.ErrorHandler.Unauthenticated(ctx, fmt.Errorf("access denied"))
		return
	}

	redirectURI, authorized, err := s.buildCallbackURL(ctx, req.AuthorizeRequest)
	if err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, err)
		return
	}

	if authorized {
		if req.ShouldRemember() { // we set the cookie only when auth code succeed and that's why we accept a special auth code handler on options instead just of a single iris Handler.
			// Store the user's consent decision in a cookie.
			cookie := &http.Cookie{
				Name:     req.ConsentCookieName(),
				Value:    req.Scope,
				Path:     "/",
				Expires:  Clock().Add(365 * 24 * time.Hour), // 1 year.
				MaxAge:   365 * 24 * 60 * 60,
				HttpOnly: true,
				Secure:   ctx.IsSSL(),
			}
			ctx.SetCookie(cookie) // if we redirect on 3xx status, this cookie is lost as browsers don't respect setting cookies on 3xx responses.
		}
	}

	ctx.Redirect(redirectURI) // login URL and then back to consent screen or auth code url.
}

// processes the request to revoke consent.
func (s *Server[UserData]) revokeConsent(ctx *context.Context) {
	req := AuthorizeRequest{
		ClientID: ctx.Query().Get("client_id"),
	}
	ctx.RemoveCookie(req.ConsentCookieName())
	ctx.StatusCode(204) // No-Content.
}

// processes the "device_code" flow requests.
func (s *Server[UserData]) handleDeviceCode(ctx *context.Context) {
	var req AuthorizeRequest
	if err := ctx.ReadForm(&req); err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, fmt.Errorf("%w: %w", ErrClientMissing, err))
		return
	}

	client, ok, err := s.options.ClientStore.GetClient(ctx, req.ClientID)
	if err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, err)
		return
	}
	if !ok {
		s.Auth.ErrorHandler.InvalidArgument(ctx, ErrClientMissing)
		return
	}
	if client.VerificationURI == "" {
		s.Auth.ErrorHandler.InvalidArgument(ctx, fmt.Errorf("client has no verification URI, not applicable for device code flow"))
		return
	}

	// If the user is not authenticated, redirect the user to the login page.
	// accessToken := opts.Auth.ExtractAccessToken(ctx)
	// // if !auth.UserAuthorized(ctx) {
	// if accessToken == "" {
	// 	redirectBackURL := ctx.Request().RequestURI
	// 	ctx.Redirect(opts.LoginPageURL + "?redirect_to=" + url.QueryEscape(redirectBackURL))
	// 	return
	// }
	// userData, _, err := opts.Auth.Verify(ctx, []byte(accessToken))
	// if err != nil {
	// 	opts.ErrorHandler.Unauthenticated(ctx, fmt.Errorf("user is not authenticated: %w", err))
	// 	return
	// }

	userData, _, err := s.Auth.Verify(ctx, []byte(s.Auth.ExtractAccessToken(ctx)))
	if err != nil { // If the user is not authenticated, redirect the user to the login page.
		redirectBackURL := ctx.Request().RequestURI
		ctx.Redirect(s.options.LoginPageURL + "?redirect_to=" + url.QueryEscape(redirectBackURL))
		return
	}

	deviceData, err := NewDeviceCode(req.ClientID, userData, s.options.DeviceInterval, req.Scopes(), client.CodeMaxAge)
	if err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, fmt.Errorf("failed to generate a new device code: %w", err))
		return
	}

	deviceAuthResponse := deviceData.AuthResponse(client.VerificationURI)
	ctx.JSON(deviceAuthResponse)
}

// Token is an alias of the oauth2.Token type.
type Token = oauth2.Token

const (
	// GrantTypeAuthorizationCode is the default grant type.
	// It's used when the client has an authorization code.
	GrantTypeAuthorizationCode = "authorization_code" // the default one.
	// GrantTypeDeviceCode is used when the client has a device code.
	GrantTypeDeviceCode = "urn:ietf:params:oauth:grant-type:device_code"
	// GrantTypePassword is used when the client has a username and password.
	GrantTypePassword = "password"
	// GrantTypeRefreshToken is used when the client wants to refresh the token.
	GrantTypeRefreshToken = "refresh_token"
)

// newTokenRequest contains the parameters required to request a token
// from the OAuth2 server, supporting multiple grant types.
type newTokenRequest struct {
	AuthorizeRequest

	// ClientSecret is used to authenticate confidential clients.
	ClientSecret string `json:"client_secret" url:"client_secret" form:"client_secret"`
	// GrantType defines which OAuth2 flow to use.
	GrantType string `json:"grant_type" url:"grant_type" form:"grant_type"`
	// Code is the authorization code for auth code grant flow.
	Code string `json:"code" url:"code" form:"code"`
	// DeviceCode is the device code used in device code grant flow.
	DeviceCode string `json:"device_code" url:"device_code" form:"device_code"`
	// RefreshToken is used to request a new access token.
	RefreshToken string `json:"refresh_token" url:"refresh_token" form:"refresh_token"`
	// Username for password grant flow.
	Username string `json:"username" form:"username"`
	// Password for password grant flow.
	Password string `json:"password" form:"password"`
	// CodeVerifier is the PKCE verifier for an authorization code that was issued with a challenge.
	CodeVerifier string `json:"code_verifier" url:"code_verifier" form:"code_verifier"`
}

// processes token issuance requests for multiple grant types.
func (s *Server[UserData]) handleTokenRequest(ctx *context.Context) {
	var req newTokenRequest
	if err := ctx.ReadForm(&req); err != nil {
		s.Auth.ErrorHandler.InvalidArgument(ctx, err)
		return
	}

	token, err := s.newToken(ctx, req)
	if err != nil {
		s.Auth.ErrorHandler.Unauthenticated(ctx, err)
		return
	}

	ctx.JSON(token)
}

// newToken creates a new OAuth2 token based on the request's grant type,
// validating client and user credentials when necessary, and returning the token.
// It supports the following grant types:
// - authorization_code
// - device_code
// - password
// - refresh_token
//
// The client must have access to the requested grant type.
func (s *Server[UserData]) newToken(ctx *context.Context, req newTokenRequest) (*oauth2.Token, error) {
	if req.GrantType == "" {
		req.GrantType = GrantTypeAuthorizationCode
	}

	if req.ClientID == "" || (req.Code == "" && req.DeviceCode == "" && req.RefreshToken == "" && (req.Username == "" || req.Password == "")) {
		return nil, fmt.Errorf("new token: %w: missing client_id or code parameters", ErrClientMissing)
	}

	client, ok, err := s.options.ClientStore.GetClient(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}

	if !ok || !secureCompare(client.Secret, req.ClientSecret) {
		return nil, fmt.Errorf("new token: %w", ErrClientMismatch)
	}

	if !client.MatchGrantType(req.GrantType) {
		return nil, fmt.Errorf("new token: client has no access to the requested grant type: %s", req.GrantType)
	}

	var (
		userData AuthUserData
		scopes   []string
	)
	switch req.GrantType {
	case GrantTypeAuthorizationCode:
		userData, scopes, err = s.extractFromAuthorizationCode(ctx, client, req.Code, req.State, req.CodeVerifier)
	case GrantTypeDeviceCode:
		userData, scopes, err = s.extractFromDeviceCode(ctx, client, req.DeviceCode)
	case GrantTypePassword:
		userData, scopes, err = s.extractFromPassword(ctx, client, req.Username, req.Password, req.Scopes())
	case GrantTypeRefreshToken:
		userData, scopes, err = s.extractFromRefreshToken(ctx, client, req.RefreshToken)
	default:
		err = fmt.Errorf("unsupported grant type: %s", req.GrantType)
	}
	if err != nil {
		return nil, fmt.Errorf("new token: %w", err)
	}

	if !client.MatchScopes(scopes) {
		return nil, fmt.Errorf("new token:%w", ErrClientInsufficientScopes)
	}

	clientClaims := client.Claims(scopes) // do not include the client secret in the token, ofc.
	return s.SignToken(client.ID, client.AccessMaxAge, client.RefreshMaxAge, userData, clientClaims)
}

func (s *Server[UserData]) extractFromAuthorizationCode(ctx stdContext.Context, client *Client, code, state, verifier string) (AuthUserData, []string, error) {
	authData, ok, err := s.options.AuthCodeStore.GetAndDeleteAuthCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if !ok || authData.ClientID != client.ID {
		return nil, nil, fmt.Errorf("invalid client credentials")
	}
	if authData.HasExpired() {
		return nil, nil, fmt.Errorf("auth code expired")
	}
	// if !client.MatchScopes(authData.Scopes) {
	// 	return nil, nil,fmt.Errorf("client has no access to the requested scopes")
	// }

	if authData.State != state {
		return nil, nil, fmt.Errorf("invalid state")
	}
	if err = checkPKCE(authData, verifier); err != nil {
		return nil, nil, err
	}

	return authData.UserData, authData.Scopes, nil
}

func (s *Server[UserData]) extractFromDeviceCode(ctx stdContext.Context, client *Client, deviceCode string) (AuthUserData, []string, error) {
	deviceData, ok, err := s.options.DeviceCodeStore.GetAndDeleteDeviceCode(ctx, deviceCode)
	if err != nil {
		return nil, nil, err
	}
	if !ok || deviceData.ClientID != client.ID {
		return nil, nil, fmt.Errorf("invalid client credentials")
	}
	if deviceData.HasExpired() {
		return nil, nil, fmt.Errorf("device code expired")
	}

	return deviceData.UserData, deviceData.Scopes, nil
}

func (s *Server[UserData]) extractFromPassword(ctx stdContext.Context, client *Client, username, password string, scopes []string) (AuthUserData, []string, error) {
	// Validate the username and password.
	userData, err := s.Auth.SigninProvider(ctx, username, password)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid username or password: %w", err)
	}

	if len(scopes) == 0 {
		scopes = client.Scopes // give all scopes if none provided, as password grant type is the most privileged one.
	}

	return userData, scopes, nil
}

func (s *Server[UserData]) extractFromRefreshToken(ctx stdContext.Context, client *Client, refreshToken string) (AuthUserData, []string, error) {
	userData, claims, _, err := s.Auth.VerifyExtra[ClientTokenClaims](ctx, []byte(refreshToken))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify refresh token: %w", err)
	}

	if claims.ClientID != client.ID {
		return nil, nil, fmt.Errorf("invalid client credentials")
	}

	if claims.Version != client.Version {
		return nil, nil, fmt.Errorf("token version mismatch")
	}

	return userData, claims.Scopes, nil
}

func (s *Server[UserData]) introspectToken(ctx *context.Context) {
	var accessToken string

	if ctx.IsMediaType(context.ContentJSONHeaderValue) {
		// try to read token from the request body.
		var tokenRequest = struct {
			Token       string `json:"token"` // or
			AccessToken string `json:"access_token"`
		}{}
		if err := ctx.ReadJSON(&tokenRequest); err != nil && !context.IsErrEmptyJSON(err) {
			s.Auth.ErrorHandler.InvalidArgument(ctx, err)
			return
		}

		if tokenRequest.Token != "" {
			accessToken = tokenRequest.Token
		} else if tokenRequest.AccessToken != "" {
			accessToken = tokenRequest.AccessToken
		} else {
			s.Auth.ErrorHandler.Unauthenticated(ctx, fmt.Errorf("json: missing access_token"))
			return
		}
	}

	if accessToken == "" { // if it's not in the body, try to read it from the header or any given extract method.
		accessToken = s.Auth.ExtractAccessToken(ctx)
		if accessToken == "" {
			s.Auth.ErrorHandler.Unauthenticated(ctx, fmt.Errorf("missing access token"))
			return
		}
	}

	_, verifiedToken, err := s.Auth.Verify(ctx, []byte(accessToken))
	if err != nil {
		s.Auth.ErrorHandler.Unauthenticated(ctx, err)
		return
	}

	ctx.SetContentType(context.ContentJSONHeaderValue)
	ctx.Write(verifiedToken.Payload)
}

// ClientTokenClaims holds client-specific data embedded in tokens (e.g., client ID and scopes).
type ClientTokenClaims struct {
	// ClientID is the unique ID of the client requesting the token.
	ClientID string `json:"client_id" form:"client_id" url:"client_id"`
	// ClientName is the name of the client application.
	ClientName string `json:"client_name" form:"client_name" url:"client_name"`
	// ClientWebsite is the website URL of the client application.
	ClientWebsite string `json:"client_website" form:"client_website" url:"client_website"`
	// ClientLogoURL is the URL of the client's logo.
	ClientLogoURL string `json:"client_logo_url" form:"client_logo_url" url:"client_logo_url"`
	// Scopes are the list of permissions or resources the client may access.
	Scopes []string `json:"scopes" form:"scopes" url:"scopes"`

	// ClientSecret is the client's secret key.
	// It's only available outside of oauth2 routes,
	// when the app is server-side and has rights to write to the users.
	ClientSecret string `json:"client_secret,omitempty" form:"client_secret,omitempty" url:"client_secret,omitempty"`

	// Used to verify the token's version. Useful for invalidating tokens.
	Version int `json:"version" form:"-" url:"-"`
}

// MatchPolicies reports whether the token satisfies at least one of the requiredPolicies.
// It returns true when requiredPolicies is empty.
func (c *ClientTokenClaims) MatchPolicies(ctx stdContext.Context, requiredPolicies []Policy) bool {
	if len(requiredPolicies) == 0 {
		return true
	}

	for _, policy := range requiredPolicies {
		if policy.Matches(ctx, c) {
			return true
		}
	}

	return false
}
