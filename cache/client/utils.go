// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"net/http"
	"time"

	"github.com/kataras/iris/cache/entry"
)

// GetMaxAge parses the "Cache-Control" header
// and returns a LifeChanger which can be passed
// to the response's Reset
func GetMaxAge(r *http.Request) entry.LifeChanger {
	return func() time.Duration {
		cacheControlHeader := r.Header.Get("Cache-Control")
		// headerCacheDur returns the seconds
		headerCacheDur := entry.ParseMaxAge(cacheControlHeader)
		return time.Duration(headerCacheDur) * time.Second
	}
}
