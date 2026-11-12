package types

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJSONSingleString(t *testing.T) {
	var soa StringOrArray
	input := `"single_string"`
	expected := "single_string"
	err := json.Unmarshal([]byte(input), &soa)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
	}
	if string(soa) != expected {
		t.Errorf("UnmarshalJSON() = %v, want %v", soa, expected)
	}
}

func TestUnmarshalJSONArrayOfStrings(t *testing.T) {
	var soa StringOrArray
	input := `["string1","string2","string3"]`
	expected := "string1,string2,string3"
	err := json.Unmarshal([]byte(input), &soa)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
	}
	if string(soa) != expected {
		t.Errorf("UnmarshalJSON() = %v, want %v", soa, expected)
	}
}

func TestUnmarshalJSONNull(t *testing.T) {
	var soa StringOrArray
	input := `null`
	expected := ""
	err := json.Unmarshal([]byte(input), &soa)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
	}
	if string(soa) != expected {
		t.Errorf("UnmarshalJSON() = %v, want %v", soa, expected)
	}
}

func TestMarshalJSONSingleString(t *testing.T) {
	soa := StringOrArray("single_string")
	expected := `"single_string"`
	output, err := json.Marshal(soa)
	if err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
	}
	if string(output) != expected {
		t.Errorf("MarshalJSON() = %v, want %v", string(output), expected)
	}
}

func TestMarshalJSONArrayOfStrings(t *testing.T) {
	soa := StringOrArray("string1,string2,string3")
	expected := `"string1,string2,string3"`
	output, err := json.Marshal(soa)
	if err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
	}
	if string(output) != expected {
		t.Errorf("MarshalJSON() = %v, want %v", string(output), expected)
	}
}
