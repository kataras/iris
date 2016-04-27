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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iriscontrol

import (
	"os"
	"strconv"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/plugin/routesinfo"
)

var pathSeperator = string(os.PathSeparator)
var pluginPath = os.Getenv("GOPATH") + pathSeperator + "src" + pathSeperator + "github.com" + pathSeperator + "kataras" + pathSeperator + "iris" + pathSeperator + "plugin" + pathSeperator + "iriscontrol" + pathSeperator
var assetsURL = "https://github.com/iris-contrib/iris-control-assets/archive/master.zip"
var assetsFolderName = "iris-control-assets-master"
var installationPath = pluginPath + assetsFolderName + pathSeperator

// for the plugin server
func (i *irisControlPlugin) startControlPanel() {

	// install the assets first
	if err := i.installAssets(); err != nil {
		i.pluginContainer.Printf("[%s] %s Error %s: Couldn't install the assets from the internet,\n make sure you are connecting to the internet the first time running the iris-control plugin", time.Now().UTC().String(), Name, err.Error())
		i.Destroy()
		return
	}

	i.server = iris.New()
	i.server.Templates(installationPath + "templates/*")
	i.setPluginsInfo()
	i.setPanelRoutes()

	go i.server.Listen(strconv.Itoa(i.options.Port))
	i.pluginContainer.Printf("[%s] %s is running at port %d with %d authenticated users", time.Now().UTC().String(), Name, i.options.Port, len(i.auth.authenticatedUsers))

}

type DashboardPage struct {
	ServerIsRunning bool
	Routes          []routesinfo.RouteInfo
	Plugins         []PluginInfo
}

func (i *irisControlPlugin) setPluginsInfo() {
	plugins := i.pluginContainer.GetAll()
	i.plugins = make([]PluginInfo, 0, len(plugins))
	for _, plugin := range plugins {
		i.plugins = append(i.plugins, PluginInfo{Name: i.pluginContainer.GetName(plugin), Description: i.pluginContainer.GetDescription(plugin)})
	}
}

// installAssets checks if must install ,if yes download the zip and unzip it, returns error.
func (i *irisControlPlugin) installAssets() (err error) {
	//we know already what is the zip folder inside it, so we can check if it's exists, if yes then don't install it again.
	if i.pluginContainer.GetDownloader().DirectoryExists(installationPath) {
		return
	}
	//set the installationPath ,although we know it but do it here too
	installationPath, err = i.pluginContainer.GetDownloader().Install(assetsURL, pluginPath)
	return err

}

func (i *irisControlPlugin) setPanelRoutes() {

	iris.Static("/public/*assets", installationPath+"static"+pathSeperator, 1)

	i.server.Get("/login", func(ctx *iris.Context) {
		ctx.RenderFile("login.html", nil)
	})

	i.server.Post("/login", func(ctx *iris.Context) {
		i.auth.login(ctx)
	})

	i.server.Use(i.auth)
	i.server.Get("/", func(ctx *iris.Context) {
		ctx.RenderFile("index.html", DashboardPage{ServerIsRunning: i.station.Server.IsListening(), Routes: i.routes.All(), Plugins: i.plugins})
	})

	i.server.Post("/logout", func(ctx *iris.Context) {
		i.auth.logout(ctx)
	})

	//the controls
	i.server.Post("/start_server", func(ctx *iris.Context) {
		//println("server start")
		old := i.stationServer
		if !old.IsSecure() {
			i.station.Listen(old.Options().ListeningAddr)
			//yes but here it does re- post listen to this plugin so ...
		} else {
			i.station.ListenTLS(old.Options().ListeningAddr, old.Options().CertFile, old.Options().KeyFile)
		}

	})

	i.server.Post("/stop_server", func(ctx *iris.Context) {
		//println("server stop")
		i.station.Close()
	})

}
