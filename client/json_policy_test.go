package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	json "encoding/json/v2"
)

type timed struct {
	Name    string        `json:"name"`
	Timeout time.Duration `json:"timeout"`
}

// echoServer answers with the request body as JSON.
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, r.Body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// bodyServer answers with a fixed JSON body and status.
func bodyServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestDurationTravelsAsNanoseconds: the Iris JSON policy encodes time.Duration
// as int64 nanoseconds and reads it back. Under plain v1 defaults json/v2 has
// no representation for Duration and both directions fail.
func TestDurationTravelsAsNanoseconds(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)

	c := New(BaseURL(srv.URL))
	got, err := c.BindJSON[timed](context.Background(), http.MethodPost, "/", timed{Name: "a", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(sent, `"timeout":2000000000`) {
		t.Fatalf("expected the duration encoded as nanoseconds, sent %s", sent)
	}
	if got.Timeout != 2*time.Second {
		t.Fatalf("expected the duration decoded from nanoseconds, got %v", got.Timeout)
	}
}

// TestMapKeysEncodeDeterministically: jsonutil.DefaultMarshalOptions sorts map keys.
func TestMapKeysEncodeDeterministically(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	payload := map[string]int{"zeta": 1, "alpha": 2, "mid": 3}
	if err := New(BaseURL(srv.URL)).Call(context.Background(), http.MethodPost, "/", payload); err != nil {
		t.Fatal(err)
	}

	if sent != `{"alpha":2,"mid":3,"zeta":1}` {
		t.Fatalf("expected sorted keys, sent %s", sent)
	}
}

// TestMemberNamesMatchCaseInsensitively: deployed clients bind under the old rules.
func TestMemberNamesMatchCaseInsensitively(t *testing.T) {
	srv := bodyServer(t, http.StatusOK, `{"NAME":"makis"}`)

	got, err := New(BaseURL(srv.URL)).BindJSON[timed](context.Background(), http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "makis" {
		t.Fatalf("expected a case-insensitive match, got %q", got.Name)
	}
}

// TestDuplicateMembersAreRejectedOnDecode: the v2 default the framework keeps.
func TestDuplicateMembersAreRejectedOnDecode(t *testing.T) {
	srv := bodyServer(t, http.StatusOK, `{"name":"a","name":"b"}`)

	if _, err := New(BaseURL(srv.URL)).BindJSON[timed](context.Background(), http.MethodGet, "/", nil); err == nil {
		t.Fatal("expected the duplicate member to be rejected")
	}
}

// TestPackageLevelBindersUseTheIrisPolicy: Bind, BindError and
// DecodeError decode a Duration from nanoseconds, so they run on jsonutil's
// options and not on the engine's v1 defaults.
func TestPackageLevelBindersUseTheIrisPolicy(t *testing.T) {
	body := `{"name":"a","timeout":1000000000}`
	newResp := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}
	}

	if got, err := Bind[timed](newResp()); err != nil || got.Timeout != time.Second {
		t.Fatalf("Bind: got %+v, %v", got, err)
	}

	apiErr := APIError{Response: newResp(), Body: []byte(body)}
	if got, err := BindError[timed](apiErr); err != nil || got.Timeout != time.Second {
		t.Fatalf("BindError: got %+v, %v", got, err)
	}

	var v timed
	if err := DecodeError(apiErr, &v); err != nil || v.Timeout != time.Second {
		t.Fatalf("DecodeError: got %+v, %v", v, err)
	}

	// A caller's option is applied on top of the policy.
	if _, err := Bind[timed](newResp(), json.RejectUnknownMembers(true)); err != nil {
		t.Fatalf("known members only, expected success, got %v", err)
	}
	strictBody := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"name":"a","timeout":1,"extra":true}`)),
	}
	if _, err := Bind[timed](strictBody, json.RejectUnknownMembers(true)); err == nil {
		t.Fatal("expected the caller's RejectUnknownMembers to apply")
	}
}

// TestUserJSONOptionsWinOverTheDefaults: an option given to New replaces the
// Iris policy for that direction, as documented.
func TestUserJSONOptionsWinOverTheDefaults(t *testing.T) {
	srv := bodyServer(t, http.StatusOK, `{"name":"a","extra":true}`)

	c := New(BaseURL(srv.URL), JSONUnmarshalOptions(json.RejectUnknownMembers(true)))
	if _, err := c.BindJSON[timed](context.Background(), http.MethodGet, "/", nil); err == nil {
		t.Fatal("expected the user's RejectUnknownMembers to apply")
	}
}
