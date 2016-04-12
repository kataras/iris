package main

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/fasthttp"
	_"time"
)

func main() {
	echo2 := echo.New()
	
	echo2.Get("/rest/hello",func (ctx echo.Context) error{
		//time.Sleep(time.Duration(500) * time.Millisecond)
		return ctx.String(200, "Hello world")
	})
	
	echo2.Run(fasthttp.New(":8080"))

}