package httpexpect

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCookieFailed(t *testing.T) {
	chain := makeChain(newMockReporter(t))

	chain.fail("fail")

	value := &Cookie{chain, nil}

	assert.True(t, value.Raw() == nil)
	assert.True(t, value.Name() != nil)
	assert.True(t, value.Value() != nil)
	assert.True(t, value.Domain() != nil)
	assert.True(t, value.Path() != nil)
	assert.True(t, value.Expires() != nil)
}

func TestCookieGetters(t *testing.T) {
	reporter := newMockReporter(t)

	NewCookie(reporter, nil).chain.assertFailed(t)

	value := NewCookie(reporter, &http.Cookie{
		Name:    "name",
		Value:   "value",
		Domain:  "example.com",
		Path:    "/path",
		Expires: time.Unix(1234, 0),
	})

	value.chain.assertOK(t)

	value.Name().chain.assertOK(t)
	value.Value().chain.assertOK(t)
	value.Domain().chain.assertOK(t)
	value.Path().chain.assertOK(t)
	value.Expires().chain.assertOK(t)

	assert.Equal(t, "name", value.Name().Raw())
	assert.Equal(t, "value", value.Value().Raw())
	assert.Equal(t, "example.com", value.Domain().Raw())
	assert.Equal(t, "/path", value.Path().Raw())
	assert.True(t, time.Unix(1234, 0).Equal(value.Expires().Raw()))

	value.chain.assertOK(t)
}
