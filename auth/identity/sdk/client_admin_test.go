package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14/auth/identity"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/client"
	irisContext "github.com/kataras/iris/v14/context"
)

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Token  string
	Body   []byte
	Count  int
}

// newClientAdminSDK returns an SDK wired to a test server that records the
// last request and answers with the given status and JSON body.
func newClientAdminSDK(t *testing.T, status int, response string) (*SDK[map[string]any], *recordedRequest) {
	t.Helper()

	rec := &recordedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.Count++
		rec.Method = r.Method
		rec.Path = r.URL.EscapedPath()
		rec.Query = r.URL.RawQuery
		rec.Token = r.Header.Get(Header)
		rec.Body, _ = io.ReadAll(r.Body)

		if response != "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		if response != "" {
			_, _ = w.Write([]byte(response))
		}
	}))
	t.Cleanup(srv.Close)

	return &SDK[map[string]any]{
		Client:      client.New(client.BaseURL(srv.URL)),
		adminOption: client.RequestHeader(true, Header, "test-token"),
	}, rec
}

func TestAdminCreateClient(t *testing.T) {
	s, rec := newClientAdminSDK(t, http.StatusCreated, `{"audience":"aud-1","kind":"server","token":"tok","encryption_key":"key","client":{"id":"aud-1","name":"My App"}}`)

	resp, err := s.AdminCreateClient(context.Background(), identity.CreateClientRequest{Name: "My App", Kind: identity.ClientKind("server")})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/clients" || rec.Token != "test-token" {
		t.Fatalf("unexpected request: %+v", rec)
	}
	var sent identity.CreateClientRequest
	if err = json.Unmarshal(rec.Body, &sent); err != nil || sent.Name != "My App" {
		t.Fatalf("unexpected body %q: %v", rec.Body, err)
	}
	if resp.Audience != "aud-1" || resp.Token != "tok" || resp.EncryptionKey != "key" || resp.Client == nil || resp.Client.Name != "My App" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAdminListClients(t *testing.T) {
	s, rec := newClientAdminSDK(t, http.StatusOK, `{"page":1,"size":5,"total_items":1,"items":[{"id":"c1","name":"Billing"}]}`)

	page, err := s.AdminListClients(context.Background(), irisContext.PageOptions{Page: 1, Size: 5}, &oauth2.ClientFilter{Name: "bill"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/clients/list" || rec.Token != "test-token" {
		t.Fatalf("unexpected request: %+v", rec)
	}
	if rec.Query == "" {
		t.Fatal("expected page options in the query string")
	}
	var sent oauth2.ClientFilter
	if err = json.Unmarshal(rec.Body, &sent); err != nil || sent.Name != "bill" {
		t.Fatalf("unexpected body %q: %v", rec.Body, err)
	}
	if page.TotalItems != 1 {
		t.Fatalf("unexpected total items: %+v", page)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "Billing" {
		t.Fatalf("unexpected page: %+v", page)
	}

	if _, err = s.AdminListClients(context.Background(), irisContext.PageOptions{Page: 1, Size: 5}, nil); err != nil {
		t.Fatal(err)
	}
	if string(rec.Body) != "{}" {
		t.Fatalf("nil filter should send an empty object, got %q", rec.Body)
	}
}

func TestAdminCountClients(t *testing.T) {
	s, rec := newClientAdminSDK(t, http.StatusOK, `{"total_count":42}`)

	n, err := s.AdminCountClients(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/clients/count" || rec.Token != "test-token" || n != 42 {
		t.Fatalf("unexpected request %+v or count %d", rec, n)
	}
}

func TestAdminGetClient(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		s, rec := newClientAdminSDK(t, http.StatusOK, `{"id":"a b","name":"Spaced"}`)

		c, err := s.AdminGetClient(context.Background(), "a b")
		if err != nil {
			t.Fatal(err)
		}
		if rec.Method != http.MethodGet || rec.Path != "/clients/a%20b" || rec.Token != "test-token" {
			t.Fatalf("unexpected request: %+v", rec)
		}
		if c.Name != "Spaced" {
			t.Fatalf("unexpected client: %+v", c)
		}
	})

	t.Run("not found", func(t *testing.T) {
		s, _ := newClientAdminSDK(t, http.StatusNotFound, `{"message":"not found"}`)

		c, err := s.AdminGetClient(context.Background(), "missing")
		if err == nil {
			t.Fatalf("expected an error, got client %+v", c)
		}
	})
}

func TestAdminDeleteClient(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   bool
	}{
		{http.StatusNoContent, true},
		{http.StatusNotModified, false},
	} {
		s, rec := newClientAdminSDK(t, tc.status, "")

		got, err := s.AdminDeleteClient(context.Background(), "c1")
		if err != nil {
			t.Fatalf("status %d: %v", tc.status, err)
		}
		if got != tc.want {
			t.Fatalf("status %d: got %v, want %v", tc.status, got, tc.want)
		}
		if rec.Method != http.MethodDelete || rec.Path != "/clients/c1" || rec.Token != "test-token" {
			t.Fatalf("unexpected request: %+v", rec)
		}
	}

	s, _ := newClientAdminSDK(t, http.StatusBadRequest, `{"message":"last admin"}`)
	if _, err := s.AdminDeleteClient(context.Background(), "c1"); err == nil {
		t.Fatal("expected an error for 400")
	}
}

func TestAdminIssueClientToken(t *testing.T) {
	s, rec := newClientAdminSDK(t, http.StatusOK, `{"audience":"c1","kind":"admin","token":"tok2","encryption_key":"key2"}`)

	creds, err := s.AdminIssueClientToken(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/clients/c1/token" || rec.Token != "test-token" {
		t.Fatalf("unexpected request: %+v", rec)
	}
	if creds.Audience != "c1" || creds.Token != "tok2" {
		t.Fatalf("unexpected credentials: %+v", creds)
	}
}

func TestAdminRevokeClientTokens(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   bool
	}{
		{http.StatusNoContent, true},
		{http.StatusNotModified, false},
	} {
		s, rec := newClientAdminSDK(t, tc.status, "")

		got, err := s.AdminRevokeClientTokens(context.Background(), "c1")
		if err != nil {
			t.Fatalf("status %d: %v", tc.status, err)
		}
		if got != tc.want {
			t.Fatalf("status %d: got %v, want %v", tc.status, got, tc.want)
		}
		if rec.Method != http.MethodPost || rec.Path != "/clients/c1/revoke-tokens" || rec.Token != "test-token" {
			t.Fatalf("unexpected request: %+v", rec)
		}
	}
}

func TestAdminClientIDRequired(t *testing.T) {
	s, rec := newClientAdminSDK(t, http.StatusNoContent, "")
	ctx := context.Background()

	calls := map[string]func(id string) error{
		"get":           func(id string) error { _, err := s.AdminGetClient(ctx, id); return err },
		"delete":        func(id string) error { _, err := s.AdminDeleteClient(ctx, id); return err },
		"issue token":   func(id string) error { _, err := s.AdminIssueClientToken(ctx, id); return err },
		"revoke tokens": func(id string) error { _, err := s.AdminRevokeClientTokens(ctx, id); return err },
	}
	for name, call := range calls {
		for _, id := range []string{"", "  "} {
			if err := call(id); !errors.Is(err, ErrClientIDRequired) {
				t.Fatalf("%s(%q): expected ErrClientIDRequired, got %v", name, id, err)
			}
		}
	}
	if rec.Count != 0 {
		t.Fatalf("expected no request to be sent, got %d", rec.Count)
	}
}
