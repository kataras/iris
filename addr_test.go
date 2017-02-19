// Black-box Testing
package iris_test

import (
	"context"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/iris-contrib/httpexpect"
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/httptest"
)

func getRandomNumber(min int, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

const (
	testEnableSubdomain = true
	testSubdomain       = "mysubdomain"
)

func testSubdomainHost(host string) string {
	s := testSubdomain + "." + host
	return s
}

func testSubdomainURL(scheme string, host string) string {
	subdomainHost := testSubdomainHost(host)
	return scheme + subdomainHost
}

func subdomainTester(e *httpexpect.Expect, app *iris.Framework) *httpexpect.Expect {
	es := e.Builder(func(req *httpexpect.Request) {
		req.WithURL(testSubdomainURL(app.Config.VScheme, app.Config.VHost))
	})
	return es
}

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

// works as
// defer listenTLS(iris.Default, hostTLS)()
func listenTLS(app *iris.Framework, hostTLS string) func() {
	// create the key and cert files on the fly, and delete them when this test finished
	certFile, ferr := ioutil.TempFile("", "cert")

	if ferr != nil {
		panic(ferr)
	}

	keyFile, ferr := ioutil.TempFile("", "key")
	if ferr != nil {
		panic(ferr)
	}

	certFile.WriteString(testTLSCert)
	keyFile.WriteString(testTLSKey)

	go app.ListenTLS(hostTLS, certFile.Name(), keyFile.Name())
	time.Sleep(200 * time.Millisecond)

	return func() {
		app.Shutdown(context.Background())

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
	app := iris.New()
	app.Adapt(newTestNativeRouter())

	host := "localhost"
	hostTLS := host + ":" + strconv.Itoa(getRandomNumber(1919, 2021))
	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from %s", hostTLS)
	})

	defer listenTLS(app, hostTLS)()

	e := httptest.New(app, t, httptest.ExplicitURL(true))
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
	app := iris.New()
	app.Adapt(newTestNativeRouter())

	domain := "localhost"
	hostTLS := domain + ":" + strconv.Itoa(getRandomNumber(2222, 2229))
	srv1Host := domain + ":" + strconv.Itoa(getRandomNumber(4446, 5444))
	srv2Host := domain + ":" + strconv.Itoa(getRandomNumber(7778, 8887))

	app.Get("/", func(ctx *iris.Context) {
		ctx.Writef("Hello from %s", hostTLS)
	})

	defer listenTLS(app, hostTLS)()

	// using the same iris' handler but not as proxy, just the same handler
	srv2 := &http.Server{Handler: app.Router, Addr: srv2Host}
	go srv2.ListenAndServe()

	// using the proxy handler
	srv1 := &http.Server{Handler: iris.ProxyHandler("https://" + hostTLS), Addr: srv1Host}
	go srv1.ListenAndServe()
	time.Sleep(200 * time.Millisecond) // wait a little for the http servers

	e := httptest.New(app, t, httptest.ExplicitURL(true))
	e.Request("GET", "/").Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

	eproxy1 := httptest.NewInsecure("http://"+srv1Host, t, httptest.ExplicitURL(true))
	eproxy1.Request("GET", "/").Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

	eproxy2 := httptest.NewInsecure("http://"+srv2Host, t)
	eproxy2.Request("GET", "/").Expect().Status(iris.StatusOK).Body().Equal("Hello from " + hostTLS)

}
