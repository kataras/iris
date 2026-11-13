package netutil

import (
	"bytes"
	"net"
	"slices"
	"strings"
)

/* Based on:
https://husobee.github.io/golang/ip-address/2015/12/17/remote-ip-go.html requested at:
https://github.com/kataras/iris/issues/1453
*/

// IPRange is a structure that holds the start and end of a range of IP Addresses.
type IPRange struct {
	Start string `ini:"start" json:"start" yaml:"Start" toml:"Start"`
	End   string `ini:"end" json:"end" yaml:"End" toml:"End"`
}

// IPInRange reports whether a given IP Address is within a range given.
func IPInRange(r IPRange, ipAddress net.IP) bool {
	return bytes.Compare(ipAddress, net.ParseIP(r.Start)) >= 0 && bytes.Compare(ipAddress, net.ParseIP(r.End)) <= 0
}

// IPIsPrivateSubnet reports whether this "ipAddress" is in a private subnet.
func IPIsPrivateSubnet(ipAddress net.IP, privateRanges []IPRange) bool {
	// IPv4 for now.
	if ipCheck := ipAddress.To4(); ipCheck != nil {
		// iterate over all our ranges.
		for _, r := range privateRanges {
			// check if this ip is in a private range.
			if IPInRange(r, ipAddress) {
				return true
			}
		}
	}
	return false
}

// GetIPAddress returns a valid public IP Address from a collection of IP Addresses
// and a range of private subnets.
//
// The collection is the value of a header such as X-Forwarded-For split on the
// comma. Each proxy appends the address it got the request from, so the entry
// on the right is the closest to the server and the entries on the left are
// whatever the client sent. The walk goes from right to left, and an entry is:
//
//   - a public address: the result. A port, as in "8.8.4.4:1234" or
//     "[2001:db8::1]:443", is left out of it.
//   - an address in "privateRanges", or one that is not a global unicast
//     address (a loopback one for example), with or without a port: one of
//     our own proxies, skipped.
//   - empty (the header was not sent, or a comma has nothing next to it): not
//     an entry, skipped.
//   - anything else: the end of the walk, with no result. An entry that is not
//     an address cannot be taken as one of our proxies, so it is never skipped
//     for an entry on its left, which the client may have written.
//
// Reports whether a valid IP was found.
func GetIPAddress(ipAddresses []string, privateRanges []IPRange) (string, bool) {
	// march from right to left until we get a public address
	// that will be the address right before our proxy.
	for _, addr := range slices.Backward(ipAddresses) {
		ip := strings.TrimSpace(addr)
		if ip == "" {
			continue
		}

		// a bare address first: an IPv6 one has colons and no port.
		realIP := net.ParseIP(ip)
		if realIP == nil {
			if host, _, err := net.SplitHostPort(ip); err == nil {
				ip, realIP = host, net.ParseIP(host)
			}
		}

		if realIP == nil {
			// not an address, stop here.
			return "", false
		}

		if !realIP.IsGlobalUnicast() || IPIsPrivateSubnet(realIP, privateRanges) {
			// bad address, go to next
			continue
		}
		return ip, true

	}

	return "", false
}

// IPRangeEqual reports whether two IPRange slices are equal.
func IPRangeEqual(a, b []IPRange) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}
