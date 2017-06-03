package unis

import (
	"reflect"
	"testing"
)

// getTestName returns the test's name, i.e "TestPrepender", "TestPrefixRemover".
func getTestName(t *testing.T) string {
	v := reflect.ValueOf(t).Elem()
	name := v.FieldByName("name").String()
	return name
}

// we may could use + build go1.8 on different files but
// let's keep the only the old way using reflection, which does the same,
// to be aligned with everything.
//
// getTestName for go release version >= 1.8
// func getTestName(t *testing.T) string {
// 	return t.Name()
// }
