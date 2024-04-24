// white-box testing

package host

import (
	"bytes"
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/iris-contrib/httpexpect/v2"
)

const (
	debugMode = false
)

func newTester(t *testing.T, baseURL string, handler http.Handler) *httpexpect.Expect {
	var transporter http.RoundTripper

	if strings.HasPrefix(baseURL, "http") { // means we are testing real serve time
		transporter = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS13},
		}
	} else { // means we are testing the handler itself
		transporter = httpexpect.NewBinder(handler)
	}

	testConfiguration := httpexpect.Config{
		BaseURL: baseURL,
		Client: &http.Client{
			Transport: transporter,
			Jar:       httpexpect.NewCookieJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	}

	if debugMode {
		testConfiguration.Printers = []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		}
	}

	return httpexpect.WithConfig(testConfiguration)
}

func testSupervisor(t *testing.T, creator func(*http.Server, []func(TaskHost)) *Supervisor) {
	loggerOutput := &bytes.Buffer{}
	logger := log.New(loggerOutput, "", 0)
	mu := new(sync.RWMutex)
	const (
		expectedHelloMessage = "Hello\n"
	)

	// http routing

	expectedBody := "this is the response body\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(expectedBody))
		if err != nil {
			t.Fatal(err)
		}
	})

	// host (server wrapper and adapter) construction

	srv := &http.Server{Handler: mux, ErrorLog: logger}
	addr := "localhost:5525"
	// serving
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		t.Fatal(err)
	}

	helloMe := func(_ TaskHost) {
		mu.Lock()
		logger.Print(expectedHelloMessage)
		mu.Unlock()
	}

	host := creator(srv, []func(TaskHost){helloMe})
	defer host.Shutdown(context.TODO())

	go host.Serve(ln)

	// http testsing and various calls
	// no need for time sleep because the following will take some time by theirselves
	tester := newTester(t, "http://"+addr, mux)
	tester.Request("GET", "/").Expect().Status(http.StatusOK).Body().IsEqual(expectedBody)

	// WARNING: Data Race here because we try to read the logs
	// but it's "safe" here.

	// testing Task (recorded) message:
	mu.RLock()
	got := loggerOutput.String()
	mu.RUnlock()
	if expectedHelloMessage != got {
		t.Fatalf("expected hello Task's message to be '%s' but got '%s'", expectedHelloMessage, got)
	}
}

func TestSupervisor(t *testing.T) {
	testSupervisor(t, func(srv *http.Server, tasks []func(TaskHost)) *Supervisor {
		su := New(srv)
		for _, t := range tasks {
			su.RegisterOnServe(t)
		}

		return su
	})
}
