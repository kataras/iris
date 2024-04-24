//go:build go1.18
// +build go1.18

package auth

import (
	stdContext "context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/kataras/jwt"
)

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
	// Example can be found at: https://github.com/kataras/iris/tree/main/_examples/auth/auth/main.go.
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
		errorHandler ErrorHandler
		// Not nil if a transformer is registered.
		transformer Transformer[T]
		// Not nil if a custom claims provider is registered.
		claimsProvider ClaimsProvider
		// True if KIDRefresh on config.Keys.
		refreshEnabled bool
	}

	// VerifyUserFunc is passed on Verify and VerifyHandler method
	// to, optionally, further validate a T user value.
	VerifyUserFunc[T User] func(t T) error

	// SigninRequest is the request body the server expects
	// on SignHandler. The Password and Username or Email should be filled.
	SigninRequest struct {
		Username string `json:"username" form:"username,omitempty"` // username OR email, username has priority over email.
		Email    string `json:"email" form:"email,omitempty"`       // email OR username.
		Password string `json:"password" form:"password"`
	}

	// SigninResponse is the response body the server sends
	// to the client on the SignHandler. It contains a pair of the access token
	// and the refresh token if the refresh jwt token id exists in the configuration.
	SigninResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
	}

	// RefreshRequest is the request body the server expects
	// on VerifyHandler to renew an access and refresh token pair.
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

	return Must(New[T](config))
}

// Must is a helper that wraps a call to a function returning (*Auth[T], error)
// and panics if the error is non-nil. It is intended for use in variable
// initializations such as
//
//	var s = auth.Must(auth.New[MyUser](config))
func Must[T User](s *Auth[T], err error) *Auth[T] {
	if err != nil {
		panic(err)
	}

	return s
}

// New initializes a new Auth instance typeof T and returns it.
// The T generic can be any custom struct.
// It accepts a Configuration value which can be constructed
// manually or through a configuration file using the
// MustGenerateConfiguration or MustLoadConfiguration
// or LoadConfiguration or MustLoad package-level functions.
//
// Example can be found at: https://github.com/kataras/iris/tree/main/_examples/auth/auth/main.go.
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
		errorHandler: new(DefaultErrorHandler),
	}

	return s, nil
}

// WithProviderAndErrorHandler registers a provider (if not nil) and
// an error handler (if not nil) and returns this "s" Auth instance.
// It's the same as calling AddProvider and SetErrorHandler at once.
// It's really useful when registering an Auth instance using the iris.Party.PartyConfigure
// method when a Provider of T and ErrorHandler is available through the registered Party's dependencies.
//
// Usage Example:
//
//	api := app.Party("/api")
//	api.EnsureStaticBindings().RegisterDependency(
//	  NewAuthProviderErrorHandler(),
//	  NewAuthCustomerProvider,
//	  auth.Must(auth.New[Customer](authConfig)).WithProviderAndErrorHandler,
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
	s.errorHandler = errHandler
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

// Signin signs a token based on the provided username and password
// and returns a pair of access and refresh tokens.
//
// Signin calls the Provider.Signin method to check if a user
// is authenticated by the given username and password combination.
func (s *Auth[T]) Signin(ctx stdContext.Context, username, password string) ([]byte, []byte, error) {
	var t T

	// get "t" from a valid provider.
	if n := len(s.providers); n > 0 {
		for i := 0; i < n; i++ {
			p := s.providers[i]

			v, err := p.Signin(ctx, username, password)
			if err != nil {
				if i == n-1 { // last provider errored.
					return nil, nil, fmt.Errorf("auth: signin: %w", err)
				}
				// keep searching.
				continue
			}

			// found.
			t = v
			break
		}
	} else {
		return nil, nil, fmt.Errorf("auth: signin: no provider")
	}

	// sign the tokens.
	accessToken, refreshToken, err := s.sign(t)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: signin: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *Auth[T]) sign(t T) ([]byte, []byte, error) {
	// sign the tokens.
	var (
		accessStdClaims  StandardClaims
		refreshStdClaims StandardClaims
	)

	if s.claimsProvider != nil {
		accessStdClaims = s.claimsProvider.GetAccessTokenClaims()
		refreshStdClaims = s.claimsProvider.GetRefreshTokenClaims(accessStdClaims)
	}

	iat := jwt.Clock().Unix()

	if accessStdClaims.IssuedAt == 0 {
		accessStdClaims.IssuedAt = iat
	}

	if accessStdClaims.ID == "" {
		accessStdClaims.ID = uuid.NewString()
	}

	if refreshStdClaims.IssuedAt == 0 {
		refreshStdClaims.IssuedAt = iat
	}

	if refreshStdClaims.ID == "" {
		refreshStdClaims.ID = uuid.NewString()
	}

	if refreshStdClaims.OriginID == "" {
		// keep a reference of the access token the refresh token is created,
		// if that access token is invalidated then
		// its refresh token should be too so the user can force-login.
		refreshStdClaims.OriginID = accessStdClaims.ID
	}

	accessToken, err := s.keys.SignToken(KIDAccess, t, accessStdClaims)
	if err != nil {
		return nil, nil, fmt.Errorf("access: %w", err)
	}

	var refreshToken []byte
	if s.refreshEnabled {
		refreshToken, err = s.keys.SignToken(KIDRefresh, t, refreshStdClaims)
		if err != nil {
			return nil, nil, fmt.Errorf("refresh: %w", err)
		}
	}

	return accessToken, refreshToken, nil
}

// SignHandler generates and sends a pair of access and refresh token to the client
// as JSON body of `SigninResponse` and cookie (if cookie setting was provided).
// See `Signin` method for more.
func (s *Auth[T]) SigninHandler(ctx *context.Context) {
	// No, let the developer decide it based on a middleware, e.g. iris.LimitRequestBodySize.
	// ctx.SetMaxRequestBodySize(s.maxRequestBodySize)

	var (
		req SigninRequest
		err error
	)

	switch ctx.GetContentTypeRequested() {
	case context.ContentFormHeaderValue, context.ContentFormMultipartHeaderValue:
		err = ctx.ReadForm(&req)
	default:
		err = ctx.ReadJSON(&req)
	}

	if err != nil {
		s.errorHandler.InvalidArgument(ctx, err)
		return
	}

	if req.Username == "" {
		req.Username = req.Email
	}

	accessTokenBytes, refreshTokenBytes, err := s.Signin(ctx, req.Username, req.Password)
	if err != nil {
		s.tryRemoveCookie(ctx) // remove cookie on invalidated.

		s.errorHandler.Unauthenticated(ctx, err)
		return
	}
	accessToken := jwt.BytesToString(accessTokenBytes)
	refreshToken := jwt.BytesToString(refreshTokenBytes)

	s.trySetCookie(ctx, accessToken)

	resp := SigninResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ctx.JSON(resp)
}

// Verify accepts a token and verifies it.
// It returns the token's custom and standard JWT claims.
func (s *Auth[T]) Verify(ctx stdContext.Context, token []byte, verifyFuncs ...VerifyUserFunc[T]) (T, StandardClaims, error) {
	t, claims, err := s.verify(ctx, token)
	if err != nil {
		return t, StandardClaims{}, fmt.Errorf("auth: verify: %w", err)
	}

	for _, verify := range verifyFuncs {
		if verify == nil {
			continue
		}

		if err = verify(t); err != nil {
			return t, StandardClaims{}, fmt.Errorf("auth: verify: %w", err)
		}
	}

	return t, claims, nil
}

func (s *Auth[T]) verify(ctx stdContext.Context, token []byte) (T, StandardClaims, error) {
	var t T

	if len(token) == 0 { // should never happen at this state.
		return t, StandardClaims{}, jwt.ErrMissing
	}

	verifiedToken, err := jwt.VerifyWithHeaderValidator(nil, nil, token, s.keys.ValidateHeader, jwt.Future(time.Minute), jwt.Leeway(time.Minute))
	if err != nil {
		return t, StandardClaims{}, err
	}

	if s.transformer != nil {
		if t, err = s.transformer.Transform(ctx, verifiedToken); err != nil {
			return t, StandardClaims{}, err
		}
	} else {
		if err = verifiedToken.Claims(&t); err != nil {
			return t, StandardClaims{}, err
		}
	}

	standardClaims := verifiedToken.StandardClaims

	if n := len(s.providers); n > 0 {
		for i := 0; i < n; i++ {
			p := s.providers[i]

			err := p.ValidateToken(ctx, standardClaims, t)
			if err != nil {
				if i == n-1 { // last provider errored.
					return t, StandardClaims{}, err
				}
				// keep searching.
				continue
			}

			// token is allowed.
			break
		}
	} else {
		// return t, StandardClaims{}, fmt.Errorf("no provider")
	}

	return t, standardClaims, nil
}

// VerifyHandler verifies and sets the necessary information about the user(claims) and
// the verified token to the Iris Context and calls the Context's Next method.
// This information is available through auth.GetAccessToken, auth.GetStandardClaims and
// auth.GetUser[T] package-level functions.
//
// See `Verify` method for more.
func (s *Auth[T]) VerifyHandler(verifyFuncs ...VerifyUserFunc[T]) context.Handler {
	return func(ctx *context.Context) {
		accessToken := s.ExtractAccessToken(ctx)

		if accessToken == "" { // if empty, fire 401.
			s.errorHandler.Unauthenticated(ctx, jwt.ErrMissing)
			return
		}

		t, claims, err := s.Verify(ctx, []byte(accessToken), verifyFuncs...)
		if err != nil {
			s.errorHandler.Unauthenticated(ctx, err)
			return
		}

		ctx.SetUser(t)

		// store the user to the request.
		ctx.Values().Set(accessTokenContextKey, accessToken)
		ctx.Values().Set(standardClaimsContextKey, claims)
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
			accessToken = ctx.GetCookie(cookieName, context.CookieEncoding(s.securecookie))
		}
	}

	return accessToken
}

// Refresh accepts a previously generated refresh token (from SigninHandler) and
// returns a new access and refresh token pair.
func (s *Auth[T]) Refresh(ctx stdContext.Context, refreshToken []byte) ([]byte, []byte, error) {
	if !s.refreshEnabled {
		return nil, nil, fmt.Errorf("auth: refresh: disabled")
	}

	t, _, err := s.verify(ctx, refreshToken)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: refresh: %w", err)
	}

	// refresh the tokens, both refresh & access tokens will be renew to prevent
	// malicious 😈 users that may hold a refresh token.
	accessTok, refreshTok, err := s.sign(t)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: refresh: %w", err)
	}

	return accessTok, refreshTok, nil
}

// RefreshHandler reads the request body which should include data for `RefreshRequest` structure
// and sends a new access and refresh token pair,
// also sets the cookie to the new encrypted access token value.
// See `Refresh` method for more.
func (s *Auth[T]) RefreshHandler(ctx *context.Context) {
	var req RefreshRequest
	err := ctx.ReadJSON(&req)
	if err != nil {
		s.errorHandler.InvalidArgument(ctx, err)
		return
	}

	accessTokenBytes, refreshTokenBytes, err := s.Refresh(ctx, []byte(req.RefreshToken))
	if err != nil {
		// s.tryRemoveCookie(ctx)
		s.errorHandler.Unauthenticated(ctx, err)
		return
	}

	accessToken := jwt.BytesToString(accessTokenBytes)
	refreshToken := jwt.BytesToString(refreshTokenBytes)

	s.trySetCookie(ctx, accessToken)

	resp := SigninResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ctx.JSON(resp)
}

// Signout accepts the access token and a boolean which reports whether
// the signout should be applied to all tokens generated for a specific user (logout from all devices)
// or just the provided token's one.
// It calls the Provider's InvalidateToken(all=false) or InvalidateTokens (all=true).
func (s *Auth[T]) Signout(ctx stdContext.Context, token []byte, all bool) error {
	t, standardClaims, err := s.verify(ctx, token)
	if err != nil {
		return fmt.Errorf("auth: signout: verify: %w", err)
	}

	for i, n := 0, len(s.providers)-1; i <= n; i++ {
		p := s.providers[i]

		if all {
			err = p.InvalidateTokens(ctx, t)
		} else {
			err = p.InvalidateToken(ctx, standardClaims, t)
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
		s.errorHandler.Unauthenticated(ctx, jwt.ErrMissing)
		return
	}

	err := s.Signout(ctx, []byte(accessToken), all)
	if err != nil {
		s.errorHandler.Unauthenticated(ctx, err)
		return
	}

	s.tryRemoveCookie(ctx)

	ctx.SetUser(nil)

	ctx.Values().Remove(accessTokenContextKey)
	ctx.Values().Remove(standardClaimsContextKey)
	ctx.Values().Remove(userContextKey)
}

func (s *Auth[T]) extractTokenFromHeader(ctx *context.Context) string {
	for _, headerKey := range s.config.Headers {
		headerValue := ctx.GetHeader(headerKey)
		if headerValue == "" {
			continue
		}

		// pure check: authorization header format must be Bearer {token}
		authHeaderParts := strings.Split(headerValue, " ")
		if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
			continue
		}

		return authHeaderParts[1]
	}

	return ""
}

func (s *Auth[T]) trySetCookie(ctx *context.Context, accessToken string) {
	if cookieName := s.config.Cookie.Name; cookieName != "" {
		maxAge := s.keys[KIDAccess].MaxAge
		if maxAge == 0 {
			maxAge = context.SetCookieKVExpiration
		}

		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    url.QueryEscape(accessToken),
			Path:     "/",
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
		ctx.RemoveCookie(cookieName)
	}
}
