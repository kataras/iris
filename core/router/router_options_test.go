package router_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/kataras/iris/v14/core/router"

	"gopkg.in/yaml.v3"
)

// allSet is a RouterOptions with every field set to a non-zero value, so that a
// dropped or misspelled tag cannot hide behind omitzero/omitempty.
var allSet = router.RouterOptions{
	Host:                             "example.com:8080",
	DisablePathCorrection:            true,
	DisablePathCorrectionRedirection: true,
	EnablePathIntelligence:           true,
	EnablePathEscape:                 true,
	ForceLowercaseRouting:            true,
	AllowQuerySemicolons:             true,
	EnableDynamicHandler:             true,
	FireMethodNotAllowed:             true,
}

// Every field survives a JSON round trip, and every json key is snake_case,
// matching the ini tags and context.ContextOptions.
func TestRouterOptionsJSONRoundTrip(t *testing.T) {
	b, err := json.Marshal(allSet)
	if err != nil {
		t.Fatal(err)
	}

	var keys map[string]any
	if err = json.Unmarshal(b, &keys); err != nil {
		t.Fatal(err)
	}

	assertKeys(t, "json", jsonKeys, keys)

	// The expected list itself must stay snake_case, so that a future edit to
	// jsonKeys cannot re-introduce a camelCase key by agreeing with a bad tag.
	for _, key := range jsonKeys {
		if key != strings.ToLower(key) || strings.ContainsAny(key, " -") {
			t.Errorf("json key %q is not snake_case", key)
		}
	}

	var got router.RouterOptions
	if err = json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, allSet) {
		t.Fatalf("json round trip = %+v, want %+v", got, allSet)
	}
}

// jsonKeys are the json object keys every field must produce: snake_case, one
// per exported field, matching the ini tags and context.ContextOptions.
// Spelled out rather than derived from the struct, so that a tag typo is caught
// instead of mirrored.
var jsonKeys = []string{
	"host",
	"disable_path_correction",
	"disable_path_correction_redirection",
	"enable_path_intelligence",
	"enable_path_escape",
	"force_lowercase_routing",
	"allow_query_semicolons",
	"enable_dynamic_handler",
	"fire_method_not_allowed",
}

// yamlKeys are the yaml keys, which are the Go field names by convention here
// and in context.ContextOptions - NOT snake_case. Asserted for the same reason:
// yaml.v3 falls back to the lower-cased field name when a tag is missing, so a
// dropped yaml tag would silently change "Host" into "host".
var yamlKeys = []string{
	"Host",
	"DisablePathCorrection",
	"DisablePathCorrectionRedirection",
	"EnablePathIntelligence",
	"EnablePathEscape",
	"ForceLowercaseRouting",
	"AllowQuerySemicolons",
	"EnableDynamicHandler",
	"FireMethodNotAllowed",
}

// assertKeys checks that "keys" holds exactly "want" and nothing else.
func assertKeys(t *testing.T, encoding string, want []string, keys map[string]any) {
	t.Helper()

	if len(keys) != len(want) {
		t.Fatalf("%s keys = %v, want exactly %v", encoding, keys, want)
	}

	for _, key := range want {
		if _, ok := keys[key]; !ok {
			t.Errorf("%s key %q is missing from %v", encoding, key, keys)
		}
	}
}

// Every field survives a YAML round trip, and every yaml key is the Go field
// name.
func TestRouterOptionsYAMLRoundTrip(t *testing.T) {
	b, err := yaml.Marshal(allSet)
	if err != nil {
		t.Fatal(err)
	}

	var keys map[string]any
	if err = yaml.Unmarshal(b, &keys); err != nil {
		t.Fatal(err)
	}

	assertKeys(t, "yaml", yamlKeys, keys)

	var got router.RouterOptions
	if err = yaml.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, allSet) {
		t.Fatalf("yaml round trip = %+v, want %+v; yaml:\n%s", got, allSet, b)
	}
}

// Every exported field carries all four tags, so that a field added later
// cannot silently fall back to the Go name in one encoding only.
func TestRouterOptionsTagsAreComplete(t *testing.T) {
	typ := reflect.TypeOf(router.RouterOptions{})

	for i := range typ.NumField() {
		field := typ.Field(i)

		for _, tag := range []string{"ini", "json", "yaml", "toml"} {
			if _, ok := field.Tag.Lookup(tag); !ok {
				t.Errorf("field %s has no %q tag", field.Name, tag)
			}
		}

		jsonKey, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		iniKey, _, _ := strings.Cut(field.Tag.Get("ini"), ",")
		if jsonKey != iniKey {
			t.Errorf("field %s: json key %q != ini key %q", field.Name, jsonKey, iniKey)
		}
	}
}
