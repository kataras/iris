package identity

import (
	"net/http"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/httptest"

	"github.com/stretchr/testify/require"
)

// TestServerGateRejectsUserClient mounts the same split as Server.Routing:
// any authenticated client may call /clients, and /u plus /cors require a server scope.
func TestServerGateRejectsUserClient(t *testing.T) {
	signer, clientAuth := newTestClientSigner(t)

	var userConfig auth.Configuration
	require.NoError(t, userConfig.BindRandom())
	userConfig.Cookie = auth.CookieConfiguration{}
	userAuth, err := auth.New[entity.FlatUser](userConfig)
	require.NoError(t, err)

	srv := oauth2.New(userAuth, oauth2.WithClientStore(oauth2.NewMemoryClientStore()))
	seed := func(name string, kind ClientKind, redirectURI string) string {
		t.Helper()
		spec, err := specForKind(kind, "")
		require.NoError(t, err)
		client, err := srv.CreateClient(t.Context(), oauth2.ClientRegistration{
			Name:          name,
			Website:       "https://example.com",
			LogoURL:       "https://example.com/logo.png",
			RedirectURI:   redirectURI,
			GrantTypes:    spec.GrantTypes,
			Scopes:        spec.Scopes,
			CodeMaxAge:    spec.CodeMaxAge,
			AccessMaxAge:  spec.AccessMaxAge,
			RefreshMaxAge: spec.RefreshMaxAge,
		})
		require.NoError(t, err)
		creds, err := signer.Credentials(client)
		require.NoError(t, err)
		return creds.Token
	}
	serverToken := seed("server-app", ClientKindServer, "")
	userToken := seed("user-app", ClientKindUser, "https://example.com/callback")

	noContent := func(ctx *context.Context) { ctx.StatusCode(http.StatusNoContent) }
	serverGate := oauth2.AllowPolicies(ServerOnlyPolicies...)

	app := iris.New()
	app.Use(clientAuth.VerifyHandler(srv.ClientVerifier()))
	app.Router("/clients", func(r router.Router) {
		r.Get("/", noContent)
	})
	app.Router("/u", serverGate, func(r router.Router) {
		r.Post("/signin", noContent)
	})
	app.Router("/cors", serverGate, func(r router.Router) {
		r.Post("/", noContent)
	})

	e := httptest.New(t, app)
	e.POST("/u/signin").WithHeader(Header, userToken).Expect().Status(http.StatusForbidden)
	e.POST("/cors").WithHeader(Header, userToken).Expect().Status(http.StatusForbidden)
	e.POST("/u/signin").WithHeader(Header, serverToken).Expect().Status(http.StatusNoContent)
	e.POST("/cors").WithHeader(Header, serverToken).Expect().Status(http.StatusNoContent)
	// Self routes stay open to a user client.
	e.GET("/clients").WithHeader(Header, userToken).Expect().Status(http.StatusNoContent)
}
