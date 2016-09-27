package iris

/*
This is the part we only care, these are end-to-end tests for the mux(router) and the server, the whole http file is made for these reasons only, so these tests are enough I think.

CONTRIBUTE & DISCUSSION ABOUT TESTS TO: https://github.com/iris-contrib/tests
*/

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/gavv/httpexpect"
)

const (
	testTLSCert = `-----BEGIN CERTIFICATE-----
MIIDATCCAemgAwIBAgIJAPdE0ZyCKwVtMA0GCSqGSIb3DQEBBQUAMBcxFTATBgNV
BAMMDG15ZG9tYWluLmNvbTAeFw0xNjA5MjQwNjU3MDVaFw0yNjA5MjIwNjU3MDVa
MBcxFTATBgNVBAMMDG15ZG9tYWluLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBAM9YJOV1Bl+NwEq8ZAcVU2YONBw5zGkUFJUZkL77XT0i1V473JTf
GEpNZisDman+6n+pXarC2mR4T9PkCfmk072HaZ2LXwYe9XSgxnLJZJA1fJMzdMMC
2XveINF+/eeoyW9+8ZjQPbZdHWcxi7RomXg1AOMAG2UWMjalK5xkTHcqDuOI2LEe
mezWHnFdBJNMTi3pNdbVr7BjexZTSGhx4LAIP2ufTUoVzk+Cvyr4IhS00zOiyyWv
tuJaO20Q0Q5n34o9vDAACKAfNRLBE8qzdRwsjMumXTX3hJzvgFp/4Lr5Hr2I2fBd
tbIWN9xIsu6IibBGFItiAfQSrKAR7IFVqDUCAwEAAaNQME4wHQYDVR0OBBYEFNvN
Yik2eBRDmDaqoMaLfvr75kGfMB8GA1UdIwQYMBaAFNvNYik2eBRDmDaqoMaLfvr7
5kGfMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggEBAEAv3pKkmDTXIChB
nVrbYwNibin9HYOPf3JCjU48V672YPgbfJM0WfTvLXNVBOdTz3aIUrhfwv/go2Jz
yDcIFdLUdwllovhj1RwI96lbgCJ4AKoO/fvJ5Rxq+/vvLYB38PNl/fVEnOOeWzCQ
qHfjckShNV5GzJPhfpYn9Gj6+Zj3O0cJXhF9L/FlbVxFhwPjPRbFDNTHYzgiHy82
zhhDhTQJVnNJXfokdlg9OjfFkySqpv9fvPi/dfk5j1KmKUiYP5SYUhZiKX1JScvx
JgesCwz1nUfVQ1TYE0dphU8mTCN4Y42i7bctx7iLcDRIFhtYstt0hhCKSxFmJSBG
y9RITRA=
-----END CERTIFICATE-----
`

	testTLSKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAz1gk5XUGX43ASrxkBxVTZg40HDnMaRQUlRmQvvtdPSLVXjvc
lN8YSk1mKwOZqf7qf6ldqsLaZHhP0+QJ+aTTvYdpnYtfBh71dKDGcslkkDV8kzN0
wwLZe94g0X7956jJb37xmNA9tl0dZzGLtGiZeDUA4wAbZRYyNqUrnGRMdyoO44jY
sR6Z7NYecV0Ek0xOLek11tWvsGN7FlNIaHHgsAg/a59NShXOT4K/KvgiFLTTM6LL
Ja+24lo7bRDRDmffij28MAAIoB81EsETyrN1HCyMy6ZdNfeEnO+AWn/guvkevYjZ
8F21shY33Eiy7oiJsEYUi2IB9BKsoBHsgVWoNQIDAQABAoIBABRhi67qY+f8nQw7
nHF9zSbY+pJTtB4YFTXav3mmZ7HcvLB4neQcUdzr4sETp4UoQ5Cs60IfySvbD626
WqipZQ7aQq1zx7FoVaRTMW6TEUmDmG03v6BzpUEhwoQVQYwF8Vb+WW01+vr0CDHe
kub26S8BtsaZehfjqKfqcHD9Au8ri+Nwbu91iT4POVzBBBwYbtwXZwaYDR5PCNOI
ld+6qLapVIVKpvLHL+tA4A/n0n4l7p8TJo/qYesFRZ7J+USt4YGFDuf15nnDge/7
9Qjyqa9WmvRGytPdgtEzc8XwJu7xhcRthSmCppdY0ExHBwVceCEz8u3QbRYFqq3U
iLXUpfkCgYEA6JMlRtLIEAPkJZGBxJBSaeWVOeaAaMnLAdcu4OFjSuxr65HXsdhM
aWHopCE44NjBROAg67qgc5vNBZvhnCwyTI8nb6k/CO5QZ4AG1d2Xe/9rPV5pdaBL
gRgOJjlG0apZpPVM4I/0JU5prwS2Z71lFmEMikwmbmngYmOysqUBfbMCgYEA5Dpw
qzn1Oq+fasSUjzUa/wvBTVMpVjnNrda7Hnlx6lssnQaPFqifFYL9Zr/+5rWdsE2O
NNCsy68tacspAUa0LQinzpeSP4dyTVCvZG/xWPaYDKE6FDmzsi8xHBTTxMneeU6p
HUKJMUD3LcvBiCLunhT2xd1/+LKzVce6ms9R3ncCgYEAv9wjdDmOMSgEnblbg/xL
AHEUmZ89bzSI9Au/8GP+tWAz5zF47o2w+355nGyLr3EgfuEmR1C97KEqkOX3SA5t
sBqoPcUw6v0t9zP2b5dN0Ez0+rtX5GFH6Ecf5Qh7E5ukOCDkOpyGnAADzw3kK9Bi
BAQrhCstyQguwvvb/uOAR2ECgYEA3nYcZrqaz7ZqVL8C88hW5S4HIKEkFNlJI97A
DAdiw4ZVqUXAady5HFXPPL1+8FEtQLGIIPEazXuWb53I/WZ2r8LVFunlcylKgBRa
sjLvdMEBGqZ5H0fTYabgXrfqZ9JBmcrTyyKU6b6icTBAF7u9DbfvhpTObZN6fO2v
dcEJ0ycCgYEAxM8nGR+pa16kZmW1QV62EN0ifrU7SOJHCOGApU0jvTz8D4GO2j+H
MsoPSBrZ++UYgtGO/dK4aBV1JDdy8ZdyfE6UN+a6dQgdNdbOMT4XwWdS0zlZ/+F4
PKvbgZnLEKHvjODJ65aQmcTVUoaa11J29iAAtA3a3TcMn6C2fYpRy1s=
-----END RSA PRIVATE KEY-----
	`
)

func TestParseAddr(t *testing.T) {

	// test hosts
	expectedHost1 := "mydomain.com:1993"
	expectedHost2 := "mydomain.com"
	expectedHost3 := DefaultServerHostname + ":9090"
	expectedHost4 := "mydomain.com:443"

	host1 := ParseHost(expectedHost1)
	host2 := ParseHost(expectedHost2)
	host3 := ParseHost(":9090")
	host4 := ParseHost(expectedHost4)

	if host1 != expectedHost1 {
		t.Fatalf("Expecting server 1's host to be %s but we got %s", expectedHost1, host1)
	}
	if host2 != expectedHost2 {
		t.Fatalf("Expecting server 2's host to be %s but we got %s", expectedHost2, host2)
	}
	if host3 != expectedHost3 {
		t.Fatalf("Expecting server 3's host to be %s but we got %s", expectedHost3, host3)
	}
	if host4 != expectedHost4 {
		t.Fatalf("Expecting server 4's host to be %s but we got %s", expectedHost4, host4)
	}

	// test hostname
	expectedHostname1 := "mydomain.com"
	expectedHostname2 := "mydomain.com"
	expectedHostname3 := DefaultServerHostname
	expectedHostname4 := "mydomain.com"

	hostname1 := ParseHostname(host1)
	hostname2 := ParseHostname(host2)
	hostname3 := ParseHostname(host3)
	hostname4 := ParseHostname(host4)
	if hostname1 != expectedHostname1 {
		t.Fatalf("Expecting server 1's hostname to be %s but we got %s", expectedHostname1, hostname1)
	}

	if hostname2 != expectedHostname2 {
		t.Fatalf("Expecting server 2's hostname to be %s but we got %s", expectedHostname2, hostname2)
	}

	if hostname3 != expectedHostname3 {
		t.Fatalf("Expecting server 3's hostname to be %s but we got %s", expectedHostname3, hostname3)
	}

	if hostname4 != expectedHostname4 {
		t.Fatalf("Expecting server 4's hostname to be %s but we got %s", expectedHostname4, hostname4)
	}

	// test scheme, no need to test fullhost(scheme+host)
	expectedScheme1 := SchemeHTTP
	expectedScheme2 := SchemeHTTP
	expectedScheme3 := SchemeHTTP
	expectedScheme4 := SchemeHTTPS
	scheme1 := ParseScheme(host1)
	scheme2 := ParseScheme(host2)
	scheme3 := ParseScheme(host3)
	scheme4 := ParseScheme(host4)
	if scheme1 != expectedScheme1 {
		t.Fatalf("Expecting server 1's hostname to be %s but we got %s", expectedScheme1, scheme1)
	}

	if scheme2 != expectedScheme2 {
		t.Fatalf("Expecting server 2's hostname to be %s but we got %s", expectedScheme2, scheme2)
	}

	if scheme3 != expectedScheme3 {
		t.Fatalf("Expecting server 3's hostname to be %s but we got %s", expectedScheme3, scheme3)
	}

	if scheme4 != expectedScheme4 {
		t.Fatalf("Expecting server 4's hostname to be %s but we got %s", expectedScheme4, scheme4)
	}
}

// Contains the server test for multi running servers
func TestMultiRunningServers_v1_PROXY(t *testing.T) {
	defer Close()
	host := "localhost" // you have to add it to your hosts file( for windows, as 127.0.0.1 mydomain.com)
	hostTLS := "localhost:9999"
	initDefault()
	Default.Config.DisableBanner = true
	// create the key and cert files on the fly, and delete them when this test finished
	certFile, ferr := ioutil.TempFile("", "cert")

	if ferr != nil {
		t.Fatal(ferr.Error())
	}

	keyFile, ferr := ioutil.TempFile("", "key")
	if ferr != nil {
		t.Fatal(ferr.Error())
	}

	defer func() {
		certFile.Close()
		time.Sleep(350 * time.Millisecond)
		os.Remove(certFile.Name())

		keyFile.Close()
		time.Sleep(350 * time.Millisecond)
		os.Remove(keyFile.Name())
	}()

	certFile.WriteString(testTLSCert)
	keyFile.WriteString(testTLSKey)

	Get("/", func(ctx *Context) {
		ctx.Write("Hello from %s", hostTLS)
	})

	go ListenTLS(hostTLS, certFile.Name(), keyFile.Name())
	if ok := <-Default.Available; !ok {
		t.Fatal("Unexpected error: server cannot start, please report this as bug!!")
	}

	closeProxy := Proxy("localhost:8080", "https://"+hostTLS)
	defer closeProxy()

	Default.Config.Tester.ExplicitURL = true
	e := Tester(t)

	e.Request("GET", "http://"+host+":8080").Expect().Status(StatusOK).Body().Equal("Hello from " + hostTLS)
	e.Request("GET", "https://"+hostTLS).Expect().Status(StatusOK).Body().Equal("Hello from " + hostTLS)

}

// Contains the server test for multi running servers
func TestMultiRunningServers_v2(t *testing.T) {
	defer Close()
	domain := "localhost"
	hostTLS := "localhost:9999"

	initDefault()
	Default.Config.DisableBanner = true

	// create the key and cert files on the fly, and delete them when this test finished
	certFile, ferr := ioutil.TempFile("", "cert")

	if ferr != nil {
		t.Fatal(ferr.Error())
	}

	keyFile, ferr := ioutil.TempFile("", "key")
	if ferr != nil {
		t.Fatal(ferr.Error())
	}

	certFile.WriteString(testTLSCert)
	keyFile.WriteString(testTLSKey)

	defer func() {
		certFile.Close()
		time.Sleep(350 * time.Millisecond)
		os.Remove(certFile.Name())

		keyFile.Close()
		time.Sleep(350 * time.Millisecond)
		os.Remove(keyFile.Name())
	}()

	Get("/", func(ctx *Context) {
		ctx.Write("Hello from %s", hostTLS)
	})

	// add a secondary server
	//Servers.Add(ServerConfiguration{ListeningAddr: domain + ":80", RedirectTo: "https://" + host, Virtual: true})
	// add our primary/main server
	//Servers.Add(ServerConfiguration{ListeningAddr: host, CertFile: certFile.Name(), KeyFile: keyFile.Name(), Virtual: true})

	//go Go()

	// using the proxy handler
	fsrv1 := &fasthttp.Server{Handler: proxyHandler(domain+":8080", "https://"+hostTLS)}
	go fsrv1.ListenAndServe(domain + ":8080")
	// using the same iris' handler but not as proxy, just the same handler
	fsrv2 := &fasthttp.Server{Handler: Default.Router}
	go fsrv2.ListenAndServe(domain + ":8888")

	go ListenTLS(hostTLS, certFile.Name(), keyFile.Name())

	if ok := <-Available; !ok {
		t.Fatal("Unexpected error: server cannot start, please report this as bug!!")
	}

	Default.Config.Tester.ExplicitURL = true
	e := Tester(t)

	e.Request("GET", "http://"+domain+":8080").Expect().Status(StatusOK).Body().Equal("Hello from " + hostTLS)
	e.Request("GET", "http://localhost:8888").Expect().Status(StatusOK).Body().Equal("Hello from " + hostTLS)
	e.Request("GET", "https://"+hostTLS).Expect().Status(StatusOK).Body().Equal("Hello from " + hostTLS)

}

const (
	testEnableSubdomain = true
	testSubdomain       = "mysubdomain"
)

func testSubdomainHost() string {
	s := testSubdomain + "." + Default.Config.VHost
	return s
}

func testSubdomainURL() string {
	subdomainHost := testSubdomainHost()
	return Default.Config.VScheme + subdomainHost
}

func subdomainTester(e *httpexpect.Expect) *httpexpect.Expect {
	es := e.Builder(func(req *httpexpect.Request) {
		req.WithURL(testSubdomainURL())
	})
	return es
}

type param struct {
	Key   string
	Value string
}

type testRoute struct {
	Method       string
	Path         string
	RequestPath  string
	RequestQuery string
	Body         string
	Status       int
	Register     bool
	Params       []param
	URLParams    []param
}

func TestMuxSimple(t *testing.T) {
	testRoutes := []testRoute{
		// FOUND - registed
		{"GET", "/test_get", "/test_get", "", "hello, get!", 200, true, nil, nil},
		{"POST", "/test_post", "/test_post", "", "hello, post!", 200, true, nil, nil},
		{"PUT", "/test_put", "/test_put", "", "hello, put!", 200, true, nil, nil},
		{"DELETE", "/test_delete", "/test_delete", "", "hello, delete!", 200, true, nil, nil},
		{"HEAD", "/test_head", "/test_head", "", "hello, head!", 200, true, nil, nil},
		{"OPTIONS", "/test_options", "/test_options", "", "hello, options!", 200, true, nil, nil},
		{"CONNECT", "/test_connect", "/test_connect", "", "hello, connect!", 200, true, nil, nil},
		{"PATCH", "/test_patch", "/test_patch", "", "hello, patch!", 200, true, nil, nil},
		{"TRACE", "/test_trace", "/test_trace", "", "hello, trace!", 200, true, nil, nil},
		// NOT FOUND - not registed
		{"GET", "/test_get_nofound", "/test_get_nofound", "", "Not Found", 404, false, nil, nil},
		{"POST", "/test_post_nofound", "/test_post_nofound", "", "Not Found", 404, false, nil, nil},
		{"PUT", "/test_put_nofound", "/test_put_nofound", "", "Not Found", 404, false, nil, nil},
		{"DELETE", "/test_delete_nofound", "/test_delete_nofound", "", "Not Found", 404, false, nil, nil},
		{"HEAD", "/test_head_nofound", "/test_head_nofound", "", "Not Found", 404, false, nil, nil},
		{"OPTIONS", "/test_options_nofound", "/test_options_nofound", "", "Not Found", 404, false, nil, nil},
		{"CONNECT", "/test_connect_nofound", "/test_connect_nofound", "", "Not Found", 404, false, nil, nil},
		{"PATCH", "/test_patch_nofound", "/test_patch_nofound", "", "Not Found", 404, false, nil, nil},
		{"TRACE", "/test_trace_nofound", "/test_trace_nofound", "", "Not Found", 404, false, nil, nil},
		// Parameters
		{"GET", "/test_get_parameter1/:name", "/test_get_parameter1/iris", "", "name=iris", 200, true, []param{{"name", "iris"}}, nil},
		{"GET", "/test_get_parameter2/:name/details/:something", "/test_get_parameter2/iris/details/anything", "", "name=iris,something=anything", 200, true, []param{{"name", "iris"}, {"something", "anything"}}, nil},
		{"GET", "/test_get_parameter2/:name/details/:something/*else", "/test_get_parameter2/iris/details/anything/elsehere", "", "name=iris,something=anything,else=/elsehere", 200, true, []param{{"name", "iris"}, {"something", "anything"}, {"else", "elsehere"}}, nil},
		// URL Parameters
		{"GET", "/test_get_urlparameter1/first", "/test_get_urlparameter1/first", "name=irisurl", "name=irisurl", 200, true, nil, []param{{"name", "irisurl"}}},
		{"GET", "/test_get_urlparameter2/second", "/test_get_urlparameter2/second", "name=irisurl&something=anything", "name=irisurl,something=anything", 200, true, nil, []param{{"name", "irisurl"}, {"something", "anything"}}},
		{"GET", "/test_get_urlparameter2/first/second/third", "/test_get_urlparameter2/first/second/third", "name=irisurl&something=anything&else=elsehere", "name=irisurl,something=anything,else=elsehere", 200, true, nil, []param{{"name", "irisurl"}, {"something", "anything"}, {"else", "elsehere"}}},
	}
	defer Close()
	initDefault()

	for idx := range testRoutes {
		r := testRoutes[idx]
		if r.Register {
			HandleFunc(r.Method, r.Path, func(ctx *Context) {
				ctx.SetStatusCode(r.Status)
				if r.Params != nil && len(r.Params) > 0 {
					ctx.SetBodyString(ctx.Params.String())
				} else if r.URLParams != nil && len(r.URLParams) > 0 {
					if len(r.URLParams) != len(ctx.URLParams()) {
						t.Fatalf("Error when comparing length of url parameters %d != %d", len(r.URLParams), len(ctx.URLParams()))
					}
					paramsKeyVal := ""
					for idxp, p := range r.URLParams {
						val := ctx.URLParam(p.Key)
						paramsKeyVal += p.Key + "=" + val + ","
						if idxp == len(r.URLParams)-1 {
							paramsKeyVal = paramsKeyVal[0 : len(paramsKeyVal)-1]
						}
					}
					ctx.SetBodyString(paramsKeyVal)
				} else {
					ctx.SetBodyString(r.Body)
				}

			})
		}
	}

	e := Tester(t)

	// run the tests (1)
	for idx := range testRoutes {
		r := testRoutes[idx]
		e.Request(r.Method, r.RequestPath).WithQueryString(r.RequestQuery).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}

}

func TestMuxSimpleParty(t *testing.T) {
	initDefault()

	h := func(c *Context) { c.WriteString(c.HostString() + c.PathString()) }

	if testEnableSubdomain {
		subdomainParty := Party(testSubdomain + ".")
		{
			subdomainParty.Get("/", h)
			subdomainParty.Get("/path1", h)
			subdomainParty.Get("/path2", h)
			subdomainParty.Get("/namedpath/:param1/something/:param2", h)
			subdomainParty.Get("/namedpath/:param1/something/:param2/else", h)
		}
	}

	// simple
	p := Party("/party1")
	{
		p.Get("/", h)
		p.Get("/path1", h)
		p.Get("/path2", h)
		p.Get("/namedpath/:param1/something/:param2", h)
		p.Get("/namedpath/:param1/something/:param2/else", h)
	}

	Default.Config.VHost = "0.0.0.0:8080"
	e := Tester(t)

	request := func(reqPath string) {
		e.Request("GET", reqPath).
			Expect().
			Status(StatusOK).Body().Equal(Default.Config.VHost + reqPath)
	}

	// run the tests
	request("/party1/")
	request("/party1/path1")
	request("/party1/path2")
	request("/party1/namedpath/theparam1/something/theparam2")
	request("/party1/namedpath/theparam1/something/theparam2/else")

	if testEnableSubdomain {
		es := subdomainTester(e)
		subdomainRequest := func(reqPath string) {
			es.Request("GET", reqPath).
				Expect().
				Status(StatusOK).Body().Equal(testSubdomainHost() + reqPath)
		}

		subdomainRequest("/")
		subdomainRequest("/path1")
		subdomainRequest("/path2")
		subdomainRequest("/namedpath/theparam1/something/theparam2")
		subdomainRequest("/namedpath/theparam1/something/theparam2/else")
	}
}

func TestMuxPathEscape(t *testing.T) {
	initDefault()

	Get("/details/:name", func(ctx *Context) {
		name := ctx.Param("name")
		highlight := ctx.URLParam("highlight")
		ctx.Text(StatusOK, fmt.Sprintf("name=%s,highlight=%s", name, highlight))
	})

	e := Tester(t)

	e.GET("/details/Sakamoto desu ga").
		WithQuery("highlight", "text").
		Expect().Status(StatusOK).Body().Equal("name=Sakamoto desu ga,highlight=text")
}

func TestMuxDecodeURL(t *testing.T) {
	initDefault()

	Get("/encoding/:url", func(ctx *Context) {
		url := DecodeURL(ctx.Param("url"))
		ctx.SetStatusCode(StatusOK)
		ctx.Write(url)
	})

	e := Tester(t)

	e.GET("/encoding/http%3A%2F%2Fsome-url.com").Expect().Status(StatusOK).Body().Equal("http://some-url.com")
}

func TestMuxCustomErrors(t *testing.T) {
	var (
		notFoundMessage        = "Iris custom message for 404 not found"
		internalServerMessage  = "Iris custom message for 500 internal server error"
		testRoutesCustomErrors = []testRoute{
			// NOT FOUND CUSTOM ERRORS - not registed
			{"GET", "/test_get_nofound_custom", "/test_get_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"POST", "/test_post_nofound_custom", "/test_post_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"PUT", "/test_put_nofound_custom", "/test_put_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"DELETE", "/test_delete_nofound_custom", "/test_delete_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"HEAD", "/test_head_nofound_custom", "/test_head_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"OPTIONS", "/test_options_nofound_custom", "/test_options_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"CONNECT", "/test_connect_nofound_custom", "/test_connect_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"PATCH", "/test_patch_nofound_custom", "/test_patch_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"TRACE", "/test_trace_nofound_custom", "/test_trace_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			// SERVER INTERNAL ERROR 500 PANIC CUSTOM ERRORS - registed
			{"GET", "/test_get_panic_custom", "/test_get_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"POST", "/test_post_panic_custom", "/test_post_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"PUT", "/test_put_panic_custom", "/test_put_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"DELETE", "/test_delete_panic_custom", "/test_delete_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"HEAD", "/test_head_panic_custom", "/test_head_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"OPTIONS", "/test_options_panic_custom", "/test_options_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"CONNECT", "/test_connect_panic_custom", "/test_connect_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"PATCH", "/test_patch_panic_custom", "/test_patch_panic_custom", "", internalServerMessage, 500, true, nil, nil},
			{"TRACE", "/test_trace_panic_custom", "/test_trace_panic_custom", "", internalServerMessage, 500, true, nil, nil},
		}
	)
	initDefault()
	// first register the testRoutes needed
	for _, r := range testRoutesCustomErrors {
		if r.Register {
			HandleFunc(r.Method, r.Path, func(ctx *Context) {
				ctx.EmitError(r.Status)
			})
		}
	}

	// register the custom errors
	OnError(404, func(ctx *Context) {
		ctx.Write("%s", notFoundMessage)
	})

	OnError(500, func(ctx *Context) {
		ctx.Write("%s", internalServerMessage)
	})

	// create httpexpect instance that will call fasthtpp.RequestHandler directly
	e := Tester(t)

	// run the tests
	for _, r := range testRoutesCustomErrors {
		e.Request(r.Method, r.RequestPath).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}
}

type testUserAPI struct {
	*Context
}

// GET /users
func (u testUserAPI) Get() {
	u.Write("Get Users\n")
}

// GET /users/:param1 which its value passed to the id argument
func (u testUserAPI) GetBy(id string) { // id equals to u.Param("param1")
	u.Write("Get By %s\n", id)
}

// PUT /users
func (u testUserAPI) Put() {
	u.Write("Put, name: %s\n", u.FormValue("name"))
}

// POST /users/:param1
func (u testUserAPI) PostBy(id string) {
	u.Write("Post By %s, name: %s\n", id, u.FormValue("name"))
}

// DELETE /users/:param1
func (u testUserAPI) DeleteBy(id string) {
	u.Write("Delete By %s\n", id)
}

func TestMuxAPI(t *testing.T) {
	initDefault()

	middlewareResponseText := "I assume that you are authenticated\n"
	API("/users", testUserAPI{}, func(ctx *Context) { // optional middleware for .API
		// do your work here, or render a login window if not logged in, get the user and send it to the next middleware, or do  all here
		ctx.Set("user", "username")
		ctx.Next()
	}, func(ctx *Context) {
		if ctx.Get("user") == "username" {
			ctx.Write(middlewareResponseText)
			ctx.Next()
		} else {
			ctx.SetStatusCode(StatusUnauthorized)
		}
	})

	e := Tester(t)

	userID := "4077"
	formname := "kataras"

	e.GET("/users").Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Get Users\n")
	e.GET("/users/" + userID).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Get By " + userID + "\n")
	e.PUT("/users").WithFormField("name", formname).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Put, name: " + formname + "\n")
	e.POST("/users/"+userID).WithFormField("name", formname).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Post By " + userID + ", name: " + formname + "\n")
	e.DELETE("/users/" + userID).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Delete By " + userID + "\n")
}

func TestMuxAPIWithParty(t *testing.T) {
	initDefault()
	siteParty := Party("sites/:site")

	middlewareResponseText := "I assume that you are authenticated\n"
	siteParty.API("/users", testUserAPI{}, func(ctx *Context) {
		ctx.Set("user", "username")
		ctx.Next()
	}, func(ctx *Context) {
		if ctx.Get("user") == "username" {
			ctx.Write(middlewareResponseText)
			ctx.Next()
		} else {
			ctx.SetStatusCode(StatusUnauthorized)
		}
	})

	e := Tester(t)
	siteID := "1"
	apiPath := "/sites/" + siteID + "/users"
	userID := "4077"
	formname := "kataras"

	e.GET(apiPath).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Get Users\n")
	e.GET(apiPath + "/" + userID).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Get By " + userID + "\n")
	e.PUT(apiPath).WithFormField("name", formname).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Put, name: " + formname + "\n")
	e.POST(apiPath+"/"+userID).WithFormField("name", formname).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Post By " + userID + ", name: " + formname + "\n")
	e.DELETE(apiPath + "/" + userID).Expect().Status(StatusOK).Body().Equal(middlewareResponseText + "Delete By " + userID + "\n")
}

type myTestHandlerData struct {
	Sysname              string // this will be the same for all requests
	Version              int    // this will be the same for all requests
	DynamicPathParameter string // this will be different for each request
}

type myTestCustomHandler struct {
	data myTestHandlerData
}

func (m *myTestCustomHandler) Serve(ctx *Context) {
	data := &m.data
	data.DynamicPathParameter = ctx.Param("myparam")
	ctx.JSON(StatusOK, data)
}

func TestMuxCustomHandler(t *testing.T) {
	initDefault()
	myData := myTestHandlerData{
		Sysname: "Redhat",
		Version: 1,
	}
	Handle("GET", "/custom_handler_1/:myparam", &myTestCustomHandler{myData})
	Handle("GET", "/custom_handler_2/:myparam", &myTestCustomHandler{myData})

	e := Tester(t)
	// two times per testRoute
	param1 := "thisimyparam1"
	expectedData1 := myData
	expectedData1.DynamicPathParameter = param1
	e.GET("/custom_handler_1/" + param1).Expect().Status(StatusOK).JSON().Equal(expectedData1)

	param2 := "thisimyparam2"
	expectedData2 := myData
	expectedData2.DynamicPathParameter = param2
	e.GET("/custom_handler_1/" + param2).Expect().Status(StatusOK).JSON().Equal(expectedData2)

	param3 := "thisimyparam3"
	expectedData3 := myData
	expectedData3.DynamicPathParameter = param3
	e.GET("/custom_handler_2/" + param3).Expect().Status(StatusOK).JSON().Equal(expectedData3)

	param4 := "thisimyparam4"
	expectedData4 := myData
	expectedData4.DynamicPathParameter = param4
	e.GET("/custom_handler_2/" + param4).Expect().Status(StatusOK).JSON().Equal(expectedData4)
}
