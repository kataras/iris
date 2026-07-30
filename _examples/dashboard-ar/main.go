package main

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/v12"
)

var (
	startTime    = time.Now()
	requestCount atomic.Int64
)

func main() {
	app := iris.New()

	app.RegisterView(iris.HTML("./views", ".html"))
	app.HandleDir("/static", "./static")

	// Middleware to count requests.
	app.Use(func(ctx iris.Context) {
		requestCount.Add(1)
		ctx.Next()
	})

	app.Get("/", dashboardHandler)
	app.Get("/routes", routesHandler)
	app.Get("/api/stats", statsAPIHandler)

	app.Listen(":8080")
}

func dashboardHandler(ctx iris.Context) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	data := iris.Map{
		"Title":        "لوحة التحكم",
		"Page":         "dashboard",
		"Version":      iris.Version,
		"Uptime":       formatDuration(time.Since(startTime)),
		"Requests":     requestCount.Load(),
		"RoutesCount":  len(ctx.Application().GetRoutes()),
		"MemoryMB":     fmt.Sprintf("%.1f", float64(mem.Alloc)/1024/1024),
		"GoVersion":    runtime.Version(),
		"NumGoroutine": runtime.NumGoroutine(),
	}

	ctx.ViewLayout("layouts/main")
	if err := ctx.View("index", data); err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString(err.Error())
	}
}

func routesHandler(ctx iris.Context) {
	type routeInfo struct {
		Method  string
		Path    string
		Handler string
	}

	routes := ctx.Application().GetRoutes()
	routeList := make([]routeInfo, 0, len(routes))
	for _, r := range routes {
		routeList = append(routeList, routeInfo{
			Method:  r.Method,
			Path:    r.Path,
			Handler: r.MainHandlerName,
		})
	}

	data := iris.Map{
		"Title":  "المسارات المسجلة",
		"Page":   "routes",
		"Routes": routeList,
	}

	ctx.ViewLayout("layouts/main")
	if err := ctx.View("routes", data); err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString(err.Error())
	}
}

func statsAPIHandler(ctx iris.Context) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	ctx.JSON(iris.Map{
		"uptime":       formatDuration(time.Since(startTime)),
		"requests":     requestCount.Load(),
		"memoryMB":     fmt.Sprintf("%.1f", float64(mem.Alloc)/1024/1024),
		"numGoroutine": runtime.NumGoroutine(),
	})
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%d س %d د %d ث", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%d د %d ث", m, s)
	}
	return fmt.Sprintf("%d ث", s)
}
