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
	station *iris.Station
	// the server is this plugin's server object, it is managed by this plugin only
	server *iris.Station
	//
	//infos
	routes  *routesinfo.RoutesInfoPlugin
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
	container.Plugin(i.routes) // add the routesinfo plugin to the main server
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

// PreListen sets the station object before the main server starts
// and starts the actual work of the plugin
func (i *irisControlPlugin) PostListen(s *iris.Station) {
	i.station = s
	i.startControlPanel()

}

func (i *irisControlPlugin) PreClose(s *iris.Station) {
	i.Destroy()
}

//

// Destroy removes entirely the plugin, the options and all of these properties, you cannot re-use this plugin after this method.
func (i *irisControlPlugin) Destroy() {
	i.pluginContainer.RemovePlugin(Name)

	i.options = IrisControlOptions{}
	i.routes = nil
	i.station = nil
	i.server.Close()
	i.pluginContainer = nil
	i.auth.Destroy()
	i.auth = nil
	i.pluginContainer.Printf("[%s] %s is turned off", time.Now().UTC().String(), Name)
}
