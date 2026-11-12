package server

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/idna"
)

// Note that all hosts will be converted to Punycode via idna.Lookup.ToASCII so that
// Manager.GetCertificate can handle the Unicode IDN and mixedcase hosts correctly.
func newHostPolicy(hostPatterns []string) autocert.HostPolicy {
	exactMatches := make(map[string]struct{})
	wildcardMatches := make(map[string]struct{})

	// Convert all hosts to ASCII first.
	for i, pattern := range hostPatterns {
		if asciiHost, err := idna.Lookup.ToASCII(pattern); err == nil { // keeps subdomains and wildcard char (*).
			hostPatterns[i] = asciiHost
		}
	}

	for _, pattern := range hostPatterns {
		if strings.HasPrefix(pattern, "*.") {
			wildcardMatches[pattern[2:]] = struct{}{}
		} else {
			exactMatches[pattern] = struct{}{}
		}
	}

	// All hosts are static.
	hasWildcards := len(exactMatches) != len(hostPatterns)

	return func(ctx context.Context, host string) error {
		if _, ok := exactMatches[host]; ok {
			return nil
		}

		if hasWildcards { // search for wildcard matches.
			parts := strings.Split(host, ".")
			for i := 1; i < len(parts); i++ { // Start from 1 to ignore the first subdomain.
				subdomain := strings.Join(parts[i:], ".")
				if _, ok := wildcardMatches[subdomain]; ok {
					return nil
				}
			}
		}

		return fmt.Errorf("server: host %q not configured in HostWhitelist", host)
	}
}
