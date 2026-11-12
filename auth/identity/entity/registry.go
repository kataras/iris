package entity

import (
	"context"
	"errors"
	"fmt"

	"github.com/kataras/pg"
)

// Registry contains all repositories.
// It is used to pass the repositories to the services.
type Registry struct {
	db *pg.DB

	clients        *ClientRepository
	clientWebhooks *ClientWebhookRepository
	users          *UserRepository
	userGroups     *UserGroupRepository
	metrics        *MetricRepository
	origins        *OriginRepository
}

// NewRegistry creates a new postgres Registry.
func NewRegistry(db *pg.DB) *Registry {
	return &Registry{
		db: db,

		clients:        NewClientRepository(db),
		clientWebhooks: NewClientWebhookRepository(db),
		users:          NewUserRepository(db),
		userGroups:     NewUserGroupRepository(db),
		metrics:        NewMetricRepository(db),
		origins:        NewOriginRepository(db),
	}
}

// Clients returns the client repository.
func (r *Registry) Clients() *ClientRepository {
	return r.clients
}

// ClientWebhooks returns the client webhook repository.
func (r *Registry) ClientWebhooks() *ClientWebhookRepository {
	return r.clientWebhooks
}

// Users returns the user repository.
func (r *Registry) Users() *UserRepository {
	return r.users
}

// UserGroups returns the user group repository.
func (r *Registry) UserGroups() *UserGroupRepository {
	return r.userGroups
}

// Metrics returns the metric repository.
func (r *Registry) Metrics() *MetricRepository {
	return r.metrics
}

// Origins returns the origin repository.
func (r *Registry) Origins() *OriginRepository {
	return r.origins
}

// InTransaction executes the given function in a transaction.
func (r *Registry) InTransaction(ctx context.Context, fn func(*Registry) error) (err error) {
	if r.db.IsTransaction() {
		return fn(r)
	}

	var tx *pg.DB
	tx, err = r.db.Begin(ctx)
	if err != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p) // re-throw panic after RollbackDatabase.
		} else if err != nil {
			if errors.Is(err, pg.ErrIntentionalRollback) { // See PlanService.InTransaction method.
				err = tx.Rollback(ctx)
				return
			}

			// pg.DB.Rollback does return nil. On Go 1.27.0 and 1.27.1 the generic
			// methods of pg.DB shift staticcheck's per-method facts onto the
			// wrong methods (golang/go#81306, fixed in Go 1.27.2).
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil { //nolint:staticcheck // SA4023 false positive, see above.
				err = fmt.Errorf("%w: %s", err, rollbackErr.Error())
			}
		} else {
			err = tx.Commit(ctx)
		}
	}()

	newRegistry := NewRegistry(tx)
	err = fn(newRegistry)

	return
}

// IsInTransaction returns true if the current request is in a transaction.
func (r *Registry) IsInTransaction() bool {
	return r.db.IsTransaction()
}
