package oauth2

import (
	stdContext "context"
	"strings"
)

type policyWrapper struct {
	policies []Policy
}

var _ Policy = &policyWrapper{}

// WrapPolicies wraps the given policies into a single policy.
// Example:
//
//	policies := append(oauth2.NewScopePolicies("users:read", "users:write"),
//		oauth2.ResourcePolicy{Resource: "orders", Action: "read"},
//	)
//	policy := oauth2.WrapPolicies(policies...)
//	app.Get("/protected", oauth2.AllowPolicies(policy), handler)
func WrapPolicies(policies ...Policy) Policy {
	return &policyWrapper{policies: policies}
}

// Matches checks if the given claims match any of the policies.
func (e *policyWrapper) Matches(ctx stdContext.Context, claims *ClientTokenClaims) bool {
	return matchPolicies(ctx, claims, e.policies)
}

// Policy is an interface that all policy types must implement.
type Policy interface {
	// String returns the scope as string representation of the policy.
	// String() string

	// Matches checks if the policy matches the given claims.
	Matches(ctx stdContext.Context, claims *ClientTokenClaims) bool
}

type policyScoper interface {
	Scopes() []string
}

// ScopePolicy represents a simple string-based scope policy.
type ScopePolicy struct {
	Scope string `json:"scope" yaml:"Scope"`
}

var _ Policy = ScopePolicy{}

// NewScopePolicies creates a new slice of string scope policies.
func NewScopePolicies(scopes ...string) []Policy {
	policies := make([]Policy, 0, len(scopes))
	for _, scope := range scopes {
		policies = append(policies, ScopePolicy{Scope: scope})
	}
	return policies
}

// String returns the scope as a string.
func (p ScopePolicy) String() string {
	return p.Scope
}

// Matches checks if the given scope matches the policy's scope.
func (p ScopePolicy) Matches(_ stdContext.Context, claims *ClientTokenClaims) bool {
	_, _, matched := matchScopes(claims.Scopes, p.Scope)
	return matched
}

func matchScopes(scopes []string, requiredScopes ...string) (string, string, bool) {
	if len(requiredScopes) == 0 {
		return "", "", true
	}

	for _, reqScope := range requiredScopes {
		for _, tokenScope := range scopes {
			if matchScope(tokenScope, reqScope) {
				return reqScope, tokenScope, true
			}
		}
	}

	return "", "", false
}

// matchScope checks if the tokenScope covers the requiredScope.
// Example wildcard scenarios:
//
//	tokenScope: "users:*", requiredScope: "users:read" => true
//	tokenScope: "users",   requiredScope: "users:write" => true
//	tokenScope: "users:*", requiredScope: "orders:read" => false
func matchScope(tokenScope, requiredScope string) bool {
	tokenParts := strings.SplitN(tokenScope, ":", 2)
	reqParts := strings.SplitN(requiredScope, ":", 2)

	// If the token scope has no ":", treat it as resource-level wildcard (e.g. "users").
	if len(tokenParts) == 1 {
		// Compare resource only. If resources differ => no match.
		if len(reqParts) == 1 {
			return tokenParts[0] == reqParts[0]
		}
		return tokenParts[0] == reqParts[0]
	}

	// If the required scope has no ":", treat it as resource only => token must match resource exactly,
	// ignoring action.
	if len(reqParts) == 1 {
		return tokenParts[0] == reqParts[0]
	}

	// Both are resource:action.
	// 1. Resources must match.
	if tokenParts[0] != reqParts[0] {
		return false
	}

	// 2. Check the action. If tokenScope's action is "*", it covers all actions for that resource.
	return tokenParts[1] == "*" || tokenParts[1] == reqParts[1]
}

// ResourcePolicy represents a fine-grained rule for a given resource/action.
// For example, Resource="users" and Action="read" => "users:read" scope.
//
// A policy of {Resource: "users", Action: "create read update delete"}}
// generates the following scopes:
// "users:create", "users:read", "users:update", "users:delete".
type ResourcePolicy struct {
	Resource string `json:"resource" yaml:"Resource"`
	Action   string `json:"action" yaml:"Action"` // more can be added by separating them with a space.
}

var _ Policy = ResourcePolicy{}

// Matches checks if a scope string matches the resource:action requirement.
// For instance, a scope "invoices:read" means resource="invoices", action="read".
// If more than one Action is provided (separated by space), at least one of them must match.
// If Actions is empty, then all scopes that match the resource are accepted.
func (rp ResourcePolicy) Matches(_ stdContext.Context, claims *ClientTokenClaims) bool {
	actions := SplitScopes(rp.Action)
	if len(actions) == 0 {
		// Against just the resource.
		_, _, matched := matchScopes(claims.Scopes, rp.Resource)
		return matched
	}

	for _, action := range actions {
		_, _, matched := matchScopes(claims.Scopes, rp.concatAction(action))
		if matched {
			return true
		}
	}

	return false
}

// String returns the resource:action scope string.
func (rp ResourcePolicy) String() string {
	return JoinScopes(rp.Scopes()...)
}

// Scopes returns the resource:action scope strings.
func (rp ResourcePolicy) Scopes() []string {
	actions := rp.Actions()

	if len(actions) == 0 {
		// users:*
		return []string{rp.concatAction("*")}
	}

	scopes := make([]string, 0, len(actions))
	for _, action := range actions {
		scope := rp.concatAction(action)
		scopes = append(scopes, scope)
	}

	// users:create users:read users:delete users:update
	return scopes
}

func (rp ResourcePolicy) concatAction(action string) string {
	return rp.Resource + ":" + action
}

// Actions returns just the actions.
func (rp ResourcePolicy) Actions() []string {
	return SplitScopes(rp.Action)
}

// PolicyScopes converts the given policies to scopes for app clients to be registered with.
func PolicyScopes(policies ...Policy) []string {
	scopes := make([]string, 0)
	for _, policy := range policies {
		if scoper, ok := policy.(policyScoper); ok {
			scopes = append(scopes, scoper.Scopes()...)
		}
	}

	return scopes
}

func matchPolicies(ctx stdContext.Context, claims *ClientTokenClaims, policies []Policy) bool {
	if len(policies) == 0 {
		return true
	}

	for _, policy := range policies {
		if policy.Matches(ctx, claims) {
			return true
		}
	}

	return false
}
