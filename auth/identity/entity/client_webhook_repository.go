package entity

import (
	"context"

	"github.com/kataras/pg"
)

// ClientWebhookRepository handles cost oauth2 client's webhook-related database operations.
type ClientWebhookRepository struct {
	*pg.Repository[ClientWebhook]
}

// NewClientWebhookRepository creates a new ClientWebhookRepository instance.
func NewClientWebhookRepository(db *pg.DB) *ClientWebhookRepository {
	return &ClientWebhookRepository{
		Repository: pg.NewRepository[ClientWebhook](db),
	}
}

// InTransaction executes a function within a database transaction.
func (r *ClientWebhookRepository) InTransaction(ctx context.Context, fn func(*ClientWebhookRepository) error) error {
	conn := r.Repository.DB()
	if conn.IsTransaction() {
		return fn(r)
	}

	return conn.InTransaction(ctx, func(db *pg.DB) error {
		return fn(NewClientWebhookRepository(db))
	})
}
