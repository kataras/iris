package httpexpect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonNumber(t *testing.T) {
	type (
		myInt int
	)

	chain := makeChain(newMockReporter(t))

	d1, ok := canonNumber(&chain, 123)
	assert.True(t, ok)
	assert.Equal(t, 123.0, d1)
	chain.assertOK(t)
	chain.reset()

	d2, ok := canonNumber(&chain, 123.0)
	assert.True(t, ok)
	assert.Equal(t, 123.0, d2)
	chain.assertOK(t)
	chain.reset()

	d3, ok := canonNumber(&chain, myInt(123))
	assert.True(t, ok)
	assert.Equal(t, 123.0, d3)
	chain.assertOK(t)
	chain.reset()

	_, ok = canonNumber(&chain, "123")
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()

	_, ok = canonNumber(&chain, nil)
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()
}

func TestCanonArray(t *testing.T) {
	type (
		myArray []interface{}
		myInt   int
	)

	chain := makeChain(newMockReporter(t))

	d1, ok := canonArray(&chain, []interface{}{123.0, 456.0})
	assert.True(t, ok)
	assert.Equal(t, []interface{}{123.0, 456.0}, d1)
	chain.assertOK(t)
	chain.reset()

	d2, ok := canonArray(&chain, myArray{myInt(123), 456.0})
	assert.True(t, ok)
	assert.Equal(t, []interface{}{123.0, 456.0}, d2)
	chain.assertOK(t)
	chain.reset()

	_, ok = canonArray(&chain, "123")
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()

	_, ok = canonArray(&chain, func() {})
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()

	_, ok = canonArray(&chain, nil)
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()

	_, ok = canonArray(&chain, []interface{}(nil))
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()
}

func TestCanonMap(t *testing.T) {
	type (
		myMap map[string]interface{}
		myInt int
	)

	chain := makeChain(newMockReporter(t))

	d1, ok := canonMap(&chain, map[string]interface{}{"foo": 123.0})
	assert.True(t, ok)
	assert.Equal(t, map[string]interface{}{"foo": 123.0}, d1)
	chain.assertOK(t)
	chain.reset()

	d2, ok := canonMap(&chain, myMap{"foo": myInt(123)})
	assert.True(t, ok)
	assert.Equal(t, map[string]interface{}{"foo": 123.0}, d2)
	chain.assertOK(t)
	chain.reset()

	_, ok = canonMap(&chain, "123")
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()

	_, ok = canonMap(&chain, func() {})
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()

	_, ok = canonMap(&chain, nil)
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()

	_, ok = canonMap(&chain, map[string]interface{}(nil))
	assert.False(t, ok)
	chain.assertFailed(t)
	chain.reset()
}

func TestDiffErrors(t *testing.T) {
	na := " (unavailable)"

	assert.Equal(t, na, diffValues(map[string]interface{}{}, []interface{}{}))
	assert.Equal(t, na, diffValues([]interface{}{}, map[string]interface{}{}))
	assert.Equal(t, na, diffValues("foo", "bar"))
	assert.Equal(t, na, diffValues(func() {}, func() {}))

	assert.NotEqual(t, na, diffValues(map[string]interface{}{}, map[string]interface{}{}))
	assert.NotEqual(t, na, diffValues([]interface{}{}, []interface{}{}))
}
