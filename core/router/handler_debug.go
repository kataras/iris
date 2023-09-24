package router

import (
	"fmt"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/pio"
)

func printRoutesInfo(logger *golog.Logger, registeredRoutes []*Route, noLogCount int) {
	if !(logger != nil && logger.Level == golog.DebugLevel && noLogCount < len(registeredRoutes)) {
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

	allMethods := append(AllMethods, []string{MethodNone, ""}...)
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

		bckpNewLine := logger.NewLine
		logger.NewLine = false
		debugLevel := golog.Levels[golog.DebugLevel]
		// Replace that in order to not transfer it to the log handler (e.g. json)
		// logger.Debugf("API: %d registered %s (", len(registeredRoutes), tr)
		// with:
		pio.WriteRich(logger.Printer, debugLevel.Title, debugLevel.ColorCode, debugLevel.Style...)
		fmt.Fprintf(logger.Printer, " %s %sAPI: %d registered %s (", time.Now().Format(logger.TimeFormat), logger.Prefix, len(registeredRoutes)-noLogCount, tr)
		//
		logger.NewLine = bckpNewLine

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
			fmt.Fprintf(logger.Printer, "%d ", len(m.routes))
			pio.WriteRich(logger.Printer, m.method, TraceTitleColorCode(m.method))
		}

		fmt.Fprint(logger.Printer, ")\n")
	}

	for i, m := range methodRoutes {
		for _, r := range m.routes {
			r.Trace(logger.Printer, -1)
		}

		if i != len(allMethods)-1 {
			logger.Printer.Write(pio.NewLine)
		}
	}
}
