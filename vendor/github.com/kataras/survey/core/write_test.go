package core

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrite_returnsErrorIfTargetNotPtr(t *testing.T) {
	// try to copy a value to a non-pointer
	err := WriteAnswer(true, "hello", true)
	// make sure there was an error
	if err == nil {
		t.Error("Did not encounter error when writing to non-pointer.")
	}
}

func TestWrite_canWriteToBool(t *testing.T) {
	// a pointer to hold the boolean value
	ptr := true

	// try to copy a false value to the pointer
	WriteAnswer(&ptr, "", false)

	// if the value is true
	if ptr {
		// the test failed
		t.Error("Could not write a false bool to a pointer")
	}
}

func TestWrite_canWriteString(t *testing.T) {
	// a pointer to hold the boolean value
	ptr := ""

	// try to copy a false value to the pointer
	err := WriteAnswer(&ptr, "", "hello")
	if err != nil {
		t.Error(err)
	}

	// if the value is not what we wrote
	if ptr != "hello" {
		t.Error("Could not write a string value to a pointer")
	}
}

func TestWrite_canWriteSlice(t *testing.T) {
	// a pointer to hold the value
	ptr := []string{}

	// copy in a value
	WriteAnswer(&ptr, "", []string{"hello", "world"})

	// make sure there are two entries
	assert.Equal(t, []string{"hello", "world"}, ptr)
}

func TestWrite_recoversInvalidReflection(t *testing.T) {
	// a variable to mutate
	ptr := false

	// write a boolean value to the string
	err := WriteAnswer(&ptr, "", "hello")

	// if there was no error
	if err == nil {
		// the test failed
		t.Error("Did not encounter error when forced invalid write.")
	}
}

func TestWriteAnswer_handlesNonStructValues(t *testing.T) {
	// the value to write to
	ptr := ""

	// write a value to the pointer
	WriteAnswer(&ptr, "", "world")

	// if we didn't change the value appropriate
	if ptr != "world" {
		// the test failed
		t.Error("Did not write value to primitive pointer")
	}
}

func TestWriteAnswer_canMutateStruct(t *testing.T) {
	// the struct to hold the answer
	ptr := struct{ Name string }{}

	// write a value to an existing field
	err := WriteAnswer(&ptr, "name", "world")
	if err != nil {
		// the test failed
		t.Errorf("Encountered error while writing answer: %v", err.Error())
		// we're done here
		return
	}

	// make sure we changed the field
	if ptr.Name != "world" {
		// the test failed
		t.Error("Did not mutate struct field when writing answer.")
	}
}

func TestWriteAnswer_canMutateMap(t *testing.T) {
	// the map to hold the answer
	ptr := make(map[string]interface{})

	// write a value to an existing field
	err := WriteAnswer(&ptr, "name", "world")
	if err != nil {
		// the test failed
		t.Errorf("Encountered error while writing answer: %v", err.Error())
		// we're done here
		return
	}

	// make sure we changed the field
	if ptr["name"] != "world" {
		// the test failed
		t.Error("Did not mutate map when writing answer.")
	}
}

func TestWrite_returnsErrorIfInvalidMapType(t *testing.T) {
	// try to copy a value to a non map[string]interface{}
	ptr := make(map[int]string)

	err := WriteAnswer(&ptr, "name", "world")
	// make sure there was an error
	if err == nil {
		t.Error("Did not encounter error when writing to invalid map.")
	}
}

func TestWrite_writesStringSliceToIntSlice(t *testing.T) {
	// make a slice of int to write to
	target := []int{}

	// write the answer
	err := WriteAnswer(&target, "name", []string{"1", "2", "3"})

	// make sure there was no error
	assert.Nil(t, err, "WriteSlice to Int Slice")
	// and we got what we wanted
	assert.Equal(t, []int{1, 2, 3}, target)
}

func TestWrite_writesStringArrayToIntArray(t *testing.T) {
	// make an array of int to write to
	target := [3]int{}

	// write the answer
	err := WriteAnswer(&target, "name", [3]string{"1", "2", "3"})

	// make sure there was no error
	assert.Nil(t, err, "WriteArray to Int Array")
	// and we got what we wanted
	assert.Equal(t, [3]int{1, 2, 3}, target)
}

func TestWriteAnswer_returnsErrWhenFieldNotFound(t *testing.T) {
	// the struct to hold the answer
	ptr := struct{ Name string }{}

	// write a value to an existing field
	err := WriteAnswer(&ptr, "", "world")

	if err == nil {
		// the test failed
		t.Error("Did not encountered error while writing answer to non-existing field.")
	}
}

func TestFindFieldIndex_canFindExportedField(t *testing.T) {
	// create a reflective wrapper over the struct to look through
	val := reflect.ValueOf(struct{ Name string }{})

	// find the field matching "name"
	fieldIndex, err := findFieldIndex(val, "name")
	// if something went wrong
	if err != nil {
		// the test failed
		t.Error(err.Error())
		return
	}

	// make sure we got the right value
	if val.Type().Field(fieldIndex).Name != "Name" {
		// the test failed
		t.Errorf("Did not find the correct field name. Expected 'Name' found %v.", val.Type().Field(fieldIndex).Name)
	}
}

func TestFindFieldIndex_canFindTaggedField(t *testing.T) {
	// the struct to look through
	val := reflect.ValueOf(struct {
		Username string `survey:"name"`
	}{})

	// find the field matching "name"
	fieldIndex, err := findFieldIndex(val, "name")
	// if something went wrong
	if err != nil {
		// the test failed
		t.Error(err.Error())
		return
	}

	// make sure we got the right value
	if val.Type().Field(fieldIndex).Name != "Username" {
		// the test failed
		t.Errorf("Did not find the correct field name. Expected 'Username' found %v.", val.Type().Field(fieldIndex).Name)
	}
}

func TestFindFieldIndex_canHandleCapitalAnswerNames(t *testing.T) {
	// create a reflective wrapper over the struct to look through
	val := reflect.ValueOf(struct{ Name string }{})

	// find the field matching "name"
	fieldIndex, err := findFieldIndex(val, "Name")
	// if something went wrong
	if err != nil {
		// the test failed
		t.Error(err.Error())
		return
	}

	// make sure we got the right value
	if val.Type().Field(fieldIndex).Name != "Name" {
		// the test failed
		t.Errorf("Did not find the correct field name. Expected 'Name' found %v.", val.Type().Field(fieldIndex).Name)
	}
}

func TestFindFieldIndex_tagOverwriteFieldName(t *testing.T) {
	// the struct to look through
	val := reflect.ValueOf(struct {
		Name     string
		Username string `survey:"name"`
	}{})

	// find the field matching "name"
	fieldIndex, err := findFieldIndex(val, "name")
	// if something went wrong
	if err != nil {
		// the test failed
		t.Error(err.Error())
		return
	}

	// make sure we got the right value
	if val.Type().Field(fieldIndex).Name != "Username" {
		// the test failed
		t.Errorf("Did not find the correct field name. Expected 'Username' found %v.", val.Type().Field(fieldIndex).Name)
	}
}

type testFieldSettable struct {
	Values map[string]string
}

type testStringSettable struct {
	Value string `survey:"string"`
}

type testTaggedStruct struct {
	TaggedValue testStringSettable `survey:"tagged"`
}

type testPtrTaggedStruct struct {
	TaggedValue *testStringSettable `survey:"tagged"`
}

func (t *testFieldSettable) WriteAnswer(name string, value interface{}) error {
	if t.Values == nil {
		t.Values = map[string]string{}
	}
	if v, ok := value.(string); ok {
		t.Values[name] = v
		return nil
	}
	return fmt.Errorf("Incompatible type %T", value)
}

func (t *testStringSettable) WriteAnswer(_ string, value interface{}) error {
	t.Value = value.(string)
	return nil
}

func TestWriteWithFieldSettable(t *testing.T) {
	testSet1 := testFieldSettable{}
	err := WriteAnswer(&testSet1, "values", "stringVal")
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"values": "stringVal"}, testSet1.Values)

	testSet2 := testFieldSettable{}
	err = WriteAnswer(&testSet2, "values", 123)
	assert.Error(t, fmt.Errorf("Incompatible type int64"), err)
	assert.Equal(t, map[string]string{}, testSet2.Values)

	testString1 := testStringSettable{}
	err = WriteAnswer(&testString1, "", "value1")
	assert.Nil(t, err)
	assert.Equal(t, testStringSettable{"value1"}, testString1)

	testSetStruct := testTaggedStruct{}
	err = WriteAnswer(&testSetStruct, "tagged", "stringVal1")
	assert.Nil(t, err)
	assert.Equal(t, testTaggedStruct{TaggedValue: testStringSettable{"stringVal1"}}, testSetStruct)

	testPtrSetStruct := testPtrTaggedStruct{&testStringSettable{}}
	err = WriteAnswer(&testPtrSetStruct, "tagged", "stringVal1")
	assert.Nil(t, err)
	assert.Equal(t, testPtrTaggedStruct{TaggedValue: &testStringSettable{"stringVal1"}}, testPtrSetStruct)
}

// CONVERSION TESTS
func TestWrite_canStringToBool(t *testing.T) {
	// a pointer to hold the boolean value
	ptr := true

	// try to copy a false value to the pointer
	WriteAnswer(&ptr, "", "false")

	// if the value is true
	if ptr {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToInt(t *testing.T) {
	// a pointer to hold the value
	var ptr int = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToInt8(t *testing.T) {
	// a pointer to hold the value
	var ptr int8 = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToInt16(t *testing.T) {
	// a pointer to hold the value
	var ptr int16 = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToInt32(t *testing.T) {
	// a pointer to hold the value
	var ptr int32 = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToInt64(t *testing.T) {
	// a pointer to hold the value
	var ptr int64 = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToUint(t *testing.T) {
	// a pointer to hold the value
	var ptr uint = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToUint8(t *testing.T) {
	// a pointer to hold the value
	var ptr uint8 = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToUint16(t *testing.T) {
	// a pointer to hold the value
	var ptr uint16 = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToUint32(t *testing.T) {
	// a pointer to hold the value
	var ptr uint32 = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToUint64(t *testing.T) {
	// a pointer to hold the value
	var ptr uint64 = 1

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2")

	// if the value is true
	if ptr != 2 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToFloat32(t *testing.T) {
	// a pointer to hold the value
	var ptr float32 = 1.0

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2.5")

	// if the value is true
	if ptr != 2.5 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canStringToFloat64(t *testing.T) {
	// a pointer to hold the value
	var ptr float64 = 1.0

	// try to copy a value to the pointer
	WriteAnswer(&ptr, "", "2.5")

	// if the value is true
	if ptr != 2.5 {
		// the test failed
		t.Error("Could not convert string to pointer type")
	}
}

func TestWrite_canConvertStructFieldTypes(t *testing.T) {
	// the struct to hold the answer
	ptr := struct {
		Name   string
		Age    uint
		Male   bool
		Height float64
	}{}

	// write the values as strings
	check(t, WriteAnswer(&ptr, "name", "Bob"))
	check(t, WriteAnswer(&ptr, "age", "22"))
	check(t, WriteAnswer(&ptr, "male", "true"))
	check(t, WriteAnswer(&ptr, "height", "6.2"))

	// make sure we changed the fields
	if ptr.Name != "Bob" {
		t.Error("Did not mutate Name when writing answer.")
	}

	if ptr.Age != 22 {
		t.Error("Did not mutate Age when writing answer.")
	}

	if !ptr.Male {
		t.Error("Did not mutate Male when writing answer.")
	}

	if ptr.Height != 6.2 {
		t.Error("Did not mutate Height when writing answer.")
	}
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Encountered error while writing answer: %v", err.Error())
	}
}
