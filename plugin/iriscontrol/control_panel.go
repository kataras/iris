package iriscontrol

import (
	"os"
	"strconv"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
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
	i.server.Config.DisableBanner = true
	i.server.Config.Render.Template.Directory = installationPath + "templates"
	//i.server.SetRenderConfig(i.server.Config.Render)
	i.setPluginsInfo()
	i.setPanelRoutes()

	go i.server.Listen(":" + strconv.Itoa(i.options.Port))

	i.pluginContainer.Printf("[%s] %s is running at port %d", time.Now().UTC().String(), Name, i.options.Port)

}

// DashboardPage is the main data struct for the index
// contains a boolean if server is running, the routes and the plugins
type DashboardPage struct {
	ServerIsRunning      bool
	Routes               []iris.Route
	Plugins              []PluginInfo
	LastOperationDateStr string
}

func (i *irisControlPlugin) setPluginsInfo() {
	plugins := i.pluginContainer.GetAll()
	i.plugins = make([]PluginInfo, 0, len(plugins))
	for _, plugin := range plugins {
		name := i.pluginContainer.GetName(plugin)
		desc := i.pluginContainer.GetDescription(plugin)
		if name == "" {
			// means an iris internaly plugin or a nameless plugin
			name = "Internal Iris Plugin"
		}
		if desc == "" {
			// means an iris internaly plugin or a descriptionless plugin
			desc = "Propably an internal Iris Plugin - no description provided"
		}

		i.plugins = append(i.plugins, PluginInfo{Name: name, Description: desc})
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

	i.server.Static("/public", installationPath+"static", 1)

	i.server.Use(i.authFunc)
	i.server.Get("/", func(ctx *iris.Context) {
		ctx.Render("index.html", DashboardPage{
			ServerIsRunning:      i.station.HTTPServer.IsListening(),
			Routes:               i.routes,
			Plugins:              i.plugins,
			LastOperationDateStr: i.lastOperationDate.Format(config.TimeFormat),
		})
	})

	//the controls
	i.server.Post("/start_server", func(ctx *iris.Context) {
		//println("server start")
		i.lastOperationDate = time.Now()
		old := i.stationServer
		if !old.IsSecure() {
			i.station.Listen(old.Config.ListeningAddr)
			//yes but here it does re- post listen to this plugin so ...
		} else {
			i.station.ListenTLS(old.Config.ListeningAddr, old.Config.CertFile, old.Config.KeyFile)
		}

	})

	i.server.Post("/stop_server", func(ctx *iris.Context) {
		//println("server stop")
		i.station.Close()
	})

}
