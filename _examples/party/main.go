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

import "github.com/kataras/iris"

func main() {
	// Let's party
	admin := iris.Party("/admin")
	{
		// add a silly middleware
		admin.UseFunc(func(c *iris.Context) {
			//your authentication logic here...
			println("from ", c.Request.URL.Path)
			authorized := true
			if authorized {
				c.Next()
			} else {
				c.SendStatus(401, c.Request.URL.Path+" is not authorized for you")
			}

		})
		admin.Get("/", func(c *iris.Context) {
			c.Write("from /admin/ or /admin if you pathcorrection on")
		})
		admin.Get("/dashboard", func(c *iris.Context) {
			c.Write("/admin/dashboard")
		})
		admin.Delete("/delete/:userId", func(c *iris.Context) {
			c.Write("admin/delete/%s", c.Param("userId"))
		})
	}

	beta := admin.Party("/beta")
	beta.Get("/hey", func(c *iris.Context) { c.Write("hey from /admin/beta/hey") })

	//for subdomains look at the: _examples/subdomains_simple/main.go

	println("Iris is listening on :8080")
	iris.Listen("8080")

}
