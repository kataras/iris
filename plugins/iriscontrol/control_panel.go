package iriscontrol

import "github.com/kataras/iris"

// for the plugin server
func (i *irisControlPlugin) StartControlPanel() {
	i.server = iris.Custom(iris.StationOptions{Cache: false, Profile: false, PathCorrection: true})
	i.server.Templates("./plugins/iriscontrol/templates/*")
}

func (i *irisControlPlugin) setPanelRoutes() {

	i.server.Use(&userAuth{})

	i.server.Get("/", func(ctx *iris.Context) {

	})

	i.server.Get("/login", func(ctx *iris.Context) {

	})

	i.server.Post("/login", func(ctx *iris.Context) {

	})

	i.server.Post("/logout", func(ctx *iris.Context) {

	})

}
