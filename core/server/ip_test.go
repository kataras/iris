package server

import (
	"net"
	"testing"
)

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

// TestIPInRange checks if IPInRange function correctly identifies IPs within a range.
func TestIPInRange(t *testing.T) {
	tests := []struct {
		ipRange   IPRange
		ipAddress string
		want      bool
	}{
		{IPRange{Start: "192.168.1.1", End: "192.168.1.10"}, "192.168.1.5", true},
		{IPRange{Start: "192.168.1.1", End: "192.168.1.10"}, "192.168.1.11", false},
	}

	for _, tt := range tests {
		t.Run(tt.ipAddress, func(t *testing.T) {
			ip := net.ParseIP(tt.ipAddress)
			if got := IPInRange(tt.ipRange, ip); got != tt.want {
				t.Errorf("IPInRange(%v, %v) = %v, want %v", tt.ipRange, ip, got, tt.want)
			}
		})
	}
}

// TestIPIsPrivateSubnet checks if IPIsPrivateSubnet function correctly identifies private subnets.
func TestIPIsPrivateSubnet(t *testing.T) {
	privateRanges := []IPRange{
		{Start: "192.168.0.0", End: "192.168.255.255"},
		{Start: "10.0.0.0", End: "10.255.255.255"},
	}

	tests := []struct {
		ipAddress string
		want      bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ipAddress, func(t *testing.T) {
			ip := net.ParseIP(tt.ipAddress)
			if got := IPIsPrivateSubnet(ip, privateRanges); got != tt.want {
				t.Errorf("IPIsPrivateSubnet(%v, %v) = %v, want %v", ip, privateRanges, got, tt.want)
			}
		})
	}
}

// TestGetIPAddress checks if GetIPAddress function correctly identifies valid public IP addresses.
func TestGetIPAddress(t *testing.T) {
	privateRanges := []IPRange{
		{Start: "192.168.0.0", End: "192.168.255.255"},
		{Start: "10.0.0.0", End: "10.255.255.255"},
	}

	tests := []struct {
		ipAddresses   []string
		privateRanges []IPRange
		want          string
		wantFound     bool
	}{
		{[]string{"192.168.1.1", "8.8.8.8"}, privateRanges, "8.8.8.8", true},
		{[]string{"192.168.1.1", "10.0.0.1"}, privateRanges, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got, gotFound := GetIPAddress(tt.ipAddresses, tt.privateRanges)
			if got != tt.want || gotFound != tt.wantFound {
				t.Errorf("GetIPAddress(%v, %v) = %v, %v, want %v, %v", tt.ipAddresses, tt.privateRanges, got, gotFound, tt.want, tt.wantFound)
			}
		})
	}
}

// TestIPRangeEqual checks if IPRangeEqual function correctly compares two IPRange slices.
func TestIPRangeEqual(t *testing.T) {
	tests := []struct {
		a    []IPRange
		b    []IPRange
		want bool
	}{
		{[]IPRange{{Start: "192.168.1.1", End: "192.168.1.10"}}, []IPRange{{Start: "192.168.1.1", End: "192.168.1.10"}}, true},
		{[]IPRange{{Start: "192.168.1.1", End: "192.168.1.10"}}, []IPRange{{Start: "192.168.1.1", End: "192.168.1.11"}}, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := IPRangeEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("IPRangeEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
