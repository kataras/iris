package entity_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/kataras/iris/v14/auth/identity/entity"

	"github.com/kataras/jwt"
)

type flatClaims struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Firstname string `json:"firstname"`
}

func verified(t *testing.T, claims map[string]any) *jwt.VerifiedToken {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	tok, err := jwt.Sign(jwt.EdDSA, priv, claims)
	if err != nil {
		t.Fatal(err)
	}
	v, err := jwt.Verify(jwt.EdDSA, pub, tok)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// A token may carry the user flat at top level with no attrs claim.
func TestFlatTransformerWithoutAttrs(t *testing.T) {
	v := verified(t, map[string]any{"id": "u1", "username": "ann@example.com", "firstname": "Ann"})
	got, err := entity.FlatTransformer[flatClaims]().Transform(context.Background(), v)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "u1" || got.Username != "ann@example.com" || got.Firstname != "Ann" {
		t.Fatalf("got %+v", got)
	}
}

// Tokens that do carry attrs keep working: attrs overlay the top level.
func TestFlatTransformerWithAttrs(t *testing.T) {
	v := verified(t, map[string]any{"id": "u1", "username": "ann@example.com", "attrs": map[string]any{"firstname": "Ann"}})
	got, err := entity.FlatTransformer[flatClaims]().Transform(context.Background(), v)
	if err != nil {
		t.Fatal(err)
	}
	if got.Firstname != "Ann" {
		t.Fatalf("got %+v", got)
	}
}
