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

package iris

// THIS FILE IS NOT YOUR CONSUME YET.

/* for v3 */

/* Notes - for me
I could use map[string]interface{} for configuration and all will be easier
but instead of this I prefer to have them staticaly typed, it's one line more for the end-user if wants to change the default configs.

OR

Make a global struct named TemplateEngine, and do the configs here, after make a TemplateRender for each one of the engines, this.
*/

import (
	"html/template"

	"github.com/flosch/pongo2"
)

type (
	TemplateEngine interface {
		// Name returns the package name for this engine
		Name() string // useful use for debugging

		//Render(ctx *Context, filename string, v interface{})
	}
	// engines

	StandarTemplateEngine struct {
		Config *standarTemplateConfig
	}

	QuickTemplateEngine struct {
		Config *quickTemplateConfig
	}

	Pongo2TemplateEngine struct {
		Config *pongo2TemplateConfig
	}

	// configs
	standarTemplateConfig struct {
		Directory string
		Funcs     template.FuncMap
	}

	quickTemplateConfig struct {
		Directory string
	}

	pongo2TemplateConfig struct {
		Directory string
		Filters   []pongo2.FilterFunction
	}
)

// TemplateEngines the struct which defines the available template engines with their default config
// these are shared in all iris instances, if you want to re-new the configs call the Standar/Quicktemplate/Pongo2.Reset()
var TemplateEngines = struct {
	Standar       StandarTemplateEngine
	Quicktemplate QuickTemplateEngine
	Pongo2        Pongo2TemplateEngine
}{defaultStandarTemplateEngine(), defaultQuickTemplateEngine(), defaultPongo2TemplateEngine()}

func defaultStandarTemplateEngine() StandarTemplateEngine {
	return StandarTemplateEngine{}
}

func defaultQuickTemplateEngine() QuickTemplateEngine {
	return QuickTemplateEngine{}
}

func defaultPongo2TemplateEngine() Pongo2TemplateEngine {
	return Pongo2TemplateEngine{}
}

func (s StandarTemplateEngine) Name() string {
	return "html/template"
}

func (s StandarTemplateEngine) Reset() {

}

func (q QuickTemplateEngine) Name() string {
	return "valyala/quicktemplate"
}

func (q QuickTemplateEngine) Reset() {

}

func (p Pongo2TemplateEngine) Name() string {
	return "flosch/pongo2"
}

func (p Pongo2TemplateEngine) Reset() {

}
