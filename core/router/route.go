// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
	"fmt"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router/macro"
)

// Route contains the information about a registered Route.
// If any of the following fields are changed then the
// caller should Refresh the router.
type Route struct {
	Name      string          // "userRoute"
	Method    string          // "GET"
	Subdomain string          // "admin."
	tmpl      *macro.Template // Tmpl().Src: "/api/user/{id:int}"
	Path      string          // "/api/user/:id"
	Handlers  context.Handlers
	// FormattedPath all dynamic named parameters (if any) replaced with %v,
	// used by Application to validate param values of a Route based on its name.
	FormattedPath string
}

// NewRoute returns a new route based on its method,
// subdomain, the path (unparsed or original),
// handlers and the macro container which all routes should share.
// It parses the path based on the "macros",
// handlers are being changed to validate the macros at serve time, if needed.
func NewRoute(method, subdomain, unparsedPath string,
	handlers context.Handlers, macros *macro.Map) (*Route, error) {

	tmpl, err := macro.Parse(unparsedPath, macros)
	if err != nil {
		return nil, err
	}

	path, handlers, err := compileRoutePathAndHandlers(handlers, tmpl)
	if err != nil {
		return nil, err
	}

	path = cleanPath(path) // maybe unnecessary here but who cares in this moment
	defaultName := method + subdomain + path
	formattedPath := formatPath(path)

	route := &Route{
		Name:          defaultName,
		Method:        method,
		Subdomain:     subdomain,
		tmpl:          tmpl,
		Path:          path,
		Handlers:      handlers,
		FormattedPath: formattedPath,
	}
	return route, nil
}

// Tmpl returns the path template, i
// it contains the parsed template
// for the route's path.
// May contain zero named parameters.
//
// Developer can get his registered path
// via Tmpl().Src, Route.Path is the path
// converted to match the underline router's specs.
func (r Route) Tmpl() macro.Template {
	return *r.tmpl
}

// IsOnline returns true if the route is marked as "online" (state).
func (r Route) IsOnline() bool {
	return r.Method != MethodNone
}

// formats the parsed to the underline path syntax.
// path = "/api/users/:id"
// return "/api/users/%v"
//
// path = "/files/*file"
// return /files/%v
//
// path = "/:username/messages/:messageid"
// return "/%v/messages/%v"
// we don't care about performance here, it's prelisten.
func formatPath(path string) string {
	if strings.Contains(path, ParamStart) || strings.Contains(path, WildcardParamStart) {
		var (
			startRune         = ParamStart[0]
			wildcardStartRune = WildcardParamStart[0]
		)

		var formattedParts []string
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if len(part) == 0 {
				continue
			}
			if part[0] == startRune || part[0] == wildcardStartRune {
				// is param or wildcard param
				part = "%v"
			}
			formattedParts = append(formattedParts, part)
		}

		return "/" + strings.Join(formattedParts, "/")
	}
	// the whole path is static just return it
	return path
}

// ResolvePath returns the formatted path's %v replaced with the args.
func (r Route) ResolvePath(args ...string) string {
	rpath, formattedPath := r.Path, r.FormattedPath
	if rpath == formattedPath {
		// static, no need to pass args
		return rpath
	}
	// check if we have /*, if yes then join all arguments to one as path and pass that as parameter
	if rpath[len(rpath)-1] == WildcardParamStart[0] {
		parameter := strings.Join(args, "/")
		return fmt.Sprintf(formattedPath, parameter)
	}
	// else return the formattedPath with its args,
	// the order matters.
	for _, s := range args {
		formattedPath = strings.Replace(formattedPath, "%v", s, 1)
	}
	return formattedPath
}
