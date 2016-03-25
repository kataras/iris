// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iriscontrol

import (
	"github.com/kataras/iris"
	"os"
	"strconv"
	"time"
)

// for the plugin server
func (i *irisControlPlugin) startControlPanel() {
	i.server = iris.Custom(iris.StationOptions{Cache: false, Profile: false, PathCorrection: true})
	i.server.Templates(os.Getenv("GOPATH") + "/src/github.com/kataras/iris/plugins/iriscontrol/templates/*")

	i.setPanelRoutes()

	go i.server.Listen(strconv.Itoa(i.options.Port))
	i.pluginContainer.Printf("[%s] %s is running at port %d with %d authenticated users", time.Now().UTC().String(), Name, i.options.Port, len(i.auth.authenticatedUsers))

}

type DashboardPage struct {
	ServerIsRunning bool
	Routes          []RouteInfo
}

func (i *irisControlPlugin) setPanelRoutes() {

	i.server.Use(i.auth)

	i.server.Get("/", func(ctx *iris.Context) {
		ctx.RenderFile("index.html", DashboardPage{ServerIsRunning: i.station.Server.IsRunning, Routes: i.routes})
	})

	i.server.Get("/login", func(ctx *iris.Context) {
		ctx.RenderFile("login.html", nil)
	})

	i.server.Post("/login", func(ctx *iris.Context) {
		i.auth.login(ctx)
	})

	i.server.Post("/logout", func(ctx *iris.Context) {
		i.auth.logout(ctx)
	})

	//the controls
	i.server.Post("/start_server", func(ctx *iris.Context) {

	})

	i.server.Post("/stop_server", func(ctx *iris.Context) {

	})

}
