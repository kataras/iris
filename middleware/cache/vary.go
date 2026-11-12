package cache

import "strings"

// VaryBuilder helps to build the Vary header value.
type VaryBuilder struct {
	directives []string
}

// NewVaryBuilder creates a new VaryBuilder.
func NewVaryBuilder() *VaryBuilder {
	return &VaryBuilder{}
}

// Add appends a directive to the Vary header value.
func (b *VaryBuilder) Add(directive string) *VaryBuilder {
	b.directives = append(b.directives, directive)
	return b
}

// Accept sets the Vary directive to 'Accept'.
func (b *VaryBuilder) Accept() *VaryBuilder {
	return b.Add("Accept")
}

// AcceptEncoding sets the Vary directive to 'Accept-Encoding'.
func (b *VaryBuilder) AcceptEncoding() *VaryBuilder {
	return b.Add("Accept-Encoding")
}

// AcceptLanguage sets the Vary directive to 'Accept-Language'.
func (b *VaryBuilder) AcceptLanguage() *VaryBuilder {
	return b.Add("Accept-Language")
}

// UserAgent sets the Vary directive to 'User-Agent'.
func (b *VaryBuilder) UserAgent() *VaryBuilder {
	return b.Add("User-Agent")
}

// Build constructs the Vary header value from the added directives.
func (b *VaryBuilder) Build() string {
	return strings.Join(b.directives, ", ")
}
