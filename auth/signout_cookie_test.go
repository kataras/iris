package auth_test

import (
	stdContext "context"
	"net/http"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/httptest"
)

type cookieUser struct {
	ID string `json:"id"`
}

// cookieProvider signs in anybody and never invalidates a token on the server
// side, so after a signout the only thing that ends the session is the
// removal of the cookie.
type cookieProvider struct{}

func (cookieProvider) Signin(_ stdContext.Context, username, _ string) (cookieUser, error) {
	return cookieUser{ID: username}, nil
}

func (cookieProvider) Refresh(_ stdContext.Context, u cookieUser) (cookieUser, error) {
	return u, nil
}

func (cookieProvider) ValidateToken(stdContext.Context, auth.StandardClaims, cookieUser) error {
	return nil
}

func (cookieProvider) InvalidateToken(stdContext.Context, auth.StandardClaims, cookieUser) error {
	return nil
}

func (cookieProvider) InvalidateTokens(stdContext.Context, cookieUser) error {
	return nil
}

// setCookieLine returns the Set-Cookie line of the response for the cookie
// with the given name.
func setCookieLine(t *testing.T, res *http.Response, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range res.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("expected a Set-Cookie line for %q, got %q", name, res.Header.Values("Set-Cookie"))
	return nil
}

// A browser keys a cookie on its name, its domain and its path. The signin
// handler sets the access token cookie with the request's root domain, so the
// signout handler has to remove it with the same domain and path: a removal
// that names another domain, or none, is a different cookie for the browser,
// which keeps the original and stays signed in.
func TestSignoutRemovesTheCookieSigninSet(t *testing.T) {
	config := auth.MustGenerateConfiguration()
	cookieName := config.Cookie.Name
	if cookieName == "" {
		t.Fatal("expected the generated configuration to name a cookie")
	}

	s := auth.Must[cookieUser](config).AddProvider(cookieProvider{})

	app := iris.New()
	app.Post("/signin", s.SigninHandler)
	app.Get("/signout", s.SignoutHandler)

	// A host under a root domain, the way an application is deployed: the
	// cookie's domain ("example.com") is then not the request's host.
	e := httptest.New(t, app, httptest.URL("http://app.example.com"))

	signin := e.POST("/signin").WithJSON(map[string]string{"username": "kataras", "password": "pass"}).
		Expect().Status(httptest.StatusOK)

	set := setCookieLine(t, signin.Raw(), cookieName)
	if set.Domain == "" || set.Path == "" {
		t.Fatalf("expected the signin cookie to carry a domain and a path, got %q", set.String())
	}

	// No Authorization header: the test client keeps cookies like a browser
	// does, so the request is authenticated by the cookie alone.
	signout := e.GET("/signout").Expect().Status(httptest.StatusOK)

	removal := setCookieLine(t, signout.Raw(), cookieName)
	if removal.MaxAge >= 0 {
		t.Fatalf("expected the signout to expire the cookie, got %q", removal.String())
	}

	if removal.Domain != set.Domain || removal.Path != set.Path {
		t.Errorf("the signout removes a cookie the signin did not set: set with Domain=%q Path=%q, removed with Domain=%q Path=%q",
			set.Domain, set.Path, removal.Domain, removal.Path)
	}

	// What that means for the client: the cookie is gone, so the next request
	// carries no token at all.
	e.GET("/signout").Expect().Status(httptest.StatusUnauthorized)
}
