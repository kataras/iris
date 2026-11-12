// Package jwt signs and verifies JSON Web Tokens.
//
// It has no single Handler entry point: NewSigner(alg, key, maxAge) issues
// tokens and NewVerifier(alg, key, validators...) validates them, and it is the
// Verifier's `Verify` method that returns the Handler. The two halves are
// separate objects because most applications sign in one place and verify in
// another.
package jwt

import "github.com/kataras/iris/v14/context"

func init() {
	context.SetHandlerName("iris/middleware/jwt.*", "iris.jwt")
}
