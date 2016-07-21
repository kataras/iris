package iris

/*
This is the part we only care, these are end-to-end tests for the mux(router) and the server, the whole http file is made for these reasons only, so these tests are enough I think.

CONTRIBUTE & DISCUSSION ABOUT TESTS TO: https://github.com/iris-contrib/tests
*/

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/gavv/httpexpect"
	"github.com/kataras/iris/config"
)

const (
	testTLSCert = `-----BEGIN CERTIFICATE-----
		MIIDAzCCAeugAwIBAgIJAPDsxtKV4v3uMA0GCSqGSIb3DQEBBQUAMBgxFjAUBgNV
		BAMMDTEyNy4wLjAuMTo0NDMwHhcNMTYwNjI5MTMxMjU4WhcNMjYwNjI3MTMxMjU4
		WjAYMRYwFAYDVQQDDA0xMjcuMC4wLjE6NDQzMIIBIjANBgkqhkiG9w0BAQEFAAOC
		AQ8AMIIBCgKCAQEA0KtAOHKrcbLwWJXgRX7XSFyu4HHHpSty4bliv8ET4sLJpbZH
		XeVX05Foex7PnrurDP6e+0H5TgqqcpQM17/ZlFcyKrJcHSCgV0ZDB3Sb8RLQSLns
		8a+MOSbn1WZ7TkC7d/cWlKmasQRHQ2V/cWlGooyKNEPoGaEz8MbY0wn2spyIJwsB
		dciERC6317VTXbiZdoD8QbAsT+tBvEHM2m2A7B7PQmHNehtyFNbSV5uZNodvv1uv
		ZTnDa6IqpjFLb1b2HNFgwmaVPmmkLuy1l9PN+o6/DUnXKKBrfPAx4JOlqTKEQpWs
		pnfacTE3sWkkmOSSFltAXfkXIJFKdS/hy5J/KQIDAQABo1AwTjAdBgNVHQ4EFgQU
		zr1df/c9+NyTpmyiQO8g3a8NswYwHwYDVR0jBBgwFoAUzr1df/c9+NyTpmyiQO8g
		3a8NswYwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQUFAAOCAQEACG5shtMSDgCd
		MNjOF+YmD+PX3Wy9J9zehgaDJ1K1oDvBbQFl7EOJl8lRMWITSws22Wxwh8UXVibL
		sscKBp14dR3e7DdbwCVIX/JyrJyOaCfy2nNBdf1B06jYFsIHvP3vtBAb9bPNOTBQ
		QE0Ztu9kCqgsmu0//sHuBEeA3d3E7wvDhlqRSxTLcLtgC1NSgkFvBw0JvwgpkX6s
		M5WpSBZwZv8qpplxhFfqNy8Uf+xrpSW0pGfkHumehkQGC6/Ry7raganS0aHhDPK9
		Z1bEJ2com1bFFAQsm9yIXrRVMGGCtihB2Au0Q4jpEjUbzWYM+ItZyvRAGRM6Qex6
		s/jogMeRsw==
		-----END CERTIFICATE-----
`
	testTLSKey = `-----BEGIN RSA PRIVATE KEY-----
	MIIEpQIBAAKCAQEA0KtAOHKrcbLwWJXgRX7XSFyu4HHHpSty4bliv8ET4sLJpbZH
	XeVX05Foex7PnrurDP6e+0H5TgqqcpQM17/ZlFcyKrJcHSCgV0ZDB3Sb8RLQSLns
	8a+MOSbn1WZ7TkC7d/cWlKmasQRHQ2V/cWlGooyKNEPoGaEz8MbY0wn2spyIJwsB
	dciERC6317VTXbiZdoD8QbAsT+tBvEHM2m2A7B7PQmHNehtyFNbSV5uZNodvv1uv
	ZTnDa6IqpjFLb1b2HNFgwmaVPmmkLuy1l9PN+o6/DUnXKKBrfPAx4JOlqTKEQpWs
	pnfacTE3sWkkmOSSFltAXfkXIJFKdS/hy5J/KQIDAQABAoIBAQDCd+bo9I0s8Fun
	4z3Y5oYSDTZ5O/CY0O5GyXPrSzCSM4Cj7EWEj1mTdb9Ohv9tam7WNHHLrcd+4NfK
	4ok5hLVs1vqM6h6IksB7taKATz+Jo0PzkzrsXvMqzERhEBo4aoGMIv2rXIkrEdas
	S+pCsp8+nAWtAeBMCn0Slu65d16vQxwgfod6YZfvMKbvfhOIOShl9ejQ+JxVZcMw
	Ti8sgvYmFUrdrEH3nCgptARwbx4QwlHGaw/cLGHdepfFsVaNQsEzc7m61fSO70m4
	NYJv48ZgjOooF5AccbEcQW9IxxikwNc+wpFYy5vDGzrBwS5zLZQFpoyMWFhtWdjx
	hbmNn1jlAoGBAPs0ZjqsfDrH5ja4dQIdu5ErOccsmoHfMMBftMRqNG5neQXEmoLc
	Uz8WeQ/QDf302aTua6E9iSjd7gglbFukVwMndQ1Q8Rwxz10jkXfiE32lFnqK0csx
	ltruU6hOeSGSJhtGWBuNrT93G2lmy23fSG6BqOzdU4rn/2GPXy5zaxM/AoGBANSm
	/E96RcBUiI6rDVqKhY+7M1yjLB41JrErL9a0Qfa6kYnaXMr84pOqVN11IjhNNTgl
	g1lwxlpXZcZh7rYu9b7EEMdiWrJDQV7OxLDHopqUWkQ+3MHwqs6CxchyCq7kv9Df
	IKqat7Me6Cyeo0MqcW+UMxlCRBxKQ9jqC7hDfZuXAoGBAJmyS8ImerP0TtS4M08i
	JfsCOY21qqs/hbKOXCm42W+be56d1fOvHngBJf0YzRbO0sNo5Q14ew04DEWLsCq5
	+EsDv0hwd7VKfJd+BakV99ruQTyk5wutwaEeJK1bph12MD6L4aiqHJAyLeFldZ45
	+TUzu8mA+XaJz+U/NXtUPvU9AoGBALtl9M+tdy6I0Fa50ujJTe5eEGNAwK5WNKTI
	5D2XWNIvk/Yh4shXlux+nI8UnHV1RMMX++qkAYi3oE71GsKeG55jdk3fFQInVsJQ
	APGw3FDRD8M4ip62ki+u+tEr/tIlcAyHtWfjNKO7RuubWVDlZFXqCiXmSdOMdsH/
	bxiREW49AoGACWev/eOzBoQJCRN6EvU2OV0s3b6f1QsPvcaH0zc6bgbBFOGmJU8v
	pXhD88tsu9exptLkGVoYZjR0n0QT/2Kkyu93jVDW/80P7VCz8DKYyAJDa4CVwZxO
	MlobQSunSDKx/CCJhWkbytCyh1bngAtwSAYLXavYIlJbAzx6FvtAIw4=
	-----END RSA PRIVATE KEY-----
`
)

func TestServerHost(t *testing.T) {
	var server1, server2, server3 Server
	var expectedHost1 = "mydomain.com:1993"
	var expectedHost2 = "mydomain.com:80"
	var expectedHost3 = config.DefaultServerHostname + ":9090"
	server1.Config.ListeningAddr = expectedHost1
	server2.Config.ListeningAddr = "mydomain.com"
	server3.Config.ListeningAddr = ":9090"

	server1.prepare()
	server2.prepare()
	server3.prepare()

	if server1.Host() != expectedHost1 {
		t.Fatalf("Expecting server 1's host to be %s but we got %s", expectedHost1, server1.Host())
	}
	if server2.Host() != expectedHost2 {
		t.Fatalf("Expecting server 2's host to be %s but we got %s", expectedHost2, server2.Host())
	}
	if server3.Host() != expectedHost3 {
		t.Fatalf("Expecting server 3's host to be %s but we got %s", expectedHost3, server3.Host())
	}
}

func TestServerHostname(t *testing.T) {
	var server Server
	var expectedHostname = "mydomain.com"
	server.Config.ListeningAddr = expectedHostname + ":1993"
	server.prepare()
	if server.Hostname() != expectedHostname {
		t.Fatalf("Expecting server's hostname to be %s but we got %s", expectedHostname, server.Hostname())
	}
}

func TestServerFullHost(t *testing.T) {
	var server1 Server
	var server2 Server
	server1.Config.ListeningAddr = "127.0.0.1:8080"
	server1.Config.CertFile = "notempty"
	server1.Config.KeyFile = "notempty"
	server2.Config.ListeningAddr = "127.0.0.1:8080"
	server1ExpectingFullhost := "https://" + server1.Config.ListeningAddr
	server2ExpectingFullhost := "http://" + server2.Config.ListeningAddr
	if server1.FullHost() != server1ExpectingFullhost {
		t.Fatalf("Expecting server 1's fullhost to be %s but got %s", server1ExpectingFullhost, server1.FullHost())
	}
	if server2.FullHost() != server2ExpectingFullhost {
		t.Fatalf("Expecting server 2's fullhost to be %s but got %s", server2ExpectingFullhost, server2.FullHost())
	}

}

func TestServerPort(t *testing.T) {
	var server1, server2 Server
	expectedPort1 := 8080
	expectedPort2 := 80
	server1.Config.ListeningAddr = "mydomain.com:8080"
	server2.Config.ListeningAddr = "mydomain.com"
	server1.prepare()
	server2.prepare()
	if server1.Port() != expectedPort1 {
		t.Fatalf("Expecting server 1's port to be %d but we got %d", expectedPort1, server1.Port())
	}
	if server2.Port() != expectedPort2 {
		t.Fatalf("Expecting server 2's port to be %d but we got %d", expectedPort2, server2.Port())
	}
}

// Contains the server test for multi running servers
func TestMultiRunningServers_v1(t *testing.T) {
	host := "mydomain.com:443" // you have to add it to your hosts file( for windows, as 127.0.0.1 mydomain.com)
	initDefault()
	Config.DisableBanner = true
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
		ctx.Write("Hello from %s", ctx.HostString())
	})

	// start the secondary server
	AddServer(config.Server{ListeningAddr: "mydomain.com:80", RedirectTo: "https://" + host, Virtual: true})
	// start the main server
	go ListenTo(config.Server{ListeningAddr: host, CertFile: certFile.Name(), KeyFile: keyFile.Name(), Virtual: true})
	// prepare test framework
	if ok := <-Available; !ok {
		t.Fatal("Unexpected error: server cannot start, please report this as bug!!")
	}

	e := Tester(t)

	e.Request("GET", "http://mydomain.com:80").Expect().Status(StatusOK).Body().Equal("Hello from " + host)
	e.Request("GET", "https://"+host).Expect().Status(StatusOK).Body().Equal("Hello from " + host)

}

// Contains the server test for multi running servers
func TestMultiRunningServers_v2(t *testing.T) {
	domain := "mydomain.com"
	host := domain + ":443"
	initDefault()
	Config.DisableBanner = true
	Config.Tester.ListeningAddr = host
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
		ctx.Write("Hello from %s", ctx.HostString())
	})

	// add a secondary server
	Servers.Add(config.Server{ListeningAddr: domain + ":80", RedirectTo: "https://" + host, Virtual: true})
	// add our primary/main server
	Servers.Add(config.Server{ListeningAddr: host, CertFile: certFile.Name(), KeyFile: keyFile.Name(), Virtual: true})

	go Go()

	// prepare test framework
	if ok := <-Available; !ok {
		t.Fatal("Unexpected error: server cannot start, please report this as bug!!")
	}

	e := Tester(t)

	e.Request("GET", "http://"+domain+":80").Expect().Status(StatusOK).Body().Equal("Hello from " + host)
	e.Request("GET", "https://"+host).Expect().Status(StatusOK).Body().Equal("Hello from " + host)

}

const (
	testEnableSubdomain = true
	testSubdomain       = "mysubdomain"
)

func testSubdomainHost() string {
	s := testSubdomain + "." + Servers.Main().Host()
	return s
}

func testSubdomainURL() (subdomainURL string) {
	subdomainHost := testSubdomainHost()
	if Servers.Main().IsSecure() {
		subdomainURL = "https://" + subdomainHost
	} else {
		subdomainURL = "http://" + subdomainHost
	}
	return
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

	e := Tester(t)

	request := func(reqPath string) {
		e.Request("GET", reqPath).
			Expect().
			Status(StatusOK).Body().Equal(Servers.Main().Host() + reqPath)
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

func TestMuxEncodeURL(t *testing.T) {
	initDefault()

	Get("/encoding/:url", func(ctx *Context) {
		url := URLEncode(ctx.Param("url"))
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
