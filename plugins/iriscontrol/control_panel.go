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

var pathSeperator = string(os.PathSeparator)
var pluginPath = os.Getenv("GOPATH") + pathSeperator + "src" + pathSeperator + "github.com" + pathSeperator + "kataras" + pathSeperator + "iris" + pathSeperator + "plugins" + pathSeperator + "iriscontrol" + pathSeperator
var assetsUrl = "https://github.com/iris-contrib/iris-control-assets/archive/master.zip"
var zipPath = pluginPath + "master.zip"
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

	i.server = iris.Custom(iris.StationOptions{Cache: false, Profile: false, PathCorrection: true})
	i.server.Templates(installationPath + "templates/*")
	i.setPluginsInfo()
	i.setPanelRoutes()

	go i.server.Listen(strconv.Itoa(i.options.Port))
	i.pluginContainer.Printf("[%s] %s is running at port %d with %d authenticated users", time.Now().UTC().String(), Name, i.options.Port, len(i.auth.authenticatedUsers))

}

type DashboardPage struct {
	ServerIsRunning bool
	Routes          []RouteInfo
	Plugins         []PluginInfo
}

func (i *irisControlPlugin) setPluginsInfo() {
	plugins := i.pluginContainer.GetAll()
	i.plugins = make([]PluginInfo, 0, len(plugins))
	for _, plugin := range plugins {
		i.plugins = append(i.plugins, PluginInfo{Name: plugin.GetName(), Description: plugin.GetDescription()})
	}
}

// installAssets checks if must install ,if yes download the zip and unzip it, returns error.
func (i *irisControlPlugin) installAssets() error {
	downloader := i.pluginContainer.GetDownloader()
	// if the directory exists then the assets already installed, just  return a nil error.
	if downloader.DirectoryExists(installationPath) {
		//remove here the  previous downloaded zip file if exists
		if downloader.DirectoryExists(zipPath) {
			os.Remove(zipPath)
		}

		return nil
	}
	var zipFile string
	var err error
	zipFile, err = downloader.DownloadZip(assetsUrl, pluginPath)
	if err == nil {
		err = downloader.Unzip(zipFile, pluginPath)
		if err == nil {
			// delete the master.zip after unzip
			// on windows I have problem because file is still open  (although I'm closing it inside the utils.unzip).
			// on linux could work without a sleep because linux can delete files even if they are open
			// so I move the os.Remove to the start of this server, so to the next start this zip (500kb) will be removed
			// os.Remove(zipFile)
		}
	}
	return err

}

func (i *irisControlPlugin) setPanelRoutes() {

	i.server.Get("/public/*assets", iris.Static(installationPath+"static"+pathSeperator, "/public/"))

	i.server.Get("/login", func(ctx *iris.Context) {
		ctx.RenderFile("login.html", nil)
	})

	i.server.Post("/login", func(ctx *iris.Context) {
		i.auth.login(ctx)
	})

	i.server.Use(i.auth)
	i.server.Get("/", func(ctx *iris.Context) {
		ctx.RenderFile("index.html", DashboardPage{ServerIsRunning: i.station.Server.IsRunning, Routes: i.routes, Plugins: i.plugins})
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
