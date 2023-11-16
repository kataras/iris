# The `apps` package

Package `github.com/kataras/iris/v12/apps` provides a globally scoped control over all registered Iris Applications of the same Program.

[Example Application](https://github.com/kataras/iris/tree/main/_examples/routing/subdomains/redirect/multi-instances)

Below you will find a use case for each feature.

## The `Get` function

The `Get` function returns an Iris Application based on its "appName".
It returns nil when no application was found with the given exact name.

If "appName" parameter is missing then it returns the last registered one.
When no application is registered yet then it creates a new on-fly
with a "Default" name and returns that instead.
The "Default" one can be used across multiple Go packages
of the same Program too.

Applications of the same program are registered automatically.

To check if at least one application is registered or not,
use the `GetAll() []*iris.Application` function instead.

```go
func Get(appName ...string) *iris.Application
```

### Features

- Access an Iris Application globally from all of your Go packages of the same Program without a global parameter and import cycle.

```go
/* myserver/api/user.go */

package userapi

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/apps"
)

func init() {
	app := apps.Get()
	app.Get("/", list)
}

func list(ctx iris.Context) {
	// [...]
	ctx.WriteString("list users")
}
```

```go
/* myserver/main.go */

package main

import (
	_ "myserver/api/user"

	"github.com/kataras/iris/v12/apps"
)

func main() {
	app := apps.Get()
	app.Listen(":80")
}
```

The empty `_` import statement will call the `userapi.init` function (before `main`) which
initializes an Iris Application, if not already exists, and registers it to the internal global store for further use across packages.

- Reference one or more Iris Applications based on their names.

The `Application.SetName` method sets a unique name to this Iris Application.
It sets a child prefix for the current Application's Logger.
Its `Application.String` method returns the given name. It returns itself.

```go
func (app *Application) SetName(appName string) *Application
```

```go
/* myserver/main.go */

package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.New().SetName("app.company.com")
}
```

```go
/* myserver/pkg/something.go */

package pkg

import "github.com/kataras/iris/v12/apps"

func DoSomething() {
    app := apps.Get("app.company.com")
}
```

The `main` function creates and registers an Iris Application on `"app.company.com"` name, after that declaration every package of the same Program can retrieve that specific Application instance through its name.

## The `Switch` function

The `Switch` function returns a new Application with the sole purpose of routing the matched Applications through the "provided cases". Read below about the available SwitchProviders and how you can create and use your own one.

The cases are filtered in order of their registration.

```go
func Switch(provider SwitchProvider, options ...SwitchOption) *iris.Application
```

Example Code:

```go
switcher := Switch(Hosts{
	{Pattern: "mydomain.com", Target: app},
	{Pattern: "test.mydomain.com", Target: testSubdomainApp},
	{Pattern: "otherdomain.com", Target: "appName"},
})
switcher.Listen(":80")
```

Note that this is NOT an alternative for a load balancer.
The filters are executed by registration order and a matched Application
handles the request, that's all it does.

The returned Switch Iris Application can register routes that will run
when neither of the registered Applications is responsible to handle the incoming request against the provided filters.

The returned Switch Iris Application can also register custom error code handlers,
e.g. to inject the 404 on not responsible Application was found.
It can also be wrapped with its `WrapRouter` method,
which is really useful for logging and statistics.

### The `SwitchProvider` interface

This is the first required input argument for the `Switch` function.

A `SwitchProvider` should return one or more `SwitchCase` values.

```go
type SwitchProvider interface {
	GetSwitchCases() []SwitchCase
}
```

The `SwitchCase` structure contains the Filter and the target Iris Application.

```go
type SwitchCase struct {
	Filter func(ctx iris.Context) bool
	App    *iris.Application
}
```

### The `SwitchOptions` structure

This is the last variadic (optional) input argument for the `Switch` function.

```go
type SwitchOptions struct {
	// RequestModifiers holds functions to run
	// if and only if at least one Filter passed.
	// They are used to modify the request object
	// of the matched Application, e.g. modify the host.
	//
	// See `SetHost` option too.
	RequestModifiers []func(*http.Request)
}
```

The `SetHost` function is a SwitchOption.
It force sets a Host field for the matched Application's request object.
Extremely useful when used with Hosts SwitchProvider.
Usecase: www. to root domain without redirection (SEO reasons)
and keep the same internal request Host for both of them so
the root app's handlers will always work with a single host no matter
what the real request Host was.

```go
func SetHost(hostField string) SwitchOptionFunc
```

Example Code:

```go
cases := Hosts{
	{"^(www.)?mydomain.com$", rootApp},
}
switcher := Switch(cases, SetHost("mydomain.com"))
```

### Join different type of SwitchProviders

Wrap with the `Join` slice to pass more than one provider at the same time.

Example Code:

```go
Switch(Join{
	SwitchCase{
		Filter: customFilter,
		App:    myapp,
	},
	Hosts{
		{Pattern: "^test.*$", Target: myapp},
	},
})
```
