// Package auth provides generic, token-based authentication for Iris applications.
//
// The central type is Auth[T], which signs and verifies JSON Web Tokens for a
// user type T through pluggable Provider implementations. It supports access and
// refresh tokens with rotation, optional secure-cookie storage, and ready-made
// HTTP handlers for the sign-in, verify, refresh and sign-out flows
// (SigninHandler, VerifyHandler, RefreshHandler, SignoutHandler).
//
// The auth/identity subpackage builds a full user-management service on top of
// this package, and auth/oauth2 implements an OAuth2 authorization server.
package auth

import (
	stdContext "context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/uid"
	"golang.org/x/oauth2"

	"github.com/gorilla/securecookie"
	"github.com/kataras/jwt"
)

func init() {
	context.SetHandlerName(`.*\(\*Auth\[.*\]\)\.VerifyHandler\..*`, "iris.auth")
}

type (
	// Auth holds the necessary functionality to authorize and optionally authenticating
	// users to access and perform actions against the resource server (Iris API).
	// It completes a secure and fast JSON Web Token signer and verifier which,
	// based on the custom application needs, can be further customized.
	// Each Auth of T instance can sign and verify a single custom <T> instance,
	// more Auth instances can share the same configuration to support multiple custom user types.
	// Initialize a new Auth of T instance using the New or MustLoad package-level functions.
	// Most important methods of the instance are:
	// - AddProvider
	// - SigninHandler
	// - VerifyHandler
	// - SignoutHandler
	// - SignoutAllHandler
	//
	// Example can be found at: https://github.com/kataras/iris/tree/main/_examples/09-authentication/auth-sdk/main.go.
	Auth[T User] struct {
		// Holds the configuration passed through the New and MustLoad
		// package-level functions. One or more Auth instance can share the
		// same configuration's values.
		config Configuration
		// Holds the result of the config.KeysConfiguration.
		keys jwt.Keys
		// This is an Iris cookie option used to encrypt and decrypt a cookie when
		// the config.Cookie.Hash & Block are not empty.
		securecookie context.SecureCookie
		// Defaults to an empty list, which cannot sign any tokens.
		// One or more custom providers should be registered through
		// the AddProvider or WithProviderAndErrorHandler methods.
		providers []Provider[T] // at least one.
		// Always not nil, set to custom error handler on SetErrorHandler.
		ErrorHandler ErrorHandler
		// Not nil if a transformer is registered.
		transformer Transformer[T]
		// Not nil if a custom claims provider is registered.
		claimsProvider ClaimsProvider
		// True if KIDRefresh on config.Keys.
		refreshEnabled bool

		// The registered validators for the token.
		// They are called in the order they are registered.
		// The first one that returns an error will stop the validation process.
		//
		// See `s.verify`.
		Validators []jwt.TokenValidatorFunc
	}

	// VerifyUserFunc is passed on Verify and VerifyHandler method
	// to, optionally, further validate a T user value.
	VerifyUserFunc[T User] func(ctx stdContext.Context, t T) error

	// SigninRequest is the request body the server expects
	// on SigninHandler. The Password and Username or Email should be filled.
	SigninRequest struct {
		Username string `json:"username" form:"username,omitempty"` // username OR email, username has priority over email.
		Email    string `json:"email" form:"email,omitempty"`       // email OR username.
		Password string `json:"password" form:"password"`
	}

	// RefreshRequest is the request body the server expects
	// on RefreshHandler to renew an access and refresh token pair.
	RefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}
)

// MustLoad binds a filename (fullpath) configuration yaml or json
// and constructs a new Auth instance. It panics on error.
func MustLoad[T User](filename string) *Auth[T] {
	var config Configuration
	if err := config.BindFile(filename); err != nil {
		panic(err)
	}

	return Must[T](config)
}

// Must calls New with the same configuration and panics if it returns an error.
// It takes what New takes and is intended for variable initializations such as
//
//	var s = auth.Must[MyUser](config)
func Must[T User](config Configuration) *Auth[T] {
	s, err := New[T](config)
	if err != nil {
		panic(err)
	}

	return s
}

var instances = make(map[string]any)

// Resolve tries to return the Auth of T instance based on the type of T.
func Resolve[T User]() (*Auth[T], bool) {
	var t T
	if v, ok := instances[reflect.TypeOf(t).String()]; ok {
		if auth, ok := v.(*Auth[T]); ok {
			return auth, true
		}
	}

	return nil, false
}

// New initializes a new Auth instance typeof T and returns it.
// The T generic can be any custom struct.
// It accepts a Configuration value which can be constructed
// manually or through a configuration file using the
// MustGenerateConfiguration or MustLoadConfiguration
// or LoadConfiguration or MustLoad package-level functions.
//
// Example can be found at: https://github.com/kataras/iris/tree/main/_examples/09-authentication/auth-sdk/main.go.
func New[T User](config Configuration) (*Auth[T], error) {
	keys, err := config.validate()
	if err != nil {
		return nil, err
	}
	_, refreshEnabled := keys[KIDRefresh]

	s := &Auth[T]{
		config:         config,
		keys:           keys,
		securecookie:   securecookie.New([]byte(config.Cookie.Hash), []byte(config.Cookie.Block)),
		refreshEnabled: refreshEnabled,
		// providers:    []Provider[T]{newProvider[T]()},
		ErrorHandler: new(DefaultErrorHandler),
	}

	var t T
	instances[reflect.TypeOf(t).String()] = s

	return s, nil
}

// Config returns the configuration of this Auth of T instance.
func (s *Auth[T]) Config() Configuration {
	return s.config
}

// Keys returns the keys loaded by the keys configuration of this Auth of T instance.
func (s *Auth[T]) Keys() jwt.Keys {
	return s.keys
}

// WithProviderAndErrorHandler registers a provider (if not nil) and
// an error handler (if not nil) and returns this "s" Auth instance.
// Unlike AddProvider, which appends, it replaces any previously registered
// providers with the given one, then it sets the error handler as SetErrorHandler does.
// It's really useful when registering an Auth instance using the iris.Router.Controller
// method when a Provider of T and ErrorHandler is available through the registered Router's dependencies.
//
// Usage Example:
//
//	api := app.Router("/api")
//	api.RegisterDependency(
//	  NewAuthProviderErrorHandler(),
//	  NewAuthCustomerProvider,
//	  auth.Must[Customer](authConfig).WithProviderAndErrorHandler,
//	)
func (s *Auth[T]) WithProviderAndErrorHandler(provider Provider[T], errHandler ErrorHandler) *Auth[T] {
	if provider != nil {
		for i := range s.providers {
			s.providers[i] = nil
		}
		s.providers = nil

		s.providers = make([]Provider[T], 0, 1)
		s.AddProvider(provider)
	}

	if errHandler != nil {
		s.SetErrorHandler(errHandler)
	}

	return s
}

// WithProvider registers a provider (if not nil) and returns this "s" Auth instance.
func (s *Auth[T]) WithProvider(provider Provider[T]) *Auth[T] {
	if provider != nil {
		s.AddProvider(provider)
	}

	return s
}

// AddProvider registers one or more providers to this Auth of T instance and returns itself.
// Look the Provider godoc for more.
func (s *Auth[T]) AddProvider(providers ...Provider[T]) *Auth[T] {
	// A provider can also implement both transformer and
	// error handler if that's the design option of the end-developer.
	for _, p := range providers {
		if s.transformer == nil {
			if transformer, ok := p.(Transformer[T]); ok {
				s.SetTransformer(transformer)
			}
		}

		if errHandler, ok := p.(ErrorHandler); ok {
			s.SetErrorHandler(errHandler)
		}

		if s.claimsProvider == nil {
			if claimsProvider, ok := p.(ClaimsProvider); ok {
				s.claimsProvider = claimsProvider
			}
		}
	}

	s.providers = append(s.providers, providers...)
	return s
}

// SetErrorHandler sets a custom error handler to this Auth of T instance and returns itself.
// Look the Provider and ErrorHandler godoc for more.
func (s *Auth[T]) SetErrorHandler(errHandler ErrorHandler) *Auth[T] {
	s.ErrorHandler = errHandler
	return s
}

// SetTransformer sets a custom transformer to this Auth of T instance and returns itself.
// Look the Provider and Transformer godoc for more.
func (s *Auth[T]) SetTransformer(transformer Transformer[T]) *Auth[T] {
	s.transformer = transformer
	return s
}

// SetTransformerFunc like SetTransformer method but accepts a function instead.
func (s *Auth[T]) SetTransformerFunc(transfermerFunc func(ctx stdContext.Context, tok *VerifiedToken) (T, error)) *Auth[T] {
	s.transformer = TransformerFunc[T](transfermerFunc)
	return s
}

// SetValidators appends the given custom validators to this Auth of T instance and
// returns itself. Repeated calls accumulate; set the exported Validators field
// directly to replace or clear them.
func (s *Auth[T]) SetValidators(validators ...jwt.TokenValidatorFunc) *Auth[T] {
	s.Validators = append(s.Validators, validators...)
	return s
}

// ErrNoProvider is returned when no provider is registered to an Auth of T instance.
var ErrNoProvider = fmt.Errorf("auth: no provider registered")

// Signin signs a token based on the provided username and password
// and returns a pair of access and refresh tokens.
//
// Signin calls the Provider.Signin method to check if a user
// is authenticated by the given username and password combination.
func (s *Auth[T]) Signin(ctx stdContext.Context, username, password string) (*oauth2.Token, error) {
	t, err := s.SigninProvider(ctx, username, password)
	if err != nil {
		return nil, err
	}

	// sign the tokens.
	tok, err := s.SignToken(t)
	if err != nil {
		return nil, fmt.Errorf("auth: signin: %w", err)
	}

	return tok, nil
}

// SigninProvider verifies the username and password against the registered providers
// and returns the custom T value.
func (s *Auth[T]) SigninProvider(ctx stdContext.Context, username, password string) (T, error) {
	var t T

	// get "t" from a valid provider.
	if n := len(s.providers); n > 0 {
		for i := range n {
			p := s.providers[i]

			v, err := p.Signin(ctx, username, password)
			if err != nil {
				if i == n-1 { // last provider errored.
					return t, fmt.Errorf("auth: signin: %w", err)
				}
				// keep searching.
				continue
			}

			// found.
			t = v
			break
		}
	} else {
		return t, ErrNoProvider
	}

	return t, nil
}

// AccessTokenOption returns a jwt.SignOptionFunc which can be used to apply options
// only to the access token's key.
// It checks the OriginID field of the claims, if it's empty then it applies the options
// to the access token's key, otherwise it does nothing.
func AccessTokenOption(opts ...jwt.SignOption) jwt.SignOption {
	return jwt.SignOptionFunc(func(claims *jwt.Claims) {
		if claims.OriginID == "" { // means access token (see SignToken below).
			for _, opt := range opts {
				if opt == nil {
					continue
				}
				opt.ApplyClaims(claims)
			}
		}
	})
}

// RefreshTokenOption returns a jwt.SignOptionFunc which can be used to apply options
// only to the refresh token's key.
func RefreshTokenOption(opts ...jwt.SignOption) jwt.SignOption {
	return jwt.SignOptionFunc(func(claims *jwt.Claims) {
		if claims.OriginID != "" { // means refresh token (see SignToken below).
			for _, opt := range opts {
				if opt == nil {
					continue
				}

				opt.ApplyClaims(claims)
			}
		}
	})
}

// SignToken signs a token based on the provided custom T value and returns a pair of access and refresh tokens.
func (s *Auth[T]) SignToken(t any /* Allow any type so more fields can be added dynamically */, opts ...jwt.SignOption) (*oauth2.Token, error) {
	// sign the tokens.
	var (
		accessStdClaims  StandardClaims
		refreshStdClaims StandardClaims
	)

	if s.claimsProvider != nil {
		accessStdClaims = s.claimsProvider.GetAccessTokenClaims()
		refreshStdClaims = s.claimsProvider.GetRefreshTokenClaims(accessStdClaims)
	}

	now := jwt.Clock()
	iat := now.Unix()

	if accessStdClaims.IssuedAt == 0 {
		accessStdClaims.IssuedAt = iat
	}

	if accessStdClaims.ID == "" {
		// Deliberately version 4 (random): a token id must not leak its mint time.
		accessStdClaims.ID = uid.NewString()
	}

	if refreshStdClaims.IssuedAt == 0 {
		refreshStdClaims.IssuedAt = iat
	}

	if refreshStdClaims.ID == "" {
		refreshStdClaims.ID = uid.NewString()
	}

	if refreshStdClaims.OriginID == "" {
		// keep a reference of the access token the refresh token is created,
		// if that access token is invalidated then
		// its refresh token should be too so the user can force-login.
		refreshStdClaims.OriginID = accessStdClaims.ID
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt.ApplyClaims(&accessStdClaims)
		opt.ApplyClaims(&refreshStdClaims)
	}

	accessToken, err := s.keys.SignToken(KIDAccess, t, accessStdClaims)
	if err != nil {
		return nil, fmt.Errorf("access: %w", err)
	}

	var refreshToken []byte
	if s.refreshEnabled {
		refreshToken, err = s.keys.SignToken(KIDRefresh, t, refreshStdClaims)
		if err != nil {
			return nil, fmt.Errorf("refresh: %w", err)
		}
	}

	// extract max age to set the oauth2.Token instance,
	// first the access token's max age option, and then fromt he configuration's keys.
	maxAgeAccessToken := accessStdClaims.Age()
	if maxAgeAccessToken == 0 {
		if accessKey, ok := s.keys.Get(KIDAccess); ok {
			maxAgeAccessToken = accessKey.MaxAge
		}
	}

	token := &oauth2.Token{
		AccessToken:  string(accessToken),
		TokenType:    "Bearer",
		RefreshToken: string(refreshToken),
		Expiry:       now.Add(maxAgeAccessToken),
		ExpiresIn:    int64(maxAgeAccessToken.Seconds()),
	}

	return token, nil
}

// SigninHandler generates and sends a pair of access and refresh token to the client
// as JSON body of `oauth2.Token` and cookie (if cookie setting was provided).
// If a safe, same-site relative "redirect_to" (or "redirect_uri") query parameter
// is present, it redirects there instead of writing the JSON body.
// See `Signin` method for more.
func (s *Auth[T]) SigninHandler(ctx *context.Context) {
	// No, let the developer decide it based on a middleware, e.g. bodylimit.New.
	// app.Use(bodylimit.New(n))

	var (
		req SigninRequest
		err error
	)

	switch ctx.MediaType() {
	case context.ContentFormHeaderValue, context.ContentFormMultipartHeaderValue:
		err = ctx.ReadForm(&req)
		if context.IsErrPath(err) {
			err = nil
		} // if any other field... this is useful to continue when decoder fails on redirect uris, so we can pass redirect uris without errors here.
		if (req.Email == "" && req.Username == "") || req.Password == "" {
			s.ErrorHandler.InvalidArgument(ctx, fmt.Errorf("missing credentials"))
			return
		}
	default:
		err = ctx.ReadJSON(&req)
	}

	if err != nil {
		s.ErrorHandler.InvalidArgument(ctx, err)
		return
	}

	if req.Username == "" {
		req.Username = req.Email
	}

	tok, err := s.Signin(ctx, req.Username, req.Password)
	if err != nil {
		s.tryRemoveCookie(ctx) // remove cookie on invalidated.

		s.ErrorHandler.Unauthenticated(ctx, err)
		return
	}

	s.trySetCookie(ctx, tok.AccessToken)

	redirectTo := ctx.Query().Get("redirect_to")
	if redirectTo == "" {
		redirectTo = ctx.Query().Get("redirect_uri")
	}
	if redirectTo != "" {
		// If the client wants to redirect to another page after the token stored in cookie.
		if unescaped, _ := url.QueryUnescape(redirectTo); unescaped != "" {
			redirectTo = unescaped
		}
		// Only redirect to safe, same-site relative paths to prevent open redirects
		// (v14). Absolute, protocol-relative or scheme-bearing targets are ignored
		// and the token is returned as JSON instead.
		if isSafeRelativeRedirect(redirectTo) {
			ctx.Redirect(redirectTo)
			return
		}
	}

	ctx.JSON(tok)
}

// isSafeRelativeRedirect reports whether target is a safe, same-site relative
// redirect: a path beginning with a single "/" with no scheme or host. It rejects
// absolute URLs ("https://evil"), scheme targets ("javascript:"), protocol-relative
// ("//evil") and backslash-obfuscated ("/\evil") targets that browsers may resolve
// to an attacker-controlled origin.
func isSafeRelativeRedirect(target string) bool {
	if target == "" || target[0] != '/' {
		return false
	}
	if strings.HasPrefix(target, "//") {
		return false
	}
	// Reject backslashes (browsers may treat "/\" as "//") and any ASCII control
	// character (NUL, CR, LF, tab, DEL) which could enable header/response
	// splitting once the value is used in a redirect.
	return !strings.ContainsFunc(target, func(r rune) bool {
		return r == '\\' || r < 0x20 || r == 0x7f
	})
}

// Verify accepts a token and verifies it.
// It returns the token's custom claims and the verified token information.
func (s *Auth[T]) Verify(ctx stdContext.Context, token []byte, verifyFuncs ...VerifyUserFunc[T]) (T, *VerifiedToken, error) {
	t, verifiedToken, err := s.verify(ctx, token)
	if err != nil {
		return t, nil, fmt.Errorf("auth: verify: %w", err)
	}

	for _, verify := range verifyFuncs {
		if verify == nil {
			continue
		}

		if err = verify(ctx, t); err != nil {
			return t, nil, fmt.Errorf("auth: verify: %w", err)
		}
	}

	return t, verifiedToken, nil
}

func (s *Auth[T]) verify(ctx stdContext.Context, token []byte) (T, *VerifiedToken, error) {
	var t T

	if len(token) == 0 { // should never happen at this state.
		return t, nil, jwt.ErrMissing
	}

	validators := make([]jwt.TokenValidator, 0, len(s.Validators)+2)
	for _, v := range s.Validators {
		if v == nil {
			continue
		}
		validators = append(validators, v)
	}
	// add default validators, afterwards.
	validators = append(validators,
		jwt.Future(time.Minute), jwt.Leeway(time.Minute))

	verifiedToken, err := jwt.VerifyWithHeaderValidator(nil, nil, token, s.keys.ValidateHeader, validators...)
	if err != nil {
		return t, nil, err
	}

	if s.transformer != nil {
		if t, err = s.transformer.Transform(ctx, verifiedToken); err != nil {
			return t, nil, err
		}
	} else {
		if err = verifiedToken.Claims(&t); err != nil {
			return t, nil, err
		}
	}

	standardClaims := verifiedToken.StandardClaims

	if n := len(s.providers); n > 0 {
		for i := range n {
			p := s.providers[i]

			err := p.ValidateToken(ctx, standardClaims, t)
			if err != nil {
				if i == n-1 { // last provider errored.
					return t, nil, err
				}
				// keep searching.
				continue
			}

			// token is allowed.
			break
		}
	}
	// With no providers registered the token stands on its signature and its standard
	// claims alone. An earlier draft rejected that case instead:
	// return t, StandardClaims{}, fmt.Errorf("no provider")

	return t, verifiedToken, nil
}

// VerifyHandler verifies and sets the necessary information about the user(claims) and
// the verified token to the Iris Context and calls the Context's Next method.
// This information is available through auth.GetToken and
// auth.GetUser[T] package-level functions.
//
// See `Verify` method for more.
func (s *Auth[T]) VerifyHandler(verifyFuncs ...VerifyUserFunc[T]) context.Handler {
	return func(ctx *context.Context) {
		accessToken := s.ExtractAccessToken(ctx)

		if accessToken == "" { // if empty, fire 401.
			s.ErrorHandler.Unauthenticated(ctx, jwt.ErrMissing)
			return
		}

		t, verifiedToken, err := s.Verify(ctx, []byte(accessToken), verifyFuncs...)
		if err != nil {
			s.ErrorHandler.Unauthenticated(ctx, err)
			return
		}

		ctx.SetUser(t)

		// store the auth instance abd user to the request.
		ctx.Values().Set(authCntextKey, s)
		ctx.Values().Set(authTokenContextKey, verifiedToken)
		ctx.Values().Set(userContextKey, t)

		ctx.Next()
	}
}

// ExtractAccessToken extracts the access token from the request's header or cookie.
func (s *Auth[T]) ExtractAccessToken(ctx *context.Context) string {
	// first try from authorization: bearer header.
	accessToken := s.extractTokenFromHeader(ctx)

	// then if no header, try try extract from cookie.
	if accessToken == "" {
		if cookieName := s.config.Cookie.Name; cookieName != "" {
			accessToken = ctx.CookieValue(cookieName, context.CookieEncoding(s.securecookie))
		}
	}

	return accessToken
}

// Refresh accepts a previously generated refresh token (from SigninHandler) and
// returns a new access and refresh token pair.
func (s *Auth[T]) Refresh(ctx stdContext.Context, refreshToken []byte) (*oauth2.Token, error) {
	if !s.refreshEnabled {
		return nil, fmt.Errorf("auth: refresh: disabled")
	}

	t, _, err := s.verify(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("auth: refresh: %w", err)
	}

	// Refresh data from storage is optional.
	if newT, err := s.RefreshProvider(ctx, t); err != nil && !errors.Is(err, ErrNoProvider) {
		return nil, err
	} else {
		t = newT
	}

	// refresh the tokens, both refresh & access tokens will be renew to prevent
	// malicious 😈 users that may hold a refresh token,
	// also this ensures the expiration is shifting on active users.
	tok, err := s.SignToken(t)
	if err != nil {
		return nil, fmt.Errorf("auth: refresh: %w", err)
	}

	return tok, nil
}

// RefreshProvider calls the registered providers to refresh the user information
// based on the provided custom T value. It returns the updated T value or an error
// if no provider was registered or if all providers errored.
func (s *Auth[T]) RefreshProvider(ctx stdContext.Context, t T) (T, error) {
	if n := len(s.providers); n > 0 {
		for i := range n {
			p := s.providers[i]

			v, err := p.Refresh(ctx, t)
			if err != nil {
				if i == n-1 { // last provider errored.
					return t, fmt.Errorf("auth: refresh: %w", err)
				}
				// keep searching.
				continue
			}

			// found.
			t = v
			break
		}
	} else {
		return t, ErrNoProvider
	}

	return t, nil
}

// RefreshHandler reads the request body which should include data for `RefreshRequest` structure
// and sends a new access and refresh token pair,
// also sets the cookie to the new encrypted access token value.
// See `Refresh` method for more.
func (s *Auth[T]) RefreshHandler(ctx *context.Context) {
	var req RefreshRequest
	err := ctx.ReadJSON(&req)
	if err != nil {
		s.ErrorHandler.InvalidArgument(ctx, err)
		return
	}

	tok, err := s.Refresh(ctx, []byte(req.RefreshToken))
	if err != nil {
		// s.tryRemoveCookie(ctx)
		s.ErrorHandler.Unauthenticated(ctx, err)
		return
	}

	s.trySetCookie(ctx, tok.AccessToken)

	ctx.JSON(tok)
}

// Signout accepts the access token and a boolean which reports whether
// the signout should be applied to all tokens generated for a specific user (logout from all devices)
// or just the provided token's one.
// It calls the Provider's InvalidateToken(all=false) or InvalidateTokens (all=true).
func (s *Auth[T]) Signout(ctx stdContext.Context, token []byte, all bool) error {
	t, verifiedToken, err := s.verify(ctx, token)
	if err != nil {
		return fmt.Errorf("auth: signout: verify: %w", err)
	}

	for i, n := 0, len(s.providers)-1; i <= n; i++ {
		p := s.providers[i]

		if all {
			err = p.InvalidateTokens(ctx, t)
		} else {
			err = p.InvalidateToken(ctx, verifiedToken.StandardClaims, t)
		}

		if err != nil {
			if i == n { // last provider errored.
				return err
			}
			// keep trying.
			continue
		}

		// token is marked as invalidated by a provider.
		break
	}

	return nil
}

// SignoutHandler verifies the request's access token and invalidates it, calling
// the Provider's InvalidateToken method.
// See `Signout` method too.
func (s *Auth[T]) SignoutHandler(ctx *context.Context) {
	s.signoutHandler(ctx, false)
}

// SignoutAllHandler verifies the request's access token and
// should invalidate all the tokens generated previously calling
// the Provider's InvalidateTokens method.
// See `Signout` method too.
func (s *Auth[T]) SignoutAllHandler(ctx *context.Context) {
	s.signoutHandler(ctx, true)
}

func (s *Auth[T]) signoutHandler(ctx *context.Context, all bool) {
	accessToken := s.ExtractAccessToken(ctx)
	if accessToken == "" {
		s.ErrorHandler.Unauthenticated(ctx, jwt.ErrMissing)
		return
	}

	err := s.Signout(ctx, []byte(accessToken), all)
	if err != nil {
		s.ErrorHandler.Unauthenticated(ctx, err)
		return
	}

	s.tryRemoveCookie(ctx)

	ctx.SetUser(nil)

	ctx.Values().Remove(authTokenContextKey)
	ctx.Values().Remove(userContextKey)
}

func (s *Auth[T]) extractTokenFromHeader(ctx *context.Context) string {
	for _, headerKey := range s.config.Headers {
		headerValue := ctx.Header(headerKey)
		if headerValue == "" {
			continue
		}

		// If HeadersNoBearer is true, accept both with or without "Bearer " prefix.
		if s.config.HeadersNoBearer {
			// Remove "bearer " prefix if present (case-insensitive), but don't require it.
			lower := strings.ToLower(headerValue)
			if len(lower) > 8 && strings.HasPrefix(lower, "bearer ") {
				return strings.TrimSpace(headerValue[7:])
			}
			return headerValue
		}

		// Else, pure check: authorization header format must be Bearer {token}
		authHeaderParts := strings.SplitN(headerValue, " ", 2)
		if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
			continue
		}

		return authHeaderParts[1]
	}

	return ""
}

// cookiePath is the Path of the access token cookie. trySetCookie and
// tryRemoveCookie must agree on it, and on the Domain (ctx.Domain()): a
// browser keys a cookie on its name, domain and path.
const cookiePath = "/"

func (s *Auth[T]) trySetCookie(ctx *context.Context, accessToken string) {
	if cookieName := s.config.Cookie.Name; cookieName != "" {
		maxAge := s.keys[KIDAccess].MaxAge
		if maxAge == 0 {
			maxAge = context.DefaultCookieExpiration
		}

		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    url.QueryEscape(accessToken),
			Path:     cookiePath,
			HttpOnly: true,
			Secure:   s.config.Cookie.Secure || ctx.IsSSL(),
			Domain:   ctx.Domain(),
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(maxAge),
			MaxAge:   int(maxAge.Seconds()),
		}

		ctx.SetCookie(cookie, context.CookieEncoding(s.securecookie), context.CookieAllowReclaim())
	}
}

func (s *Auth[T]) tryRemoveCookie(ctx *context.Context) {
	if cookieName := s.config.Cookie.Name; cookieName != "" {
		// The Path and the Domain trySetCookie gave it. A removal without the
		// domain names another cookie, and the browser keeps the one that
		// holds the token.
		ctx.RemoveCookie(cookieName, context.CookiePath(cookiePath), context.CookieDomain(ctx.Domain()))
	}
}

// JWKSHandler returns a handler that can be used to expose the JSON Web Key Set (JWKS) of the Auth instance.
// The JWKS is a JSON object that represents a set of JWKs.
//
// See https://tools.ietf.org/html/rfc7517#section-5 for more.
//
// Example:
//
//	app.Get("/.well-known/jwks.json", AuthInstance.JWKSHandler())
func (s *Auth[T]) JWKSHandler() context.Handler {
	sets, err := s.keys.JWKS()
	if err != nil {
		panic(err)
	}

	contents, err := json.Marshal(sets)
	if err != nil {
		panic(err)
	}

	return func(ctx *context.Context) {
		ctx.SetContentType(context.ContentJSONHeaderValue)
		ctx.Write(contents)
	}
}

// Clone returns a new Auth instance sharing this one's configuration, keys
// and cookie transcoder but typed for a different User claims type. Useful
// when one keypair authenticates more than one account shape.
//
// It could not exist before Go 1.27: the New claims type is independent of
// the receiver's type parameter, which only a generic method can express.
func (a *Auth[Original]) Clone[New User]() *Auth[New] {
	return &Auth[New]{
		config:         a.config,
		keys:           a.keys,
		securecookie:   a.securecookie,
		ErrorHandler:   a.ErrorHandler,
		claimsProvider: a.claimsProvider,
		refreshEnabled: a.refreshEnabled,
	}
}

// VerifyExtra verifies a token like Verify and additionally decodes and
// optionally validates a second, independent claims value of type New.
func (a *Auth[Original]) VerifyExtra[New User](ctx stdContext.Context, token []byte, verifyFuncs ...VerifyUserFunc[New]) (Original, New, *VerifiedToken, error) {
	var n New
	t, verifiedToken, err := a.Verify(ctx, token)
	if err != nil {
		return t, n, nil, fmt.Errorf("auth: verify: %w", err)
	}

	if err := verifiedToken.Claims(&n); err != nil {
		return t, n, nil, fmt.Errorf("auth: verify: claims: %w", err)
	}

	for _, verify := range verifyFuncs {
		if verify == nil {
			continue
		}

		if err = verify(ctx, n); err != nil {
			return t, n, nil, fmt.Errorf("auth: verify: %w", err)
		}
	}

	return t, n, verifiedToken, nil
}
