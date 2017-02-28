// Package websocket provides an easy way to setup server and client side rich websocket experience for Iris
// As originally written by me at https://github.com/kataras/go-websocket based on v0.1.1
package websocket

import (
	"strings"

	"gopkg.in/kataras/iris.v6"
)

// New returns a new websocket server policy adaptor.
func New(cfg Config) Server {
	return &server{
		config: cfg.Validate(),
		rooms:  make(map[string][]string, 0),
		onConnectionListeners: make([]ConnectionFunc, 0),
	}
}

func fixPath(s string) string {
	if s == "" {
		return ""
	}

	if s[0] != '/' {
		s = "/" + s
	}

	s = strings.Replace(s, "//", "/", -1)
	return s
}

// Adapt implements the iris' adaptor, it adapts the websocket server to an Iris station.
func (s *server) Adapt(frame *iris.Policies) {
	// bind the server's Handler to Iris at Boot state
	evt := iris.EventPolicy{
		Boot: func(f *iris.Framework) {
			wsPath := fixPath(s.config.Endpoint)
			if wsPath == "" {
				f.Log(iris.DevMode, "websocket's configuration field 'Endpoint' cannot be empty, websocket server stops")
				return
			}

			wsClientSidePath := fixPath(s.config.ClientSourcePath)
			if wsClientSidePath == "" {
				f.Log(iris.DevMode, "websocket's configuration field 'ClientSourcePath' cannot be empty, websocket server stops")
				return
			}

			// set the routing for client-side source (javascript) (optional)
			clientSideLookupName := "iris-websocket-client-side"
			wsHandler := s.Handler()
			f.Get(wsPath, wsHandler)
			// check if client side doesn't already exists
			if f.Routes().Lookup(clientSideLookupName) == nil {
				// serve the client side on domain:port/iris-ws.js
				f.StaticContent(wsClientSidePath, "application/javascript", ClientSource).ChangeName(clientSideLookupName)
			}

			// If we want to show configuration fields... I'm not sure for this yet, so let it commented: f.Logf(iris.DevMode, "%#v", s.config)
		},

		Build: func(f *iris.Framework) {
			f.Log(iris.DevMode, "Serving Websockets on "+f.Config.VHost+s.config.Endpoint)
		},
	}

	evt.Adapt(frame)
}
