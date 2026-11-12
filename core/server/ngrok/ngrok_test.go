package ngrok

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14/core/server"
)

// https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables.
var runsInCI = os.Getenv("CI") == "true"

func skipTestError(err error) bool {
	return err != nil && runsInCI && strings.Contains(err.Error(), "permission denied")
}

func loadNgrokAuthTokenAndDomain() (string, string, bool) {
	authToken := strings.TrimSpace(os.Getenv("NGROK_AUTHTOKEN"))
	staticDomain := strings.TrimSpace(os.Getenv("NGROK_DOMAIN"))
	return authToken, staticDomain, authToken != "" && staticDomain != ""
}

// TestImportRegistersListener checks that importing the package is enough to
// register the listener constructor with core/server.
func TestImportRegistersListener(t *testing.T) {
	if server.RegisteredNgrokListener() == nil {
		t.Fatal("importing core/server/ngrok did not register a listener with core/server")
	}
}

// TestNewNgrokListener tests the newNgrokListener function.
func TestNewNgrokListener(t *testing.T) {
	authToken, staticDomain, ok := loadNgrokAuthTokenAndDomain()
	if !ok { // skip this test, as it doesn't have the necessary environment variable.
		/*
			Go to your GitHub repository.

			1. Click on Settings.
			2. On the left sidebar, click on Secrets.
			3. Choose Actions.
			4. Click on New repository secret.
			5. Name your secret (e.g. NGROK_AUTHTOKEN, NGROK_DOMAIN).
			6. Enter the value of your ngrok authtoken.
			7. Click Add secret.

			Once you’ve added the secret, you can use it in your GitHub Actions workflow by referencing it with the ${{ secrets.NGROK_AUTHTOKEN }}
			syntax in your .yml configuration file.
			This ensures that your authtoken is securely stored and not exposed in your repository’s code or logs.
		*/
		t.Skipf("Skipping TestNewNgrokListener as it can't retrieve the auth token or the static domain to test from the environment.")
		return
	}

	tests := []struct {
		name         string
		hostPatterns []string
		ngrokConfig  *server.NgrokOptions
		wantErr      bool
	}{
		{"NoHostPatternNoAuth", []string{}, &server.NgrokOptions{AuthToken: authToken, Compression: true}, false}, // generates random URL.
		{"WithHostPattern", []string{staticDomain}, &server.NgrokOptions{AuthToken: authToken, Compression: false}, false},
		{"WithOAuth", []string{staticDomain}, &server.NgrokOptions{AuthToken: authToken, Oauth: map[string]string{"github": "github.com"}}, false},
		{"InvalidHostPattern", []string{"..invalid"}, &server.NgrokOptions{AuthToken: authToken}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			tunnel, err := newNgrokListener(ctx, tt.ngrokConfig, tt.hostPatterns, "")
			if tunnel != nil {
				tunnel.Close()
			}

			if (err != nil) != tt.wantErr {
				if skipTestError(err) {
					return
				}

				t.Errorf("newNgrokListener() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tunnel != nil {
				if len(tt.hostPatterns) > 0 { // compare the expected domain, it should match the given one (if any).
					wantDomain := tt.hostPatterns[0]
					// URL is part of the type newNgrokListener returns
					// (server.NgrokTunnelListener), it is what the server reads.
					if domain := tunnel.URL().Hostname(); domain != wantDomain {
						t.Errorf("newNgrokListener() got URL host = %v, want %v", domain, wantDomain)
					}
				}
			}
		})
	}
}

// TestServerNgrokListener starts a server with ngrok options through the
// registered listener and checks the public URL it reports.
func TestServerNgrokListener(t *testing.T) {
	_, staticDomain, ok := loadNgrokAuthTokenAndDomain()
	if !ok {
		t.Skip("Skipping TestServerNgrokListener as it can't retrieve the auth token or the static domain to test from the environment.")
	}

	s := server.NewServer(server.ServerOptions{
		Ngrok: &server.NgrokOptions{ /* AuthToken from OS variable. */ },
		Hosts: []string{staticDomain},
	})

	created := make(chan *url.URL, 1)
	s.OnListenerCreated(func(_ net.Listener, u *url.URL) error {
		created <- u
		return nil
	})

	go s.ListenAndServe(http.NotFoundHandler())

	select {
	case u := <-created:
		if want := "https://" + staticDomain; u.String() != want {
			t.Errorf("got URL = %v, want %v", u, want)
		}
	case <-time.After(15 * time.Second):
		t.Error("the ngrok listener was not created in time")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}
