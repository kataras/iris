package server

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// IsRootDomain reports whether the host or ip is a root domain.
func IsRootDomain(hostOrIP string) bool {
	if host, _, err := net.SplitHostPort(hostOrIP); err == nil {
		hostOrIP = host
	}

	if _, err := netip.ParseAddr(hostOrIP); err == nil { // if it's an IP, see #1945.
		return true
	}

	if rootDomain, err := publicsuffix.EffectiveTLDPlusOne(hostOrIP); err == nil {
		return rootDomain == hostOrIP
	}

	return hostOrIP == "localhost"
	// return false
}

// IsLoopback reports whether the host is a loopback address.
// It supports both IP addresses and hostnames.
// It returns true if the host is a loopback address, false otherwise.
// Note that the function might block while performing DNS queries.
func IsLoopback(hostOrIP string) bool {
	if host, _, err := net.SplitHostPort(hostOrIP); err == nil {
		hostOrIP = host
	}

	if addr, err := netip.ParseAddr(hostOrIP); err == nil {
		return addr.IsLoopback()
	}

	// If the input is not an IP address, it might be a hostname.
	// Resolve the hostname to IP addresses.
	// Note that the net.LookupIP function can potentially block while performing the DNS query,
	// so to consider this when using the function in a context where blocking is undesirable.
	ips, err := net.LookupIP(hostOrIP)
	if err != nil {
		// If there's an error resolving the hostname, it's not a loopback.
		return false
	}

	// Check if any of the resolved IP addresses is a loopback address.
	for _, ip := range ips {
		if addr, err := netip.ParseAddr(ip.String()); err == nil {
			if addr.IsLoopback() {
				return true
			}
		}
	}

	// If none of the IP addresses is a loopback, return false.
	return false
}

var loopbackHostnames = map[string]struct{}{
	"localhost":       {},
	"local":           {},
	"loopback":        {},
	"ip6-localhost":   {},
	"ip6-loopback":    {},
	"ipv6-localhost":  {},
	"ipv6-loopback":   {},
	"127.0.0.1":       {},
	"0.0.0.0":         {},
	"::1":             {},
	"[::1]":           {},
	"0:0:0:0:0:0:0:0": {},
	"0:0:0:0:0:0:0:1": {},
}

func isLoopbackMap(hostOrIP string) bool {
	_, ok := loopbackHostnames[hostOrIP]
	if !ok {
		return strings.HasSuffix(hostOrIP, ".local") || strings.HasSuffix(hostOrIP, ".localdomain") || strings.HasSuffix(hostOrIP, ".localhost")
	}
	return ok
}

// GetRootDomain returns the root domain of the given host.
var GetRootDomain = func(hostOrIP string) string {
	if host, _, err := net.SplitHostPort(hostOrIP); err == nil {
		hostOrIP = host
	}

	// Check if the input is a known loopback hostname.
	if isLoopbackMap(hostOrIP) { // this is used on request handlers, so we need a fast check instead of the IsLoopback (IsLoopback is slower and blocks for DNS queries).
		return hostOrIP
	}

	if _, err := netip.ParseAddr(hostOrIP); err == nil { // there's no root domain to extract, return the IP address.
		return hostOrIP
	}

	if rootDomain, err := publicsuffix.EffectiveTLDPlusOne(hostOrIP); err == nil {
		return rootDomain
	}

	return "" // the root domain cannot be determined.
}

// GetSubdomain returns the subdomain of the given host.
func GetSubdomain(hostOrIP string) string {
	// Extract the hostname from host:port if necessary.
	if host, _, err := net.SplitHostPort(hostOrIP); err == nil {
		hostOrIP = host
	}

	// Check if the input is an IP address.
	if _, err := netip.ParseAddr(hostOrIP); err == nil {
		return "" // IP addresses do not have subdomains.
	}

	// Extract the root domain using the publicsuffix package.
	rootDomain, err := publicsuffix.EffectiveTLDPlusOne(hostOrIP)
	if err != nil {
		return "" // Return the error if the root domain cannot be determined.
	}

	// Find the subdomain by removing the root domain from the host.
	subdomain := strings.TrimSuffix(hostOrIP, "."+rootDomain)

	// If there's no subdomain, return an empty string.
	if subdomain == hostOrIP {
		return ""
	}

	return subdomain
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
	// if loopbackToLocalhost && IsLoopback(host) {
	// 	host = "localhost"
	// }
	// No...let's use the simple map naming here.
	if isLoopbackMap(host) {
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
	// if IsLoopback(host) {
	// 	host = "localhost"
	// }
	// No...let's use the simple map naming here.
	if isLoopbackMap(host) {
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
