package examples

import (
	"net/http"
	"os"
	"testing"

	"google.golang.org/appengine/aetest"

	"github.com/gavv/httpexpect"
)

// init() is used by GAE to start serving the app
// added here for illustration purposes
//
// func init() {
//     http.Handle("/", GaeHandler())
// }

// gaeInstance is our global dev_appserver instance.
var gaeInstance aetest.Instance

// TestMain is called first to create the gaeInstance.
func TestMain(m *testing.M) {
	// INFO: Remove the return to actually run the tests.
	// Requires installed Google Appengine SDK.
	// https://cloud.google.com/appengine/downloads
	return

	var err error
	gaeInstance, err = aetest.NewInstance(nil)
	if err != nil {
		panic(err)
	}

	c := m.Run() // call all actual tests
	gaeInstance.Close()
	os.Exit(c)
}

// newHttpExpect returns a new Expect instance for testing.
func newHttpExpect(t *testing.T) *httpexpect.Expect {
	return httpexpect.WithConfig(httpexpect.Config{
		// Use gaeInstance to create requests.
		// aetest.Instance is compatible with httpexpect.RequestFactory.
		RequestFactory: gaeInstance,

		// Pass requests directly to GaeHandler.
		Client: &http.Client{
			Transport: httpexpect.NewBinder(GaeHandler()),
			Jar:       httpexpect.NewJar(),
		},

		// Report errors using testify.
		Reporter: httpexpect.NewAssertReporter(t),
	})
}

// TestPing is an actual tests, using the global gaeInstance
func TestPing(t *testing.T) {
	e := newHttpExpect(t)
	e.GET("/ping").Expect().Status(200)
}
