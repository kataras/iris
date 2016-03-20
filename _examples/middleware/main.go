// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package main

import (
	"fmt"
	"github.com/kataras/iris"
)

func main() {

	// register global middleware, you can pass more than one handler comma separated
	iris.UseFunc(func(c *iris.Context) {
		fmt.Println("(1)Global logger: ", c.Request.URL.Path)
		c.Next()
	})

	// register a global structed iris.Handler as middleware
	myglobal := MyGlobalMiddlewareStructed{loggerId: "my logger id"}
	iris.Use(myglobal)

	// register route's middleware
	iris.Get("/home", func(c *iris.Context) {
		fmt.Println("(1)HOME logger for /home")
		c.Next()
	}, func(c *iris.Context) {
		fmt.Println("(2)HOME logger for /home")
		c.Next()
	}, func(c *iris.Context) {
		c.Write("Hello from /home")
	})

	// register a structed iris.Handler as middleware to the route
	iris.Get("/hello", iris.ToHandlerFunc(myglobal))

	println("Iris is listening on :8080")
	iris.Listen("8080")
}

// a silly example
type MyGlobalMiddlewareStructed struct {
	loggerId string
}

var _ iris.Handler = &MyGlobalMiddlewareStructed{}

//Important staff, iris middleware must implement the iris.Handler interface which is:
func (m MyGlobalMiddlewareStructed) Serve(c *iris.Context) {
	fmt.Println("Hello from logger with id: ", m.loggerId)
	c.Next()
}
