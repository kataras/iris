package iris

import (
	"fmt"
)

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
	Activate(*PluginContainer) error

	// PreHandle it's being called every time BEFORE a Route is registed to the Router
	//
	// first parameter is the HTTP method
	// second is the Route
	PreHandle(string, *Route)
	// PostHandle it's being called every time AFTER a Route successfuly registed to the Router
	//
	// first parameter is the HTTP method
	// second is the Route
	PostHandle(string, *Route)

	// PreBuild it's being called only one time, BEFORE the Build state, this is the most useful event
	PreBuild(*Station)
	// PostBuild it's being called only one time, AFTER the Build state finished, BEFORE the Listen
	PostBuild(*Station)

	// PostListen it's being called only one time, AFTER the Server is started (if .Listen called)
	// is used to do work when the server is running
	PostListen(*Station, error)

	// PreClose it's being called only one time, BEFORE the Iris .Close method
	// any plugin cleanup/clear memory happens here
	//
	// The plugin is deactivated after this state
	PreClose(*Station)
}

type PluginContainer struct {
	activatedPlugins []IPlugin
}

// Plugin activates the plugins and if succeed then adds it to the activated plugins list
func (p *PluginContainer) Plugin(plugin IPlugin) error {
	if p.activatedPlugins == nil {
		p.activatedPlugins = make([]IPlugin, 0)
	}

	// Check if the plugin already exists
	if p.GetByName(plugin.GetName()) != nil {
		return fmt.Errorf("[Iris] Error on Plugin: %s is already exists: %s", plugin.GetName(), plugin.GetDescription)
	}
	// Activate the plugin, if no error then add it to the plugins
	err := plugin.Activate(p)
	if err != nil {
		return err
	}
	// All ok, add it to the plugins list
	p.activatedPlugins = append(p.activatedPlugins, plugin)

	return nil
}

// RemovePlugin DOES NOT calls the plugin.PreClose method but it removes it completely from the plugins list
func (p *PluginContainer) RemovePlugin(pluginName string) {
	if p.activatedPlugins == nil {
		return
	}
	indexToRemove := -1
	for i := 0; i < len(p.activatedPlugins); i++ {
		if p.activatedPlugins[i].GetName() == pluginName {
			indexToRemove = i
		}
	}

	if indexToRemove != -1 {
		p.activatedPlugins = append(p.activatedPlugins[:indexToRemove], p.activatedPlugins[indexToRemove+1:]...)
	}
}

// GetByName returns a plugin instance by it's name
func (p *PluginContainer) GetByName(pluginName string) IPlugin {
	if p.activatedPlugins == nil {
		return nil
	}

	for i := 0; i < len(p.activatedPlugins); i++ {
		if p.activatedPlugins[i].GetName() == pluginName {
			return p.activatedPlugins[i]
		}
	}

	return nil
}

// Printf sends plain text to any registed logger (future), some plugins maybe want use this method
// maybe at the future I change it, instead of sync even-driven to async channels...
func (p *PluginContainer) Printf(format string, a ...interface{}) {
	fmt.Printf(format, a...) //for now just this.
}

// These methods are the callers for all plugins' events

func (p *PluginContainer) doPreHandle(method string, route *Route) {
	for i := 0; i < len(p.activatedPlugins); i++ {
		p.activatedPlugins[i].PreHandle(method, route)
	}
}

func (p *PluginContainer) doPostHandle(method string, route *Route) {
	for i := 0; i < len(p.activatedPlugins); i++ {
		p.activatedPlugins[i].PostHandle(method, route)
	}
}

func (p *PluginContainer) doPreBuild(station *Station) {
	for i := 0; i < len(p.activatedPlugins); i++ {
		p.activatedPlugins[i].PreBuild(station)
	}
}

func (p *PluginContainer) doPostBuild(station *Station) {
	for i := 0; i < len(p.activatedPlugins); i++ {
		p.activatedPlugins[i].PostBuild(station)
	}
}

func (p *PluginContainer) doPostListen(station *Station, err error) {
	for i := 0; i < len(p.activatedPlugins); i++ {
		p.activatedPlugins[i].PostListen(station, err)
	}
}

func (p *PluginContainer) doPreClose(station *Station) {
	for i := 0; i < len(p.activatedPlugins); i++ {
		p.activatedPlugins[i].PreClose(station)
	}
}
