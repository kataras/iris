package context_test

import (
	"net/http"
	nethttptest "net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
)

// RemoteAddr with X-Forwarded-For trusted. The connection address of a
// net/http/httptest request is 192.0.2.1:1234.
func TestRemoteAddrForwardedFor(t *testing.T) {
	newApp := func(force bool) *iris.Application {
		app := iris.New(iris.Defaults(func(opts *iris.Options) {
			opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
			opts.Context.RemoteAddrHeadersForce = force
		}))
		app.Get("/", func(ctx iris.Context) {
			ctx.WriteString(ctx.RemoteAddr())
		})
		if err := app.Build(); err != nil {
			t.Fatal(err)
		}
		return app
	}

	remoteAddr := func(app *iris.Application, forwardedFor string) string {
		req := nethttptest.NewRequest(http.MethodGet, "/", nil)
		if forwardedFor != "" {
			req.Header.Set("X-Forwarded-For", forwardedFor)
		}
		rec := nethttptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Body.String()
	}

	const conn = "192.0.2.1"

	app := newApp(false)
	for _, tt := range []struct {
		name         string
		forwardedFor string
		want         string
	}{
		// unchanged since v14.0.0.
		{"no header", "", conn},
		{"right-most public", "1.2.3.4, 8.8.4.4", "8.8.4.4"},
		{"private on the right", "1.2.3.4, 8.8.4.4, 10.0.0.5", "8.8.4.4"},
		{"private only", "10.0.0.1, 192.168.1.1", conn},
		{"bare IPv6", "1.2.3.4, 2001:db8::1", "2001:db8::1"},
		// an entry with a port.
		{"IPv4 with a port", "1.2.3.4, 8.8.4.4:1234", "8.8.4.4"},
		{"IPv6 with a port", "1.2.3.4, [2001:db8::1]:443", "2001:db8::1"},
		{"private with a port", "1.2.3.4, 8.8.4.4, 10.0.0.5:8080", "8.8.4.4"},
		// not an address: the connection address, never the entry on its left.
		{"not an address", "1.2.3.4, unknown", conn},
		{"not an address between", "1.2.3.4, unknown, 10.0.0.5", conn},
	} {
		if got := remoteAddr(app, tt.forwardedFor); got != tt.want {
			t.Errorf("%s: RemoteAddr with %q = %q, want %q", tt.name, tt.forwardedFor, got, tt.want)
		}
	}

	// RemoteAddrHeadersForce takes the first entry of a header that has no
	// public address. See TestRemoteAddrHeadersForceNotAnAddress for a header
	// with an entry that is not an address.
	forced := newApp(true)
	for _, tt := range []struct {
		name         string
		forwardedFor string
		want         string
	}{
		{"private only", "10.0.0.1, 192.168.1.1", "10.0.0.1"},
		{"IPv4 with a port", "1.2.3.4, 8.8.4.4:1234", "8.8.4.4"},
	} {
		if got := remoteAddr(forced, tt.forwardedFor); got != tt.want {
			t.Errorf("force, %s: RemoteAddr with %q = %q, want %q", tt.name, tt.forwardedFor, got, tt.want)
		}
	}
}

// RemoteAddrHeadersForce does not apply to a header whose reading ended at an
// entry that is not an address: RemoteAddr then gives what it gives with the
// option off, the next header and at last the address of the connection. It
// still applies to a header whose entries are all addresses and none of them
// public.
func TestRemoteAddrHeadersForceNotAnAddress(t *testing.T) {
	newApp := func(force bool) *iris.Application {
		app := iris.New(iris.Defaults(func(opts *iris.Options) {
			opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For", "X-Real-Ip"}
			opts.Context.RemoteAddrHeadersForce = force
		}))
		app.Get("/", func(ctx iris.Context) {
			ctx.WriteString(ctx.RemoteAddr())
		})
		if err := app.Build(); err != nil {
			t.Fatal(err)
		}
		return app
	}

	remoteAddr := func(app *iris.Application, forwardedFor, realIP string) string {
		req := nethttptest.NewRequest(http.MethodGet, "/", nil)
		if forwardedFor != "" {
			req.Header.Set("X-Forwarded-For", forwardedFor)
		}
		if realIP != "" {
			req.Header.Set("X-Real-Ip", realIP)
		}
		rec := nethttptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Body.String()
	}

	const conn = "192.0.2.1"

	forced, notForced := newApp(true), newApp(false)
	for _, tt := range []struct {
		name         string
		forwardedFor string
		realIP       string
		want         string // with the option on.
		wantOff      string // with the option off.
	}{
		// a non-address ended the reading: the option changes nothing.
		{"not an address on the right", "6.6.6.6, 203.0.113.7, unknown", "", conn, conn},
		{"a unix socket on the right", "6.6.6.6, 203.0.113.7, unix:", "", conn, conn},
		{"an address with a zone on the right", "6.6.6.6, 203.0.113.7, fe80::1%eth0", "", conn, conn},
		{"not an address before a private one", "6.6.6.6, 203.0.113.7, unknown, 10.0.0.5", "", conn, conn},
		{"not an address left of a private one", "unknown, 10.0.0.5", "", conn, conn},
		{"not an address left of a private one, no space", "unknown,10.0.0.5", "", conn, conn},
		{"private left of not an address", "10.0.0.1, unknown", "", conn, conn},
		{"not an address and empty entries", "6.6.6.6, unknown, ,", "", conn, conn},
		{"not an address, then the next header", "1.2.3.4, unknown", "9.9.9.9", "9.9.9.9", "9.9.9.9"},
		// the option is decided for each header: it is off for the first one
		// and on for the second.
		{"not an address, then a private next header", "10.0.0.1, unknown", "10.0.0.9", "10.0.0.9", conn},
		{"private, then not an address in the next header", "10.0.0.1", "unknown", "10.0.0.1", conn},
		// all entries are addresses and none is public: the first one from the left.
		{"private only", "10.0.0.1, 10.0.0.2", "", "10.0.0.1", conn},
		{"private only with a port", "10.0.0.1:8080, 10.0.0.2", "", "10.0.0.1", conn},
		{"private only with an empty entry", "10.0.0.1, , 10.0.0.2", "", "10.0.0.1", conn},
		// a public address is the result with the option on or off.
		{"right-most public", "1.2.3.4, 8.8.4.4", "", "8.8.4.4", "8.8.4.4"},
		{"empty entries on the right", "1.2.3.4, 8.8.4.4, ,", "", "8.8.4.4", "8.8.4.4"},
		{"empty entry on the left", ", 8.8.4.4", "", "8.8.4.4", "8.8.4.4"},
		// the reading stops at the public address and never reaches the
		// entry on its left.
		{"not an address left of a public one", "unknown, 8.8.4.4", "", "8.8.4.4", "8.8.4.4"},
	} {
		if got := remoteAddr(forced, tt.forwardedFor, tt.realIP); got != tt.want {
			t.Errorf("force, %s: RemoteAddr with %q and %q = %q, want %q", tt.name, tt.forwardedFor, tt.realIP, got, tt.want)
		}
		if got := remoteAddr(notForced, tt.forwardedFor, tt.realIP); got != tt.wantOff {
			t.Errorf("no force, %s: RemoteAddr with %q and %q = %q, want %q", tt.name, tt.forwardedFor, tt.realIP, got, tt.wantOff)
		}
	}
}
