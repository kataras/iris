package main

import (
	"gopkg.in/yaml.v2"

	"github.com/kataras/iris"
)

func main() {
	app := newApp()

	// use Postman or whatever to do a POST request
	// (however you are always free to use app.Get and GET http method requests to read body of course)
	// to the http://localhost:8080 with RAW BODY:
	/*
		addr: localhost:8080
		serverName: Iris
	*/
	//
	// The response should be:
	// Received: main.config{Addr:"localhost:8080", ServerName:"Iris"}
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed), iris.WithOptimizations)
}

func newApp() *iris.Application {
	app := iris.New()
	app.Post("/", handler)

	return app
}

// simple yaml stuff, read more at https://github.com/go-yaml/yaml
type config struct {
	Addr       string `yaml:"addr"`
	ServerName string `yaml:"serverName"`
}

// Decode implements the `kataras/iris/context#BodyDecoder` optional interface
// that any go type can implement in order to be self-decoded when reading the request's body.
func (c *config) Decode(body []byte) error {
	return yaml.Unmarshal(body, c)
}

func handler(ctx iris.Context) {
	var c config

	//
	// Note:
	// second parameter is nil because our &c implements the `context#BodyDecoder`
	// which has a priority over the context#Unmarshaler (which can be a more global option for reading request's body)
	// see the `http_request/read-custom-via-unmarshaler/main.go` example to learn how to use the context#Unmarshaler too.
	//
	// Note 2:
	// If you need to read the body again for any reason
	// you should disable the body consumption via `app.Run(..., iris.WithoutBodyConsumptionOnUnmarshal)`.
	//

	if err := ctx.UnmarshalBody(&c, nil); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.WriteString(err.Error())
		return
	}

	ctx.Writef("Received: %#+v", c)
}
