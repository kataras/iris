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
	"github.com/kataras/iris"
)

func main() {
	api := iris.New()
	//api := iris.Custom(iris.StationOptions{Cache: false})
	//ok it's working both cached and normal router, optimization done before listen

	//the subdomains are working like parties, the only difference is that you CANNOT HAVE party of party with both of them subdomains
	//YOU CANNOT DO THAT AND YOU MUSTN'T DO IT DIRECTLY: api.Party("admin.yourhost.com").Party("other.admin.yuorhost.com")
	//Do that: api.Party("other.admin.yourhost.com") .... and different/new party with api.Party("admin.yourhost.com")
	admin := api.Party("admin.yourhost.com")
	{
		//this will only success on admin.yourhost.com/hey
		admin.Get("/hey", func(c *iris.Context) {
			c.Write("HEY FROM admin.omicronware.com")
		})
		//this will only success on admin.yourhost.com/hey2
		admin.Get("/hey2", func(c *iris.Context) {
			c.Write("HEY SECOND FROM admin.omicronware.com")
		})
	}

	// this will serve on yourhost.com/hey and not on admin.yourhost.com/hey
	api.Get("/hey", func(c *iris.Context) {
		c.Write("HEY FROM no-subdomain hey")
	})

	api.Listen(":80")
}
