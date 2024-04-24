package basicauth

import "testing"

func TestHeaderEncode(t *testing.T) {
	var tests = []struct {
		username string
		password string
		header   string
		ok       bool
	}{
		{
			username: "user",
			password: "pass",
			header:   "Basic dXNlcjpwYXNz",
			ok:       true,
		},
		{
			username: "user",
			password: "p:(notallowed)ass",
			header:   "",
			ok:       false,
		},
		{
			username: "123u%ser",
			password: "pass132$",
			header:   "Basic MTIzdSVzZXI6cGFzczEzMiQ=",
			ok:       true,
		},
	}

	for i, tt := range tests {
		got, ok := encodeHeader(tt.username, tt.password)
		if tt.ok != ok {
			t.Fatalf("[%d] expected: %v but got: %v (username=%s,password=%s)", i, tt.ok, ok, tt.username, tt.password)
		}
		if tt.header != got {
			t.Fatalf("[%d] expected result header: %q but got: %q", i, tt.header, got)
		}
	}
}

func TestHeaderDecode(t *testing.T) {
	var tests = []struct {
		header   string
		ok       bool
		username string
		password string
	}{
		{
			header:   "Basic dXNlcjpwYXNz",
			ok:       true,
			username: "user",
			password: "pass",
		},
		{
			header: "dXNlcjpwYXNz",
			ok:     false,
		},
		{
			header: "Basic ",
			ok:     false,
		},
		{
			header: "Basic dXNlcjp",
			ok:     false,
		},
		{
			header: "dXNlcjpwYXNz Basic",
			ok:     false,
		},
		{
			header: "dXNlcjpwYXNzBasic",
			ok:     false,
		},
	}

	for i, tt := range tests {
		fullUser, username, password, ok := decodeHeader(tt.header)
		if expected, got := tt.ok, ok; expected != got {
			t.Fatalf("[%d] expected: %v but got: %v (header=%s)", i, expected, got, tt.header)
		}

		if expected, got := tt.username, username; expected != got {
			t.Fatalf("[%d] expected username: %q but got: %q", i, expected, got)
		}

		if expected, got := tt.password, password; expected != got {
			t.Fatalf("[%d] expected password: %q but got: %q", i, expected, got)
		}

		if tt.username != "" || tt.password != "" {
			if expected, got := tt.username+colonLiteral+tt.password, fullUser; expected != got {
				t.Fatalf("[%d] expected username:password to be: %q but got: %q", i, expected, got)
			}
		} else {
			if fullUser != "" {
				t.Fatalf("[%d] expected username:password to be empty but got: %q", i, fullUser)
			}
		}

	}
}
