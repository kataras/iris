package identity

import (
	"github.com/kataras/iris/v14/auth/identity/entity"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/middleware/cache/httpcache"
)

// UserGroupController is the API controller for the UserGroup entity.
type UserGroupController struct {
	cache  *httpcache.Cache
	groups *UserGroupService
}

// NewUserGroupController creates and returns a new UserGroupController API.
func NewUserGroupController(cache *httpcache.Cache, groups *UserGroupService) *UserGroupController {
	return &UserGroupController{
		cache:  cache,
		groups: groups,
	}
}

// Routing registers the API routes.
// It implements the router.Controller interface.
func (c *UserGroupController) Routing(r router.Router) {
	// caching := c.cache.Handler()
	// invalidating := c.cache.InvalidateRoutes(r)
	// invalidating := c.cache.InvalidateHandler("httpcache:" + r.GetRelPath())
	caching, invalidating := c.cache.Pair(r)

	r.Get("/{user_group_id:string}", caching, c.getUserGroup)
	r.Delete("/{user_group_id:string}", invalidating, c.deleteUserGroup)
	r.Post("/", invalidating, c.upsertUserGroup)
	// QUERY (RFC 10008), and POST for SDKs built before it.
	r.Handle(router.MethodPost+" "+router.MethodQuery, "/list", caching, c.listUserGroups)
	r.Get("/count", caching, c.countUserGroups)
}

func getUserGroupID(ctx *context.Context) string {
	return ctx.Params().GetString("user_group_id")
}

func (c *UserGroupController) getUserGroup(ctx *context.Context) {
	groupID := getUserGroupID(ctx)

	group, err := c.groups.GetUserGroup(ctx, groupID)
	ctx.OK(group, err)
}

func (c *UserGroupController) deleteUserGroup(ctx *context.Context) {
	groupID := getUserGroupID(ctx)

	deleted, err := c.groups.DeleteUserGroup(ctx, groupID)
	ctx.NoContent(deleted, err)
}

func (c *UserGroupController) upsertUserGroup(ctx *context.Context) {
	group, ok := ctx.BindJSON[entity.UserGroup]()
	if !ok {
		return
	}

	id, created, err := c.groups.UpsertUserGroup(ctx, group)
	if created {
		ctx.Created(id, err)
		return
	}

	ctx.NoContent(true, err)
}

func (c *UserGroupController) listUserGroups(ctx *context.Context) {
	ctx.Paginated(c.groups.ListUserGroupsPaginated)
}

type CountUserGroupsResponse struct {
	TotalCount int64 `json:"total_count"`
}

func (c *UserGroupController) countUserGroups(ctx *context.Context) {
	count, err := c.groups.CountUserGroups(ctx)
	ctx.OK(CountUserGroupsResponse{count}, err)
}
