package opa

import (
	"context"
	"fmt"

	"github.com/kataras/iris/v14/auth/oauth2"
)

// externalPolicy represents a policy evaluated by an external policy engine.
type externalPolicy struct {
	*Client

	Resource string
	Action   string
	Context  map[string]string
}

var _ oauth2.Policy = (*externalPolicy)(nil)

func (ep *externalPolicy) String() string {
	return fmt.Sprintf("%s:%s", ep.Resource, ep.Action)
}

func (ep *externalPolicy) Matches(ctx context.Context, claims *oauth2.ClientTokenClaims) bool {
	allowed, err := ep.Client.CheckPolicy(ctx, PolicyRequest{
		ClientID: claims.ClientID,
		Resource: ep.Resource,
		Action:   ep.Action,
		Scopes:   claims.Scopes,
		Context:  ep.Context,
	})
	if err != nil {
		return false
	}
	return allowed
}
