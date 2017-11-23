package mvc2

import (
	"reflect"
	"testing"
)

type (
	testService interface {
		say(string)
	}
	testServiceImpl struct {
		prefix string
	}
)

func (s *testServiceImpl) say(message string) string {
	return s.prefix + ": " + message
}

func TestMakeServiceInputBinder(t *testing.T) {
	expectedService := &testServiceImpl{"say"}
	b := MustMakeServiceInputBinder(expectedService)
	// in
	var (
		intType          = reflect.TypeOf(1)
		availableBinders = []*InputBinder{b}
	)

	// 1
	testCheck(t, "test1", true, testGetBindersForInput(t, availableBinders,
		[]interface{}{expectedService}, reflect.TypeOf(expectedService)))
	// 2
	testCheck(t, "test2-fail", false, testGetBindersForInput(t, availableBinders,
		[]interface{}{42}))
	// 3
	testCheck(t, "test3-fail", false, testGetBindersForInput(t, availableBinders,
		[]interface{}{42}, intType))
	// 4
	testCheck(t, "test4-fail", false, testGetBindersForInput(t, availableBinders,
		[]interface{}{42}))
	// 5 - check if nothing passed, so no valid binders at all.
	testCheck(t, "test5", true, testGetBindersForInput(t, availableBinders,
		[]interface{}{}))

}
