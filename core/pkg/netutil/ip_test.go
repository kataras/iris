package netutil

import (
	"strings"
	"testing"
)

// forwardedPrivateRanges are the ranges an application has by default
// (context.DefaultContextOptions).
func forwardedPrivateRanges() []IPRange {
	return []IPRange{
		{Start: "10.0.0.0", End: "10.255.255.255"},
		{Start: "100.64.0.0", End: "100.127.255.255"},
		{Start: "172.16.0.0", End: "172.31.255.255"},
		{Start: "192.0.0.0", End: "192.0.0.255"},
		{Start: "192.168.0.0", End: "192.168.255.255"},
		{Start: "198.18.0.0", End: "198.19.255.255"},
	}
}

type forwardedTest struct {
	name   string
	header string // an X-Forwarded-For value, split the way Context.RemoteAddr splits it.
	want   string
	found  bool
}

func runForwardedTests(t *testing.T, privateRanges []IPRange, tests []forwardedTest) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := strings.Split(strings.TrimSpace(tt.header), ",")
			got, found := GetIPAddress(entries, privateRanges)
			if got != tt.want || found != tt.found {
				t.Errorf("GetIPAddress(%q) = %q, %v, want %q, %v", tt.header, got, found, tt.want, tt.found)
			}
		})
	}
}

// What v14.0.0 returns for entries that are addresses without a port.
// None of these may change.
func TestGetIPAddressParsedEntries(t *testing.T) {
	runForwardedTests(t, forwardedPrivateRanges(), []forwardedTest{
		{"one public", "8.8.4.4", "8.8.4.4", true},
		{"right-most public", "1.2.3.4, 8.8.4.4", "8.8.4.4", true},
		{"no space after the comma", "1.2.3.4,8.8.4.4", "8.8.4.4", true},
		{"spaces around", "  1.2.3.4 ,\t8.8.4.4  ", "8.8.4.4", true},
		{"private on the right", "1.2.3.4, 8.8.4.4, 10.0.0.5", "8.8.4.4", true},
		{"every default range", "8.8.4.4, 10.0.0.5, 100.64.0.1, 172.16.0.1, 192.0.0.1, 192.168.1.1, 198.18.0.1", "8.8.4.4", true},
		{"private only", "10.0.0.1, 192.168.1.1", "", false},
		{"loopback", "8.8.4.4, 127.0.0.1", "8.8.4.4", true},
		{"IPv6 loopback", "8.8.4.4, ::1", "8.8.4.4", true},
		{"unspecified", "8.8.4.4, 0.0.0.0", "8.8.4.4", true},
		{"link-local", "8.8.4.4, 169.254.1.1", "8.8.4.4", true},
		{"IPv6 link-local", "8.8.4.4, fe80::1", "8.8.4.4", true},
		{"multicast", "8.8.4.4, 224.0.0.1", "8.8.4.4", true},
		{"broadcast", "8.8.4.4, 255.255.255.255", "8.8.4.4", true},
		{"bare IPv6", "1.2.3.4, 2001:db8::1", "2001:db8::1", true},
		{"bare IPv6 as written", "2001:DB8:0:0:0:0:0:1", "2001:DB8:0:0:0:0:0:1", true},
		{"IPv6 then private", "2001:db8::1, 10.0.0.5", "2001:db8::1", true},
		{"IPv4-mapped private", "8.8.4.4, ::ffff:10.0.0.5", "8.8.4.4", true},
		{"IPv4-mapped public as written", "1.2.3.4, ::ffff:8.8.4.4", "::ffff:8.8.4.4", true},
		{"header not sent", "", "", false},
		{"stray commas", "8.8.4.4, ,", "8.8.4.4", true},
		{"not an address left of the answer", "unknown, 8.8.4.4", "8.8.4.4", true},
	})

	// Without ranges a private address is a public one.
	runForwardedTests(t, nil, []forwardedTest{
		{"no ranges", "8.8.4.4, 10.0.0.5", "10.0.0.5", true},
	})

	if got, found := GetIPAddress(nil, forwardedPrivateRanges()); got != "" || found {
		t.Errorf("GetIPAddress(nil) = %q, %v, want \"\", false", got, found)
	}
}

// An entry with a port is the address before the port. v14.0.0 skipped such an
// entry, so "1.2.3.4, 8.8.4.4:1234" returned 1.2.3.4, which the client wrote.
func TestGetIPAddressEntryWithPort(t *testing.T) {
	runForwardedTests(t, forwardedPrivateRanges(), []forwardedTest{
		{"IPv4", "1.2.3.4, 8.8.4.4:1234", "8.8.4.4", true},
		{"IPv6", "1.2.3.4, [2001:db8::1]:443", "2001:db8::1", true},
		{"only entry", "8.8.4.4:1234", "8.8.4.4", true},
		{"IPv6 only entry", "[2001:db8::1]:443", "2001:db8::1", true},
		{"spaces around", "1.2.3.4,  8.8.4.4:1234 ", "8.8.4.4", true},
		{"left of a private one", "1.2.3.4, 8.8.4.4:1234, 10.0.0.5", "8.8.4.4", true},
		// a private or a loopback address stays one of our proxies with a port too.
		{"private", "1.2.3.4, 8.8.4.4, 10.0.0.5:8080", "8.8.4.4", true},
		{"private on both sides", "8.8.4.4, 10.0.0.5:8080, 192.168.1.1", "8.8.4.4", true},
		{"loopback", "8.8.4.4, 127.0.0.1:8080", "8.8.4.4", true},
		{"IPv6 loopback", "8.8.4.4, [::1]:8080", "8.8.4.4", true},
		{"private only", "10.0.0.1:80, 192.168.1.1:80", "", false},
	})
}

// An entry that is not an address, with or without a port, ends the walk with
// no result. v14.0.0 skipped it and returned an entry on its left, so the
// client chose the result.
func TestGetIPAddressNotAnAddress(t *testing.T) {
	runForwardedTests(t, forwardedPrivateRanges(), []forwardedTest{
		{"word", "1.2.3.4, unknown", "", false},
		{"word right of a public one", "1.2.3.4, 8.8.4.4, unknown", "", false},
		{"word between", "1.2.3.4, unknown, 10.0.0.5", "", false},
		{"host name with a port", "1.2.3.4, example.com:80", "", false},
		{"port only", "1.2.3.4, :80", "", false},
		{"two ports", "1.2.3.4, 8.8.4.4:1234:5", "", false},
		{"out of range", "1.2.3.4, 300.1.1.1", "", false},
		{"two addresses in one entry", "1.2.3.4, 8.8.4.4 8.8.8.8", "", false},
		{"only entry", "unknown", "", false},
	})
}

func TestIP(t *testing.T) {
	privateRanges := []IPRange{
		{
			Start: "10.0.0.0",
			End:   "10.255.255.255",
		},
		{
			Start: "100.64.0.0",
			End:   "100.127.255.255",
		},
		{
			Start: "172.16.0.0",
			End:   "172.31.255.255",
		},
		{
			Start: "192.0.0.0",
			End:   "192.0.0.255",
		},
		{
			Start: "192.168.0.0",
			End:   "192.168.255.255",
		},
		{
			Start: "198.18.0.0",
			End:   "198.19.255.255",
		},
	}

	addresses := []string{
		"201.37.138.59",
		"159.117.3.153",
		"166.192.97.84",
		"225.181.213.210",
		"124.50.84.134",
		"87.53.250.102",
		"106.79.33.62",
		"242.120.17.144",
		"131.179.101.254",
		"103.11.11.174",
		"115.97.0.114",
		"219.202.120.251",
		"37.72.123.120",
		"154.94.78.101",
		"126.105.144.250",
	}

	got, ok := GetIPAddress(addresses, privateRanges)
	if !ok {
		t.Logf("expected addr to be matched")
	}

	if expected := "126.105.144.250"; expected != got {
		t.Logf("expected addr to be found: %s but got: %s", expected, got)
	}

	addresses = []string{
		"10.10.233.1",
		"126.105.144.250",
		"192.168.99.33",
		"172.18.22.23",
		"10.0.0.0",
		"10.255.255.255",
	}

	got, ok = GetIPAddress(addresses, privateRanges)
	if !ok {
		t.Logf("expected addr to be matched")
	}

	if expected := "126.105.144.250"; expected != got {
		t.Logf("expected addr to be found: %s but got: %s", expected, got)
	}

	addresses = []string{
		"10.0.0.0",
		"10.10.233.1",
		"192.168.99.33",
		"172.18.22.23",
	}

	_, ok = GetIPAddress(addresses, privateRanges)
	if ok {
		t.Logf("expected addr to not be matched")
	}
}
