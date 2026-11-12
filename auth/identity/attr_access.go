package identity

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

const (
	// AttrAccessSelf is a profile field a client with users:update may change.
	AttrAccessSelf = "self"
	// AttrAccessPrivileged is written by an admin client, or by a server client
	// whose scopes include users:attr:<name>.
	AttrAccessPrivileged = "privileged"
)

// AttrScope is the scope a server client needs to write a privileged attribute.
func AttrScope(name string) string {
	return "users:attr:" + name
}

func normalizeAttrAccess(access string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(access)) {
	case "", AttrAccessSelf:
		return AttrAccessSelf, nil
	case AttrAccessPrivileged:
		return AttrAccessPrivileged, nil
	default:
		return "", fmt.Errorf("%w: access %q, want %q or %q", entity.ErrInvalidArgument, access, AttrAccessSelf, AttrAccessPrivileged)
	}
}

func (s *UserSchema) markPrivileged(field string) {
	if s.privilegedFields == nil {
		s.privilegedFields = make(map[string]struct{})
	}
	s.privilegedFields[field] = struct{}{}
}

// IsPrivileged reports whether name is a privileged attribute.
func (s *UserSchema) IsPrivileged(name string) bool {
	if s == nil || s.privilegedFields == nil {
		return false
	}
	_, ok := s.privilegedFields[name]
	return ok
}

// clientCanWritePrivilegedAttr reports whether client may write the privileged attribute name.
// An admin client may. A server client may when it holds users:attr:<name>.
func clientCanWritePrivilegedAttr(client *oauth2.Client, name string) bool {
	if client == nil {
		return false
	}
	if KindOfClient(client) == ClientKindAdmin {
		return true
	}
	for _, scope := range client.Scopes {
		if scope == AttrScope(name) {
			return true
		}
	}
	return false
}

// attrRoot returns the top-level attribute of a dotted path.
func attrRoot(path string) string {
	if i := strings.IndexByte(path, '.'); i >= 0 {
		return path[:i]
	}
	return path
}

func attrNames(attrs json.RawMessage) ([]string, error) {
	if len(attrs) == 0 || string(attrs) == "null" {
		return nil, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(attrs, &fields); err != nil {
		return nil, fmt.Errorf("%w: attrs: %w", entity.ErrInvalidArgument, err)
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	return names, nil
}

// denyPrivilegedWrite answers 403 when names include a privileged attribute
// the client cannot write. It answers 400 when attrs cannot be read.
// It returns true when the response is already written.
func (c *UserController) denyPrivilegedWrite(ctx *context.Context, client *oauth2.Client, names []string) bool {
	if c.userSchema == nil {
		return false
	}
	for _, name := range names {
		root := attrRoot(name)
		if !c.userSchema.IsPrivileged(root) {
			continue
		}
		if clientCanWritePrivilegedAttr(client, root) {
			continue
		}
		errors.PermissionDenied.Detail(ctx,
			fmt.Sprintf("field %q is privileged", root),
			"an admin client, or a client with scope "+AttrScope(root)+", can write it",
		)
		return true
	}
	return false
}

// denyPrivilegedAttrs is denyPrivilegedWrite for a stored attrs object.
func (c *UserController) denyPrivilegedAttrs(ctx *context.Context, client *oauth2.Client, attrs json.RawMessage) bool {
	names, err := attrNames(attrs)
	if err != nil {
		errors.InvalidArgument.Err(ctx, err)
		return true
	}
	return c.denyPrivilegedWrite(ctx, client, names)
}

// denyPrivilegedClaims answers 403 when extra token claims name a privileged attribute.
// Those claims are issued from the stored user. Enrich does not copy them.
func (c *UserController) denyPrivilegedClaims(ctx *context.Context, names []string) bool {
	if c.userSchema == nil {
		return false
	}
	for _, name := range names {
		root := attrRoot(name)
		if !c.userSchema.IsPrivileged(root) {
			continue
		}
		errors.PermissionDenied.Detail(ctx,
			fmt.Sprintf("field %q is privileged", root),
			"privileged attributes are issued from the stored user and cannot be set as extra claims",
		)
		return true
	}
	return false
}
