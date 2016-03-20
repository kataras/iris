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
package domain

import ()

// IPlugin is the interface which all Plugins must implement.
//
// A Plugin can register other plugins also from it's Activate state
type IPlugin interface {

	// GetName has to returns the name of the plugin, a name is unique
	// name has to be not dependent from other methods of the plugin,
	// because it is being called even before the Activate
	GetName() string
	// GetDescription has to returns the description of what the plugins is used for
	GetDescription() string

	// Activate called BEFORE the plugin being added to the plugins list,
	// if Activate returns none nil error then the plugin is not being added to the list
	// it is being called only one time
	//
	// PluginContainer parameter used to add other plugins if that's necessary by the plugin
	Activate(IPluginContainer) error

	// PreHandle it's being called every time BEFORE a Route is registed to the Router
	//
	// first parameter is the HTTP method
	// second is the Route
	PreHandle(string, IRoute)
	// PostHandle it's being called every time AFTER a Route successfully registed to the Router
	//
	// first parameter is the HTTP method
	// second is the Route
	PostHandle(string, IRoute)
	// PreListen it's being called only one time, BEFORE the Server is started (if .Listen called)
	// is used to do work at the time all other things are ready to go
	PreListen(IStation)
	// PostListen it's being called only one time, AFTER the Server is started (if .Listen called)
	// is used to do work when the server is running
	PostListen(IStation, error)

	// PreClose it's being called only one time, BEFORE the Iris .Close method
	// any plugin cleanup/clear memory happens here
	//
	// The plugin is deactivated after this state
	PreClose(IStation)
}

type IPluginContainer interface {
	Plugin(plugin IPlugin) error
	RemovePlugin(pluginName string)
	GetByName(pluginName string) IPlugin
	Printf(format string, a ...interface{})
	DoPreHandle(method string, route IRoute)
	DoPostHandle(method string, route IRoute)
	DoPreListen(station IStation)
	DoPostListen(station IStation, err error)
	DoPreClose(station IStation)
}
