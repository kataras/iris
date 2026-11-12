package cache

import (
	"fmt"
	"strings"
)

// ProxyAuthenticateBuilder helps to build the Proxy-Authenticate header value.
type ProxyAuthenticateBuilder struct {
	schemes []string
}

// NewProxyAuthenticateBuilder creates a new ProxyAuthenticateBuilder.
func NewProxyAuthenticateBuilder() *ProxyAuthenticateBuilder {
	return &ProxyAuthenticateBuilder{}
}

// Basic sets the Proxy-Authenticate scheme to 'Basic'.
func (b *ProxyAuthenticateBuilder) Basic(realm string) *ProxyAuthenticateBuilder {
	b.schemes = append(b.schemes, fmt.Sprintf("Basic realm=\"%s\"", realm))
	return b
}

// Digest sets the Proxy-Authenticate scheme to 'Digest'.
func (b *ProxyAuthenticateBuilder) Digest(realm, nonce, opaque string) *ProxyAuthenticateBuilder {
	b.schemes = append(b.schemes, fmt.Sprintf("Digest realm=\"%s\", nonce=\"%s\", opaque=\"%s\"", realm, nonce, opaque))
	return b
}

// Build constructs the Proxy-Authenticate header value from the added schemes.
func (b *ProxyAuthenticateBuilder) Build() string {
	return strings.Join(b.schemes, ", ")
}
