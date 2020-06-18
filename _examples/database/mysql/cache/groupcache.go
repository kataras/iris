package cache

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"myapp/entity"
	"myapp/sql"

	"github.com/mailgun/groupcache/v2"
)

// Service that cache will use to retrieve data.
type Service interface {
	RecordInfo() sql.Record
	GetByID(ctx context.Context, dest interface{}, id int64) error
	List(ctx context.Context, dest interface{}, opts sql.ListOptions) error
}

// Cache is a simple structure which holds the groupcache and the database service, exposes
// `GetByID` and `List` which returns cached (or stores new) items.
type Cache struct {
	service Service
	maxAge  time.Duration
	group   *groupcache.Group
}

// Size default size to use on groupcache, defaults to 3MB.
var Size int64 = 3 << (10 * 3)

// New returns a new cache service which exposes `GetByID` and `List` methods to work with.
// The "name" should be unique, "maxAge" for cache expiration.
func New(service Service, name string, maxAge time.Duration) *Cache {
	c := new(Cache)
	c.service = service
	c.maxAge = maxAge
	c.group = groupcache.NewGroup(name, Size, c)
	return c
}

const (
	prefixID   = "#"
	prefixList = "["
)

// Get implements the groupcache.Getter interface.
// Use `GetByID` and `List` instead.
func (c *Cache) Get(ctx context.Context, key string, dest groupcache.Sink) error {
	if len(key) < 2 { // empty or missing prefix+key, should never happen.
		return sql.ErrUnprocessable
	}

	var v interface{}

	prefix := key[0:1]
	key = key[1:]
	switch prefix {
	case prefixID:
		// Get by ID.
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil || id <= 0 {
			return err
		}

		switch c.service.RecordInfo().(type) {
		case *entity.Category:
			v = new(entity.Category)
		case *entity.Product:
			v = new(entity.Product)
		}

		err = c.service.GetByID(ctx, v, id)
		if err != nil {
			return err
		}

	case prefixList:
		// Get a set of records, list.
		q, err := url.ParseQuery(key)
		if err != nil {
			return err
		}
		opts := sql.ParseListOptions(q)

		switch c.service.RecordInfo().(type) {
		case *entity.Category:
			v = new(entity.Categories)
		case *entity.Product:
			v = new(entity.Products)
		}

		err = c.service.List(ctx, v, opts)
		if err != nil {
			return err
		}

	default:
		return sql.ErrUnprocessable
	}

	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return dest.SetBytes(b, time.Now().Add(c.maxAge))
}

// GetByID binds an item to "dest" an item based on its "id".
func (c *Cache) GetByID(ctx context.Context, id string, dest *[]byte) error {
	return c.group.Get(ctx, prefixID+id, groupcache.AllocatingByteSliceSink(dest))
}

// List binds item to "dest" based on the "rawQuery" of `url.Values` for `ListOptions`.
func (c *Cache) List(ctx context.Context, rawQuery string, dest *[]byte) error {
	return c.group.Get(ctx, prefixList+rawQuery, groupcache.AllocatingByteSliceSink(dest))
}
