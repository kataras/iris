package jwt_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/jwt"

	kjwt "github.com/kataras/jwt"
)

const testKid = "NjVBRjY5MDlCMUIwNzU4RTA2QzZFMDQ4QzQ2MDAyQjVDNjk1RTM2Qg"

// kidHeader is the token header an identity provider such as Auth0 sends:
// the algorithm, the type and the id of the key that signed the token.
type kidHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid,omitempty"`
}

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	return key
}

// signWithKid signs fooClaims{Foo: "bar"} under a header that names the given kid.
func signWithKid(t *testing.T, alg jwt.Alg, key any, kid string, encrypt kjwt.InjectFunc) string {
	t.Helper()

	header := kidHeader{Alg: alg.Name(), Typ: "JWT", Kid: kid}
	token, err := kjwt.SignEncryptedWithHeader(alg, key, encrypt, fooClaims{Foo: "bar"}, header, jwt.MaxAge(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	return string(token)
}

// serveToken sends the token to a route protected by the verifier. It returns
// the response status, the response body and the error the verifier reported.
func serveToken(t *testing.T, verifier *jwt.Verifier, token string) (int, string, error) {
	t.Helper()

	var verifyErr error
	verifier.ErrorHandler = func(ctx iris.Context, err error) {
		verifyErr = err
		ctx.StopWithStatus(iris.StatusUnauthorized)
	}

	app := iris.New()
	app.Get("/protected", verifier.Verify(func() any { return new(fooClaims) }), func(ctx iris.Context) {
		claims, ok := jwt.Get[*fooClaims](ctx)
		if !ok {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}
		ctx.WriteString(claims.Foo)
	})

	resp := httptest.New(t, app).GET("/protected").WithHeader("Authorization", "Bearer "+token).Expect()
	return resp.Raw().StatusCode, resp.Body().Raw(), verifyErr
}

// algHeader is a header validator that checks the algorithm and nothing else,
// so the header may carry more fields, a "kid" among them.
func algHeader(alg string, headerDecoded []byte) (kjwt.Alg, kjwt.PublicKey, kjwt.InjectFunc, error) {
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerDecoded, &header); err != nil {
		return nil, nil, nil, err
	}

	if header.Alg != alg {
		return nil, nil, nil, jwt.ErrTokenAlg
	}

	return nil, nil, nil, nil
}

// anyHeader is a header validator that accepts every header.
func anyHeader(string, []byte) (kjwt.Alg, kjwt.PublicKey, kjwt.InjectFunc, error) {
	return nil, nil, nil, nil
}

// Without a HeaderValidator the header check is byte exact, so a token whose
// header carries a "kid" is refused. This is the behavior before the field
// existed and it stays the default.
func TestVerifierKidHeaderWithoutHeaderValidator(t *testing.T) {
	key := newRSAKey(t)
	verifier := jwt.NewVerifier(jwt.RS256, &key.PublicKey)

	status, _, err := serveToken(t, verifier, signWithKid(t, jwt.RS256, key, testKid, nil))
	if status != iris.StatusUnauthorized {
		t.Fatalf("expected status %d but got %d", iris.StatusUnauthorized, status)
	}
	if !errors.Is(err, jwt.ErrTokenAlg) {
		t.Fatalf("expected jwt.ErrTokenAlg but got: %v", err)
	}

	// A token of the same key without a "kid" is accepted, as always.
	status, body, err := serveToken(t, verifier, signWithKid(t, jwt.RS256, key, "", nil))
	if status != iris.StatusOK || body != "bar" {
		t.Fatalf("token without kid: expected status %d and body bar but got %d, %q: %v", iris.StatusOK, status, body, err)
	}
}

// https://github.com/kataras/iris/issues/2192
func TestVerifierHeaderValidator(t *testing.T) {
	key, otherKey := newRSAKey(t), newRSAKey(t)

	verifier := jwt.NewVerifier(jwt.RS256, &key.PublicKey)
	verifier.HeaderValidator = algHeader

	status, body, err := serveToken(t, verifier, signWithKid(t, jwt.RS256, key, testKid, nil))
	if status != iris.StatusOK || body != "bar" {
		t.Fatalf("token with kid: expected status %d and body bar but got %d, %q: %v", iris.StatusOK, status, body, err)
	}

	rejected := []struct {
		name      string
		validator kjwt.HeaderValidator
		token     string
		err       error
	}{
		{"HS256", algHeader, signWithKid(t, jwt.HS256, testSecret, testKid, nil), jwt.ErrTokenAlg},
		{"RS384, same key", algHeader, signWithKid(t, jwt.RS384, key, testKid, nil), jwt.ErrTokenAlg},
		{"RS256, other key", algHeader, signWithKid(t, jwt.RS256, otherKey, testKid, nil), jwt.ErrTokenSignature},
		// The Alg of the Verifier still decides how the signature is checked,
		// even when the validator lets every header through.
		{"HS256, validator accepts any header", anyHeader, signWithKid(t, jwt.HS256, testSecret, testKid, nil), jwt.ErrTokenSignature},
		{"RS384, validator accepts any header", anyHeader, signWithKid(t, jwt.RS384, key, testKid, nil), jwt.ErrTokenSignature},
	}

	for _, tt := range rejected {
		verifier.HeaderValidator = tt.validator

		status, _, err := serveToken(t, verifier, tt.token)
		if status != iris.StatusUnauthorized {
			t.Fatalf("%s: expected status %d but got %d", tt.name, iris.StatusUnauthorized, status)
		}
		if !errors.Is(err, tt.err) {
			t.Fatalf("%s: expected %v but got: %v", tt.name, tt.err, err)
		}
	}
}

// The ValidateHeader method of a kataras/jwt key registry is a header validator
// that selects the algorithm and the key by the "kid" of each token.
func TestVerifierHeaderValidatorKeys(t *testing.T) {
	key, otherKey := newRSAKey(t), newRSAKey(t)

	keys := make(kjwt.Keys)
	keys.Register(kjwt.RS256, testKid, &key.PublicKey, nil)

	verifier := jwt.NewVerifier(nil, nil)
	verifier.HeaderValidator = keys.ValidateHeader

	status, body, err := serveToken(t, verifier, signWithKid(t, jwt.RS256, key, testKid, nil))
	if status != iris.StatusOK || body != "bar" {
		t.Fatalf("registered kid: expected status %d and body bar but got %d, %q: %v", iris.StatusOK, status, body, err)
	}

	rejected := []struct {
		name  string
		token string
		err   error
	}{
		{"unknown kid", signWithKid(t, jwt.RS256, key, "unknown", nil), kjwt.ErrUnknownKid},
		{"no kid", signWithKid(t, jwt.RS256, key, "", nil), kjwt.ErrEmptyKid},
		{"other key", signWithKid(t, jwt.RS256, otherKey, testKid, nil), jwt.ErrTokenSignature},
		{"other algorithm", signWithKid(t, jwt.HS256, testSecret, testKid, nil), jwt.ErrTokenAlg},
	}

	for _, tt := range rejected {
		status, _, err := serveToken(t, verifier, tt.token)
		if status != iris.StatusUnauthorized {
			t.Fatalf("%s: expected status %d but got %d", tt.name, iris.StatusUnauthorized, status)
		}
		if !errors.Is(err, tt.err) {
			t.Fatalf("%s: expected %v but got: %v", tt.name, tt.err, err)
		}
	}
}

// The header validator is used for tokens with an encrypted payload too.
func TestVerifierHeaderValidatorWithDecryption(t *testing.T) {
	key := newRSAKey(t)

	encryptionKey := jwt.MustGenerateRandom(32)
	encrypt, _, err := jwt.GCM(encryptionKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	token := signWithKid(t, jwt.RS256, key, testKid, encrypt)

	verifier := jwt.NewVerifier(jwt.RS256, &key.PublicKey).WithDecryption(encryptionKey, nil)

	status, _, err := serveToken(t, verifier, token)
	if status != iris.StatusUnauthorized || !errors.Is(err, jwt.ErrTokenAlg) {
		t.Fatalf("without a header validator: expected status %d and jwt.ErrTokenAlg but got %d: %v", iris.StatusUnauthorized, status, err)
	}

	verifier.HeaderValidator = algHeader

	status, body, err := serveToken(t, verifier, token)
	if status != iris.StatusOK || body != "bar" {
		t.Fatalf("with a header validator: expected status %d and body bar but got %d, %q: %v", iris.StatusOK, status, body, err)
	}
}
