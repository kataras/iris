package httpexpect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBooleanFailed(t *testing.T) {
	chain := makeChain(newMockReporter(t))

	chain.fail("fail")

	value := &Boolean{chain, false}

	value.chain.assertFailed(t)

	value.Path("$").chain.assertFailed(t)
	value.Schema("")

	value.Equal(false)
	value.NotEqual(false)
	value.True()
	value.False()
}

func TestBooleanGetters(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewBoolean(reporter, true)

	assert.Equal(t, true, value.Raw())
	value.chain.assertOK(t)
	value.chain.reset()

	assert.Equal(t, true, value.Path("$").Raw())
	value.chain.assertOK(t)
	value.chain.reset()

	value.Schema(`{"type": "boolean"}`)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Schema(`{"type": "object"}`)
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestBooleanTrue(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewBoolean(reporter, true)

	assert.Equal(t, true, value.Raw())

	value.Equal(true)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Equal(false)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.NotEqual(false)
	value.chain.assertOK(t)
	value.chain.reset()

	value.NotEqual(true)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.True()
	value.chain.assertOK(t)
	value.chain.reset()

	value.False()
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestBooleanFalse(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewBoolean(reporter, false)

	assert.Equal(t, false, value.Raw())

	value.Equal(true)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.Equal(false)
	value.chain.assertOK(t)
	value.chain.reset()

	value.NotEqual(false)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.NotEqual(true)
	value.chain.assertOK(t)
	value.chain.reset()

	value.True()
	value.chain.assertFailed(t)
	value.chain.reset()

	value.False()
	value.chain.assertOK(t)
	value.chain.reset()
}
