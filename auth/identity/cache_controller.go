package identity

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
)

type CacheController struct {
	cache *httpcache.Cache
}

// NewCacheController creates a new CacheController with the given HTTP cache.
func NewCacheController(cache *httpcache.Cache) *CacheController {
	return &CacheController{
		cache: cache,
	}
}

func (c *CacheController) Routing(r router.Router) {
	r.Get("/", c.getCacheInformation)
	r.Delete("/", c.clearCache)
}

type CacheInformationResponse struct {
	TotalCount        int64             `json:"total_count"`
	PasswordProtected bool              `json:"password_protected"`
	Options           httpcache.Options `json:"options"`
}

func (c *CacheController) getCacheInformation(ctx *context.Context) {
	if c.cache.Options.Disable {
		resp := CacheInformationResponse{
			TotalCount:        0,
			PasswordProtected: false,
			Options: httpcache.Options{
				Disable: true,
			},
		}
		ctx.OK(resp, nil)
		return
	}

	totalCount, err := c.cache.CountAll(ctx)

	options := *c.cache.Options
	options.Password = ""
	passwordProtected := c.cache.Options.Password != ""
	resp := CacheInformationResponse{
		TotalCount:        int64(totalCount),
		PasswordProtected: passwordProtected,
		Options:           options,
	}

	ctx.OK(resp, err)
}

func (c *CacheController) clearCache(ctx *context.Context) {
	if c.cache.Options.Disable {
		ctx.NoContent(false, nil)
		return
	}

	n, err := c.cache.InvalidateAll(ctx)
	ctx.NoContent(n > 0, err)
}
