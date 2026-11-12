package identity

import (
	"github.com/kataras/iris/v14/auth/identity/entity"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
)

type BillingController struct {
	metrics *entity.MetricRepository
	cache   *httpcache.Cache
}

func NewBillingController(r *entity.Registry, cache *httpcache.Cache) *BillingController {
	return &BillingController{metrics: r.Metrics(), cache: cache}
}

func (c *BillingController) Routing(r router.Router) {
	// No invalidation, the response is cached, and refresh only every cache ttl (10m).
	caching := c.cache.Handler()
	r.Get("/", caching, c.getBilling)
}

func (c *BillingController) getBilling(ctx *context.Context) {
	billing, err := c.metrics.GetBilling(ctx)
	ctx.OK(billing, err)
}
