package iriscontrol

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/plugin/routesinfo"
	"github.com/kataras/iris/server"
)

// Name the name(string) of this plugin which is Iris Control
const Name = "Iris Control"

type irisControlPlugin struct {
	options config.IrisControl
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
// receives config.IrisControl
func New(cfg ...config.IrisControl) iris.IPlugin {
	c := config.DefaultIrisControl()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	auth := newUserAuth(c.Users)
	if auth == nil {
		panic(Name + " Error: you should pass authenticated users map to the options, refer to the docs!")
	}

	return &irisControlPlugin{options: c, auth: auth, routes: routesinfo.RoutesInfo()}
}

// Web set the options for the plugin and return the plugin which is ready-to-use inside iris.Plugin method
// first parameter is port
// second parameter is map of users (username:password)
func Web(port int, users map[string]string) iris.IPlugin {
	return New(config.IrisControl{port, users})
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
		i.stationServer = i.station.Server()
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

	i.options = config.IrisControl{}
	i.routes = nil
	i.station = nil
	i.server.Close()
	i.pluginContainer = nil
	i.auth.Destroy()
	i.auth = nil
	i.pluginContainer.Printf("[%s] %s is turned off", time.Now().UTC().String(), Name)
}
