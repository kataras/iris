package reflex

import (
	"reflect"
	"testing"
)

// Configuration is the main structure that holds the application settings.
type Configuration struct {
	Field1       string                   // A simple field.
	Field2       *NestedStruct            // A pointer to a nested struct.
	ComplexSlice []*NestedStruct          // A slice of pointers to NestedStruct.
	ComplexMap   map[string]*NestedStruct // A map with string keys and *NestedStruct values.
}

// NestedStruct is a nested structure within Configuration.
type NestedStruct struct {
	NestedField1 int            // A simple field within the nested struct.
	NestedField2 *AnotherStruct // A pointer to another struct, showcasing deeper nesting.
}

// AnotherStruct is another structure that can be nested within NestedStruct.
type AnotherStruct struct {
	SomeField string // Example field in AnotherStruct.
}

// DeepClone creates a deep copy of the Configuration, including nested pointer fields.
func (c *Configuration) DeepClone() *Configuration {
	// Use reflection to clone the struct.
	val := reflect.ValueOf(c)
	clonedVal := deepClone(val)
	return clonedVal.Interface().(*Configuration)
}

// TestDeepClone tests the DeepClone method for various scenarios.
func TestDeepClone(t *testing.T) {
	// Test data setup
	nestedStruct := NestedStruct{
		NestedField1: 100,
		NestedField2: &AnotherStruct{},
	}
	config := Configuration{
		Field1: "test",
		Field2: &nestedStruct,
	}

	// Perform the deep clone.
	clonedConfig := config.DeepClone()

	// Test if the top-level fields are equal.
	if clonedConfig.Field1 != config.Field1 {
		t.Errorf("Field1 was not cloned correctly, got: %s, want: %s", clonedConfig.Field1, config.Field1)
	}

	// Test if the pointer fields are not the same (deep clone check).
	if clonedConfig.Field2 == config.Field2 {
		t.Error("Field2 was not deep cloned correctly, pointers are equal")
	}

	// Test if the nested struct fields are equal.
	if clonedConfig.Field2.NestedField1 != config.Field2.NestedField1 {
		t.Errorf("NestedField1 was not cloned correctly, got: %d, want: %d", clonedConfig.Field2.NestedField1, config.Field2.NestedField1)
	}

	// Test if the nested pointer fields are not the same (deep clone check).
	if clonedConfig.Field2.NestedField2 == config.Field2.NestedField2 {
		t.Error("NestedField2 was not deep cloned correctly, pointers are equal")
	}

	// Test if the deep clone did not alter the original configuration.
	const alteredInt = 1000

	config.Field1 = "altered"
	nestedStruct.NestedField1 = alteredInt // Alter the struct the original config points at.
	if clonedConfig.Field1 == "altered" {
		t.Error("Deep clone altered the original configuration's Field1")
	}
	if clonedConfig.Field2.NestedField1 == alteredInt {
		t.Error("Deep clone altered the original configuration's NestedField1")
	}
}

// TestDeepCloneNilPointer tests the DeepClone method when there are nil pointers in the struct.
func TestDeepCloneNilPointer(t *testing.T) {
	config := Configuration{
		Field1: "test",
		Field2: nil, // Nil pointer field.
	}

	// Perform the deep clone.
	clonedConfig := config.DeepClone()

	// Test if the nil pointer field remains nil after cloning.
	if clonedConfig.Field2 != nil {
		t.Error("Nil pointer field was not cloned correctly, expected nil")
	}
}

// TestDeepCloneComplexStruct tests the DeepClone method for a struct with complex fields like slices and maps.
func TestDeepCloneComplexStruct(t *testing.T) {
	// Test data setup for complex fields.
	anotherStruct := AnotherStruct{
		// Initialize fields of AnotherStruct.
	}
	nestedStruct := NestedStruct{
		NestedField1: 100,
		NestedField2: &anotherStruct,
	}
	complexSlice := []*NestedStruct{&nestedStruct}
	complexMap := map[string]*NestedStruct{
		"key": &nestedStruct,
	}

	config := Configuration{
		Field1: "test",
		Field2: &nestedStruct,
		// Assuming Configuration has these fields for the purpose of the test.
		ComplexSlice: complexSlice,
		ComplexMap:   complexMap,
	}

	// Perform the deep clone.
	clonedConfig := config.DeepClone()

	// Test if the slice has been deep cloned.
	if &clonedConfig.ComplexSlice[0] == &config.ComplexSlice[0] {
		t.Error("ComplexSlice was not deep cloned correctly, slice element pointers are equal")
	}

	// Test if the map has been deep cloned.
	if clonedConfig.ComplexMap["key"] == config.ComplexMap["key"] {
		t.Error("ComplexMap was not deep cloned correctly, map element pointers are equal")
	}

	// Test if the nested struct within the slice has been deep cloned.
	if clonedConfig.ComplexSlice[0].NestedField2 == config.ComplexSlice[0].NestedField2 {
		t.Error("NestedField2 within slice was not deep cloned correctly, pointers are equal")
	}

	// Test if the nested struct within the map has been deep cloned.
	if clonedConfig.ComplexMap["key"].NestedField2 == config.ComplexMap["key"].NestedField2 {
		t.Error("NestedField2 within map was not deep cloned correctly, pointers are equal")
	}

	// Test if the deep clone did not alter the original configuration's complex fields
	const alteredInt = 1000

	nestedStruct.NestedField1 = alteredInt // Alter the struct the original config points at.
	if clonedConfig.Field2.NestedField1 == alteredInt {
		t.Error("Deep clone altered the original configuration's NestedField1 within ComplexSlice")
	}
	if clonedConfig.ComplexMap["key"].NestedField1 == alteredInt {
		t.Error("Deep clone altered the original configuration's NestedField1 within ComplexMap")
	}
}
