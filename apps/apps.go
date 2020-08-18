// Package apps is responsible to control many Iris Applications.
// This package directly imports the iris root package and cannot be used
// inside Iris' codebase itself. Only external packages/programs can make use of it.
package apps

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

// Get returns an existing Iris Application based on its "appName".
// Applications of the same program
// are registered automatically.
func Get(appName string) *iris.Application {
	if app, ok := context.GetApplication(appName); ok {
		return app.(*iris.Application)
	}

	return nil
}

// GetAll returns a slice of all registered Iris Applications.
func GetAll() []*iris.Application {
	appsReadOnly := context.GetApplications()
	apps := make([]*iris.Application, 0, len(appsReadOnly))

	for _, app := range appsReadOnly {
		apps = append(apps, app.(*iris.Application))
	}

	return apps
}

// Configure applies one or more configurator to the
// applications with the given "appNames".
//
// See `ConfigureAll` too.
func Configure(appNames []string, configurators ...iris.Configurator) {
	for _, appName := range appNames {
		if app := Get(appName); app != nil {
			app.Configure(configurators...)
		}
	}
}

// ConfigureAll applies one or more configurator to all
// registered applications so far.
//
// See `Configure` too.
func ConfigureAll(configurators ...iris.Configurator) {
	for _, app := range context.GetApplications() {
		app.(*iris.Application).Configure(configurators...)
	}
}
