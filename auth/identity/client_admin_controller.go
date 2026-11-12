package identity

import (
	stdContext "context"
	"fmt"
	"slices"

	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

// CreateClientRequest is the request body of POST /clients.
type CreateClientRequest struct {
	Name    string     `json:"name"`
	Website string     `json:"website"`
	LogoURL string     `json:"logo_url"`
	Kind    ClientKind `json:"kind"`
	// RedirectURI is required for the user kind.
	RedirectURI string `json:"redirect_uri,omitempty"`
	// VerificationURI adds the device code grant to a user client.
	VerificationURI string `json:"verification_uri,omitempty"`
	// The max ages are in seconds. Zero means the default of the kind.
	CodeMaxAge    int `json:"code_max_age,omitempty"`
	AccessMaxAge  int `json:"access_max_age,omitempty"`
	RefreshMaxAge int `json:"refresh_max_age,omitempty"`
}

// Validate reports the missing required fields. BindJSON calls it and answers 400 on error.
func (r *CreateClientRequest) Validate() error {
	if r.Name == "" || r.Website == "" || r.LogoURL == "" || r.Kind == "" {
		return fmt.Errorf("%w: name, website, logo_url and kind are required", entity.ErrInvalidArgument)
	}

	if r.Kind == ClientKindUser && r.RedirectURI == "" {
		return fmt.Errorf("%w: redirect_uri is required for a user client", entity.ErrInvalidArgument)
	}

	return nil
}

// CreateClientResponse is the response body of POST /clients:
// the credentials of the new client and the client itself, secret included.
type CreateClientResponse struct {
	ClientCredentials
	Client *oauth2.Client `json:"client"`
}

// CountClientsResponse is the response body of GET /clients/count.
type CountClientsResponse struct {
	TotalCount int64 `json:"total_count"`
}

func (c *ClientController) createClient(ctx *context.Context) {
	req, ok := ctx.BindJSON[CreateClientRequest]()
	if !ok {
		return
	}

	spec, err := specForKind(req.Kind, req.VerificationURI)
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}

	codeMaxAge := withDefault(req.CodeMaxAge, spec.CodeMaxAge)
	accessMaxAge := withDefault(req.AccessMaxAge, spec.AccessMaxAge)
	refreshMaxAge := withDefault(req.RefreshMaxAge, spec.RefreshMaxAge)

	reg := oauth2.ClientRegistration{
		Name:            req.Name,
		Website:         req.Website,
		LogoURL:         req.LogoURL,
		RedirectURI:     req.RedirectURI,
		VerificationURI: req.VerificationURI,
		GrantTypes:      spec.GrantTypes,
		Scopes:          spec.Scopes,
		CodeMaxAge:      codeMaxAge,
		AccessMaxAge:    accessMaxAge,
		RefreshMaxAge:   refreshMaxAge,
	}

	// CreateClient returns the validation errors of oauth2.NewClient as plain errors,
	// next to the store ones. Run the same validation first so a bad request maps to 400.
	if _, err = oauth2.NewClient(reg); err != nil {
		errors.HandleError(ctx, fmt.Errorf("%w: %w", entity.ErrInvalidArgument, err))
		return
	}

	client, err := c.oauth2Server.CreateClient(ctx, reg)
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}

	creds, err := c.signer.Credentials(client)
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}

	ctx.Created(CreateClientResponse{ClientCredentials: *creds, Client: client}, nil)
}

// withDefault returns def when value is zero.
func withDefault(value, def int) int {
	if value == 0 {
		return def
	}
	return value
}

func (c *ClientController) listClients(ctx *context.Context) {
	ctx.Paginated(func(ctx stdContext.Context, page context.PageOptions, filter *oauth2.ClientFilter) ([]*oauth2.Client, int64, error) {
		clients, total, err := c.oauth2Server.ClientStore().ListClients(ctx, page, filter)
		if err != nil {
			return nil, 0, err
		}

		// The store returns detached copies, so blanking the secret does not touch the stored client.
		for _, client := range clients {
			client.Secret = ""
		}
		return clients, total, nil
	})
}

func (c *ClientController) countClients(ctx *context.Context) {
	total, err := c.oauth2Server.ClientStore().CountClients(ctx, nil)
	ctx.OK(CountClientsResponse{TotalCount: total}, err)
}

// findClient fetches the client of the {id} path parameter.
// It writes the error response and reports false when there is no such client.
func (c *ClientController) findClient(ctx *context.Context) (*oauth2.Client, bool) {
	id := ctx.Params().GetString("id")
	client, found, err := c.oauth2Server.ClientStore().GetClient(ctx, id)
	if err != nil {
		errors.HandleError(ctx, err)
		return nil, false
	}
	if !found {
		errors.HandleError(ctx, oauth2.ErrClientNotFound)
		return nil, false
	}

	return client, true
}

func (c *ClientController) getClientByID(ctx *context.Context) {
	client, ok := c.findClient(ctx)
	if !ok {
		return
	}

	ctx.OK(client, nil)
}

func (c *ClientController) deleteClient(ctx *context.Context) {
	caller, ok := getClient(ctx)
	if !ok {
		return
	}

	target, ok := c.findClient(ctx)
	if !ok {
		return
	}

	if target.ID == caller.ID {
		errors.FailedPrecondition.Message(ctx, "a client cannot delete itself")
		return
	}

	// Only an admin client that can create clients counts towards the last one.
	if KindOfClient(target) == ClientKindAdmin && slices.Contains(target.Scopes, createClientsScope) {
		admins, err := c.oauth2Server.ClientStore().CountClients(ctx, &oauth2.ClientFilter{Scope: createClientsScope})
		if err != nil {
			errors.HandleError(ctx, err)
			return
		}
		if admins <= 1 {
			errors.FailedPrecondition.Message(ctx, "cannot delete the last client that can create clients")
			return
		}
	}

	deleted, err := c.oauth2Server.ClientStore().DeleteClient(ctx, target.ID)
	ctx.NoContent(deleted, err)
}

// createClientsScope is the scope of CreateClientsPolicy.
// The last client that holds it cannot be deleted, so client management is never locked out.
var createClientsScope = oauth2.PolicyScopes(CreateClientsPolicy)[0]

func (c *ClientController) issueClientToken(ctx *context.Context) {
	client, ok := c.findClient(ctx)
	if !ok {
		return
	}

	creds, err := c.signer.Credentials(client)
	ctx.OK(creds, err)
}

func (c *ClientController) revokeClientTokens(ctx *context.Context) {
	id := ctx.Params().GetString("id")
	modified, err := c.oauth2Server.ClientStore().IncrementClientVersion(ctx, id)
	if err == nil && !modified {
		err = oauth2.ErrClientNotFound
	}

	ctx.NoContent(modified, err)
}
