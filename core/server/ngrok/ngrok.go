// Package ngrok opens ngrok tunnels for the Iris server. Import it for its
// side effects when ServerOptions.Ngrok (iris.NgrokOptions) is used:
//
//	import _ "github.com/kataras/iris/v14/core/server/ngrok"
//
// The package registers its listener constructor with the core/server package
// from its init function, there is nothing to call. Without the import the
// server fails to start when Ngrok options are set, and the ngrok SDK stays
// out of the application's dependency graph.
package ngrok

import (
	"context"
	"fmt"
	"strings"

	"github.com/kataras/iris/v14/core/server"

	"golang.ngrok.com/ngrok/v2"
)

func init() {
	server.RegisterNgrokListener(newNgrokListener)
}

func newNgrokListener(ctx context.Context, ngrokConfig *server.NgrokOptions, hostPatterns []string, addr string) (server.NgrokTunnelListener, error) {
	// Create agent options for authentication
	var agentOpts []ngrok.AgentOption
	if ngrokConfig.AuthToken != "" {
		agentOpts = append(agentOpts, ngrok.WithAuthtoken(ngrokConfig.AuthToken))
	}
	// For v2, if no auth token is provided, we'll use the DefaultAgent which uses NGROK_AUTHTOKEN env var

	// Create agent if we have specific options, otherwise use default
	var agent ngrok.Agent
	var err error
	if len(agentOpts) > 0 {
		agent, err = ngrok.NewAgent(agentOpts...)
		if err != nil {
			return nil, err
		}
	} else {
		agent = ngrok.DefaultAgent
	}

	// Prepare endpoint options
	var endpointOpts []ngrok.EndpointOption
	// forwarder, _ := ngrok.Forward(ctx, ngrok.WithUpstream(addr))
	// forwarder.Agent().Listen()
	// Domain configuration - if specified, use it; otherwise ngrok will assign a random one
	if len(hostPatterns) > 0 && !server.IsLoopback(hostPatterns[0]) {
		endpointOpts = append(endpointOpts, ngrok.WithURL("https://"+hostPatterns[0]))
	}

	// Build Traffic Policy for v2 features (compression, OAuth, etc.)
	trafficPolicy := buildTrafficPolicy(ngrokConfig)
	if trafficPolicy != "" {
		endpointOpts = append(endpointOpts, ngrok.WithTrafficPolicy(trafficPolicy))
	}

	// Create listener
	listener, err := agent.Listen(ctx, endpointOpts...)
	if err != nil {
		return nil, err
	}

	return &endpointListenerWrapper{EndpointListener: listener}, nil
}

// buildTrafficPolicy creates a Traffic Policy configuration for ngrok v2
// to replicate the features that were available in v1 (compression, OAuth, etc.)
func buildTrafficPolicy(ngrokConfig *server.NgrokOptions) string {
	if ngrokConfig == nil {
		return ""
	}

	// Check if we need any policies
	needsPolicy := ngrokConfig.Compression || len(ngrokConfig.Oauth) > 0
	if !needsPolicy {
		return ""
	}

	// Build Traffic Policy JSON configuration
	policy := `{
  "inbound": [`

	var rules []string

	// Add compression policy if enabled
	if ngrokConfig.Compression {
		compressionRule := `{
    "name": "compression",
    "actions": [{
      "type": "compress",
      "config": {
        "algorithm": ["br", "gzip", "deflate"]
      }
    }]
  }`
		rules = append(rules, compressionRule)
	}

	// Add OAuth policy if configured
	if len(ngrokConfig.Oauth) > 0 {
		for provider, domain := range ngrokConfig.Oauth {
			var oauthConfig string
			if domain != "" {
				// OAuth with domain restriction
				oauthConfig = fmt.Sprintf(`{
    "name": "oauth_%s",
    "actions": [{
      "type": "oauth",
      "config": {
        "provider": "%s",
        "allowed_domains": ["%s"]
      }
    }]
  }`, provider, provider, domain)
			} else {
				// OAuth without domain restriction
				oauthConfig = fmt.Sprintf(`{
    "name": "oauth_%s",
    "actions": [{
      "type": "oauth",
      "config": {
        "provider": "%s"
      }
    }]
  }`, provider, provider)
			}
			rules = append(rules, oauthConfig)
		}
	}

	// Join rules and complete the policy
	if len(rules) > 0 {
		policy += strings.Join(rules, ",\n")
	}

	policy += `
  ]
}`

	return policy
}

type endpointListenerWrapper struct {
	ngrok.EndpointListener
}

// Close wraps the listener's Close method.
func (w *endpointListenerWrapper) Close() error {
	return w.EndpointListener.Close()
}
