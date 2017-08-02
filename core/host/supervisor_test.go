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
	"testing"

	"github.com/iris-contrib/httpexpect"
)

const (
	debug = false
)

func newTester(t *testing.T, baseURL string, handler http.Handler) *httpexpect.Expect {

	var transporter http.RoundTripper

	if strings.HasPrefix(baseURL, "http") { // means we are testing real serve time
		transporter = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else { // means we are testing the handler itself
		transporter = httpexpect.NewBinder(handler)
	}

	testConfiguration := httpexpect.Config{
		BaseURL: baseURL,
		Client: &http.Client{
			Transport: transporter,
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	}

	if debug {
		testConfiguration.Printers = []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		}
	}

	return httpexpect.WithConfig(testConfiguration)
}

func testSupervisor(t *testing.T, creator func(*http.Server, []func(TaskHost)) *Supervisor) {
	loggerOutput := &bytes.Buffer{}
	logger := log.New(loggerOutput, "", 0)
	const (
		expectedHelloMessage = "Hello\n"
	)

	// http routing
	var (
		expectedBody = "this is the response body\n"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedBody))
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
		logger.Print(expectedHelloMessage)
	}

	host := creator(srv, []func(TaskHost){helloMe})
	defer host.Shutdown(context.TODO())

	go host.Serve(ln)

	// http testsing and various calls
	// no need for time sleep because the following will take some time by theirselves
	tester := newTester(t, "http://"+addr, mux)
	tester.Request("GET", "/").Expect().Status(http.StatusOK).Body().Equal(expectedBody)

	// WARNING: Data Race here because we try to read the logs
	// but it's "safe" here.

	// testing Task (recorded) message:
	if got := loggerOutput.String(); expectedHelloMessage != got {
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
