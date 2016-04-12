package main

import (
	"github.com/kataras/iris"
	_"time"
	"log"
)

func main() {
	iris.Get("/rest/hello",func (ctx *iris.Context){
		//time.Sleep(time.Duration(500) * time.Millisecond)
		ctx.Write("Hello world")
	})
	
	log.Fatal(iris.Listen("127.0.0.1:8080"))
}