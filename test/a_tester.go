//Package test | cd $GOPATH/src/github.com/kataras/iris/test && go test -v
package test

import (
	"net/http"
	"strconv"

	"testing"

	"github.com/gavv/httpexpect"
	"github.com/kataras/iris"
)

// Configuration
const (
	scheme = "http://"
	domain = "mydomain.com"
	port   = 8080 // this will go as test flag some day.

	// will start  the server to real listen , this is useful ONLY WHEN TEST (AND) SUBDOMAINS,
	// the hosts file (on windows) must be setted as '127.0.0.1 mydomain.com' & '127.0.0.1 mysubdomain.mydomain.com'
	enable_subdomain_tests = false // this will go as test flag some day also.
	subdomain              = "mysubdomain"
	enable_debug           = false // this will go as test flag some day also.
)

var (
	host          = domain + ":" + strconv.Itoa(port)
	subdomainHost = subdomain + domain + "." + host
	hostURL       = scheme + host
	subdomainURL  = scheme + subdomainHost
)

// Prepare the test framework based on the Configuration
func tester(api *iris.Framework, t *testing.T) *httpexpect.Expect {
	api.Config.DisableBanner = true
	go func() { // no need goroutine here, we could just add go api.Listen(addr) but newcomers can see easier that these will run in a non-blocking way
		if enable_subdomain_tests {
			api.Listen(host)
		} else {
			api.NoListen(host)
		}
	}()

	if ok := <-api.Available; !ok {
		t.Fatal("Unexpected error: server cannot start, please report this as bug!!")
	}
	close(api.Available)

	handler := api.HTTPServer.Handler

	testConfiguration := httpexpect.Config{
		BaseURL: hostURL,
		Client: &http.Client{
			Transport: httpexpect.NewFastBinder(handler),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	}

	if enable_debug {
		testConfiguration.Printers = []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		}
	}

	return httpexpect.WithConfig(testConfiguration)
}

// hosts[windows]
/*

# Copyright (c) 1993-2009 Microsoft Corp.
#
# This is a sample HOSTS file used by Microsoft TCP/IP for Windows.
#
# This file contains the mappings of IP addresses to host names. Each
# entry should be kept on an individual line. The IP address should
# be placed in the first column followed by the corresponding host name.
# The IP address and the host name should be separated by at least one
# space.
#
# Additionally, comments (such as these) may be inserted on individual
# lines or following the machine name denoted by a '#' symbol.
#
# For example:
#
#      102.54.94.97     rhino.acme.com          # source server
#       38.25.63.10     x.acme.com              # x client host

# localhost name resolution is handled within DNS itself.
127.0.0.1       localhost
::1             localhost
# for custom domain and subdomains
127.0.0.1				mydomain.com
127.0.0.1       mysubdomain.mydomain.com
127.0.0.1       kataras.mydomain.com
127.0.0.1       username1.mydomain.com
127.0.0.1       username2.mydomain.com
*/
