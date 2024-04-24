package main

import (
	"time"

	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/middleware/monitor"
)

func main() {
	app := iris.New()

	// Initialize and start the monitor middleware.
	m := monitor.New(monitor.Options{
		RefreshInterval:     2 * time.Second,
		ViewRefreshInterval: 2 * time.Second,
		ViewTitle:           "MyServer Monitor",
	})
	// Manually stop monitoring on CMD/CTRL+C.
	iris.RegisterOnInterrupt(m.Stop)

	// Serve the actual server's process and operating system statistics as JSON.
	app.Post("/monitor", m.Stats)
	// Render with the default page.
	app.Get("/monitor", m.View)

	/* You can protect the /monitor under an /admin group of routes
	with basic authentication or any type authorization and authentication system.
	Example Code:

	  app.Post("/monitor", myProtectMiddleware, m.Stats)
	  app.Get("/monitor", myProtectMiddleware, m.View)
	*/

	/* You can also get the OS statistics using the Holder.GetStats method.
	Example Code:
	for {
		stats := m.Holder.GetStats()
		fmt.Printf("%#+v\n", stats)
		time.Sleep(time.Second)
	}

	Note that the same stats are also stored in the expvar metrics:
		- pid_cpu
		- pid_ram
		- pid_conns
		- os_cpu
		- os_ram
		- os_total_ram
		- os_load_avg
		- os_conns
	Check https://github.com/iris-contrib/middleware/tree/master/expmetric
	which can be integrated with datadog or other platforms.
	*/

	app.Get("/", handler)

	app.Listen(":8080")
}

func handler(ctx iris.Context) {
	ctx.WriteString("Test Index Handler")
}
