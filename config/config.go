// Package config defines the default settings and semantic variables
package config

import (
	"time"
)

var (
	// StaticCacheDuration expiration duration for INACTIVE file handlers
	StaticCacheDuration = 20 * time.Second
	// CompressedFileSuffix is the suffix to add to the name of
	// cached compressed file when using the .StaticFS function.
	//
	// Defaults to iris-fasthttp.gz
	CompressedFileSuffix = "iris-fasthttp.gz"
)
