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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package pongo2

import (
	"os"

	pongo "github.com/flosch/pongo2"
	"github.com/kataras/iris"
)

// Pongo2Middleware the middleware for pongo2
// Conflicts with the render if pongo2 directory is './templates' we have issue see -> Serve ->p.once
// so please just change the iris.Config().Render.Directory = "tosomethingdifferent"

type Pongo2Middleware struct {
	templateDir string
}

func (p *Pongo2Middleware) Serve(ctx *iris.Context) {
	/*p.once.Do(func() {
		// https://github.com/kataras/iris/issues/94
		if ctx.GetStation().Config.Render.Directory == p.templateDir {
			panic("[IRIS Pongo2 middleware] You cannot use the same template directory ('" + p.templateDir + "') for pongo2 and html/template")
		}
	})
	// Changed my mind, let users just change the iris.Config().Render.Directory = "tosomethingdifferent"
	*/

	ctx.Next()

	templateName := ctx.GetString("template")
	if templateName != "" {
		templateData := ctx.Get("data")
		if templateData != nil {

			var template = pongo.Must(pongo.FromFile(p.templateDir + templateName))

			contents, err := template.Execute(getPongoContext(templateData))
			if err != nil {
				ctx.Text(500, err.Error())
				return
			}
			// set the content to html
			ctx.SetContentType([]string{iris.ContentHTML + " ;charset=" + iris.Charset})
			ctx.SetBodyString(contents)

		}

	}

}

// getPongoContext returns the pongo.Context from data, used internaly by the middleware
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

// Pongo2 creates and returns the middleware, same as New()
func Pongo2(templateDirectory ...string) *Pongo2Middleware {
	var templateDir = "." + string(os.PathSeparator)
	if templateDirectory != nil && len(templateDirectory) > 0 {
		templateDir = templateDirectory[0]
	}
	if templateDir[len(templateDir)-1] != os.PathSeparator {
		templateDir += string(os.PathSeparator)
	}
	return &Pongo2Middleware{templateDir: templateDir}
}

// New creates and returns the middleware, same as Pongo2()
func New(templateDirectory ...string) *Pongo2Middleware {
	return Pongo2(templateDirectory...)
}

/* example */
/*

package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/middleware/pongo2"
)

func main() {
    iris.Use(pongo2.Pongo2("./mypongo2templates")) // or .Pongo2() defaults to "./"

    iris.Get("/", func(ctx *iris.Context) {
        ctx.Set("template", "index.html")
        ctx.Set("data", map[string]interface{}{"message": "Hello World!"})
    })

    iris.Listen(":8080")
}

*/
