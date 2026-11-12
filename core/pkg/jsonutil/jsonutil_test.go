package jsonutil

import (
	json "encoding/json/v2"
	"testing"
)

type duplicateEmitter struct{}

func (duplicateEmitter) MarshalJSON() ([]byte, error) { return []byte(`{"a":1,"a":2}`), nil }

// The marshal and unmarshal defaults agree on duplicate object names: both
// reject them, so nothing the framework writes is unreadable by its own
// client, and the JSON-smuggling defense on reads has no write-side hole.
func TestDefaultOptionsRejectDuplicateNamesBothWays(t *testing.T) {
	if _, err := json.Marshal(duplicateEmitter{}, DefaultMarshalOptionsSlice...); err == nil {
		t.Fatal("DefaultMarshalOptions accepted duplicate object names on output")
	}

	var v map[string]int
	if err := json.Unmarshal([]byte(`{"a":1,"a":2}`), &v, DefaultUnmarshalOptionsSlice...); err == nil {
		t.Fatal("DefaultUnmarshalOptions accepted duplicate object names on input")
	}
}

// Case-insensitive matching keeps v1's EqualFold semantics: '_' and '-' still
// count. Under plain v2 folding a camelCased "vO2PEAK" matches both fields
// below and the whole body is rejected as an ambiguous name.
func TestDefaultUnmarshalOptionsKeepDelimitersInCaseInsensitiveNames(t *testing.T) {
	var v struct {
		Peak         float64 `json:"VO2PEAK"`
		RecoveryPeak float64 `json:"VO2_PEAK"`
	}

	if err := json.Unmarshal([]byte(`{"vO2PEAK":27,"vO2_PEAK":1829.5}`), &v, DefaultUnmarshalOptionsSlice...); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v.Peak != 27 || v.RecoveryPeak != 1829.5 {
		t.Fatalf("got VO2PEAK=%v VO2_PEAK=%v, want 27 and 1829.5", v.Peak, v.RecoveryPeak)
	}
}
