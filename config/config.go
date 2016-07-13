// Package config defines the default settings and semantic variables
package config

import (
	"time"
)

var (
	// Charset character encoding for template rendering
	Charset = "UTF-8"
)

var (
	// TimeFormat default time format for any kind of datetime parsing
	TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
	// StaticCacheDuration expiration duration for INACTIVE file handlers
	StaticCacheDuration = 20 * time.Second
	// CompressedFileSuffix is the suffix to add to the name of
	// cached compressed file when using the .StaticFS function.
	//
	// Defaults to iris-fasthttp.gz
	CompressedFileSuffix = "iris-fasthttp.gz"

	// ContentTypeHTML defaults to text/html but you can change it, changes the template's content type also
	ContentTypeHTML = "text/html"
)

const (
	// NoLayout to disable layout for a particular template file
	NoLayout = "@.|.@iris_no_layout@.|.@"
	// TemplateLayoutContextKey is the name of the user values which can be used to set a template layout from a middleware and override the parent's
	TemplateLayoutContextKey = "templateLayout"
)
