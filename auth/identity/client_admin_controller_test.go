package identity

import (
	"net/http"
	"slices"
	"testing"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/httptest"

	"github.com/iris-contrib/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clientAdminFixture is the clients controller mounted the way Server.Routing mounts it,
// over an in-memory client store, with one client of each kind already created.
type clientAdminFixture struct {
	e          *httpexpect.Expect
	srv        *oauth2.Server[entity.FlatUser]
	clientAuth *auth.Auth[oauth2.ClientTokenClaims]
	signer     *ClientTokenSigner

	admin, server, user                *oauth2.Client
	adminToken, serverToken, userToken string
}

func newClientAdminFixture(t *testing.T) *clientAdminFixture {
	t.Helper()

	var userConfig auth.Configuration
	require.NoError(t, userConfig.BindRandom())
	userConfig.Cookie = auth.CookieConfiguration{}
	userAuth, err := auth.New[entity.FlatUser](userConfig)
	require.NoError(t, err)

	signer, clientAuth := newTestClientSigner(t)
	srv := oauth2.New(userAuth, oauth2.WithClientStore(oauth2.NewMemoryClientStore()))

	f := &clientAdminFixture{srv: srv, clientAuth: clientAuth, signer: signer}
	f.admin, f.adminToken = f.seed(t, "admin-app", ClientKindAdmin, "", "")
	f.server, f.serverToken = f.seed(t, "server-app", ClientKindServer, "", "")
	f.user, f.userToken = f.seed(t, "user-app", ClientKindUser, "https://example.com/callback", "")

	f.e = httptest.NewController(t, "/clients", &ClientController{oauth2Server: srv, signer: signer},
		RegisterCodeErrors().Handler(),
		clientAuth.VerifyHandler(srv.ClientVerifier()),
		oauth2.AllowPolicies(ServerAppPolicies...),
	)
	return f
}

// seed creates a client of the given kind through the oauth2 server and signs its token.
func (f *clientAdminFixture) seed(t *testing.T, name string, kind ClientKind, redirectURI, verificationURI string) (*oauth2.Client, string) {
	t.Helper()

	spec, err := specForKind(kind, verificationURI)
	require.NoError(t, err)
	return f.seedScopes(t, name, spec.Scopes, spec, redirectURI, verificationURI)
}

func (f *clientAdminFixture) seedScopes(t *testing.T, name string, scopes []string, spec clientKindSpec, redirectURI, verificationURI string) (*oauth2.Client, string) {
	t.Helper()

	client, err := f.srv.CreateClient(t.Context(), oauth2.ClientRegistration{
		Name:            name,
		Website:         "https://example.com",
		LogoURL:         "https://example.com/logo.png",
		RedirectURI:     redirectURI,
		VerificationURI: verificationURI,
		GrantTypes:      spec.GrantTypes,
		Scopes:          scopes,
		CodeMaxAge:      spec.CodeMaxAge,
		AccessMaxAge:    spec.AccessMaxAge,
		RefreshMaxAge:   spec.RefreshMaxAge,
	})
	require.NoError(t, err)

	creds, err := f.signer.Credentials(client)
	require.NoError(t, err)
	return client, creds.Token
}

func (f *clientAdminFixture) as(token string, method, path string, args ...any) *httpexpect.Request {
	return f.e.Request(method, path, args...).WithHeader(Header, token)
}

// TestClientAdminNonAdminTokensForbidden covers a server token and a user token. The user
// token holds user:read, so it passes the outer group guard and must still be refused.
func TestClientAdminNonAdminTokensForbidden(t *testing.T) {
	f := newClientAdminFixture(t)
	tokens := map[string]string{"server": f.serverToken, "user": f.userToken}
	for name, token := range tokens {
		t.Run(name, func(t *testing.T) {
			assertClientAdminForbidden(t, f, token)
		})
	}

	// The self route still answers the server client.
	f.as(f.serverToken, http.MethodGet, "/clients").Expect().Status(http.StatusOK).
		JSON().Object().HasValue("id", f.server.ID)
}

func assertClientAdminForbidden(t *testing.T, f *clientAdminFixture, token string) {
	t.Helper()

	body := map[string]any{"name": "x", "website": "https://example.com", "logo_url": "https://example.com/logo.png", "kind": "server"}
	f.as(token, http.MethodPost, "/clients").WithJSON(body).Expect().Status(http.StatusForbidden)
	f.as(token, http.MethodPost, "/clients/list").Expect().Status(http.StatusForbidden)
	f.as(token, "QUERY", "/clients/list").Expect().Status(http.StatusForbidden)
	f.as(token, http.MethodGet, "/clients/count").Expect().Status(http.StatusForbidden)
	f.as(token, http.MethodGet, "/clients/{id}", f.user.ID).Expect().Status(http.StatusForbidden)
	f.as(token, http.MethodDelete, "/clients/{id}", f.user.ID).Expect().Status(http.StatusForbidden)
	f.as(token, http.MethodPost, "/clients/{id}/token", f.user.ID).Expect().Status(http.StatusForbidden)
	f.as(token, http.MethodPost, "/clients/{id}/revoke-tokens", f.user.ID).Expect().Status(http.StatusForbidden)
}

func TestClientAdminCreate(t *testing.T) {
	f := newClientAdminFixture(t)

	body := map[string]any{"name": "billing-app", "website": "https://example.com", "logo_url": "https://example.com/logo.png", "kind": "admin"}
	resp := f.as(f.adminToken, http.MethodPost, "/clients").WithJSON(body).Expect().Status(http.StatusCreated)

	var created CreateClientResponse
	resp.JSON().Decode(&created)
	require.NotNil(t, created.Client)
	assert.Equal(t, created.Client.ID, created.Audience)
	assert.Equal(t, ClientKindAdmin, created.Kind)
	assert.Equal(t, "billing-app", created.Client.Name)
	assert.NotEmpty(t, created.Client.Secret)
	assert.Equal(t, oauth2.PolicyScopes(AdminAppPolicies...), created.Client.Scopes)
	assert.Equal(t, 7200, created.Client.AccessMaxAge) // the kind default.
	assert.True(t, created.EncryptionKey == testEncryptionKey)

	_, verified, err := f.clientAuth.Verify(t.Context(), []byte(created.Token))
	require.NoError(t, err)
	assert.Equal(t, []string{created.Client.ID}, []string(verified.StandardClaims.Audience))

	// The new token is accepted by the API itself.
	f.as(created.Token, http.MethodGet, "/clients/count").Expect().Status(http.StatusOK).
		JSON().Object().HasValue("total_count", 4)

	// Explicit max ages win over the kind defaults.
	body = map[string]any{"name": "short-app", "website": "https://example.com", "logo_url": "https://example.com/logo.png", "kind": "server", "access_max_age": 600, "refresh_max_age": 1200}
	f.as(f.adminToken, http.MethodPost, "/clients").WithJSON(body).Expect().Status(http.StatusCreated).
		JSON().Object().Value("client").Object().HasValue("access_max_age", 600).HasValue("refresh_max_age", 1200)

	// A user client needs only a redirect URI; a verification URI adds the device code grant.
	body = map[string]any{"name": "web-app", "website": "https://example.com", "logo_url": "https://example.com/logo.png", "kind": "user", "redirect_uri": "https://example.com/callback"}
	web := f.as(f.adminToken, http.MethodPost, "/clients").WithJSON(body).Expect().Status(http.StatusCreated).JSON().Object()
	web.HasValue("kind", "user").NotContainsKey("encryption_key")
	web.Value("client").Object().HasValue("grant_types", []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeRefreshToken})

	body = map[string]any{"name": "tv-app", "website": "https://example.com", "logo_url": "https://example.com/logo.png", "kind": "user", "redirect_uri": "https://example.com/callback", "verification_uri": "https://example.com/device"}
	f.as(f.adminToken, http.MethodPost, "/clients").WithJSON(body).Expect().Status(http.StatusCreated).
		JSON().Object().Value("client").Object().HasValue("grant_types", []string{oauth2.GrantTypeAuthorizationCode, oauth2.GrantTypeRefreshToken, oauth2.GrantTypeDeviceCode})

	// The same name again.
	f.as(f.adminToken, http.MethodPost, "/clients").WithJSON(body).Expect().Status(http.StatusConflict)
}

func TestClientAdminCreateBadRequest(t *testing.T) {
	f := newClientAdminFixture(t)

	valid := func() map[string]any {
		return map[string]any{"name": "new-app", "website": "https://example.com", "logo_url": "https://example.com/logo.png", "kind": "server"}
	}

	tests := []struct {
		name string
		edit func(map[string]any)
	}{
		{"missing name", func(b map[string]any) { delete(b, "name") }},
		{"missing website", func(b map[string]any) { delete(b, "website") }},
		{"missing logo", func(b map[string]any) { delete(b, "logo_url") }},
		{"missing kind", func(b map[string]any) { delete(b, "kind") }},
		{"unknown kind", func(b map[string]any) { b["kind"] = "bogus" }},
		{"user without redirect_uri", func(b map[string]any) { b["kind"] = "user" }},
		{"refresh not longer than access", func(b map[string]any) { b["access_max_age"] = 600; b["refresh_max_age"] = 601 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := valid()
			tt.edit(body)
			f.as(f.adminToken, http.MethodPost, "/clients").WithJSON(body).Expect().Status(http.StatusBadRequest)
		})
	}

	f.as(f.adminToken, http.MethodPost, "/clients").WithBytes([]byte("{not json")).
		WithHeader("Content-Type", "application/json").Expect().Status(http.StatusBadRequest)
}

func TestClientAdminList(t *testing.T) {
	f := newClientAdminFixture(t)

	for _, method := range []string{http.MethodPost, "QUERY"} {
		t.Run(method, func(t *testing.T) {
			page := f.as(f.adminToken, method, "/clients/list").Expect().Status(http.StatusOK).JSON().Object()
			page.HasValue("total_items", 3)
			items := page.Value("items").Array()
			items.Length().IsEqual(3)
			for _, item := range items.Iter() {
				item.Object().NotContainsKey("secret").ContainsKey("id")
			}

			page = f.as(f.adminToken, method, "/clients/list").WithJSON(map[string]any{"name": "SERVER"}).
				Expect().Status(http.StatusOK).JSON().Object()
			page.HasValue("total_items", 1)
			page.Value("items").Array().Value(0).Object().HasValue("id", f.server.ID).NotContainsKey("secret")
		})
	}

	// The listing never blanks the stored secret.
	stored, found, err := f.srv.ClientStore().GetClient(t.Context(), f.server.ID)
	require.NoError(t, err)
	require.True(t, found)
	assert.True(t, stored.Secret == f.server.Secret)

	f.as(f.adminToken, http.MethodPost, "/clients/list").WithJSON(map[string]any{"sort": "bogus"}).
		Expect().Status(http.StatusBadRequest)
}

func TestClientAdminCountAndGet(t *testing.T) {
	f := newClientAdminFixture(t)

	f.as(f.adminToken, http.MethodGet, "/clients/count").Expect().Status(http.StatusOK).
		JSON().Object().IsEqual(map[string]any{"total_count": 3})

	var got oauth2.Client
	f.as(f.adminToken, http.MethodGet, "/clients/{id}", f.user.ID).Expect().Status(http.StatusOK).JSON().Decode(&got)
	assert.Equal(t, f.user.ID, got.ID)
	assert.Equal(t, f.user.Name, got.Name)
	assert.NotEmpty(t, got.Secret)
	assert.True(t, got.Secret == f.user.Secret)

	f.as(f.adminToken, http.MethodGet, "/clients/{id}", "00000000-0000-4000-8000-000000000000").Expect().Status(http.StatusNotFound)
}

// TestClientAdminNonUUID shows that a non-UUID id never reaches the {id} handlers:
// the uuid macro rejects it first. httptest.NewController installs the path parameter
// error handler of iris.NewBuilder, so the answer here is 400 "invalid path parameter";
// an application built with plain iris.New and no errors.ApplyPatches answers 404.
func TestClientAdminNonUUID(t *testing.T) {
	f := newClientAdminFixture(t)

	requests := []struct{ method, path string }{
		{http.MethodGet, "/clients/not-a-uuid"},
		{http.MethodDelete, "/clients/not-a-uuid"},
		{http.MethodPost, "/clients/not-a-uuid/token"},
		{http.MethodPost, "/clients/not-a-uuid/revoke-tokens"},
	}
	for _, r := range requests {
		f.as(f.adminToken, r.method, r.path).Expect().Status(http.StatusBadRequest).
			JSON().Object().HasValue("message", "invalid path parameter")
	}
}

func TestClientAdminDelete(t *testing.T) {
	f := newClientAdminFixture(t)

	// Self.
	f.as(f.adminToken, http.MethodDelete, "/clients/{id}", f.admin.ID).Expect().Status(http.StatusBadRequest).
		Body().Contains("a client cannot delete itself")

	// The last client that can create clients, deleted by one that may only delete them.
	deleterSpec, err := specForKind(ClientKindServer, "")
	require.NoError(t, err)
	deleterScopes := slices.Concat(deleterSpec.Scopes, oauth2.PolicyScopes(DeleteClientsPolicy))
	_, deleterToken := f.seedScopes(t, "deleter-app", deleterScopes, deleterSpec, "", "")
	f.as(deleterToken, http.MethodDelete, "/clients/{id}", f.admin.ID).Expect().Status(http.StatusBadRequest)

	// With a second admin the first one may go.
	second, _ := f.seed(t, "second-admin-app", ClientKindAdmin, "", "")
	f.as(deleterToken, http.MethodDelete, "/clients/{id}", second.ID).Expect().Status(http.StatusNoContent)
	f.as(deleterToken, http.MethodDelete, "/clients/{id}", f.admin.ID).Expect().Status(http.StatusBadRequest)

	// A server client.
	f.as(f.adminToken, http.MethodDelete, "/clients/{id}", f.server.ID).Expect().Status(http.StatusNoContent)
	f.as(f.adminToken, http.MethodGet, "/clients/{id}", f.server.ID).Expect().Status(http.StatusNotFound)
	f.as(f.adminToken, http.MethodDelete, "/clients/{id}", f.server.ID).Expect().Status(http.StatusNotFound)
	// Its token died with it.
	f.as(f.serverToken, http.MethodGet, "/clients").Expect().Status(http.StatusUnauthorized)
}

func TestClientAdminIssueToken(t *testing.T) {
	f := newClientAdminFixture(t)

	user := f.as(f.adminToken, http.MethodPost, "/clients/{id}/token", f.user.ID).Expect().Status(http.StatusOK).JSON().Object()
	user.HasValue("audience", f.user.ID).HasValue("kind", "user").NotContainsKey("encryption_key")
	userToken := user.Value("token").String().NotEmpty().Raw()
	_, verified, err := f.clientAuth.Verify(t.Context(), []byte(userToken))
	require.NoError(t, err)
	assert.Equal(t, []string{f.user.ID}, []string(verified.StandardClaims.Audience))

	server := f.as(f.adminToken, http.MethodPost, "/clients/{id}/token", f.server.ID).Expect().Status(http.StatusOK).JSON().Object()
	server.HasValue("audience", f.server.ID).HasValue("kind", "server").ContainsKey("encryption_key")
	serverToken := server.Value("token").String().NotEmpty().Raw()
	f.as(serverToken, http.MethodGet, "/clients").Expect().Status(http.StatusOK)

	f.as(f.adminToken, http.MethodPost, "/clients/{id}/token", "00000000-0000-4000-8000-000000000000").Expect().Status(http.StatusNotFound)
}

func TestClientAdminRevokeTokens(t *testing.T) {
	f := newClientAdminFixture(t)

	f.as(f.serverToken, http.MethodGet, "/clients").Expect().Status(http.StatusOK)
	f.as(f.adminToken, http.MethodPost, "/clients/{id}/revoke-tokens", f.server.ID).Expect().Status(http.StatusNoContent)
	f.as(f.serverToken, http.MethodGet, "/clients").Expect().Status(http.StatusUnauthorized)

	// A token issued after the revocation carries the new version.
	fresh := f.as(f.adminToken, http.MethodPost, "/clients/{id}/token", f.server.ID).Expect().Status(http.StatusOK).
		JSON().Object().Value("token").String().Raw()
	f.as(fresh, http.MethodGet, "/clients").Expect().Status(http.StatusOK)

	f.as(f.adminToken, http.MethodPost, "/clients/{id}/revoke-tokens", "00000000-0000-4000-8000-000000000000").Expect().Status(http.StatusNotFound)
}
