package netutil

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// isLocalAddress checks if the host is a local address.
func isLocalAddress(host string) bool {
	return host == "127.0.0.1" || host == "0.0.0.0" || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "172.") || strings.HasSuffix(host, ".local")
}

// SplitURL splits the URL into scheme, host, and port.
func SplitURL(inputURL string) (scheme, host string, port int, err error) {
	// Prepend scheme if missing for special addresses
	if strings.HasPrefix(inputURL, "localhost") || strings.HasPrefix(inputURL, "127.0.0.1") || strings.HasPrefix(inputURL, "0.0.0.0") || strings.Contains(inputURL, ".local") {
		if !strings.Contains(inputURL, "://") {
			inputURL = "http://" + inputURL
		}
	}

	u, err := url.Parse(inputURL)
	if err != nil {
		return "", "", 0, err
	}

	scheme = u.Scheme
	host = u.Hostname()

	// Convert local addresses to localhost
	if isLocalAddress(host) {
		host = "localhost"
	}

	// Determine the port
	portStr := u.Port()
	if portStr == "" {
		switch scheme {
		case "https":
			port = 443
		case "http":
			port = 80
		default:
			port = 0 // Unknown scheme
		}
	} else {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "", "", 0, err
		}
	}

	// Special handling for localhost with HTTPS
	if host == "localhost" && port == 443 {
		scheme = "https"
	}

	return scheme, host, port, nil
}

// JoinURL constructs a URL from the given parts.
func JoinURL(scheme, host string, port int) string {
	// Convert local addresses to localhost
	if isLocalAddress(host) {
		host = "localhost"
	}

	// Omit the port if it's the default for the given scheme.
	if (scheme == "https" && port == 443) || (scheme == "http" && port == 80) {
		port = 0
	}

	// Construct the URL
	if port > 0 {
		return fmt.Sprintf("%s://%s:%d", scheme, host, port)
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}
