//go:build go1.18

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
	Auth[T User] struct {
		config Configuration

		keys         jwt.Keys
		securecookie context.SecureCookie

		providers      []Provider[T] // at least one.
		errorHandler   ErrorHandler
		transformer    Transformer[T]
		claimsProvider ClaimsProvider
		refreshEnabled bool // if KIDRefresh exists in keys.
	}

	TVerify[T User] func(t T) error

	SigninRequest struct {
		Username string `json:"username" form:"username,omitempty"` // username OR email, username has priority over email.
		Email    string `json:"email" form:"email,omitempty"`       // email OR username.
		Password string `json:"password" form:"password"`
	}

	SigninResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
	}

	RefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}
)

func MustLoad[T User](filename string) *Auth[T] {
	var config Configuration
	if err := config.BindFile(filename); err != nil {
		panic(err)
	}

	s, err := New[T](config)
	if err != nil {
		panic(err)
	}

	return s
}

func Must[T User](s *Auth[T], err error) *Auth[T] {
	if err != nil {
		panic(err)
	}

	return s
}

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

func (s *Auth[T]) SetErrorHandler(errHandler ErrorHandler) *Auth[T] {
	s.errorHandler = errHandler
	return s
}

func (s *Auth[T]) SetTransformer(transformer Transformer[T]) *Auth[T] {
	s.transformer = transformer
	return s
}

func (s *Auth[T]) SetTransformerFunc(transfermerFunc func(ctx stdContext.Context, tok *VerifiedToken) (T, error)) *Auth[T] {
	s.transformer = TransformerFunc[T](transfermerFunc)
	return s
}

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

func (s *Auth[T]) Verify(ctx stdContext.Context, token []byte) (T, StandardClaims, error) {
	t, claims, err := s.verify(ctx, token)
	if err != nil {
		return t, StandardClaims{}, fmt.Errorf("auth: verify: %w", err)
	}

	return t, claims, nil
}

func (s *Auth[T]) verify(ctx stdContext.Context, token []byte) (T, StandardClaims, error) {
	var t T

	if len(token) == 0 { // should never happen at this state.
		return t, StandardClaims{}, jwt.ErrMissing
	}

	verifiedToken, err := jwt.VerifyWithHeaderValidator(nil, nil, token, s.keys.ValidateHeader, jwt.Leeway(time.Minute))
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

func (s *Auth[T]) VerifyHandler(verifyFuncs ...TVerify[T]) context.Handler {
	return func(ctx *context.Context) {
		accessToken := s.extractAccessToken(ctx)

		if accessToken == "" { // if empty, fire 401.
			s.errorHandler.Unauthenticated(ctx, jwt.ErrMissing)
			return
		}

		t, claims, err := s.Verify(ctx, []byte(accessToken))
		if err != nil {
			s.errorHandler.Unauthenticated(ctx, err)
			return
		}

		for _, verify := range verifyFuncs {
			if verify == nil {
				continue
			}

			if err = verify(t); err != nil {
				err = fmt.Errorf("auth: verify: %v", err)
				s.errorHandler.Unauthenticated(ctx, err)
				return
			}
		}

		ctx.SetUser(t)

		// store the user to the request.
		ctx.Values().Set(accessTokenContextKey, accessToken)

		ctx.Values().Set(userContextKey, t)
		ctx.Values().Set(standardClaimsContextKey, claims)

		ctx.Next()
	}
}

func (s *Auth[T]) extractAccessToken(ctx *context.Context) string {
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

func (s *Auth[T]) SignoutHandler(ctx *context.Context) {
	s.signoutHandler(ctx, false)
}

func (s *Auth[T]) SignoutAllHandler(ctx *context.Context) {
	s.signoutHandler(ctx, true)
}

func (s *Auth[T]) signoutHandler(ctx *context.Context, all bool) {
	accessToken := s.extractAccessToken(ctx)
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
	ctx.Values().Remove(userContextKey)
	ctx.Values().Remove(standardClaimsContextKey)
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
			Path:     "/",
			Name:     cookieName,
			Value:    url.QueryEscape(accessToken),
			HttpOnly: true,
			Secure:   s.config.Cookie.Secure || ctx.IsSSL(),
			Domain:   ctx.Domain(),
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(maxAge),
			MaxAge:   int(maxAge.Seconds()),
		}

		ctx.SetCookie(cookie, context.CookieEncoding(s.securecookie))
	}
}

func (s *Auth[T]) tryRemoveCookie(ctx *context.Context) {
	if cookieName := s.config.Cookie.Name; cookieName != "" {
		ctx.RemoveCookie(cookieName)
	}
}
