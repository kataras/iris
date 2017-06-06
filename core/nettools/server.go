// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nettools

import (
	"net/http"
)

// used on host/supervisor/task and router/path

// IsTLS returns true if the "srv" contains any certificates
// or a get certificate function, meaning that is secure.
func IsTLS(srv *http.Server) bool {
	if cfg := srv.TLSConfig; cfg != nil &&
		(len(cfg.Certificates) > 0 || cfg.GetCertificate != nil) {
		return true
	}

	return false
}

// ResolveSchemeFromServer tries to resolve a url scheme
// based on the server's configuration.
// Returns "https" on secure server,
// otherwise "http".
func ResolveSchemeFromServer(srv *http.Server) string {
	if IsTLS(srv) {
		return SchemeHTTPS
	}

	return SchemeHTTP
}

// ResolveURLFromServer returns the scheme+host from a server.
func ResolveURLFromServer(srv *http.Server) string {
	scheme := ResolveSchemeFromServer(srv)
	host := ResolveVHost(srv.Addr)
	return scheme + "://" + host
}
