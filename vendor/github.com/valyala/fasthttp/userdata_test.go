package fasthttp

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUserData(t *testing.T) {
	var u userData

	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key_%d", i))
		u.SetBytes(key, i+5)
		testUserDataGet(t, &u, key, i+5)
		u.SetBytes(key, i)
		testUserDataGet(t, &u, key, i)
	}

	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key_%d", i))
		testUserDataGet(t, &u, key, i)
	}

	u.Reset()

	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key_%d", i))
		testUserDataGet(t, &u, key, nil)
	}
}

func testUserDataGet(t *testing.T, u *userData, key []byte, value interface{}) {
	v := u.GetBytes(key)
	if v == nil && value != nil {
		t.Fatalf("cannot obtain value for key=%q", key)
	}
	if !reflect.DeepEqual(v, value) {
		t.Fatalf("unexpected value for key=%q: %d. Expecting %d", key, v, value)
	}
}

func TestUserDataValueClose(t *testing.T) {
	var u userData

	closeCalls := 0

	// store values implementing io.Closer
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		u.Set(key, &closerValue{&closeCalls})
	}

	// store values without io.Closer
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key_noclose_%d", i)
		u.Set(key, i)
	}

	u.Reset()

	if closeCalls != 5 {
		t.Fatalf("unexpected number of Close calls: %d. Expecting 10", closeCalls)
	}
}

type closerValue struct {
	closeCalls *int
}

func (cv *closerValue) Close() error {
	(*cv.closeCalls)++
	return nil
}
