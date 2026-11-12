// The jwt middleware: a Signer/Verifier round trip.
//
// middleware/jwt is the thin layer beneath the auth SDK: no providers, no user
// store. A Signer mints tokens, a Verifier turns "Authorization: Bearer <token>"
// into typed claims for the routes it guards. Reach for it when you only need to
// sign or verify, for example a resource server that trusts tokens issued
// elsewhere.
//
// v14 default: the verifier reads the Authorization header ONLY. The ?token=
// query extractor is opt-in (verifier.WithQueryToken()), because tokens in URLs
// leak into access logs, proxy logs, browser history and Referer headers.
//
// The signing secret comes from the JWT_SECRET environment variable; without it
// the example falls back to a development-only constant so it still runs.
//
// Learn: mint tokens with a Signer, guard a router with a Verifier that reads
// only the Authorization header, and revoke one token through the blocklist.
// Before: ../api-key
//
// Run:
//
//	JWT_SECRET=a-32-byte-or-longer-random-secret go run main.go
//
// Try:
//
//	TOKEN=$(curl -s -X POST -H "Content-Type: application/json" \
//	  http://localhost:8080/token -d '{"username":"makis"}' | sed 's/.*"token":"//;s/".*//')   # 200, token and expires_in
//	curl -i http://localhost:8080/api/me                                          # 401, no Authorization header
//	curl -i "http://localhost:8080/api/me?token=$TOKEN"                           # 401, query tokens are opt-in
//	curl -i -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/me        # 200, the claims
//	curl -i -H "Authorization: Bearer $TOKEN" -X POST http://localhost:8080/api/logout   # 200, token invalidated
//	curl -i -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/me        # 401, blocklisted now
//
// Next: ../sessions
package main

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/jwt"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("development-only-secret-set-JWT_SECRET-in-production")
	}

	// HS256 is symmetric: the same secret signs and verifies. Production
	// systems that verify tokens in other services should prefer an asymmetric
	// algorithm (jwt.EdDSA, RS256, ES256) so a verifier cannot mint tokens.
	signer := jwt.NewSigner(jwt.HS256, secret, 15*time.Minute)

	verifier := jwt.NewVerifier(jwt.HS256, secret)
	// An in-memory blocklist gives short-lived tokens server-side revocation;
	// see the /api/logout route below. It is per-process and resets on restart;
	// use middleware/jwt/blocklist/redis to share it across instances.
	verifier.WithDefaultBlocklist()

	app := iris.New()

	// The sign endpoint. A real one authenticates the caller first (password
	// check, API key, an upstream identity provider) and only then signs.
	app.Post("/token", func(ctx iris.Context) {
		req, ok := ctx.BindJSON[TokenRequest]()
		if !ok {
			return // 400 already written for a malformed body.
		}

		// The "jti" matters: the blocklist keys on it, and falls back to the
		// whole token string when it is empty. Signing is deterministic, so two
		// tokens minted for the same subject within the same second would be
		// byte-identical, and revoking one would revoke the other.
		token, err := signer.Sign(&UserClaims{
			Claims:   jwt.Claims{ID: newTokenID(), Subject: "user-123"},
			Username: req.Username,
			Roles:    []string{"member"},
		})
		if err != nil {
			// A signing failure is the server's: HandleError renders the
			// generic 500 body and keeps the cause in the log.
			errors.HandleError(ctx, err)
			return
		}

		// signer.MaxAge decided the token's "exp"; tell the client about it.
		ctx.OK(iris.Map{"token": string(token), "expires_in": int(signer.MaxAge.Seconds())}, nil)
	})

	// verifier.Verify extracts, verifies and decodes the token into a fresh
	// value of the claims type. Invalid, expired, blocklisted or missing tokens
	// never reach the routes: the verifier's ErrorHandler answers 401.
	api := app.Router("/api", verifier.Verify(func() any { return new(UserClaims) }))

	api.Get("/me", func(ctx iris.Context) {
		// jwt.Get[T] returns the value the claimsType function produced, a
		// *UserClaims here. ok is false only if the type does not match.
		claims, ok := jwt.Get[*UserClaims](ctx)
		if !ok {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		// The verified token also carries the standard claims and the raw bytes.
		verified := jwt.GetVerifiedToken(ctx)

		ctx.JSON(iris.Map{
			"username":   claims.Username,
			"roles":      claims.Roles,
			"subject":    verified.StandardClaims.Subject,
			"token_id":   verified.StandardClaims.ID,
			"expires_at": verified.StandardClaims.ExpiresAt().Format(time.RFC3339),
		})
	})

	api.Post("/logout", func(ctx iris.Context) {
		// Revocation goes through the blocklist the verifier was given above:
		// the token stays cryptographically valid, but every later request
		// carrying it is rejected until it expires on its own.
		verified := jwt.GetVerifiedToken(ctx)
		err := verifier.Blocklist.InvalidateToken(verified.Token, verified.StandardClaims)
		// ctx.OK carries the error branch: the 500 on failure, the body on
		// success. No status code and no error check at the call site.
		ctx.OK(iris.Map{"message": "token invalidated"}, err)
	})

	app.Listen(":8080")
}

// UserClaims is the token payload. Embedding jwt.Claims adds the standard
// registered claims (sub, exp, iat, jti...) next to the custom fields.
// A JWT is signed, not encrypted: anyone holding it can read every claim,
// so keep secrets out of this struct.
type UserClaims struct {
	jwt.Claims
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// TokenRequest is the /token payload. ctx.BindJSON needs a named type to
// instantiate, which is also where a Validate method would go.
type TokenRequest struct {
	Username string `json:"username"`
}

// newTokenID returns a unique "jti" claim. Any collision-free source works
// (a UUID library, a database sequence) as long as it is unique per token.
func newTokenID() string {
	b := make([]byte, 16)
	rand.Read(b) // crypto/rand.Read never returns an error.

	return hex.EncodeToString(b)
}
