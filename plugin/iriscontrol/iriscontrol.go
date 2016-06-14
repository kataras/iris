package iriscontrol

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/middleware/basicauth"
)

// Name the name(string) of this plugin which is Iris Control
const Name = "Iris Control"

type irisControlPlugin struct {
	options config.IrisControl
	// the pluginContainer is the container which keeps this plugin from the main user's iris instance
	pluginContainer iris.PluginContainer
	// the station object of the main  user's iris instance
	station *iris.Framework
	//a copy of the server which the main user's iris is listening for
	stationServer *iris.Server

	// the server is this plugin's server object, it is managed by this plugin only
	server *iris.Framework
	//
	//infos
	routes  []iris.Route
	plugins []PluginInfo
	// last time the server was on
	lastOperationDate time.Time
	//

	authFunc iris.HandlerFunc
}

// New returns the plugin which is ready-to-use inside iris.Plugin method
// receives config.IrisControl
func New(cfg ...config.IrisControl) iris.Plugin {
	c := config.DefaultIrisControl()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.Users == nil || len(c.Users) == 0 {
		panic(Name + " Error: you should pass authenticated users map to the options, refer to the docs!")
	}

	auth := basicauth.Default(c.Users)

	return &irisControlPlugin{options: c, authFunc: auth, routes: make([]iris.Route, 0)}
}

// Web set the options for the plugin and return the plugin which is ready-to-use inside iris.Plugin method
// first parameter is port
// second parameter is map of users (username:password)
func Web(port int, users map[string]string) iris.Plugin {
	return New(config.IrisControl{Port: port, Users: users})
}

// implement the base IPlugin

func (i *irisControlPlugin) Activate(container iris.PluginContainer) error {
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

// PostListen sets the station object after the main server starts
// starts the actual work of the plugin
func (i *irisControlPlugin) PostListen(s *iris.Framework) {
	//if the first time, because other times start/stop of the server so listen and no listen will be only from the control panel
	if i.station == nil {
		i.station = s
		i.stationServer = i.station.HTTPServer
		i.lastOperationDate = time.Now()
		i.routes = s.Lookups()
		i.startControlPanel()
	}

}

func (i *irisControlPlugin) PreClose(s *iris.Framework) {
	// Do nothing. This is a wrapper of the main server if we destroy when users stop the main server then we cannot continue the control panel i.Destroy()
}

//

// Destroy removes entirely the plugin, the options and all of these properties, you cannot re-use this plugin after this method.
func (i *irisControlPlugin) Destroy() {
	i.pluginContainer.Remove(Name)

	i.options = config.IrisControl{}
	i.routes = nil
	i.station = nil
	i.lastOperationDate = config.CookieExpireNever
	i.server.Close()
	i.pluginContainer = nil
	i.authFunc = nil
	i.pluginContainer.Printf("[%s] %s is turned off", time.Now().UTC().String(), Name)
}
