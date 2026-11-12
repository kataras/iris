package identity

import (
	"slices"

	"github.com/kataras/iris/v14/auth/oauth2"
)

var (
	// ReadUserPolicy is the policy that allows reading the self-user data.
	ReadUserPolicy = oauth2.ResourcePolicy{Resource: "user", Action: "read"}

	// OIDC Scope Policies
	// OpenIDPolicy  = oauth2.ScopePolicy{Scope: ScopeOpenID}
	// ProfilePolicy = oauth2.ScopePolicy{Scope: ScopeProfile}
	// EmailPolicy   = oauth2.ScopePolicy{Scope: ScopeEmail}
	// AddressPolicy = oauth2.ScopePolicy{Scope: ScopeAddress}
	// PhonePolicy   = oauth2.ScopePolicy{Scope: ScopePhone}

	// UserAppPolicies are the policies that the app user client can use.
	// Now includes OIDC standard scopes
	UserAppPolicies = []oauth2.Policy{
		ReadUserPolicy,
		// OpenIDPolicy,
		// ProfilePolicy,
		// EmailPolicy,
		// AddressPolicy,
		// PhonePolicy,
	}

	// UserAppScopes are the scopes that the app user client can use.
	UserAppScopes = oauth2.PolicyScopes(UserAppPolicies...)
)

var (
	// CreateUsersPolicy is the policy that allows creating new users.
	CreateUsersPolicy = oauth2.ResourcePolicy{Resource: "users", Action: "create"}
	// ReadUsersPolicy is the policy that allows reading one or more users.
	ReadUsersPolicy = oauth2.ResourcePolicy{Resource: "users", Action: "read"}
	// UpdateUsersPolicy is the policy that allows updating one or more user data.
	UpdateUsersPolicy = oauth2.ResourcePolicy{Resource: "users", Action: "update"}
	// DeleteUsersPolicy is the policy that allows deleting one or more users.
	DeleteUsersPolicy = oauth2.ResourcePolicy{Resource: "users", Action: "delete"}
	// BillingPolicy is the policy that allows reading the billing data.
	BillingPolicy = oauth2.ResourcePolicy{Resource: "billing", Action: "read"}

	// ServerOnlyPolicies gate /u, /u/groups, /cors, /billing and /cache.
	// AllowPolicies matches any one of them. A user client holds user:read, which is not here.
	ServerOnlyPolicies = []oauth2.Policy{
		CreateUsersPolicy,
		ReadUsersPolicy,
		UpdateUsersPolicy,
		DeleteUsersPolicy,
		BillingPolicy,
	}

	// CreateClientsPolicy is the policy that allows creating new clients.
	CreateClientsPolicy = oauth2.ResourcePolicy{Resource: "clients", Action: "create"}
	// ReadClientsPolicy is the policy that allows reading one or more clients.
	ReadClientsPolicy = oauth2.ResourcePolicy{Resource: "clients", Action: "read"}
	// UpdateClientsPolicy is the policy that allows updating one or more clients.
	UpdateClientsPolicy = oauth2.ResourcePolicy{Resource: "clients", Action: "update"}
	// DeleteClientsPolicy is the policy that allows deleting one or more clients.
	DeleteClientsPolicy = oauth2.ResourcePolicy{Resource: "clients", Action: "delete"}

	// ServerAppPolicies are the policies that the app server client can use, including the user client app's ones.
	ServerAppPolicies = slices.Concat(ServerOnlyPolicies, UserAppPolicies)
)

var (
	// ManageClientsPolicies are the policies that allow managing the client applications.
	ManageClientsPolicies = []oauth2.Policy{
		CreateClientsPolicy,
		ReadClientsPolicy,
		UpdateClientsPolicy,
		DeleteClientsPolicy,
	}

	// ManageClientsScopes are the scopes of the ManageClientsPolicies.
	ManageClientsScopes = oauth2.PolicyScopes(ManageClientsPolicies...)

	// AdminAppPolicies are the policies that the admin app client can use:
	// the server ones plus the client management ones.
	// slices.Concat allocates a new slice, so it never aliases the backing array of ServerAppPolicies.
	AdminAppPolicies = slices.Concat(ServerAppPolicies, ManageClientsPolicies)
)

// AllPolicies contains all policies.
// Note: to invalidate all previous client tokens,
// just change its client secret value on database and on the client-side.
var AllPolicies = slices.Concat(AdminAppPolicies, UserAppPolicies)
