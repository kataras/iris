package identity

import (
	"testing"

	"github.com/kataras/iris/v14/auth/oauth2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrivilegedAttrAccess(t *testing.T) {
	schema, err := parseUserSchema([]*UserAttribute{
		{Name: "firstname", Type: "text"},
		{Name: "role", Type: "number", Required: true, Access: "privileged"},
	})
	require.NoError(t, err)
	assert.Equal(t, AttrAccessPrivileged, schema.Attrs[1].Access)
	assert.True(t, schema.IsPrivileged("role"))
	assert.False(t, schema.IsPrivileged("firstname"))
	assert.Equal(t, "users:attr:role", AttrScope("role"))

	admin := &oauth2.Client{
		GrantTypes: []string{oauth2.GrantTypePassword, oauth2.GrantTypeRefreshToken},
		Scopes:     oauth2.PolicyScopes(AdminAppPolicies...),
	}
	server := &oauth2.Client{
		GrantTypes: []string{oauth2.GrantTypePassword, oauth2.GrantTypeRefreshToken},
		Scopes:     oauth2.PolicyScopes(ServerAppPolicies...),
	}
	granted := &oauth2.Client{
		GrantTypes: server.GrantTypes,
		Scopes:     append(append([]string{}, server.Scopes...), AttrScope("role")),
	}

	assert.True(t, clientCanWritePrivilegedAttr(admin, "role"))
	assert.False(t, clientCanWritePrivilegedAttr(server, "role"))
	assert.True(t, clientCanWritePrivilegedAttr(granted, "role"))
	assert.False(t, clientCanWritePrivilegedAttr(nil, "role"))

	_, err = parseUserSchema([]*UserAttribute{{Name: "role", Type: "number", Access: "owner"}})
	assert.Error(t, err)

	// An empty access is the self default.
	selfSchema, err := parseUserSchema([]*UserAttribute{{Name: "phone", Type: "text"}})
	require.NoError(t, err)
	assert.Equal(t, AttrAccessSelf, selfSchema.Attrs[0].Access)
	assert.False(t, selfSchema.IsPrivileged("phone"))
}
