// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"strings"

	"github.com/kataras/iris"
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

// Attach adapts the websocket server to one or more Iris instances.
func (s *server) Attach(app *iris.Application) {
	wsPath := fixPath(s.config.Endpoint)
	if wsPath == "" {
		app.Log("websocket's configuration field 'Endpoint' cannot be empty, websocket server stops")
		return
	}

	wsClientSidePath := fixPath(s.config.ClientSourcePath)
	if wsClientSidePath == "" {
		app.Log("websocket's configuration field 'ClientSourcePath' cannot be empty, websocket server stops")
		return
	}

	// set the routing for client-side source (javascript) (optional)
	clientSideLookupName := "iris-websocket-client-side"
	wsHandler := s.Handler()
	app.Get(wsPath, wsHandler)
	// check if client side doesn't already exists
	if app.GetRoute(clientSideLookupName) == nil {
		// serve the client side on domain:port/iris-ws.js
		r, err := app.StaticContent(wsClientSidePath, "application/javascript", ClientSource)
		if err != nil {
			app.Log("websocket's route for javascript client-side library failed with: %v", err)
			return
		}
		r.Name = clientSideLookupName
	}
}
