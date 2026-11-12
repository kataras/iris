package oauth2

import (
	stdContext "context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/kataras/iris/v14/core/pkg/cache"
	"github.com/kataras/iris/v14/core/pkg/pagination"
	"github.com/kataras/iris/v14/rest/types"
)

// ClientFilter holds the optional conditions and the ordering of ListClients and CountClients.
// The zero value matches every client, ordered by creation time ascending.
type ClientFilter struct {
	// Name matches the clients whose name contains this text, ignoring case.
	Name string `json:"name,omitempty"`
	// Scope matches the clients that have exactly this scope, e.g. "clients:create".
	Scope string `json:"scope,omitempty"`
	// Sort is the field to order by: "name", "created_at" or "updated_at".
	// Empty means "created_at". Any other value is an ErrInvalidClientFilter.
	Sort string `json:"sort,omitempty"`
	// SortDescending reverses the order.
	SortDescending bool `json:"sort_descending,omitzero"`
}

// ErrInvalidClientFilter is returned by ListClients and CountClients when the filter is not valid,
// for example an unknown Sort field.
var ErrInvalidClientFilter = errors.New("oauth2: invalid client filter")

// sortColumn validates the filter and returns the sort column from a fixed whitelist.
// A nil filter is valid.
func (f *ClientFilter) sortColumn() (string, error) {
	if f == nil {
		return "created_at", nil
	}

	switch f.Sort {
	case "":
		return "created_at", nil
	case "name", "created_at", "updated_at":
		return f.Sort, nil
	default:
		return "", ErrInvalidClientFilter
	}
}

// match reports whether the client satisfies the Name and Scope conditions of the filter.
func (f *ClientFilter) match(c *Client) bool {
	if f == nil {
		return true
	}
	if f.Name != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(f.Name)) {
		return false
	}
	if f.Scope != "" && !slices.Contains(c.Scopes, f.Scope) {
		return false
	}
	return true
}

// ClientStore describes methods to insert and retrieve OAuth2 clients.
type ClientStore interface {
	// InsertClient adds a new client to the store.
	InsertClient(ctx stdContext.Context, client *Client) error
	// GetClient fetches a client by its ID.
	GetClient(ctx stdContext.Context, id string) (*Client, bool, error)
	// GetClientByName fetches a client by its name.
	GetClientByName(ctx stdContext.Context, name string) (*Client, bool, error)
	// ListClients returns one page of the clients that match the filter (nil matches all)
	// and the total number of matches. Clients are returned as stored, secrets included.
	ListClients(ctx stdContext.Context, page pagination.PageOptions, filter *ClientFilter) ([]*Client, int64, error)
	// CountClients returns the number of clients that match the filter (nil matches all).
	CountClients(ctx stdContext.Context, filter *ClientFilter) (int64, error)
	// DeleteClient removes a client by its ID. It reports false when there is no such client.
	DeleteClient(ctx stdContext.Context, id string) (bool, error)
	// UpdateClientScopes replaces the scopes of a client. The secret and the version are untouched.
	// It reports false when there is no such client.
	UpdateClientScopes(ctx stdContext.Context, id string, scopes []string) (bool, error)
	// IncrementClientVersion adds one to the version of a client, which invalidates the tokens
	// issued for the previous version. The secret is untouched.
	// It reports false when there is no such client.
	IncrementClientVersion(ctx stdContext.Context, id string) (bool, error)
}

// memClientStore is an in-memory implementation of ClientStore using a cache.
type memClientStore struct {
	cache cache.CacheIterator[Client]
}

// NewMemoryClientStore creates an in-memory client store.
func NewMemoryClientStore() ClientStore {
	return &memClientStore{
		cache: cache.NewMemoryCache[Client](0),
	}
}

// InsertClient inserts a client record into the in-memory cache.
// It fills the CreatedAt and UpdatedAt timestamps when they are empty.
func (s *memClientStore) InsertClient(ctx stdContext.Context, client *Client) error {
	now := types.ISO8601(time.Now())
	if time.Time(client.CreatedAt).IsZero() {
		client.CreatedAt = now
	}
	if time.Time(client.UpdatedAt).IsZero() {
		client.UpdatedAt = now
	}
	return s.cache.Set(ctx, client.ID, client)
}

// GetClient retrieves a client by its ID from the in-memory cache.
func (s *memClientStore) GetClient(ctx stdContext.Context, id string) (*Client, bool, error) {
	client, found, err := s.cache.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	cp := *client // a detached copy, like a database row.
	return &cp, true, nil
}

// GetClientByName retrieves a client by its name from the in-memory cache.
func (s *memClientStore) GetClientByName(ctx stdContext.Context, name string) (*Client, bool, error) {
	_, client, found, err := s.cache.First(ctx, func(item *Client) bool {
		return item.Name == name
	})
	if err != nil || !found {
		return nil, false, err
	}
	cp := *client // a detached copy, like a database row.
	return &cp, true, nil
}

// filtered returns the sorted clients that match the filter.
func (s *memClientStore) filtered(ctx stdContext.Context, filter *ClientFilter) ([]*Client, error) {
	column, err := filter.sortColumn()
	if err != nil {
		return nil, err
	}

	var clients []*Client
	err = s.cache.Walk(ctx, func(_ string, c *Client) error {
		if filter.match(c) {
			cp := *c // a detached copy, callers may modify it.
			clients = append(clients, &cp)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	desc := filter != nil && filter.SortDescending
	slices.SortFunc(clients, func(a, b *Client) int {
		var cmp int
		switch column {
		case "name":
			cmp = strings.Compare(a.Name, b.Name)
		case "updated_at":
			cmp = time.Time(a.UpdatedAt).Compare(time.Time(b.UpdatedAt))
		default:
			cmp = time.Time(a.CreatedAt).Compare(time.Time(b.CreatedAt))
		}
		if cmp == 0 { // keep the order deterministic.
			cmp = strings.Compare(a.ID, b.ID)
		}
		if desc {
			return -cmp
		}
		return cmp
	})
	return clients, nil
}

// ListClients returns one page of the matching clients and the total number of matches.
func (s *memClientStore) ListClients(ctx stdContext.Context, page pagination.PageOptions, filter *ClientFilter) ([]*Client, int64, error) {
	clients, err := s.filtered(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(clients))
	offset, limit := page.GetOffset(), page.GetLimit()
	if offset >= len(clients) {
		return []*Client{}, total, nil
	}
	end := min(offset+limit, len(clients))
	return clients[offset:end], total, nil
}

// CountClients returns the number of matching clients.
func (s *memClientStore) CountClients(ctx stdContext.Context, filter *ClientFilter) (int64, error) {
	clients, err := s.filtered(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int64(len(clients)), nil
}

// DeleteClient removes a client from the in-memory cache.
func (s *memClientStore) DeleteClient(ctx stdContext.Context, id string) (bool, error) {
	_, found, err := s.cache.Get(ctx, id)
	if err != nil || !found {
		return false, err
	}
	if err = s.cache.Delete(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

// UpdateClientScopes replaces the scopes of a stored client.
func (s *memClientStore) UpdateClientScopes(ctx stdContext.Context, id string, scopes []string) (bool, error) {
	return s.mutate(ctx, id, func(c *Client) { c.Scopes = slices.Clone(scopes) })
}

// IncrementClientVersion adds one to the version of a stored client.
func (s *memClientStore) IncrementClientVersion(ctx stdContext.Context, id string) (bool, error) {
	return s.mutate(ctx, id, func(c *Client) { c.Version++ })
}

// mutate gets the client, applies fn on a copy, and stores the copy.
func (s *memClientStore) mutate(ctx stdContext.Context, id string, fn func(*Client)) (bool, error) {
	client, found, err := s.cache.Get(ctx, id)
	if err != nil || !found {
		return false, err
	}
	updated := *client
	fn(&updated)
	updated.UpdatedAt = types.ISO8601(time.Now())
	if err = s.cache.Set(ctx, id, &updated); err != nil {
		return false, err
	}
	return true, nil
}
