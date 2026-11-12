package oauth2

import (
	stdContext "context"
	"testing"
	"time"

	"github.com/kataras/iris/v14/core/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

func TestMemClientStore(t *testing.T) {
	store := NewMemoryClientStore()
	ctx := stdContext.Background()

	client, err := NewClient(ClientRegistration{
		Name:          "test-client",
		Website:       "http://example.com",
		LogoURL:       "http://example.com/logo.png",
		RedirectURI:   "http://example.com/callback",
		CodeMaxAge:    3600,
		AccessMaxAge:  3600,
		RefreshMaxAge: 604800,
	})
	assert.NoError(t, err)

	err = store.InsertClient(ctx, client)
	assert.NoError(t, err)

	retrievedClient, ok, err := store.GetClient(ctx, client.ID)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, client, retrievedClient)

	_, ok, err = store.GetClient(ctx, "non-existent-id")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func newTestClient(t *testing.T, name string, scopes ...string) *Client {
	t.Helper()
	c, err := NewClient(ClientRegistration{Name: name, Website: "http://example.com", LogoURL: "http://example.com/logo.png", Scopes: scopes})
	assert.NoError(t, err)
	return c
}

func seedClients(t *testing.T, store ClientStore) {
	t.Helper()
	ctx := stdContext.Background()
	for _, c := range []*Client{
		newTestClient(t, "Charlie App", "clients:create"),
		newTestClient(t, "alpha app", "clients:read", "clients:create"),
		newTestClient(t, "Bravo", "clients:read"),
	} {
		assert.NoError(t, store.InsertClient(ctx, c))
		time.Sleep(2 * time.Millisecond) // distinct created_at values.
	}
}

func clientNames(clients []*Client) []string {
	names := make([]string, len(clients))
	for i, c := range clients {
		names[i] = c.Name
	}
	return names
}

func TestMemClientStoreList(t *testing.T) {
	ctx := stdContext.Background()
	store := NewMemoryClientStore()
	seedClients(t, store)

	all := pagination.PageOptions{Page: 1, Size: 10}

	items, total, err := store.ListClients(ctx, all, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, items, 3)
	assert.NotEmpty(t, items[0].Secret, "the store returns secrets as stored")

	// Name filter: case-insensitive contains.
	items, total, err = store.ListClients(ctx, all, &ClientFilter{Name: "APP", Sort: "name"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, []string{"Charlie App", "alpha app"}, clientNames(items))

	// Scope filter: exact membership.
	items, total, err = store.ListClients(ctx, all, &ClientFilter{Scope: "clients:read", Sort: "name"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, []string{"Bravo", "alpha app"}, clientNames(items))

	_, total, err = store.ListClients(ctx, all, &ClientFilter{Scope: "clients"})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)

	// Sort by name ascending and descending.
	items, _, err = store.ListClients(ctx, all, &ClientFilter{Sort: "name"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"Bravo", "Charlie App", "alpha app"}, clientNames(items))
	items, _, err = store.ListClients(ctx, all, &ClientFilter{Sort: "name", SortDescending: true})
	assert.NoError(t, err)
	assert.Equal(t, []string{"alpha app", "Charlie App", "Bravo"}, clientNames(items))

	// Paging: offset/limit with the total of the whole match.
	items, total, err = store.ListClients(ctx, pagination.PageOptions{Page: 2, Size: 2}, &ClientFilter{Sort: "name"})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, []string{"alpha app"}, clientNames(items))
	items, total, err = store.ListClients(ctx, pagination.PageOptions{Page: 3, Size: 2}, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Empty(t, items)

	// Default sort is created_at, in insertion order.
	items, _, err = store.ListClients(ctx, all, nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Charlie App", "alpha app", "Bravo"}, clientNames(items))

	// Invalid sort.
	_, _, err = store.ListClients(ctx, all, &ClientFilter{Sort: "secret"})
	assert.ErrorIs(t, err, ErrInvalidClientFilter)
	_, err = store.CountClients(ctx, &ClientFilter{Sort: "secret"})
	assert.ErrorIs(t, err, ErrInvalidClientFilter)
}

func TestMemClientStoreCount(t *testing.T) {
	ctx := stdContext.Background()
	store := NewMemoryClientStore()
	seedClients(t, store)

	n, err := store.CountClients(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), n)
	n, err = store.CountClients(ctx, &ClientFilter{Name: "app"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), n)
	n, err = store.CountClients(ctx, &ClientFilter{Scope: "clients:create"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestMemClientStoreDelete(t *testing.T) {
	ctx := stdContext.Background()
	store := NewMemoryClientStore()
	c := newTestClient(t, "doomed", "a")
	assert.NoError(t, store.InsertClient(ctx, c))

	ok, err := store.DeleteClient(ctx, c.ID)
	assert.NoError(t, err)
	assert.True(t, ok)
	_, found, _ := store.GetClient(ctx, c.ID)
	assert.False(t, found)

	ok, err = store.DeleteClient(ctx, c.ID)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestMemClientStoreUpdateScopesAndVersion(t *testing.T) {
	ctx := stdContext.Background()
	store := NewMemoryClientStore()
	c := newTestClient(t, "app", "a")
	secret := c.Secret
	assert.NoError(t, store.InsertClient(ctx, c))

	ok, err := store.UpdateClientScopes(ctx, c.ID, []string{"a", "b"})
	assert.NoError(t, err)
	assert.True(t, ok)
	got, _, _ := store.GetClient(ctx, c.ID)
	assert.Equal(t, []string{"a", "b"}, got.Scopes)
	assert.Equal(t, secret, got.Secret)
	assert.Equal(t, 0, got.Version)

	ok, err = store.IncrementClientVersion(ctx, c.ID)
	assert.NoError(t, err)
	assert.True(t, ok)
	got, _, _ = store.GetClient(ctx, c.ID)
	assert.Equal(t, 1, got.Version)
	assert.Equal(t, secret, got.Secret)
	assert.Equal(t, []string{"a", "b"}, got.Scopes)

	ok, err = store.UpdateClientScopes(ctx, "missing", nil)
	assert.NoError(t, err)
	assert.False(t, ok)
	ok, err = store.IncrementClientVersion(ctx, "missing")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestMemClientStoreReturnsDetachedCopies(t *testing.T) {
	ctx := stdContext.Background()
	store := NewMemoryClientStore()
	c := newTestClient(t, "app", "a")
	secret := c.Secret
	assert.NoError(t, store.InsertClient(ctx, c))

	items, _, err := store.ListClients(ctx, pagination.PageOptions{Page: 1, Size: 10}, nil)
	assert.NoError(t, err)
	items[0].Secret = ""

	fetched, _, _ := store.GetClient(ctx, c.ID)
	assert.Equal(t, secret, fetched.Secret, "blanking a listed client must not change the store")
	fetched.Secret = ""

	byName, _, _ := store.GetClientByName(ctx, "app")
	assert.Equal(t, secret, byName.Secret, "blanking a fetched client must not change the store")
}
