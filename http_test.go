// Black-box Testing
package iris_test

import (
	"fmt"
	"github.com/gavv/httpexpect"
	"github.com/kataras/go-errors"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
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
	expectedHost3 := iris.DefaultServerHostname + ":9090"
	expectedHost4 := "mydomain.com:443"

	host1 := iris.ParseHost(expectedHost1)
	host2 := iris.ParseHost(expectedHost2)
	host3 := iris.ParseHost(":9090")
	host4 := iris.ParseHost(expectedHost4)

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
	expectedHostname3 := iris.DefaultServerHostname
	expectedHostname4 := "mydomain.com"

	hostname1 := iris.ParseHostname(host1)
	hostname2 := iris.ParseHostname(host2)
	hostname3 := iris.ParseHostname(host3)
	hostname4 := iris.ParseHostname(host4)
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
	expectedScheme1 := iris.SchemeHTTP
	expectedScheme2 := iris.SchemeHTTP
	expectedScheme3 := iris.SchemeHTTP
	expectedScheme4 := iris.SchemeHTTPS
	scheme1 := iris.ParseScheme(host1)
	scheme2 := iris.ParseScheme(host2)
	scheme3 := iris.ParseScheme(host3)
	scheme4 := iris.ParseScheme(host4)
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

func getRandomNumber(min int, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

// works as
// defer listenTLS(iris.Default, hostTLS)()
func listenTLS(api *iris.Framework, hostTLS string) func() {
	api.Close() // close any previous server
	api.Config.DisableBanner = true
	// create the key and cert files on the fly, and delete them when this test finished
	certFile, ferr := ioutil.TempFile("", "cert")

	if ferr != nil {
		api.Logger.Panic(ferr.Error())
	}

	keyFile, ferr := ioutil.TempFile("", "key")
	if ferr != nil {
		api.Logger.Panic(ferr.Error())
	}

	certFile.WriteString(testTLSCert)
	keyFile.WriteString(testTLSKey)

	go api.ListenTLS(hostTLS, certFile.Name(), keyFile.Name())
	if ok := <-api.Available; !ok {
		api.Logger.Panic("Unexpected error: server cannot start, please report this as bug!!")
	}

	return func() {
		certFile.Close()
		time.Sleep(150 * time.Millisecond)
		os.Remove(certFile.Name())

		keyFile.Close()
		time.Sleep(150 * time.Millisecond)
		os.Remove(keyFile.Name())

		api.Close()
	}
}

// Contains the server test for multi running servers
func TestMultiRunningServers_v1_PROXY(t *testing.T) {
	iris.ResetDefault()

	host := "localhost" // you have to add it to your hosts file( for windows, as 127.0.0.1 mydomain.com)
	hostTLS := host + ":" + strconv.Itoa(getRandomNumber(8886, 8889))

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Write("Hello from %s", hostTLS)
	})

	proxyHost := host + ":" + strconv.Itoa(getRandomNumber(3300, 3332))
	closeProxy := iris.Proxy(proxyHost, "https://"+hostTLS)
	defer closeProxy()

	defer listenTLS(iris.Default, hostTLS)()

	e := httptest.New(iris.Default, t, httptest.ExplicitURL(true))

	e.Request("GET", "http://"+proxyHost).Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)
	e.Request("GET", "https://"+hostTLS).Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

}

// Contains the server test for multi running servers
func TestMultiRunningServers_v2(t *testing.T) {
	iris.ResetDefault()

	domain := "localhost"
	hostTLS := domain + ":" + strconv.Itoa(getRandomNumber(3333, 4444))
	srv1Host := domain + ":" + strconv.Itoa(getRandomNumber(4446, 5444))
	srv2Host := domain + ":" + strconv.Itoa(getRandomNumber(7778, 8887))

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Write("Hello from %s", hostTLS)
	})

	// using the proxy handler
	fsrv1 := &fasthttp.Server{Handler: iris.ProxyHandler(srv1Host, "https://"+hostTLS)}
	go fsrv1.ListenAndServe(srv1Host)
	// using the same iris' handler but not as proxy, just the same handler
	fsrv2 := &fasthttp.Server{Handler: iris.Default.Router}
	go fsrv2.ListenAndServe(srv2Host)

	defer listenTLS(iris.Default, hostTLS)()

	e := httptest.New(iris.Default, t, httptest.ExplicitURL(true))

	e.Request("GET", "http://"+srv1Host).Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)
	e.Request("GET", "http://"+srv2Host).Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)
	e.Request("GET", "https://"+hostTLS).Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

}

const (
	testEnableSubdomain = true
	testSubdomain       = "mysubdomain"
)

func testSubdomainHost() string {
	s := testSubdomain + "." + iris.Default.Config.VHost
	return s
}

func testSubdomainURL() string {
	subdomainHost := testSubdomainHost()
	return iris.Default.Config.VScheme + subdomainHost
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
	defer iris.Close()
	iris.ResetDefault()

	for idx := range testRoutes {
		r := testRoutes[idx]
		if r.Register {
			iris.HandleFunc(r.Method, r.Path, func(ctx *iris.Context) {
				ctx.SetStatusCode(r.Status)
				if r.Params != nil && len(r.Params) > 0 {
					ctx.SetBodyString(ctx.ParamsSentence())
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

	e := httptest.New(iris.Default, t)

	// run the tests (1)
	for idx := range testRoutes {
		r := testRoutes[idx]
		e.Request(r.Method, r.RequestPath).WithQueryString(r.RequestQuery).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}

}

func TestMuxSimpleParty(t *testing.T) {
	iris.ResetDefault()

	h := func(c *iris.Context) { c.WriteString(c.HostString() + c.PathString()) }

	if testEnableSubdomain {
		subdomainParty := iris.Party(testSubdomain + ".")
		{
			subdomainParty.Get("/", h)
			subdomainParty.Get("/path1", h)
			subdomainParty.Get("/path2", h)
			subdomainParty.Get("/namedpath/:param1/something/:param2", h)
			subdomainParty.Get("/namedpath/:param1/something/:param2/else", h)
		}
	}

	// simple
	p := iris.Party("/party1")
	{
		p.Get("/", h)
		p.Get("/path1", h)
		p.Get("/path2", h)
		p.Get("/namedpath/:param1/something/:param2", h)
		p.Get("/namedpath/:param1/something/:param2/else", h)
	}

	iris.Default.Config.VHost = "0.0.0.0:" + strconv.Itoa(getRandomNumber(2222, 2399))
	// iris.Default.Config.Tester.Debug = true
	// iris.Default.Config.Tester.ExplicitURL = true
	e := httptest.New(iris.Default, t)

	request := func(reqPath string) {

		e.Request("GET", reqPath).
			Expect().
			Status(iris.StatusOK).Body().Equal(iris.Default.Config.VHost + reqPath)
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
				Status(iris.StatusOK).Body().Equal(testSubdomainHost() + reqPath)
		}

		subdomainRequest("/")
		subdomainRequest("/path1")
		subdomainRequest("/path2")
		subdomainRequest("/namedpath/theparam1/something/theparam2")
		subdomainRequest("/namedpath/theparam1/something/theparam2/else")
	}
}

func TestMuxPathEscape(t *testing.T) {
	iris.ResetDefault()

	iris.Get("/details/:name", func(ctx *iris.Context) {
		name := ctx.Param("name")
		highlight := ctx.URLParam("highlight")
		ctx.Text(iris.StatusOK, fmt.Sprintf("name=%s,highlight=%s", name, highlight))
	})

	e := httptest.New(iris.Default, t)

	e.GET("/details/Sakamoto desu ga").
		WithQuery("highlight", "text").
		Expect().Status(iris.StatusOK).Body().Equal("name=Sakamoto desu ga,highlight=text")
}

func TestMuxDecodeURL(t *testing.T) {
	iris.ResetDefault()

	iris.Get("/encoding/:url", func(ctx *iris.Context) {
		url := iris.DecodeURL(ctx.Param("url"))
		ctx.SetStatusCode(iris.StatusOK)
		ctx.Write(url)
	})

	e := httptest.New(iris.Default, t)

	e.GET("/encoding/http%3A%2F%2Fsome-url.com").Expect().Status(iris.StatusOK).Body().Equal("http://some-url.com")
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
	iris.ResetDefault()
	// first register the testRoutes needed
	for _, r := range testRoutesCustomErrors {
		if r.Register {
			iris.HandleFunc(r.Method, r.Path, func(ctx *iris.Context) {
				ctx.EmitError(r.Status)
			})
		}
	}

	// register the custom errors
	iris.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.Write("%s", notFoundMessage)
	})

	iris.OnError(iris.StatusInternalServerError, func(ctx *iris.Context) {
		ctx.Write("%s", internalServerMessage)
	})

	// create httpexpect instance that will call fasthtpp.RequestHandler directly
	e := httptest.New(iris.Default, t)

	// run the tests
	for _, r := range testRoutesCustomErrors {
		e.Request(r.Method, r.RequestPath).
			Expect().
			Status(r.Status).Body().Equal(r.Body)
	}
}

type testUserAPI struct {
	*iris.Context
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
	iris.ResetDefault()

	middlewareResponseText := "I assume that you are authenticated\n"
	h := []iris.HandlerFunc{func(ctx *iris.Context) { // optional middleware for .API
		// do your work here, or render a login window if not logged in, get the user and send it to the next middleware, or do  all here
		ctx.Set("user", "username")
		ctx.Next()
	}, func(ctx *iris.Context) {
		if ctx.Get("user") == "username" {
			ctx.Write(middlewareResponseText)
			ctx.Next()
		} else {
			ctx.SetStatusCode(iris.StatusUnauthorized)
		}
	}}

	iris.API("/users", testUserAPI{}, h...)
	// test a simple .Party  with compination of .API
	iris.Party("sites/:site").API("/users", testUserAPI{}, h...)

	e := httptest.New(iris.Default, t)

	siteID := "1"
	apiPath := "/sites/" + siteID + "/users"
	userID := "4077"
	formname := "kataras"

	// .API
	e.GET("/users").Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Get Users\n")
	e.GET("/users/" + userID).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Get By " + userID + "\n")
	e.PUT("/users").WithFormField("name", formname).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Put, name: " + formname + "\n")
	e.POST("/users/"+userID).WithFormField("name", formname).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Post By " + userID + ", name: " + formname + "\n")
	e.DELETE("/users/" + userID).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Delete By " + userID + "\n")

	// .Party
	e.GET(apiPath).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Get Users\n")
	e.GET(apiPath + "/" + userID).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Get By " + userID + "\n")
	e.PUT(apiPath).WithFormField("name", formname).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Put, name: " + formname + "\n")
	e.POST(apiPath+"/"+userID).WithFormField("name", formname).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Post By " + userID + ", name: " + formname + "\n")
	e.DELETE(apiPath + "/" + userID).Expect().Status(iris.StatusOK).Body().Equal(middlewareResponseText + "Delete By " + userID + "\n")

}

type myTestHandlerData struct {
	Sysname              string // this will be the same for all requests
	Version              int    // this will be the same for all requests
	DynamicPathParameter string // this will be different for each request
}

type myTestCustomHandler struct {
	data myTestHandlerData
}

func (m *myTestCustomHandler) Serve(ctx *iris.Context) {
	data := &m.data
	data.DynamicPathParameter = ctx.Param("myparam")
	ctx.JSON(iris.StatusOK, data)
}

func TestMuxCustomHandler(t *testing.T) {
	iris.ResetDefault()
	myData := myTestHandlerData{
		Sysname: "Redhat",
		Version: 1,
	}
	iris.Handle("GET", "/custom_handler_1/:myparam", &myTestCustomHandler{myData})
	iris.Handle("GET", "/custom_handler_2/:myparam", &myTestCustomHandler{myData})

	e := httptest.New(iris.Default, t)
	// two times per testRoute
	param1 := "thisimyparam1"
	expectedData1 := myData
	expectedData1.DynamicPathParameter = param1
	e.GET("/custom_handler_1/" + param1).Expect().Status(iris.StatusOK).JSON().Equal(expectedData1)

	param2 := "thisimyparam2"
	expectedData2 := myData
	expectedData2.DynamicPathParameter = param2
	e.GET("/custom_handler_1/" + param2).Expect().Status(iris.StatusOK).JSON().Equal(expectedData2)

	param3 := "thisimyparam3"
	expectedData3 := myData
	expectedData3.DynamicPathParameter = param3
	e.GET("/custom_handler_2/" + param3).Expect().Status(iris.StatusOK).JSON().Equal(expectedData3)

	param4 := "thisimyparam4"
	expectedData4 := myData
	expectedData4.DynamicPathParameter = param4
	e.GET("/custom_handler_2/" + param4).Expect().Status(iris.StatusOK).JSON().Equal(expectedData4)
}

func TestMuxFireMethodNotAllowed(t *testing.T) {
	iris.ResetDefault()
	iris.Default.Config.FireMethodNotAllowed = true
	h := func(ctx *iris.Context) {
		ctx.Write("%s", ctx.MethodString())
	}

	iris.Default.OnError(iris.StatusMethodNotAllowed, func(ctx *iris.Context) {
		ctx.Write("Hello from my custom 405 page")
	})

	iris.Get("/mypath", h)
	iris.Put("/mypath", h)

	e := httptest.New(iris.Default, t)

	e.GET("/mypath").Expect().Status(iris.StatusOK).Body().Equal("GET")
	e.PUT("/mypath").Expect().Status(iris.StatusOK).Body().Equal("PUT")
	// this should fail with 405 and catch by the custom http error

	e.POST("/mypath").Expect().Status(iris.StatusMethodNotAllowed).Body().Equal("Hello from my custom 405 page")
	iris.Close()
}

var (
	cacheDuration      = 2 * time.Second
	errCacheTestFailed = errors.New("Expected the main handler to be executed %d times instead of %d.")
)

// ~14secs
func runCacheTest(e *httpexpect.Expect, path string, counterPtr *uint32, expectedBodyStr, expectedContentType string) error {
	e.GET(path).Expect().Status(iris.StatusOK).Body().Equal(expectedBodyStr)
	time.Sleep(cacheDuration / 5) // lets wait for a while, cache should be saved and ready
	e.GET(path).Expect().Status(iris.StatusOK).Body().Equal(expectedBodyStr)
	counter := atomic.LoadUint32(counterPtr)
	if counter > 1 {
		// n should be 1 because it doesn't changed after the first call
		return errCacheTestFailed.Format(1, counter)
	}
	time.Sleep(cacheDuration)

	// cache should be cleared now
	e.GET(path).Expect().Status(iris.StatusOK).ContentType(expectedContentType, "utf-8").Body().Equal(expectedBodyStr)
	time.Sleep(cacheDuration / 5)
	// let's call again , the cache should be saved
	e.GET(path).Expect().Status(iris.StatusOK).ContentType(expectedContentType, "utf-8").Body().Equal(expectedBodyStr)
	counter = atomic.LoadUint32(counterPtr)
	if counter != 2 {
		return errCacheTestFailed.Format(2, counter)
	}

	return nil
}

/*
Inside github.com/geekypanda/httpcache are enough, no need to add 10+ seconds of testing here.
func TestCache(t *testing.T) {

	iris.ResetDefault()

	expectedBodyStr := "Imagine it as a big message to achieve x20 response performance!"
	var textCounter, htmlCounter uint32

	iris.Get("/text", iris.Cache(func(ctx *iris.Context) {
		atomic.AddUint32(&textCounter, 1)
		ctx.Text(iris.StatusOK, expectedBodyStr)
	}, cacheDuration))

	iris.Get("/html", iris.Cache(func(ctx *iris.Context) {
		atomic.AddUint32(&htmlCounter, 1)
		ctx.HTML(iris.StatusOK, expectedBodyStr)
	}, cacheDuration))

	e := httptest.New(iris.Default, t)

	// test cache on text/plain
	if err := runCacheTest(e, "/text", &textCounter, expectedBodyStr, "text/plain"); err != nil {
		t.Fatal(err)
	}

	// text cache on text/html
	if err := runCacheTest(e, "/html", &htmlCounter, expectedBodyStr, "text/html"); err != nil {
		t.Fatal(err)
	}
}
*/

func TestRedirectHTTPS(t *testing.T) {
	iris.ResetDefault()
	host := "localhost:5700"
	expectedBody := "Redirected to https://" + host + "/redirected"

	iris.Get("/redirect", func(ctx *iris.Context) { ctx.Redirect("/redirected") })
	iris.Get("/redirected", func(ctx *iris.Context) { ctx.Text(iris.StatusOK, "Redirected to "+ctx.URI().String()) })

	defer listenTLS(iris.Default, host)()

	e := httptest.New(iris.Default, t, httptest.ExplicitURL(true))
	e.Request("GET", "https://"+host+"/redirect").Expect().Status(iris.StatusOK).Body().Equal(expectedBody)
}
