// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package context

import (
	"io"
	"net/http"

	"github.com/kataras/iris/sessions"
)

// Application is the context's available Application instance, only things that are allowed to be happen inside the request are lived here.
type Application interface {
	// ConfigurationReadOnly returns all the available configuration values can be used on a request.
	ConfigurationReadOnly() ConfigurationReadOnly

	// Log uses the user's defined logger to log a warning or error message.
	Log(format string, a ...interface{})

	// View executes and write the result of a template file to the writer.
	//
	// Use context.View to render templates to the client instead.
	// Returns an error on failure, otherwise nil.
	View(writer io.Writer, filename string, layout string, bindingData interface{}) error

	// SessionManager returns the session manager which contain a Start and Destroy methods
	// used inside the context.Session().
	//
	// It's ready to use after the RegisterSessions.
	SessionManager() (sessions.Sessions, error)

	// ServeHTTPC is the internal router, it's visible because it can be used for advanced use cases,
	// i.e: routing within a foreign context.
	//
	// It is ready to use after Build state.
	ServeHTTPC(ctx Context)

	// ServeHTTP is the main router handler which calls the .Serve and acquires a new context from the pool.
	//
	// It is ready to use after Build state.
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// FireErrorCode executes an error http status code handler
	// based on the context's status code.
	//
	// If a handler is not already registered,
	// then it creates & registers a new trivial handler on the-fly.
	FireErrorCode(ctx Context)
}
