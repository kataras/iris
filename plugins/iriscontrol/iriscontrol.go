package iriscontrol

import (
	"time"

	"github.com/kataras/iris"
)

const Name = "Iris Control Panel"

type IrisControlOptions struct {
	Port  uint8
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
	routes []iris.IRoute
}

// Web set the options for the plugin and return the plugin which is ready-to-use inside iris.Plugin method
func (i *irisControlPlugin) Web(options IrisControlOptions) iris.IPlugin {
	i.options = options
	return i
}

// implement the base IPlugin

func (i *irisControlPlugin) Activate(container iris.IPluginContainer) error {
	i.pluginContainer = container
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

// PostHandle collect the registed routes information,append the whole route instance
func (i *irisControlPlugin) PostHandle(route iris.IRoute) {
	if i.routes == nil {
		i.routes = make([]iris.IRoute, 0)
	}
	i.routes = append(i.routes, route)
}

// PreListen sets the station object before the main server starts
// and starts the actual work of the plugin
func (i *irisControlPlugin) PreListen(s *iris.Station, addr string) {
	i.station = s
	i.StartControlPanel()
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

	i.pluginContainer.Printf("[%s] %s is turned off", time.Now().UTC().String(), Name)
}
