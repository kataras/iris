package iriscontrol

import (
	"strconv"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/middleware/basicauth"
)

type (
	// IrisControl is the interface which the iriscontrol should implements
	// it's empty for now because no need any public API
	IrisControl interface{}
	iriscontrol struct {
		port  int
		users map[string]string

		// child is the plugin's standalone station
		child *iris.Framework
		// the station which this plugins is registed to
		parent       *iris.Framework
		parentLastOp time.Time
	}

	pluginInfo struct {
		Name        string
		Description string
	}
)

var _ IrisControl = &iriscontrol{}

func (i *iriscontrol) listen(f *iris.Framework) {
	i.parent = f
	i.parentLastOp = time.Now()
	i.initializeChild()
}

func (i *iriscontrol) initializeChild() {
	i.child = iris.New()
	i.child.Config.DisableBanner = true
	i.child.Config.Render.Template.Directory = assetsPath + "templates"

	// set the assets
	i.child.Static("/public", assetsPath+"static", 1)

	// set the authentication middleware
	i.child.Use(basicauth.New(config.BasicAuth{
		Users:      i.users,
		ContextKey: "user",
		Realm:      config.DefaultBasicAuthRealm,
		Expires:    time.Duration(1) * time.Hour,
	}))

	i.child.Get("/", func(ctx *iris.Context) {
		ctx.MustRender("index.html", iris.Map{
			"ServerIsRunning":      i.parentIsRunning(),
			"Routes":               i.parentLookups(),
			"Plugins":              i.infoPlugins(),
			"LastOperationDateStr": i.infoLastOp(),
		})
	})

	i.child.Post("/start_server", func(ctx *iris.Context) {

		if !i.parentIsRunning() {
			// starts the server with its old configuration
			go func() {
				if err := i.parent.HTTPServer.Open(); err != nil {
					i.parent.Logger.Warningf(err.Error())
				}
			}()
			i.parentLastOp = time.Now()
		}
	})

	i.child.Post("/stop_server", func(ctx *iris.Context) {

		if i.parentIsRunning() {
			i.parentLastOp = time.Now()

			go func() {
				if err := i.parent.CloseWithErr(); err != nil {
					i.parent.Logger.Warningf(err.Error())
				}
			}()
		}
	})

	go i.child.Listen(i.parent.HTTPServer.VirtualHostname() + ":" + strconv.Itoa(i.port))
}

func (i *iriscontrol) parentIsRunning() bool {
	return i.parent != nil && i.parent.HTTPServer.IsListening()
}

func (i *iriscontrol) parentLookups() []iris.Route {
	if i.parent == nil {
		return nil
	}
	return i.parent.Lookups()
}

func (i *iriscontrol) infoPlugins() (info []pluginInfo) {
	plugins := i.parent.Plugins
	for _, p := range plugins.GetAll() {
		name := plugins.GetName(p)
		description := plugins.GetDescription(p)
		if name == "" {
			name = "Unknown plugin name"
		}
		if description == "" {
			description = "description is not available"
		}

		info = append(info, pluginInfo{Name: name, Description: description})
	}
	return
}

func (i *iriscontrol) infoLastOp() string {
	return i.parentLastOp.Format(config.TimeFormat)
}
