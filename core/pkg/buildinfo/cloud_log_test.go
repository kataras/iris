package buildinfo

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

type captureHandler struct{ records []slog.Record }

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func TestLogCloudInfoMessageHasNoPrintfVerbs(t *testing.T) {
	h := &captureHandler{}
	logCloudInfo(slog.New(h), &ContainerInfo{
		IsContainer: true,
		Provider:    ProviderAWS,
		Platform:    "ecs",
	})

	if len(h.records) == 0 {
		t.Fatal("no log records")
	}
	first := h.records[0]
	if strings.Contains(first.Message, "%") {
		t.Fatalf("message contains a printf verb: %q", first.Message)
	}

	got := map[string]string{}
	first.Attrs(func(a slog.Attr) bool {
		got[a.Key] = a.Value.String()
		return true
	})
	if got["cloud_provider"] != string(ProviderAWS) {
		t.Errorf("cloud_provider = %q, want %q (attrs %v)", got["cloud_provider"], ProviderAWS, got)
	}
	if got["platform"] != "ecs" {
		t.Errorf("platform = %q, want ecs (attrs %v)", got["platform"], got)
	}
	if len(got) != 2 {
		t.Errorf("want exactly 2 attributes, got %v", got)
	}
}
