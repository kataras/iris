// Package apps is responsible to control many Iris Applications.
// This package directly imports the iris root package and cannot be used
// inside Iris' codebase itself. Only external packages/programs can make use of it.
package apps

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

// Get returns an Iris Application based on its "appName".
// It returns nil when no application was found with the given exact name.
//
// If "appName" parameter is missing then it returns the last registered one.
// When no application is registered yet then it creates a new on-fly
// with a "Default" name and returns that instead.
// The "Default" one can be used across multiple Go packages
// of the same Program too.
//
// Applications of the same program are registered automatically.
//
// To check if at least one application is registered or not
// use the `GetAll` function instead.
func Get(appName ...string) *iris.Application {
	if len(appName) == 0 || appName[0] == "" {
		if app := context.LastApplication(); app != nil {
			return app.(*iris.Application)
		}

		return iris.New().SetName("Default")
	}

	if app, ok := context.GetApplication(appName[0]); ok {
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
