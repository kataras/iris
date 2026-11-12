package router

import (
	"fmt"
	"slices"
	"time"

	"github.com/kataras/golog"
)

func printRoutesInfo(logger *golog.Logger, registeredRoutes []*Route, noLogCount int) { // MUST be called after build of routes.
	if !(logger != nil && logger.Level == golog.DebugLevel && noLogCount < len(registeredRoutes)) {
		return
	}

	terminal, ok := logger.Printer.Terminal()
	if !ok {
		return
	}

	// group routes by method and print them without the [DBUG] and time info,
	// the route logs are colorful.
	// Note: don't use map, we need to keep registered order, use
	// different slices for each method.

	collect := func(method string) (methodRoutes []*Route) {
		for _, r := range registeredRoutes {
			if r.NoLog {
				continue
			}
			if r.Method == method {
				methodRoutes = append(methodRoutes, r)
			}
		}

		return
	}

	type MethodRoutes struct {
		method string
		routes []*Route
	}

	// slices.Concat allocates a fresh backing array, so we never accidentally
	// mutate the exported AllMethods slice if a caller has expanded its capacity
	// via RegisterMethods.
	allMethods := slices.Concat(AllMethods, []string{MethodNone, ""})
	methodRoutes := make([]MethodRoutes, 0, len(allMethods))

	for _, method := range allMethods {
		routes := collect(method)
		if len(routes) > 0 {
			methodRoutes = append(methodRoutes, MethodRoutes{method, routes})
		}
	}

	if n := len(methodRoutes); n > 0 {
		tr := "routes"
		if len(registeredRoutes) == 1 {
			tr = tr[0 : len(tr)-1]
		}

		debugLevel := golog.Levels[golog.DebugLevel]
		// Replace that in order to not transfer it to the log handler (e.g. json)
		// logger.Debugf("API: %d registered %s (", len(registeredRoutes), tr)
		// with:
		terminal.WriteRich(debugLevel.Title, debugLevel.ColorCode, debugLevel.Style...)
		terminal.Printf(" %s %sAPI: %d registered %s (", time.Now().Format(logger.TimeFormat), logger.Prefix, len(registeredRoutes)-noLogCount, tr)
		//

		for i, m := range methodRoutes {
			// @method: @count
			if i > 0 {
				if i == n-1 {
					fmt.Fprint(logger.Printer, " and ")
				} else {
					fmt.Fprint(logger.Printer, ", ")
				}
			}
			if m.method == "" {
				m.method = "ERROR"
			}
			terminal.Printf("%d ", len(m.routes))
			terminal.WriteRich(m.method, TraceTitleColorCode(m.method))
		}

		terminal.WriteString(")\n")
	}

	for i, m := range methodRoutes {
		for _, r := range m.routes {
			r.Trace(terminal, -1)
		}

		if i != len(allMethods)-1 {
			terminal.WriteString("\n")
		}
	}
}
