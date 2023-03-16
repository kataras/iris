package netutil

import (
	"bytes"
	"net"
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
// Reports whether a valid IP was found.
func GetIPAddress(ipAddresses []string, privateRanges []IPRange) (string, bool) {
	// march from right to left until we get a public address
	// that will be the address right before our proxy.
	for i := len(ipAddresses) - 1; i >= 0; i-- {
		ip := strings.TrimSpace(ipAddresses[i])
		realIP := net.ParseIP(ip)
		if !realIP.IsGlobalUnicast() || IPIsPrivateSubnet(realIP, privateRanges) {
			// bad address, go to next
			continue
		}
		return ip, true

	}

	return "", false
}
