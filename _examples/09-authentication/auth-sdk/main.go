// The auth SDK: auth.Auth[T] with a provider, ready-made token handlers and
// per-router verification.
//
// You supply two things: a Configuration (the JWT keys) and a Provider[T] over
// your user store. In return the SDK gives you the whole token lifecycle
// (SigninHandler, RefreshHandler, SignoutHandler, SignoutAllHandler) plus the
// VerifyHandler middleware, EdDSA signatures and encrypted refresh tokens.
//
// This example is self-contained: Configuration.BindRandom generates an
// ephemeral EdDSA key pair for access and refresh tokens, so every restart
// invalidates the tokens it issued. Production loads persistent keys from a
// file instead, with auth.MustLoad[Customer]("auth.yml"), and the loader prints a
// ready-to-edit example configuration when the file is missing.
//
// SigninHandler and RefreshHandler switch on the request's media type, so a
// JSON body needs the Content-Type header; otherwise send a form instead.
// tom@example.com (password tom_password) signs in but is inactive, so the
// requireActive check below answers 401 "account disabled" on /me.
//
// Learn: give a service sign-in, refresh, sign-out and JWKS endpoints by
// implementing one Provider[T] over your own user store.
// Before: ../sessions
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST -H "Content-Type: application/json" \
//	  http://localhost:8080/signin -d '{"username":"maria@example.com","password":"maria_password"}'   # 200, access_token and refresh_token
//	TOKEN=<access_token from the response above>
//	curl -i http://localhost:8080/me                                    # 401, no token
//	curl -i -H "Authorization: Bearer $TOKEN" http://localhost:8080/me  # 200, the Customer
//	curl -i -X POST -H "Content-Type: application/json" \
//	  http://localhost:8080/refresh -d '{"refresh_token":"<refresh_token>"}'   # 200, a fresh pair; both tokens rotate
//	curl -i -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/me/signout   # 200, empty body
//	curl -i -H "Authorization: Bearer $TOKEN" http://localhost:8080/me  # 401, the token is revoked
//	curl http://localhost:8080/.well-known/jwks.json                    # 200, the public key, for other services
//
// Next: ../../10-configuration/configuration
package main

import (
	stdContext "context"
	"crypto/subtle"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/auth"
)

func main() {
	// Development and tests only: random, non-persistent keys. BindRandom also
	// fills the Headers (Authorization, X-Authorization) and a cookie
	// configuration, so browser clients get the access token in an encrypted
	// HttpOnly cookie too and VerifyHandler falls back to it when no
	// Authorization header is present.
	var config auth.Configuration
	if err := config.BindRandom(); err != nil {
		log.Fatal(err)
	}
	// BindRandom leaves Cookie.Secure false so the cookie also works over plain
	// HTTP on localhost. Set it to true once you serve over HTTPS (the SDK marks
	// the cookie Secure automatically for requests that already arrive over TLS).
	// config.Cookie.Secure = true

	sso := auth.Must[Customer](config)
	sso.AddProvider(newCustomerProvider())

	app := iris.New()

	// Public: exchange credentials for an access/refresh token pair. The body is
	// JSON or a form with {"username" (or "email"), "password"}.
	app.Post("/signin", sso.SigninHandler)
	// Public: exchange {"refresh_token": "..."} for a fresh pair. Both rotate.
	app.Post("/refresh", sso.RefreshHandler)

	// Protected. VerifyHandler verifies the signature, runs the provider's
	// ValidateToken, applies the optional claims checks below and stores the
	// Customer on the request. Any failure answers 401 through the instance's
	// ErrorHandler (replaceable with SetErrorHandler).
	me := app.Router("/me", sso.VerifyHandler(requireActive))
	me.Get("/", profile)
	me.Post("/signout", sso.SignoutHandler)        // this token only.
	me.Post("/signout-all", sso.SignoutAllHandler) // every token of this customer.

	// Optional: publishes the public key so other services can verify the
	// tokens this application issues without holding the signing key.
	app.Get("/.well-known/jwks.json", sso.JWKSHandler())

	app.Listen(":8080")
}

// Customer is the T of auth.Auth[T]: it becomes the JWT payload, so keep it
// small and free of secrets: a signed token is readable by anyone holding it.
type Customer struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	Active bool     `json:"active"`
}

// customerProvider implements auth.Provider[Customer] over an in-memory store.
// A real one queries a database; see book chapter 10 for the SQL version.
type customerProvider struct {
	mu sync.RWMutex

	customers map[string]Customer // by email.
	passwords map[string]string   // by email.

	revokedTokens map[string]bool      // token (or origin token) id -> revoked.
	revokedBefore map[string]time.Time // customer id -> "signed out everywhere" moment.
}

func newCustomerProvider() *customerProvider {
	return &customerProvider{
		customers: map[string]Customer{
			"maria@example.com": {ID: "1", Email: "maria@example.com", Roles: []string{"customer"}, Active: true},
			"tom@example.com":   {ID: "2", Email: "tom@example.com", Roles: []string{"customer"}, Active: false},
		},
		passwords: map[string]string{
			"maria@example.com": "maria_password",
			"tom@example.com":   "tom_password",
		},
		revokedTokens: make(map[string]bool),
		revokedBefore: make(map[string]time.Time),
	}
}

// Signin runs on SigninHandler. Return one error text for both "no such user"
// and "wrong password" so the response cannot be used to enumerate accounts.
func (p *customerProvider) Signin(ctx stdContext.Context, username, password string) (Customer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	customer, ok := p.customers[username]
	// A real store keeps a bcrypt hash and calls
	// bcrypt.CompareHashAndPassword; ConstantTimeCompare stands in for it here
	// and keeps the comparison itself timing-safe.
	if !ok || subtle.ConstantTimeCompare([]byte(p.passwords[username]), []byte(password)) != 1 {
		return Customer{}, fmt.Errorf("invalid credentials")
	}

	return customer, nil
}

// Refresh reloads the customer before a new token pair is signed, so profile
// and role changes propagate on the next refresh.
func (p *customerProvider) Refresh(ctx stdContext.Context, c Customer) (Customer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	customer, ok := p.customers[c.Email]
	if !ok {
		return Customer{}, fmt.Errorf("customer not found")
	}

	return customer, nil
}

// ValidateToken runs on every verified request. Returning nil here (a fully
// stateless deployment) means tokens stay valid until they expire.
func (p *customerProvider) ValidateToken(ctx stdContext.Context, claims auth.StandardClaims, c Customer) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// OriginID is the access token a refresh token was issued from, so
	// revoking the access token kills its refresh token as well.
	if p.revokedTokens[claims.ID] || (claims.OriginID != "" && p.revokedTokens[claims.OriginID]) {
		return fmt.Errorf("token revoked")
	}

	if at, ok := p.revokedBefore[c.ID]; ok && claims.IssuedAt <= at.Unix() {
		return fmt.Errorf("token revoked")
	}

	return nil
}

// InvalidateToken runs on SignoutHandler.
func (p *customerProvider) InvalidateToken(ctx stdContext.Context, claims auth.StandardClaims, c Customer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.revokedTokens[claims.ID] = true

	return nil
}

// InvalidateTokens runs on SignoutAllHandler.
func (p *customerProvider) InvalidateTokens(ctx stdContext.Context, c Customer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.revokedBefore[c.ID] = time.Now()

	return nil
}

// requireActive is an auth.VerifyUserFunc: a cheap, declarative check over the
// already-verified user, applied to the whole router it is registered on.
func requireActive(ctx stdContext.Context, c Customer) error {
	if !c.Active {
		return fmt.Errorf("account disabled")
	}

	return nil
}

func profile(ctx iris.Context) {
	// The verified user, decoded from the token payload.
	customer := auth.GetUser[Customer](ctx)

	// The verified token itself: raw bytes plus the standard claims.
	// auth.UserAuthorized(ctx) reports whether a user is stored at all.
	token, _ := auth.GetToken(ctx)

	ctx.OK(iris.Map{
		"customer":   customer,
		"token_id":   token.StandardClaims.ID,
		"expires_at": token.StandardClaims.ExpiresAt().Format(time.RFC3339),
	}, nil)
}
