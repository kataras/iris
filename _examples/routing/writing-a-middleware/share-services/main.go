package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
)

func main() {
	app := newApp()

	// GET: http://localhost:8080
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()
	app.Use(middleware)
	// OR app.UseRouter(middleware)
	// to register it everywhere,
	// including the HTTP errors.

	app.Get("/", handler)

	return app
}

func middleware(ctx iris.Context) {
	service := &helloService{
		Greeting: "Hello",
	}
	setService(ctx, service)

	ctx.Next()
}

func handler(ctx iris.Context) {
	service, ok := getService(ctx)
	if !ok {
		ctx.StopWithStatus(iris.StatusInternalServerError)
		return
	}

	response := service.Greet("Gophers")
	ctx.WriteString(response)
}

/*
| ---------------------- |
| service implementation |
| ---------------------- |
*/

const serviceContextKey = "app.service"

func setService(ctx iris.Context, service GreetService) {
	ctx.Values().Set(serviceContextKey, service)
}

func getService(ctx iris.Context) (GreetService, bool) {
	v := ctx.Values().Get(serviceContextKey)
	if v == nil {
		return nil, false
	}

	service, ok := v.(GreetService)
	if !ok {
		return nil, false
	}

	return service, true
}

// A GreetService example.
type GreetService interface {
	Greet(name string) string
}

type helloService struct {
	Greeting string
}

func (m *helloService) Greet(name string) string {
	return fmt.Sprintf("%s, %s!", m.Greeting, name)
}
