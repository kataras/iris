package main

import (
	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/_examples/routing/party-controller/pkg/weatherapi"
)

// Example of usage of Party Controllers.
// The method of zero-performance cost at serve-time, APIs run as fast as common Iris handlers.
func main() {
	app := iris.New()

	// Define a group under /api request path.
	api := app.Party("/api")
	// Register one or more dependencies.
	api.RegisterDependency(weatherapi.NewClient(weatherapi.Options{
		APIKey: "{YOUR_API_KEY}",
	}))

	// Register a party controller under the "/weather" sub request path.
	api.PartyConfigure("/weather", new(WeatherController))

	// Start the local server at 8080 port.
	app.Listen(":8080")
}

// Just like the MVC controllers, route group(aka Party) controller's
// fields are injected by the parent or current party's RegisterDependency method.
//
// This controller structure could be live to another sub-package of our application as well.
type WeatherController struct {
	Client *weatherapi.Client // This is automatically injected by .RegisterDependency.
}

func (api *WeatherController) Configure(r iris.Party) {
	// Register routes under /api/weather.
	r.Get("/current", api.getCurrentData)
}

// Normal Iris Handler.
func (api *WeatherController) getCurrentData(ctx iris.Context) {
	city := ctx.URLParamDefault("city", "Athens")

	// Call the controller's "Client"'s GetCurrentByCity method
	// to retrieve data from external provider and push them to our clients.
	data, err := api.Client.GetCurrentByCity(ctx, city)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(data)
}
