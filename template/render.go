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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package template

import "github.com/kataras/iris/context"

type (
	Config struct {
		Gzip          bool
		IsDevelopment bool
	}

	// notes
	// and here we are keeping all cache templates from all engines/parsers
	// do standarTemplates *template.Template; pongoTemplates   *pongo2.TemplateSet or just Templates interface{} and after do .(...) in runtime on engine.Execute/Gzip?
	// type assertion makes a lot of checks, although in go 1.6 is faster than previous, look here: http://stackoverflow.com/questions/31577540/golang-how-to-explain-the-type-assertion-efficiency
	// so I just use properties for templates for each one template engine, only for performance, I dont like it but iris must be fast at all states...
	//
	// OR
	// have their prepare funcs inside each engine and have them as properties here? this will be faster than render's method uses. Yes, that.
	// also, it would be nice just to have all that inside engine, no need to prepare build and configs, move all them inside engines' also.
	// and the decision to do the correct Render/Execute it will be taken by the iris at pre-listen state. So here we will only have all templates ready as global var structs.
	// Ok now I decided for real, have them as properties to decide what renderer should be execute
	// end notes
	Render struct {
		config *Config
		engine Engine
	}
)

func newRender(config *Config) *Render {
	return &Render{config: &Config{}}
}

// SetEngine called once before server's listen, sets the engine and try to build them on startup, returns engine.BuildTemplate's error
func (r *Render) SetEngine(engine Engine) error {
	r.engine = engine
	return r.engine.BuildTemplates()
}

func (r *Render) Render(ctx context.IContext, name string, bindings interface{}) error {

	// build templates again on each render if IsDevelopment.
	if r.config.IsDevelopment {
		if err := r.engine.BuildTemplates(); err != nil {
			return err
		}
	}

	if r.config.Gzip {
		return r.engine.ExecuteGzip(ctx, name, bindings)
	}

	return r.engine.Execute(ctx, name, bindings)
}
