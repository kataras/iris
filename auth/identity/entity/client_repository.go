package entity

import (
	"context"

	"github.com/kataras/iris/v14/auth/oauth2"

	"github.com/kataras/pg"
)

type ClientRepository struct {
	*pg.Repository[oauth2.Client]
}

// NewClientRepository creates a new ClientRepository instance.
// The most things are handled by the oauth2's postgres client store.
func NewClientRepository(db *pg.DB) *ClientRepository {
	return &ClientRepository{
		Repository: pg.NewRepository[oauth2.Client](db),
	}
}

// InTransaction executes a function within a database transaction.
func (r *ClientRepository) InTransaction(ctx context.Context, fn func(*ClientRepository) error) error {
	conn := r.Repository.DB()
	if conn.IsTransaction() {
		return fn(r)
	}

	return conn.InTransaction(ctx, func(db *pg.DB) error {
		return fn(NewClientRepository(db))
	})
}

func (r *ClientRepository) UpdateTokenMaxAge(ctx context.Context, clientID string, accessMaxAge, refreshMaxAge int) (bool, error) {
	query := `UPDATE oauth2_clients SET access_max_age = $1, refresh_max_age = $2 WHERE id = $3;`
	return r.MutateSingle(ctx, query, accessMaxAge, refreshMaxAge, clientID)
}

func (r *ClientRepository) RestoreTokenMaxAge(ctx context.Context, clientID string) (bool, error) {
	query := `UPDATE oauth2_clients SET access_max_age = DEFAULT, refresh_max_age = DEFAULT WHERE id = $1;`
	return r.MutateSingle(ctx, query, clientID)
}

func (r *ClientRepository) IncrementClientVersion(ctx context.Context, clientID string) (bool, error) {
	query := `UPDATE oauth2_clients SET version = version + 1 WHERE id = $1;`
	return r.MutateSingle(ctx, query, clientID)
}
