package rewrite_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kataras/iris/v14"
	. "github.com/kataras/iris/v14/middleware/rewrite"
)

func TestPrimarySubdomainRedirectRootToWWW(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		expectedStatus int
		expectedHost   string
	}{
		{
			name:           "Root domain should redirect",
			host:           "example.com",
			expectedStatus: http.StatusMovedPermanently,
			expectedHost:   "www.example.com",
		},
		{
			name:           "Subdomain should not redirect",
			host:           "blog.example.com",
			expectedStatus: http.StatusOK,
			expectedHost:   "",
		},
		{
			name:           "Localhost should redirect",
			host:           "localhost.dev",
			expectedStatus: http.StatusMovedPermanently,
			expectedHost:   "www.localhost.dev",
		},
		{
			name: "IP address should not redirect",
			// IP addresses don't have subdomains: Unlike domain names, IP addresses don’t have a hierarchical structure that includes subdomains.
			// Redirecting an IP address to a “www” subdomain doesn’t make sense because it’s not a valid operation for an IP address.
			host:           "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
	}

	e, err := New(Options{
		PrimarySubdomain: "www",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "http://"+tc.host, nil)
			originalHost, originalURLHost := request.Host, request.URL.Host
			responseRecorder := httptest.NewRecorder()

			app := iris.New()
			app.UseRouter(e.Handler)

			app.Get("/", func(ctx iris.Context) {
				ctx.Writef("Host: %s", ctx.Host())
			})

			if err := app.Build(); err != nil {
				t.Fatal(err)
			}
			app.ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, responseRecorder.Code)
			}

			if tc.expectedStatus == http.StatusMovedPermanently {
				location, err := responseRecorder.Result().Location()
				if err != nil {
					t.Fatal(err)
				}
				if location.Host != tc.expectedHost {
					t.Errorf("Expected host %s, got %s", tc.expectedHost, location.Host)
				}

				// The redirect URL must be built on a clone,
				// the incoming request stays intact.
				if request.Host != originalHost || request.URL.Host != originalURLHost {
					t.Errorf("the redirect must not mutate the incoming request, got Host: %q and URL.Host: %q", request.Host, request.URL.Host)
				}
			}
		})
	}
}
