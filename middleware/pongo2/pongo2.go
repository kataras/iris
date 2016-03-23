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
package pongo2

import (
	pongo "github.com/flosch/pongo2"
	"github.com/kataras/iris"
)

type pongo2Middleware struct {
}

func (p *pongo2Middleware) Serve(ctx *iris.Context) {
	ctx.Next()

	templateName := ctx.GetString("template")
	if templateName != "" {
		templateData := ctx.Get("data")
		if templateData != nil {
			var template = pongo.Must(pongo.FromFile(templateName))
			err := template.ExecuteWriter(getPongoContext(templateData), ctx.ResponseWriter)
			if err != nil {
				ctx.SendStatus(500, err.Error())
			}
		}

	}

}

func getPongoContext(templateData interface{}) pongo.Context {
	if templateData == nil {
		return nil
	}
	contextData, isMap := templateData.(map[string]interface{})
	if isMap {
		return contextData
	}
	return nil
}

func Pongo2() *pongo2Middleware {
	return &pongo2Middleware{}
}

/* example */
/*

package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/middleware/pongo2"
)

func main() {
    iris.Use(pongo2.Pongo2())

    iris.Get("/", func(ctx *iris.Context) {
        ctx.Set("template", "index.html")
        ctx.Set("data", map[string]interface{}{"message": "Hello World!"})
    })

    iris.Listen(":8080")
}

*/
