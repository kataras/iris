package netutil

import (
	"net"
	"testing"
)

func TestIP(t *testing.T) {
	privateRanges := []IPRange{
		{
			Start: net.ParseIP("10.0.0.0"),
			End:   net.ParseIP("10.255.255.255"),
		},
		{
			Start: net.ParseIP("100.64.0.0"),
			End:   net.ParseIP("100.127.255.255"),
		},
		{
			Start: net.ParseIP("172.16.0.0"),
			End:   net.ParseIP("172.31.255.255"),
		},
		{
			Start: net.ParseIP("192.0.0.0"),
			End:   net.ParseIP("192.0.0.255"),
		},
		{
			Start: net.ParseIP("192.168.0.0"),
			End:   net.ParseIP("192.168.255.255"),
		},
		{
			Start: net.ParseIP("198.18.0.0"),
			End:   net.ParseIP("198.19.255.255"),
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
}
