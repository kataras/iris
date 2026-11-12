package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMergeJSON(t *testing.T) {
	tests := []struct {
		name   string
		inputs []any
		want   map[string]any
	}{
		{
			name:   "Single Value",
			inputs: []any{map[string]any{"key1": "value1"}},
			want:   map[string]any{"key1": "value1"},
		},
		{
			name:   "Two Values",
			inputs: []any{map[string]any{"key1": "value1"}, map[string]any{"key2": "value2"}},
			want:   map[string]any{"key1": "value1", "key2": "value2"},
		},
		{
			name: "Three Values with Struct",
			inputs: []any{map[string]any{"key1": "value1"}, map[string]any{"key2": "value2"}, struct {
				Key3 string `json:"key3"`
			}{Key3: "value3"}},
			want: map[string]any{"key1": "value1", "key2": "value2", "key3": "value3"},
		},
		{
			name: "Four Values with Structs",
			inputs: []any{map[string]any{"key1": "value1"}, map[string]any{"key2": "value2"}, struct {
				Key3 string `json:"key3"`
			}{Key3: "value3"}, struct {
				Key4 string `json:"key4"`
			}{Key4: "value4"}},
			want: map[string]any{"key1": "value1", "key2": "value2", "key3": "value3", "key4": "value4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBytes, err := MergeJSON(tt.inputs...)
			if err != nil {
				t.Errorf("MergeJSON() error = %v", err)
				return
			}

			// gotBytesFromMerged, _ := Merge(tt.inputs...).MarshalJSON()
			// if !bytes.Equal(gotBytes, gotBytesFromMerged) {
			// 	t.Errorf("MergeJSON() = %v, Merge() = %v", gotBytes, gotBytesFromMerged)
			// 	return
			// }

			var got map[string]any
			if err := json.Unmarshal(gotBytes, &got); err != nil {
				t.Errorf("Unmarshal error = %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergedMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input []any
		want  map[string]any
	}{
		{
			name:  "Single Value",
			input: []any{map[string]any{"key1": "value1"}},
			want:  map[string]any{"key1": "value1"},
		},
		{
			name:  "Two Values",
			input: []any{map[string]any{"key1": "value1"}, map[string]any{"key2": "value2"}},
			want:  map[string]any{"key1": "value1", "key2": "value2"},
		},
		{
			name: "Three Values with Struct",
			input: []any{map[string]any{"key1": "value1"}, map[string]any{"key2": "value2"}, struct {
				Key3 string `json:"key3"`
			}{Key3: "value3"}},
			want: map[string]any{"key1": "value1", "key2": "value2", "key3": "value3"},
		},
		{
			name: "Four Values with Structs",
			input: []any{map[string]any{"key1": "value1"}, map[string]any{"key2": "value2"}, struct {
				Key3 string `json:"key3"`
			}{Key3: "value3"}, struct {
				Key4 string `json:"key4"`
			}{Key4: "value4"}},
			want: map[string]any{"key1": "value1", "key2": "value2", "key3": "value3", "key4": "value4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Merge(tt.input...)
			gotBytes, err := m.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			var got map[string]any
			if err := json.Unmarshal(gotBytes, &got); err != nil {
				t.Errorf("Unmarshal error = %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeJSONNilValues(t *testing.T) {
	// When all values are nil, expect an empty object.
	merged, err := MergeJSON(nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := "{}"
	if string(merged) != expected {
		t.Errorf("Expected %s, got %s", expected, string(merged))
	}
}

func TestMergeJSONEmptyObjects(t *testing.T) {
	// When all values are empty objects, expect an empty object.
	merged, err := MergeJSON(map[string]int{}, map[string]int{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := "{}"
	if string(merged) != expected {
		t.Errorf("Expected %s, got %s", expected, string(merged))
	}
}

func TestMergeJSONSingleObject(t *testing.T) {
	// Single non-empty object should be returned as is.
	input := map[string]int{"a": 1}
	merged, err := MergeJSON(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expectedBytes, _ := json.Marshal(input)
	if string(merged) != string(expectedBytes) {
		t.Errorf("Expected %s, got %s", string(expectedBytes), string(merged))
	}
}

func TestMergeJSONTwoObjects(t *testing.T) {
	// Merge two objects.
	obj1 := map[string]int{"a": 1}
	obj2 := map[string]int{"b": 2}
	merged, err := MergeJSON(obj1, obj2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	var result map[string]int
	if err := json.Unmarshal(merged, &result); err != nil {
		t.Fatalf("Error unmarshalling merged JSON: %v", err)
	}
	expected := map[string]int{"a": 1, "b": 2}
	if len(result) != len(expected) {
		t.Errorf("Expected %d keys, got %d", len(expected), len(result))
	}
	for k, v := range expected {
		if result[k] != v {
			t.Errorf("For key %s, expected %d, got %d", k, v, result[k])
		}
	}
}

func TestMergeJSONMultipleObjects(t *testing.T) {
	// Test merging with a mix of nil, empty, and valid objects.
	obj1 := map[string]int{"a": 1}
	obj2 := map[string]int{"b": 2}
	obj3 := map[string]int{"c": 3}
	merged, err := MergeJSON(obj1, nil, map[string]int{}, obj2, obj3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	var result map[string]int
	if err := json.Unmarshal(merged, &result); err != nil {
		t.Fatalf("Error unmarshalling merged JSON: %v", err)
	}
	expected := map[string]int{"a": 1, "b": 2, "c": 3}
	if len(result) != len(expected) {
		t.Errorf("Expected %d keys, got %d", len(expected), len(result))
	}
	for k, v := range expected {
		if result[k] != v {
			t.Errorf("For key %s, expected %d, got %d", k, v, result[k])
		}
	}
}

func TestMergeJSONOverlappingKeys(t *testing.T) {
	// In case of overlapping keys, the later object will contribute its value.
	obj1 := map[string]int{"a": 1, "b": 2}
	obj2 := map[string]int{"b": 3, "c": 4}
	merged, err := MergeJSON(obj1, obj2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	var result map[string]int
	if err := json.Unmarshal(merged, &result); err != nil {
		t.Fatalf("Error unmarshalling merged JSON: %v", err)
	}
	// In Go's map unmarshalling, duplicate keys result in the last one taking effect.
	expected := map[string]int{"a": 1, "b": 3, "c": 4}
	if len(result) != len(expected) {
		t.Errorf("Expected %d keys, got %d", len(expected), len(result))
	}
	for k, v := range expected {
		if result[k] != v {
			t.Errorf("For key %s, expected %d, got %d", k, v, result[k])
		}
	}
}

func TestMergeJSONNestedObjects(t *testing.T) {
	// Test merging objects that contain nested objects.
	obj1 := map[string]any{"a": map[string]int{"x": 1}}
	obj2 := map[string]any{"b": map[string]int{"y": 2}}
	merged, err := MergeJSON(obj1, obj2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(merged, &result); err != nil {
		t.Fatalf("Error unmarshalling merged JSON: %v", err)
	}
	// Validate nested content.
	a, ok := result["a"].(map[string]any)
	if !ok {
		t.Errorf("Expected key 'a' to be a nested object")
	} else if a["x"].(float64) != 1 {
		t.Errorf("Expected nested key 'x' to be 1, got %v", a["x"])
	}
	b, ok := result["b"].(map[string]any)
	if !ok {
		t.Errorf("Expected key 'b' to be a nested object")
	} else if b["y"].(float64) != 2 {
		t.Errorf("Expected nested key 'y' to be 2, got %v", b["y"])
	}
}

func TestMergeJSONNonObject(t *testing.T) {
	// If a value does not marshal to a JSON object, expect an error.
	_, err := MergeJSON("not an object")
	if err == nil {
		t.Error("Expected error when value is not a JSON object")
	}
}

func TestMergeJSONArray(t *testing.T) {
	// If a value is an array, it is not considered a JSON object.
	_, err := MergeJSON([]int{1, 2, 3})
	if err == nil {
		t.Error("Expected error when value is not a JSON object")
	}
}

// Overlapping keys are merged, later part wins, and the output carries each
// key once: a client decoding with the v2 default (duplicate names rejected)
// must be able to read it.
func TestMergeJSONNoDuplicateKeys(t *testing.T) {
	merged, err := MergeJSON(map[string]int{"a": 1, "b": 2}, map[string]int{"b": 3, "c": 4})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(merged), `{"a":1,"b":3,"c":4}`; got != want {
		t.Fatalf("MergeJSON() = %s, want %s", got, want)
	}
}

// Parts are marshaled with the framework's JSON policy, so a struct renders
// the same inside Merge as it does through ctx.JSON: nil slices are [].
func TestMergeJSONNilSliceRendersAsEmptyArray(t *testing.T) {
	merged, err := MergeJSON(struct {
		Tags []string `json:"tags"`
	}{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(merged), `{"tags":[]}`; got != want {
		t.Fatalf("MergeJSON() = %s, want %s", got, want)
	}
}
