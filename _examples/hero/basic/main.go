package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/hero"
)

func main() {

	app := iris.New()

	// 1
	helloHandler := hero.Handler(hello)
	app.Get("/{to:string}", helloHandler)

	// 2
	hero.Register(&myTestService{
		prefix: "Service: Hello",
	})

	helloServiceHandler := hero.Handler(helloService)
	app.Get("/service/{to:string}", helloServiceHandler)

	// http://localhost:8080/your_name
	// http://localhost:8080/service/your_name
	app.Run(iris.Addr(":8080"))
}

func hello(to string) string {
	return "Hello " + to
}

type Service interface {
	SayHello(to string) string
}

type myTestService struct {
	prefix string
}

func (s *myTestService) SayHello(to string) string {
	return s.prefix + " " + to
}

func helloService(to string, service Service) string {
	return service.SayHello(to)
}
