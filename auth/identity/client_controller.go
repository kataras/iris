package identity

import (
	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/rest/errors"
)

// ClientController serves the client application routes under /clients:
// the self routes of the calling client and the client management ones.
type ClientController struct {
	clients      *entity.ClientRepository
	oauth2Server *oauth2.Server[entity.FlatUser]
	signer       *ClientTokenSigner
}

// NewClientController returns a client controller.
// The oauth2 server and the signer serve the client management routes.
func NewClientController(r *entity.Registry, oauth2Server *oauth2.Server[entity.FlatUser], signer *ClientTokenSigner) *ClientController {
	return &ClientController{
		clients:      r.Clients(),
		oauth2Server: oauth2Server,
		signer:       signer,
	}
}

func (c *ClientController) Routing(r router.Router) {
	r.Get("/", c.getClient)
	r.Post("/token-max-age", c.updateClientTokenMaxAge)
	r.Post("/restore-token-max-age", c.restoreClientTokenMaxAge)
	// Endpoint to revoke client and user tokens for the client.
	r.Post("/revoke-tokens", c.revokeTokens)

	// Client management, each route guarded by its own policy (see client_admin_controller.go).
	r.Post("/", oauth2.AllowPolicies(CreateClientsPolicy), c.createClient)
	// QUERY (RFC 10008), and POST for SDKs built before it.
	r.Handle(router.MethodPost+" "+router.MethodQuery, "/list", oauth2.AllowPolicies(ReadClientsPolicy), c.listClients)
	r.Get("/count", oauth2.AllowPolicies(ReadClientsPolicy), c.countClients)
	r.Get("/{id:uuid}", oauth2.AllowPolicies(ReadClientsPolicy), c.getClientByID)
	r.Delete("/{id:uuid}", oauth2.AllowPolicies(DeleteClientsPolicy), c.deleteClient)
	r.Post("/{id:uuid}/token", oauth2.AllowPolicies(UpdateClientsPolicy), c.issueClientToken)
	r.Post("/{id:uuid}/revoke-tokens", oauth2.AllowPolicies(UpdateClientsPolicy), c.revokeClientTokens)
}

func getClient(ctx *context.Context) (*oauth2.Client, bool) {
	client, ok := oauth2.GetClient(ctx)
	if !ok {
		errors.PermissionDenied.Message(ctx, "client not found or not authenticated")
		return nil, false
	}

	return client, true
}

func (c *ClientController) getClient(ctx *context.Context) {
	client, ok := getClient(ctx)
	if !ok {
		return
	}

	ctx.OK(client, nil)
}

type UpdateClientTokenMaxAgeRequest struct {
	AccessMaxAge  int `json:"access_max_age"`
	RefreshMaxAge int `json:"refresh_max_age"`
}

func (c *ClientController) updateClientTokenMaxAge(ctx *context.Context) {
	client, ok := getClient(ctx)
	if !ok {
		return
	}

	req, ok := ctx.BindJSON[UpdateClientTokenMaxAgeRequest]()
	if !ok {
		return
	}

	modified, err := c.clients.UpdateTokenMaxAge(ctx, client.ID, req.AccessMaxAge, req.RefreshMaxAge)
	ctx.NoContent(modified, err)
}

func (c *ClientController) restoreClientTokenMaxAge(ctx *context.Context) {
	client, ok := getClient(ctx)
	if !ok {
		return
	}

	modified, err := c.clients.RestoreTokenMaxAge(ctx, client.ID)
	ctx.NoContent(modified, err)
}

func (c *ClientController) revokeTokens(ctx *context.Context) {
	client, ok := getClient(ctx)
	if !ok {
		return
	}

	modified, err := c.clients.IncrementClientVersion(ctx, client.ID)
	ctx.NoContent(modified, err)
}
