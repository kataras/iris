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
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/plugin/routesinfo"
	"github.com/kataras/iris/server"
)

const Name = "Iris Control"

type IrisControlOptions struct {
	Port  int
	Users map[string]string
}

type irisControlPlugin struct {
	options IrisControlOptions
	// the pluginContainer is the container which keeps this plugin from the main user's iris instance
	pluginContainer iris.IPluginContainer
	// the station object of the main  user's iris instance
	station *iris.Iris
	//a copy of the server which the main user's iris is listening for
	stationServer *server.Server

	// the server is this plugin's server object, it is managed by this plugin only
	server *iris.Iris
	//
	//infos
	routes  *routesinfo.Plugin
	plugins []PluginInfo
	//

	auth *userAuth
}

// New returns the plugin which is ready-to-use inside iris.Plugin method
// parameter is IrisControlOptions
func New(options IrisControlOptions) iris.IPlugin {
	i := &irisControlPlugin{}
	i.options = options
	i.routes = routesinfo.RoutesInfo()
	if auth := newUserAuth(options.Users); auth != nil {
		i.auth = auth
	} else {
		panic(Name + " Error: you should pass authenticated users map to the options, refer to the docs!")
	}

	return i
}

// Web set the options for the plugin and return the plugin which is ready-to-use inside iris.Plugin method
// first parameter is port
// second parameter is map of users (username:password)
func Web(port int, users map[string]string) iris.IPlugin {
	return New(IrisControlOptions{port, users})
}

// implement the base IPlugin

func (i *irisControlPlugin) Activate(container iris.IPluginContainer) error {
	i.pluginContainer = container
	container.Add(i.routes) // add the routesinfo plugin to the main server
	return nil
}

func (i irisControlPlugin) GetName() string {
	return Name
}

func (i irisControlPlugin) GetDescription() string {
	return Name + " is just a web interface which gives you control of your Iris.\n"
}

//

// implement the rest of the plugin

// PostHandle
func (i *irisControlPlugin) PostHandle(route iris.IRoute) {

}

// PostListen sets the station object after the main server starts
// starts the actual work of the plugin
func (i *irisControlPlugin) PostListen(s *iris.Iris) {
	//if the first time, because other times start/stop of the server so listen and no listen will be only from the control panel
	if i.station == nil {
		i.station = s
		i.stationServer = i.station.Server
		i.startControlPanel()
	}

}

func (i *irisControlPlugin) PreClose(s *iris.Iris) {
	// Do nothing. This is a wrapper of the main server if we destroy when users stop the main server then we cannot continue the control panel i.Destroy()
}

//

// Destroy removes entirely the plugin, the options and all of these properties, you cannot re-use this plugin after this method.
func (i *irisControlPlugin) Destroy() {
	i.pluginContainer.Remove(Name)

	i.options = IrisControlOptions{}
	i.routes = nil
	i.station = nil
	i.server.Close()
	i.pluginContainer = nil
	i.auth.Destroy()
	i.auth = nil
	i.pluginContainer.Printf("[%s] %s is turned off", time.Now().UTC().String(), Name)
}
