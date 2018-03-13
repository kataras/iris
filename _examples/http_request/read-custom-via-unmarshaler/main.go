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

/*
type myBodyDecoder struct{}

var DefaultBodyDecoder = myBodyDecoder{}

// Implements the `kataras/iris/context#Unmarshaler` but at our example
// we will use the simplest `context#UnmarshalerFunc` to pass just the yaml.Unmarshal.
//
// Can be used as: ctx.UnmarshalBody(&c, DefaultBodyDecoder)
func (r *myBodyDecoder) Unmarshal(data []byte, outPtr interface{}) error {
	return yaml.Unmarshal(data, outPtr)
}
*/

func handler(ctx iris.Context) {
	var c config

	//
	// Note:
	// yaml.Unmarshal already implements the `context#Unmarshaler`
	// so we can use it directly, like the json.Unmarshal(ctx.ReadJSON), xml.Unmarshal(ctx.ReadXML)
	// and every library which follows the best practises and is aligned with the Go standards.
	//
	// Note 2:
	// If you need to read the body again for any reason
	// you should disable the body consumption via `app.Run(..., iris.WithoutBodyConsumptionOnUnmarshal)`.
	//

	if err := ctx.UnmarshalBody(&c, iris.UnmarshalerFunc(yaml.Unmarshal)); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.WriteString(err.Error())
		return
	}

	ctx.Writef("Received: %#+v", c)
}
