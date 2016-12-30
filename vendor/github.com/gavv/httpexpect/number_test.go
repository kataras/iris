package httpexpect

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumberFailed(t *testing.T) {
	chain := makeChain(newMockReporter(t))

	chain.fail("fail")

	value := &Number{chain, 0}

	value.chain.assertFailed(t)

	value.Path("$").chain.assertFailed(t)
	value.Schema("")

	value.Equal(0)
	value.NotEqual(0)
	value.Gt(0)
	value.Ge(0)
	value.Lt(0)
	value.Le(0)
	value.InRange(0, 0)
}

func TestNumberGetters(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 123.0)

	assert.Equal(t, 123.0, value.Raw())
	value.chain.assertOK(t)
	value.chain.reset()

	assert.Equal(t, 123.0, value.Path("$").Raw())
	value.chain.assertOK(t)
	value.chain.reset()

	value.Schema(`{"type": "number"}`)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Schema(`{"type": "object"}`)
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestNumberEqual(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234)

	assert.Equal(t, 1234, int(value.Raw()))

	value.Equal(1234)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Equal(4321)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.NotEqual(4321)
	value.chain.assertOK(t)
	value.chain.reset()

	value.NotEqual(1234)
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestNumberEqualDelta(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234.5)

	value.EqualDelta(1234.3, 0.3)
	value.chain.assertOK(t)
	value.chain.reset()

	value.EqualDelta(1234.7, 0.3)
	value.chain.assertOK(t)
	value.chain.reset()

	value.EqualDelta(1234.3, 0.1)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.EqualDelta(1234.7, 0.1)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.NotEqualDelta(1234.3, 0.3)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.NotEqualDelta(1234.7, 0.3)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.NotEqualDelta(1234.3, 0.1)
	value.chain.assertOK(t)
	value.chain.reset()

	value.NotEqualDelta(1234.7, 0.1)
	value.chain.assertOK(t)
	value.chain.reset()
}

func TestNumberEqualNaN(t *testing.T) {
	reporter := newMockReporter(t)

	v1 := NewNumber(reporter, math.NaN())
	v1.Equal(1234.5)
	v1.chain.assertFailed(t)

	v2 := NewNumber(reporter, 1234.5)
	v2.Equal(math.NaN())
	v2.chain.assertFailed(t)

	v3 := NewNumber(reporter, math.NaN())
	v3.EqualDelta(1234.0, 0.1)
	v3.chain.assertFailed(t)

	v4 := NewNumber(reporter, 1234.5)
	v4.EqualDelta(math.NaN(), 0.1)
	v4.chain.assertFailed(t)

	v5 := NewNumber(reporter, 1234.5)
	v5.EqualDelta(1234.5, math.NaN())
	v5.chain.assertFailed(t)

	v6 := NewNumber(reporter, math.NaN())
	v6.NotEqualDelta(1234.0, 0.1)
	v6.chain.assertFailed(t)

	v7 := NewNumber(reporter, 1234.5)
	v7.NotEqualDelta(math.NaN(), 0.1)
	v7.chain.assertFailed(t)

	v8 := NewNumber(reporter, 1234.5)
	v8.NotEqualDelta(1234.5, math.NaN())
	v8.chain.assertFailed(t)
}

func TestNumberGreater(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234)

	value.Gt(1234 - 1)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Gt(1234)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.Ge(1234 - 1)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Ge(1234)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Ge(1234 + 1)
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestNumberLesser(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234)

	value.Lt(1234 + 1)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Lt(1234)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.Le(1234 + 1)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Le(1234)
	value.chain.assertOK(t)
	value.chain.reset()

	value.Le(1234 - 1)
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestNumberInRange(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234)

	value.InRange(1234, 1234)
	value.chain.assertOK(t)
	value.chain.reset()

	value.InRange(1234-1, 1234)
	value.chain.assertOK(t)
	value.chain.reset()

	value.InRange(1234, 1234+1)
	value.chain.assertOK(t)
	value.chain.reset()

	value.InRange(1234+1, 1234+2)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.InRange(1234-2, 1234-1)
	value.chain.assertFailed(t)
	value.chain.reset()

	value.InRange(1234+1, 1234-1)
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestNumberConvertEqual(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234)

	value.Equal(int64(1234))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Equal(float32(1234))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Equal("1234")
	value.chain.assertFailed(t)
	value.chain.reset()

	value.NotEqual(int64(4321))
	value.chain.assertOK(t)
	value.chain.reset()

	value.NotEqual(float32(4321))
	value.chain.assertOK(t)
	value.chain.reset()

	value.NotEqual("4321")
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestNumberConvertGreater(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234)

	value.Gt(int64(1233))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Gt(float32(1233))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Gt("1233")
	value.chain.assertFailed(t)
	value.chain.reset()

	value.Ge(int64(1233))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Ge(float32(1233))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Ge("1233")
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestNumberConvertLesser(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234)

	value.Lt(int64(1235))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Lt(float32(1235))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Lt("1235")
	value.chain.assertFailed(t)
	value.chain.reset()

	value.Le(int64(1235))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Le(float32(1235))
	value.chain.assertOK(t)
	value.chain.reset()

	value.Le("1235")
	value.chain.assertFailed(t)
	value.chain.reset()
}

func TestNumberConvertInRange(t *testing.T) {
	reporter := newMockReporter(t)

	value := NewNumber(reporter, 1234)

	value.InRange(int64(1233), float32(1235))
	value.chain.assertOK(t)
	value.chain.reset()

	value.InRange(int64(1233), "1235")
	value.chain.assertFailed(t)
	value.chain.reset()

	value.InRange(nil, 1235)
	value.chain.assertFailed(t)
	value.chain.reset()
}
