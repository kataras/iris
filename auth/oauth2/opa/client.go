// Open Policy Agent (OPA) is an open-source, general-purpose policy engine
// that enables unified, context-aware policy enforcement across the stack.
// OPA moves the policy decision out of the application logic, so one policy can be
// written once and enforced the same way everywhere.
//
// Create a policy file (e.g., policy.rego) that defines the rules for access control.
//
//	package authz
//
//	default allow = false
//
//	allow {
//	    input.client_id == "example-client"
//	    input.resource == "users"
//	    input.action == "read"
//	}
//
//	allow {
//	    input.client_id == "example-client"
//	    input.resource == "orders"
//	    input.action == "write"
//	}
//
// Start OPA with the policy file.
// $ opa run --server --addr :8181 policy.rego
//
// Create an OPA client and a policy.
//
//	client := opa.NewClient("http://localhost:8181/v1/data/authz/allow")
//	policy := client.NewPolicy("users", "read")
//	app.Get("/protected", oauth2.AllowPolicies(policy), handler)
package opa

import (
	"context"
	"maps"

	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/client"
)

// Client is an OPA client.
//
// Example:
//
//	client := opa.NewClient("http://localhost:8181/v1/data/authz/allow")
//	policy := client.NewPolicy("users", "read")
//	app.Get("/protected", oauth2.AllowPolicies(policy), handler)
type Client struct {
	*client.Client
}

// NewClient creates a new OPA client.
// The URL should point to the OPA policy endpoint.
// Example:
//
//	NewClient("http://localhost:8181/v1/data/authz/allow")
func NewClient(url string, opts ...client.Option) *Client {
	options := append([]client.Option{client.BaseURL(url)}, opts...)
	return &Client{client.New(options...)}
}

type (
	// PolicyRequest represents the input data sent to OPA for policy evaluation.
	//
	// In the context of the PolicyRequest sent to OPA,
	// the Scopes field represents the scopes that the client has been granted and is trying to use.
	// The Action and Resource fields specify what the client is attempting to do and on which resource.
	PolicyRequest struct {
		// ClientID is the ID of the OAuth2 client making the request.
		ClientID string `json:"client_id" yaml:"ClientID"`
		// Resource is the resource being accessed (e.g., "users", "orders").
		Resource string `json:"resource" yaml:"Resource"`
		// Action is the action being performed (e.g., "read", "write").
		Action string `json:"action" yaml:"Action"`
		// Scopes is the scopes associated with the client's token. These are the scopes that the client has been granted and is trying to use.
		Scopes []string `json:"scopes" yaml:"Scopes"`
		// Context is the additional context that might be relevant for the policy decision (optional).
		Context map[string]string `json:"context,omitempty" yaml:"Context,omitempty"`
	}

	// PolicyResponse represents the response from OPA.
	PolicyResponse struct {
		Result bool `json:"result" yaml:"Result"`
	}
)

// NewPolicy creates a new policy for the resource and action.
// Example:
//
//	policy := client.NewPolicy("resource", "read", nil)
//	if policy.Matches(ctx, claims) { // claims is an *oauth2.ClientTokenClaims.
//	    // Allow
//	}
func (c *Client) NewPolicy(resource, action string, optionalDatas ...map[string]string) oauth2.Policy {
	var contextData map[string]string

	for _, optionalData := range optionalDatas {
		if optionalData == nil {
			continue
		}

		if contextData == nil {
			contextData = make(map[string]string)
		}

		maps.Copy(contextData, optionalData)
	}

	return &externalPolicy{
		Client:   c,
		Action:   action,
		Resource: resource,
		Context:  contextData,
	}
}

// CheckPolicy queries OPA to evaluate the policy for the given client, action, and resource.
// Example:
//
//	allow, err := client.CheckPolicy(context.Background(), PolicyRequest{
//	    ClientID: "clientID",
//	    Resource: "resource",
//	    Action:   "read",
//	    Scopes:   []string{"scope1", "scope2"},
//	    Context:  map[string]string{"key": "value"},
//	})
func (c *Client) CheckPolicy(ctx context.Context, req PolicyRequest) (bool, error) {
	policyResp, err := c.Client.BindJSON[PolicyResponse](ctx, "POST", "", req)
	if err != nil {
		return false, err
	}

	return policyResp.Result, nil
}
