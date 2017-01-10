// Black-box Testing
package iris_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gavv/httpexpect"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
)

const (
	testTLSCert = `-----BEGIN CERTIFICATE-----
MIIDAzCCAeugAwIBAgIJAP0pWSuIYyQCMA0GCSqGSIb3DQEBBQUAMBgxFjAUBgNV
BAMMDWxvY2FsaG9zdDozMzEwHhcNMTYxMjI1MDk1OTI3WhcNMjYxMjIzMDk1OTI3
WjAYMRYwFAYDVQQDDA1sb2NhbGhvc3Q6MzMxMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEA5vETjLa+8W856rWXO1xMF/CLss9vn5xZhPXKhgz+D7ogSAXm
mWP53eeBUGC2r26J++CYfVqwOmfJEu9kkGUVi8cGMY9dHeIFPfxD31MYX175jJQe
tu0WeUII7ciNsSUDyBMqsl7yi1IgN7iLONM++1+QfbbmNiEbghRV6icEH6M+bWlz
3YSAMEdpK3mg2gsugfLKMwJkaBKEehUNMySRlIhyLITqt1exYGaggRd1zjqUpqpD
sL2sRVHJ3qHGkSh8nVy8MvG8BXiFdYQJP3mCQDZzruCyMWj5/19KAyu7Cto3Bcvu
PgujnwRtU+itt8WhZUVtU1n7Ivf6lMJTBcc4OQIDAQABo1AwTjAdBgNVHQ4EFgQU
MXrBvbILQmiwjUj19aecF2N+6IkwHwYDVR0jBBgwFoAUMXrBvbILQmiwjUj19aec
F2N+6IkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQUFAAOCAQEA4zbFml1t9KXJ
OijAV8gALePR8v04DQwJP+jsRxXw5zzhc8Wqzdd2hjUd07mfRWAvmyywrmhCV6zq
OHznR+aqIqHtm0vV8OpKxLoIQXavfBd6axEXt3859RDM4xJNwIlxs3+LWGPgINud
wjJqjyzSlhJpQpx4YZ5Da+VMiqAp8N1UeaZ5lBvmSDvoGh6HLODSqtPlWMrldRW9
AfsXVxenq81MIMeKW2fSOoPnWZ4Vjf1+dSlbJE/DD4zzcfbyfgY6Ep/RrUltJ3ag
FQbuNTQlgKabe21dSL9zJ2PengVKXl4Trl+4t/Kina9N9Jw535IRCSwinD6a/2Ca
m7DnVXFiVA==
-----END CERTIFICATE-----
`

	testTLSKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA5vETjLa+8W856rWXO1xMF/CLss9vn5xZhPXKhgz+D7ogSAXm
mWP53eeBUGC2r26J++CYfVqwOmfJEu9kkGUVi8cGMY9dHeIFPfxD31MYX175jJQe
tu0WeUII7ciNsSUDyBMqsl7yi1IgN7iLONM++1+QfbbmNiEbghRV6icEH6M+bWlz
3YSAMEdpK3mg2gsugfLKMwJkaBKEehUNMySRlIhyLITqt1exYGaggRd1zjqUpqpD
sL2sRVHJ3qHGkSh8nVy8MvG8BXiFdYQJP3mCQDZzruCyMWj5/19KAyu7Cto3Bcvu
PgujnwRtU+itt8WhZUVtU1n7Ivf6lMJTBcc4OQIDAQABAoIBAQCTLE0eHpPevtg0
+FaRUMd5diVA5asoF3aBIjZXaU47bY0G+SO02x6wSMmDFK83a4Vpy/7B3Bp0jhF5
DLCUyKaLdmE/EjLwSUq37ty+JHFizd7QtNBCGSN6URfpmSabHpCjX3uVQqblHIhF
mki3BQCdJ5CoXPemxUCHjDgYSZb6JVNIPJExjekc0+4A2MYWMXV6Wr86C7AY3659
KmveZpC3gOkLA/g/IqDQL/QgTq7/3eloHaO+uPBihdF56do4eaOO0jgFYpl8V7ek
PZhHfhuPZV3oq15+8Vt77ngtjUWVI6qX0E3ilh+V5cof+03q0FzHPVe3zBUNXcm0
OGz19u/FAoGBAPSm4Aa4xs/ybyjQakMNix9rak66ehzGkmlfeK5yuQ/fHmTg8Ac+
ahGs6A3lFWQiyU6hqm6Qp0iKuxuDh35DJGCWAw5OUS/7WLJtu8fNFch6iIG29rFs
s+Uz2YLxJPebpBsKymZUp7NyDRgEElkiqsREmbYjLrc8uNKkDy+k14YnAoGBAPGn
ZlN0Mo5iNgQStulYEP5pI7WOOax9KOYVnBNguqgY9c7fXVXBxChoxt5ebQJWG45y
KPG0hB0bkA4YPu4bTRf5acIMpjFwcxNlmwdc4oCkT4xqAFs9B/AKYZgkf4IfKHqW
P9PD7TbUpkaxv25bPYwUSEB7lPa+hBtRyN9Wo6qfAoGAPBkeISiU1hJE0i7YW55h
FZfKZoqSYq043B+ywo+1/Dsf+UH0VKM1ZSAnZPpoVc/hyaoW9tAb98r0iZ620wJl
VkCjgYklknbY5APmw/8SIcxP6iVq1kzQqDYjcXIRVa3rEyWEcLzM8VzL8KFXbIQC
lPIRHFfqKuMEt+HLRTXmJ7MCgYAHGvv4QjdmVl7uObqlG9DMGj1RjlAF0VxNf58q
NrLmVG2N2qV86wigg4wtZ6te4TdINfUcPkmQLYpLz8yx5Z2bsdq5OPP+CidoD5nC
WqnSTIKGR2uhQycjmLqL5a7WHaJsEFTqHh2wego1k+5kCUzC/KmvM7MKmkl6ICp+
3qZLUwKBgQCDOhKDwYo1hdiXoOOQqg/LZmpWOqjO3b4p99B9iJqhmXN0GKXIPSBh
5nqqmGsG8asSQhchs7EPMh8B80KbrDTeidWskZuUoQV27Al1UEmL6Zcl83qXD6sf
k9X9TwWyZtp5IL1CAEd/Il9ZTXFzr3lNaN8LCFnU+EIsz1YgUW8LTg==
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
	api.Close() // close any prev listener
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
		time.Sleep(50 * time.Millisecond)
		os.Remove(certFile.Name())

		keyFile.Close()
		time.Sleep(50 * time.Millisecond)
		os.Remove(keyFile.Name())
	}
}

// Contains the server test for multi running servers
func TestMultiRunningServers_v1_PROXY(t *testing.T) {
	api := iris.New()

	host := "localhost"
	hostTLS := host + ":" + strconv.Itoa(getRandomNumber(1919, 2221))
	api.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from %s", hostTLS)
	})
	// println("running main on: " + hostTLS)

	defer listenTLS(api, hostTLS)()

	e := httptest.New(api, t, httptest.ExplicitURL(true))
	e.Request("GET", "/").Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

	// proxy http to https
	proxyHost := host + ":" + strconv.Itoa(getRandomNumber(3300, 3340))
	// println("running proxy on: " + proxyHost)

	iris.Proxy(proxyHost, "https://"+hostTLS)

	//	proxySrv := &http.Server{Addr: proxyHost, Handler: iris.ProxyHandler("https://" + hostTLS)}
	//	go proxySrv.ListenAndServe()
	//	time.Sleep(3 * time.Second)

	eproxy := httptest.NewInsecure("http://"+proxyHost, t, httptest.ExplicitURL(true))
	eproxy.Request("GET", "/").Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)
}

// Contains the server test for multi running servers
func TestMultiRunningServers_v2(t *testing.T) {
	api := iris.New()

	domain := "localhost"
	hostTLS := domain + ":" + strconv.Itoa(getRandomNumber(2222, 2229))
	srv1Host := domain + ":" + strconv.Itoa(getRandomNumber(4446, 5444))
	srv2Host := domain + ":" + strconv.Itoa(getRandomNumber(7778, 8887))

	api.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from %s", hostTLS)
	})

	defer listenTLS(api, hostTLS)()

	// using the same iris' handler but not as proxy, just the same handler
	srv2 := &http.Server{Handler: api.Router, Addr: srv2Host}
	go srv2.ListenAndServe()

	// using the proxy handler
	srv1 := &http.Server{Handler: iris.ProxyHandler("https://" + hostTLS), Addr: srv1Host}
	go srv1.ListenAndServe()
	time.Sleep(500 * time.Millisecond) // wait a little for the http servers

	e := httptest.New(api, t, httptest.ExplicitURL(true))
	e.Request("GET", "/").Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

	eproxy1 := httptest.NewInsecure("http://"+srv1Host, t, httptest.ExplicitURL(true))
	eproxy1.Request("GET", "/").Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

	eproxy2 := httptest.NewInsecure("http://"+srv2Host, t)
	eproxy2.Request("GET", "/").Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

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
		// FOUND - registered
		{"GET", "/test_get", "/test_get", "", "hello, get!", 200, true, nil, nil},
		{"POST", "/test_post", "/test_post", "", "hello, post!", 200, true, nil, nil},
		{"PUT", "/test_put", "/test_put", "", "hello, put!", 200, true, nil, nil},
		{"DELETE", "/test_delete", "/test_delete", "", "hello, delete!", 200, true, nil, nil},
		{"HEAD", "/test_head", "/test_head", "", "hello, head!", 200, true, nil, nil},
		{"OPTIONS", "/test_options", "/test_options", "", "hello, options!", 200, true, nil, nil},
		{"CONNECT", "/test_connect", "/test_connect", "", "hello, connect!", 200, true, nil, nil},
		{"PATCH", "/test_patch", "/test_patch", "", "hello, patch!", 200, true, nil, nil},
		{"TRACE", "/test_trace", "/test_trace", "", "hello, trace!", 200, true, nil, nil},
		// NOT FOUND - not registered
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

	iris.ResetDefault()

	for idx := range testRoutes {
		r := testRoutes[idx]
		if r.Register {
			iris.HandleFunc(r.Method, r.Path, func(ctx *iris.Context) {
				ctx.SetStatusCode(r.Status)
				if r.Params != nil && len(r.Params) > 0 {
					ctx.WriteString(ctx.ParamsSentence())
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
					ctx.WriteString(paramsKeyVal)
				} else {
					ctx.WriteString(r.Body)
				}

			})
		}
	}

	e := httptest.New(iris.Default, t, httptest.Debug(true))

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

	h := func(c *iris.Context) { c.WriteString(c.Request.URL.Host + c.Request.RequestURI) }

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

// TestRealSubdomainSimple exists because the local examples some times passed but...
// hope that travis will not has problem with this
func TestRealSubdomainSimple(t *testing.T) {

	api := iris.New()
	host := "localhost:" + strconv.Itoa(getRandomNumber(4732, 4958))
	subdomain := "admin"
	subdomainHost := subdomain + "." + host

	// no order, you can register subdomains at the end also.
	admin := api.Party(subdomain + ".")
	{
		// admin.mydomain.com
		admin.Get("/", func(c *iris.Context) {
			c.Writef("INDEX FROM %s", subdomainHost)
		})
		// admin.mydomain.com/hey
		admin.Get("/hey", func(c *iris.Context) {
			c.Writef(subdomainHost + c.Request.RequestURI)
		})
		// admin.mydomain.com/hey2
		admin.Get("/hey2", func(c *iris.Context) {
			c.Writef(subdomainHost + c.Request.RequestURI)
		})
	}

	// mydomain.com/
	api.Get("/", func(c *iris.Context) {
		c.Writef("INDEX FROM no-subdomain hey")
	})

	// mydomain.com/hey
	api.Get("/hey", func(c *iris.Context) {
		c.Writef("HEY FROM no-subdomain hey")
	})

	api.Config.DisableBanner = true
	go api.Listen(host)

	<-api.Available

	e := httptest.New(api, t, httptest.ExplicitURL(true))

	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal("INDEX FROM no-subdomain hey")
	e.GET("/hey").Expect().Status(iris.StatusOK).Body().Equal("HEY FROM no-subdomain hey")

	sub := e.Builder(func(req *httpexpect.Request) {
		req.WithURL("http://admin." + host)
	})

	sub.GET("/").Expect().Status(iris.StatusOK).Body().Equal("INDEX FROM " + subdomainHost)
	sub.GET("/hey").Expect().Status(iris.StatusOK).Body().Equal(subdomainHost + "/hey")
	sub.GET("/hey2").Expect().Status(iris.StatusOK).Body().Equal(subdomainHost + "/hey2")
}

func TestMuxPathEscape(t *testing.T) {
	iris.ResetDefault()
	iris.Config.EnablePathEscape = true

	iris.Get("/details/:name", func(ctx *iris.Context) {
		name := ctx.ParamDecoded("name")
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
		url := ctx.ParamDecoded("url")

		ctx.SetStatusCode(iris.StatusOK)
		ctx.WriteString(url)
	})

	e := httptest.New(iris.Default, t)

	e.GET("/encoding/http%3A%2F%2Fsome-url.com").Expect().Status(iris.StatusOK).Body().Equal("http://some-url.com")
}

func TestMuxCustomErrors(t *testing.T) {
	var (
		notFoundMessage        = "Iris custom message for 404 not found"
		internalServerMessage  = "Iris custom message for 500 internal server error"
		testRoutesCustomErrors = []testRoute{
			// NOT FOUND CUSTOM ERRORS - not registered
			{"GET", "/test_get_nofound_custom", "/test_get_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"POST", "/test_post_nofound_custom", "/test_post_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"PUT", "/test_put_nofound_custom", "/test_put_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"DELETE", "/test_delete_nofound_custom", "/test_delete_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"HEAD", "/test_head_nofound_custom", "/test_head_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"OPTIONS", "/test_options_nofound_custom", "/test_options_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"CONNECT", "/test_connect_nofound_custom", "/test_connect_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"PATCH", "/test_patch_nofound_custom", "/test_patch_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			{"TRACE", "/test_trace_nofound_custom", "/test_trace_nofound_custom", "", notFoundMessage, 404, false, nil, nil},
			// SERVER INTERNAL ERROR 500 PANIC CUSTOM ERRORS - registered
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
		ctx.Writef("%s", notFoundMessage)
	})

	iris.OnError(iris.StatusInternalServerError, func(ctx *iris.Context) {
		ctx.Writef("%s", internalServerMessage)
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
	u.WriteString("Get Users\n")
}

// GET /users/:param1 which its value passed to the id argument
func (u testUserAPI) GetBy(id string) { // id equals to u.Param("param1")
	u.Writef("Get By %s\n", id)
}

// PUT /users
func (u testUserAPI) Put() {
	u.Writef("Put, name: %s\n", u.FormValue("name"))
}

// POST /users/:param1
func (u testUserAPI) PostBy(id string) {
	u.Writef("Post By %s, name: %s\n", id, u.FormValue("name"))
}

// DELETE /users/:param1
func (u testUserAPI) DeleteBy(id string) {
	u.Writef("Delete By %s\n", id)
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
			ctx.WriteString(middlewareResponseText)
			ctx.Next()
		} else {
			ctx.SetStatusCode(iris.StatusUnauthorized)
		}
	}}

	iris.API("/users", testUserAPI{}, h...)
	// test a simple .Party  with combination of .API
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
		ctx.WriteString(ctx.Method())
	}

	iris.Default.OnError(iris.StatusMethodNotAllowed, func(ctx *iris.Context) {
		ctx.WriteString("Hello from my custom 405 page")
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

func TestRedirectHTTPS(t *testing.T) {

	api := iris.New(iris.OptionIsDevelopment(true))

	host := "localhost:" + strconv.Itoa(getRandomNumber(1717, 9281))

	expectedBody := "Redirected to /redirected"

	api.Get("/redirect", func(ctx *iris.Context) { ctx.Redirect("/redirected") })
	api.Get("/redirected", func(ctx *iris.Context) { ctx.Text(iris.StatusOK, "Redirected to "+ctx.Path()) })
	defer listenTLS(api, host)()

	e := httptest.New(api, t)
	e.GET("/redirect").Expect().Status(iris.StatusOK).Body().Equal(expectedBody)
}
