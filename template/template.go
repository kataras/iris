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

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/template/engine"
)

type (
	Template struct {
		Engine engine.Engine
	}
)

func New(e engine.Engine) *Template {

	if err := e.BuildTemplates(); err != nil { // first build the templates, if error panic because this is called before server's run
		panic(err)
	}

	return &Template{e}
}

func (t *Template) Render(ctx context.IContext, name string, bindings interface{}, layout ...string) error {
	config := t.Engine.GetConfig()
	// build templates again on each render if IsDevelopment.
	if config.IsDevelopment {
		if err := t.Engine.BuildTemplates(); err != nil {
			return err
		}
	}
	ctx.GetRequestCtx().Response.Header.Set("Content-Type", config.ContentType+"; charset="+config.Charset)
	// I don't like this, something feels wrong
	_layout := ""
	if len(layout) > 0 {
		_layout = layout[0]
	}
	if _layout == "" {
		_layout = config.Layout
	}

	//
	if config.Gzip {
		return t.Engine.ExecuteGzip(ctx, name, bindings, _layout)
	}

	return t.Engine.Execute(ctx, name, bindings, _layout)
}
