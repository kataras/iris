package oauth2

import (
	stdContext "context"
	"testing"

	"github.com/kataras/iris/v14/auth"
	"github.com/stretchr/testify/assert"
)

type testUser struct {
	ID string `json:"id"`
}

func newTestServer(t *testing.T) *Server[testUser] {
	t.Helper()
	var config auth.Configuration
	assert.NoError(t, config.BindRandom())
	a, err := auth.New[testUser](config)
	assert.NoError(t, err)
	return New(a, WithClientStore(NewMemoryClientStore()))
}

func TestServerCreateClientConflict(t *testing.T) {
	ctx := stdContext.Background()
	s := newTestServer(t)

	c, err := s.CreateClient(ctx, ClientRegistration{Name: "app", Website: "http://example.com", LogoURL: "http://example.com/logo.png", Scopes: []string{"a"}})
	assert.NoError(t, err)
	stored, found, err := s.ClientStore().GetClient(ctx, c.ID)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, c.Name, stored.Name)

	_, err = s.CreateClient(ctx, ClientRegistration{Name: "app", Website: "http://example.com", LogoURL: "http://example.com/logo.png"})
	assert.ErrorIs(t, err, ErrClientAlreadyExists)

	// RegisterClient keeps its get-or-create behavior.
	again, err := s.RegisterClient(ctx, ClientRegistration{Name: "app", Website: "http://example.com", LogoURL: "http://example.com/logo.png"})
	assert.NoError(t, err)
	assert.Equal(t, c.ID, again.ID)
}

func TestServerEnsureClientScopes(t *testing.T) {
	ctx := stdContext.Background()
	s := newTestServer(t)

	c, err := s.CreateClient(ctx, ClientRegistration{Name: "app", Website: "http://example.com", LogoURL: "http://example.com/logo.png", Scopes: []string{"a", "b"}})
	assert.NoError(t, err)
	secret := c.Secret

	changed, err := s.EnsureClientScopes(ctx, c, []string{"b", "c", "a", "d"})
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, []string{"a", "b", "c", "d"}, c.Scopes)

	stored, _, _ := s.ClientStore().GetClient(ctx, c.ID)
	assert.Equal(t, []string{"a", "b", "c", "d"}, stored.Scopes)
	assert.Equal(t, 0, stored.Version)
	assert.Equal(t, secret, stored.Secret)

	changed, err = s.EnsureClientScopes(ctx, c, []string{"a", "d"})
	assert.NoError(t, err)
	assert.False(t, changed)
}
