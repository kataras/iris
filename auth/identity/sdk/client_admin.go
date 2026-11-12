package sdk

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/kataras/iris/v14/auth/identity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/client"
	irisContext "github.com/kataras/iris/v14/context"
)

// ErrClientIDRequired is returned by the client management methods that take a client id when the id is empty.
// An empty id would collapse the path into a different route, e.g. /clients//revoke-tokens into the caller's own /clients/revoke-tokens.
var ErrClientIDRequired = errors.New("identity sdk: client id is required")

// AdminCreateClient fires a POST request to the identity server to create a client application.
// The response carries the credentials of the new client, including its token and,
// for server and admin clients, the encryption key. They are shown once, store them.
func (s *SDK[T]) AdminCreateClient(ctx context.Context, req identity.CreateClientRequest) (*identity.CreateClientResponse, error) {
	resp, err := s.Client.BindJSON[identity.CreateClientResponse](ctx, http.MethodPost, "/clients", req, s.adminOption)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// AdminListClients fires a POST request to the identity server to list the client applications with pagination.
// The server also accepts QUERY (RFC 10008), but proxies such as the Azure Container Apps ingress reject it, so the SDK sends POST.
// A nil filter lists every client. The secrets of the listed clients are blanked by the server.
func (s *SDK[T]) AdminListClients(ctx context.Context, page irisContext.PageOptions, filter *oauth2.ClientFilter) (*irisContext.Page[oauth2.Client], error) {
	if filter == nil {
		filter = &oauth2.ClientFilter{}
	}

	resp, err := s.Client.BindJSON[irisContext.Page[oauth2.Client]](ctx, http.MethodPost, "/clients/list", filter, s.adminOption, page.Bind)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// AdminCountClients fires a GET request to the identity server to count the client applications.
func (s *SDK[T]) AdminCountClients(ctx context.Context) (int64, error) {
	resp, err := s.Client.BindJSON[identity.CountClientsResponse](ctx, http.MethodGet, "/clients/count", nil, s.adminOption)
	if err != nil {
		return 0, err
	}

	return resp.TotalCount, nil
}

// AdminGetClient fires a GET request to the identity server to retrieve a client application by its ID.
func (s *SDK[T]) AdminGetClient(ctx context.Context, id string) (*oauth2.Client, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrClientIDRequired
	}

	resp, err := s.Client.BindJSON[oauth2.Client](ctx, http.MethodGet, clientPath(id), nil, s.adminOption)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// AdminDeleteClient fires a DELETE request to the identity server to delete a client application by its ID.
// It reports false when nothing was deleted.
func (s *SDK[T]) AdminDeleteClient(ctx context.Context, id string) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, ErrClientIDRequired
	}

	return s.clientNoContent(ctx, http.MethodDelete, clientPath(id))
}

// AdminIssueClientToken fires a POST request to the identity server to issue a new token for a client application.
func (s *SDK[T]) AdminIssueClientToken(ctx context.Context, id string) (*identity.ClientCredentials, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrClientIDRequired
	}

	resp, err := s.Client.BindJSON[identity.ClientCredentials](ctx, http.MethodPost, clientPath(id)+"/token", nil, s.adminOption)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// AdminRevokeClientTokens fires a POST request to the identity server to revoke every token issued for a client application.
// It returns true on 204 and an error carrying status 404 for an unknown client.
func (s *SDK[T]) AdminRevokeClientTokens(ctx context.Context, id string) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, ErrClientIDRequired
	}

	return s.clientNoContent(ctx, http.MethodPost, clientPath(id)+"/revoke-tokens")
}

func clientPath(id string) string {
	return "/clients/" + url.PathEscape(id)
}

// clientNoContent sends a bodiless admin request and maps 204 to true and 304 to false.
// Any status from 400 up is returned as a client.APIError.
func (s *SDK[T]) clientNoContent(ctx context.Context, method, path string) (bool, error) {
	resp, err := s.Client.Do(ctx, method, path, nil, s.adminOption)
	if err != nil {
		return false, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return false, client.ExtractError(resp)
	}
	defer s.Client.DrainResponseBody(resp)

	return resp.StatusCode == http.StatusNoContent, nil
}
