package httpexpect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChainFail(t *testing.T) {
	chain := makeChain(newMockReporter(t))

	assert.False(t, chain.failed())

	chain.fail("fail")
	assert.True(t, chain.failed())

	chain.fail("fail")
	assert.True(t, chain.failed())
}

func TestChainCopy(t *testing.T) {
	chain1 := makeChain(newMockReporter(t))
	chain2 := chain1

	assert.False(t, chain1.failed())
	assert.False(t, chain2.failed())

	chain1.fail("fail")

	assert.True(t, chain1.failed())
	assert.False(t, chain2.failed())

	chain2.fail("fail")

	assert.True(t, chain1.failed())
	assert.True(t, chain2.failed())
}

func TestChainReport(t *testing.T) {
	r0 := newMockReporter(t)

	chain := makeChain(r0)

	r1 := newMockReporter(t)

	chain.assertOK(r1)
	assert.False(t, r1.reported)

	chain.assertFailed(r1)
	assert.True(t, r1.reported)

	assert.False(t, chain.failed())

	chain.fail("fail")
	assert.True(t, r0.reported)

	r2 := newMockReporter(t)

	chain.assertFailed(r2)
	assert.False(t, r2.reported)

	chain.assertOK(r2)
	assert.True(t, r2.reported)
}
