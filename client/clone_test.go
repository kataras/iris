package client

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestNoOptionKeepsTheIrisJSONPolicy: NoOption drops the parent's options but
// the Iris JSON defaults survive it, across two generations of clones.
func TestNoOptionKeepsTheIrisJSONPolicy(t *testing.T) {
	srv := echoServer(t)

	parent := New(BaseURL("http://parent.invalid"))
	child := parent.Clone(NoOption, BaseURL(srv.URL))
	grand := child.Clone()

	if grand.BaseURL != srv.URL {
		t.Fatalf("expected the clone's base URL, got %q", grand.BaseURL)
	}

	got, err := grand.BindJSON[timed](context.Background(), http.MethodPost, "/", timed{Timeout: time.Minute})
	if err != nil {
		t.Fatalf("the Iris JSON policy must survive NoOption: %v", err)
	}
	if got.Timeout != time.Minute {
		t.Fatalf("expected the duration round trip, got %v", got.Timeout)
	}
}
