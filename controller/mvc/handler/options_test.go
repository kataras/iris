package handler

import "testing"

func TestNoOptionSkipsDefaultRequestOptions(t *testing.T) {
	opts := makeOptions([]Option{
		WithRequestOptions(withPathParameter[string]("id"), withStatusCode(200)),
		NoOption,
	})

	if !opts.skipDefaults {
		t.Fatal("expected NoOption to mark the options as skipping defaults")
	}

	before := len(opts.request)
	opts.SetDefaultRequestOptions(withStatusCode(201))
	if got := len(opts.request); got != before {
		t.Fatalf("expected SetDefaultRequestOptions to be a no-op after NoOption, request options grew from %d to %d", before, got)
	}
}

func TestSetDefaultRequestOptionsAppliesWithoutNoOption(t *testing.T) {
	opts := makeOptions([]Option{
		WithRequestOptions(withPathParameter[string]("id")),
	})

	before := len(opts.request)
	opts.SetDefaultRequestOptions(withStatusCode(201))
	if got := len(opts.request); got != before+1 {
		t.Fatalf("expected the default request option to be prepended, request options went from %d to %d", before, got)
	}
}
