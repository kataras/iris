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

package standar

import (
	"compress/gzip"
	"html/template"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/utils"
)

var (
	buffer *utils.BufferPool
)

type (
	Config struct {
		Directory string
		// Funcs for Standar
		Funcs template.FuncMap
	}

	Engine struct {
		Config    *Config
		Templates *template.Template
	}
)

func New() *Engine {
	if buffer == nil {
		buffer = utils.NewBufferPool(64)
	}
	return &Engine{Config: &Config{Directory: "templates", Funcs: template.FuncMap{}}}
}

func (s *Engine) BuildTemplates() error {
	if s.Config.Directory == "" {
		return nil
	}
	return nil
}

func (s *Engine) Execute(ctx context.IContext, name string, binding interface{}) error {
	// Retrieve a buffer from the pool to write to.
	out := buffer.Get()
	err := s.Templates.ExecuteTemplate(out, name, binding)
	if err != nil {
		buffer.Put(out)
		return err
	}
	w := ctx.GetRequestCtx().Response.BodyWriter()
	out.WriteTo(w)

	// Return the buffer to the pool.
	buffer.Put(out)
	return nil
}

func (s *Engine) ExecuteGzip(ctx context.IContext, name string, binding interface{}) error {

	// Retrieve a buffer from the pool to write to.
	out := gzip.NewWriter(ctx.GetRequestCtx().Response.BodyWriter())
	err := s.Templates.ExecuteTemplate(out, name, binding)
	if err != nil {
		return err
	}
	//out.Flush()
	out.Close()
	ctx.GetRequestCtx().Response.Header.Add("Content-Encoding", "gzip")
	return nil
}
