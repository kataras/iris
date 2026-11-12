package identity

import (
	"context"
	"fmt"

	"github.com/kataras/iris/v14/auth/identity/entity"
)

type UserGroupService struct {
	r *entity.Registry
}

func NewUserGroupService(r *entity.Registry) *UserGroupService {
	return &UserGroupService{
		r: r,
	}
}

func (s *UserGroupService) InTransaction(ctx context.Context, fn func(*UserGroupService) error) error {
	if s.r.IsInTransaction() { // do not initialize a new service instance if not necessary.
		return fn(s)
	}

	return s.r.InTransaction(ctx, func(r *entity.Registry) error {
		newService := NewUserGroupService(r)
		return fn(newService)
	})
}

func (s *UserGroupService) GetUserGroup(ctx context.Context, groupID string) (entity.UserGroup, error) {
	return s.r.UserGroups().GetUserGroupByID(ctx, groupID)
}

func (s *UserGroupService) DeleteUserGroup(ctx context.Context, groupID string) (bool, error) {
	return s.r.UserGroups().DeleteUserGroupByID(ctx, groupID)
}

func (s *UserGroupService) UpsertUserGroup(ctx context.Context, group entity.UserGroup) (string, bool, error) {
	if group.Name == "" {
		return "", false, fmt.Errorf("%w: user group name cannot be empty", entity.ErrInvalidArgument)
	}

	if len(group.Filter.Terms) == 0 {
		return "", false, fmt.Errorf("%w: user group filter cannot be empty", entity.ErrInvalidArgument)
	}

	return s.r.UserGroups().UpsertUserGroup(ctx, group)
}

func (s *UserGroupService) ListUserGroupsPaginated(ctx context.Context, l entity.PageOptions, f *entity.UserGroupFilterOptions) ([]entity.UserGroup, int64, error) {
	if l.Page < 0 || l.Size < 0 {
		return nil, 0, fmt.Errorf("%w: page and size must be non-negative", entity.ErrInvalidArgument)
	}

	return s.r.UserGroups().ListUserGroupsPaginated(ctx, l, f)
}

func (s *UserGroupService) CountUserGroups(ctx context.Context) (int64, error) {
	return s.r.UserGroups().CountUserGroups(ctx, &entity.UserGroupFilterOptions{})
}
