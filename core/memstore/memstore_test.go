// white-box testing
package memstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

type myTestObject struct {
	Name string `json:"name"`
}

func TestMuttable(t *testing.T) {
	var p Store

	// slice
	p.Set("slice", []myTestObject{{"value 1"}, {"value 2"}})
	v := p.Get("slice").([]myTestObject)
	v[0].Name = "modified"

	vv := p.Get("slice").([]myTestObject)
	if vv[0].Name != "modified" {
		t.Fatalf("expected slice to be muttable but caller was not able to change its value")
	}

	// map

	p.Set("map", map[string]myTestObject{"key 1": {"value 1"}, "key 2": {"value 2"}})
	vMap := p.Get("map").(map[string]myTestObject)
	vMap["key 1"] = myTestObject{"modified"}

	vvMap := p.Get("map").(map[string]myTestObject)
	if vvMap["key 1"].Name != "modified" {
		t.Fatalf("expected map to be muttable but caller was not able to change its value")
	}

	// object pointer of a value, it can change like maps or slices and arrays.
	p.Set("objp", &myTestObject{"value"})
	// we expect pointer here, as we set it.
	vObjP := p.Get("objp").(*myTestObject)

	vObjP.Name = "modified"

	vvObjP := p.Get("objp").(*myTestObject)
	if vvObjP.Name != "modified" {
		t.Fatalf("expected objp to be muttable but caller was able to change its value")
	}
}

func TestImmutable(t *testing.T) {
	var p Store

	// slice
	p.SetImmutable("slice", []myTestObject{{"value 1"}, {"value 2"}})
	v := p.Get("slice").([]myTestObject)
	v[0].Name = "modified"

	vv := p.Get("slice").([]myTestObject)
	if vv[0].Name == "modified" {
		t.Fatalf("expected slice to be immutable but caller was able to change its value")
	}

	// map
	p.SetImmutable("map", map[string]myTestObject{"key 1": {"value 1"}, "key 2": {"value 2"}})
	vMap := p.Get("map").(map[string]myTestObject)
	vMap["key 1"] = myTestObject{"modified"}

	vvMap := p.Get("map").(map[string]myTestObject)
	if vvMap["key 1"].Name == "modified" {
		t.Fatalf("expected map to be immutable but caller was able to change its value")
	}

	// object value, it's immutable at all cases.
	p.SetImmutable("obj", myTestObject{"value"})
	vObj := p.Get("obj").(myTestObject)
	vObj.Name = "modified"

	vvObj := p.Get("obj").(myTestObject)
	if vvObj.Name == "modified" {
		t.Fatalf("expected obj to be immutable but caller was able to change its value")
	}

	// object pointer of a value, it's immutable at all cases.
	p.SetImmutable("objp", &myTestObject{"value"})
	// we expect no pointer here if SetImmutable.
	// so it can't be changed by-design
	vObjP := p.Get("objp").(myTestObject)

	vObjP.Name = "modified"

	vvObjP := p.Get("objp").(myTestObject)
	if vvObjP.Name == "modified" {
		t.Fatalf("expected objp to be immutable but caller was able to change its value")
	}
}

func TestImmutableSetOnlyWithSetImmutable(t *testing.T) {
	var p Store

	p.SetImmutable("objp", &myTestObject{"value"})

	p.Set("objp", &myTestObject{"modified"})
	vObjP := p.Get("objp").(myTestObject)
	if vObjP.Name == "modified" {
		t.Fatalf("caller should not be able to change the immutable entry with a simple `Set`")
	}

	p.SetImmutable("objp", &myTestObject{"value with SetImmutable"})
	vvObjP := p.Get("objp").(myTestObject)
	if vvObjP.Name != "value with SetImmutable" {
		t.Fatalf("caller should be able to change the immutable entry with a `SetImmutable`")
	}
}

func TestGetInt64Default(t *testing.T) {
	var p Store

	p.Set("a uint16", uint16(2))
	if v := p.GetInt64Default("a uint16", 0); v != 2 {
		t.Fatalf("unexpected value of %d", v)
	}
}

func TestJSON(t *testing.T) {
	var p Store

	p.Set("key1", "value1")
	p.Set("key2", 2)
	p.Set("key3", myTestObject{Name: "makis"})

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	expectedJSON := []byte(`[{"key":"key1","value":"value1"},{"key":"key2","value":2},{"key":"key3","value":{"name":"makis"}}]`)

	if !bytes.Equal(b, expectedJSON) {
		t.Fatalf("expected: %s but got: %s", string(expectedJSON), string(b))
	}

	var newStore Store
	if err = json.Unmarshal(b, &newStore); err != nil {
		t.Fatal(err)
	}

	for i, v := range newStore {
		expected, got := p.Get(v.Key), v.ValueRaw

		if ex, g := fmt.Sprintf("%v", expected), fmt.Sprintf("%v", got); ex != g {
			if _, isMap := got.(map[string]interface{}); isMap {
				// was struct but converted into map (as expected).
				b1, _ := json.Marshal(expected)
				b2, _ := json.Marshal(got)

				if !bytes.Equal(b1, b2) {
					t.Fatalf("[%d] JSON expected: %s but got: %s", i, string(b1), string(b2))
				}

				continue
			}
			t.Fatalf("[%d] expected: %s but got: %s", i, ex, g)
		}
	}
}
