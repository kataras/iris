package identity

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
)

type CORSController struct {
	cors  *CORSService
	cache *httpcache.Cache
}

func NewCORSController(cors *CORSService, cache *httpcache.Cache) *CORSController {
	return &CORSController{
		cors:  cors,
		cache: cache,
	}
}

func (c *CORSController) Routing(r router.Router) {
	caching, invalidating := c.cache.Pair(r)

	r.Get("/", caching, c.listOriginPatterns)
	r.Post("/", invalidating, c.upsertOriginPatterns)
	r.Delete("/", invalidating, c.deleteOriginPatterns)
}

func (c *CORSController) listOriginPatterns(ctx *context.Context) {
	patterns := c.cors.Patterns()
	ctx.OK(patterns, nil)
}

func (c *CORSController) upsertOriginPatterns(ctx *context.Context) {
	patterns, ok := ctx.BindJSON[[]string]()
	if !ok {
		return
	}

	modified, err := c.cors.UpsertOriginPatterns(ctx, patterns...)
	ctx.NoContent(modified, err)
}

func (c *CORSController) deleteOriginPatterns(ctx *context.Context) {
	patterns, ok := ctx.BindJSON[[]string]()
	if !ok {
		return
	}

	modified, err := c.cors.DeleteOriginPatterns(ctx, patterns...)
	ctx.NoContent(modified, err)
}
