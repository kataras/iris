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

package pongo

/* TODO:
1. Find if pongo2 supports layout, it should have extends or something like django but I don't know yet, if exists then do something with the layour parameter in Exeucte/Gzip.

*/
import (
	"compress/gzip"

	"github.com/flosch/pongo2"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/template/engine"
	"github.com/kataras/iris/utils"
)

var (
	buffer *utils.BufferPool
)

type (
	PongoConfig struct {
		Filters map[string]pongo2.FilterFunction
	}
	Config struct {
		*engine.Config
		// Filters for pongo2, map[name of the filter] the filter function . The filters are auto register
		*PongoConfig
	}

	Engine struct {
		Config    *Config
		Templates *pongo2.TemplateSet
	}
)

func WrapConfig(common *engine.Config, pongo *PongoConfig) *Config {
	return &Config{Config: common, PongoConfig: pongo}
}

// DefaultPongoConfig returns the default pongo specific options, no the whole Config
func DefaultPongoConfig() *PongoConfig {
	return &PongoConfig{Filters: make(map[string]pongo2.FilterFunction, 0)}
}

func New(config *Config) *Engine {
	if buffer == nil {
		buffer = utils.NewBufferPool(64)
	}
	if config == nil {
		config = WrapConfig(engine.Common(), DefaultPongoConfig())
	}
	return &Engine{Config: config}
}

func (p *Engine) GetConfig() *engine.Config {
	return p.Config.Config
}

func (p *Engine) Execute(ctx context.IContext, name string, binding interface{}, layout string) error {
	// get the template from cache, I never used pongo2 but I think reading its code helps me to understand that this is the best way to do it with the best performance.
	tmpl, err := p.Templates.FromCache(name)
	if err != nil {
		return err
	}
	// Retrieve a buffer from the pool to write to.
	out := buffer.Get()

	err = tmpl.ExecuteWriter(binding.(pongo2.Context), out)

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

func (p *Engine) ExecuteGzip(ctx context.IContext, name string, binding interface{}, layout string) error {
	tmpl, err := p.Templates.FromCache(name)
	if err != nil {
		return err
	}
	// Retrieve a buffer from the pool to write to.
	out := gzip.NewWriter(ctx.GetRequestCtx().Response.BodyWriter())
	err = tmpl.ExecuteWriter(binding.(pongo2.Context), out)

	if err != nil {
		return err
	}
	//out.Flush()
	out.Close()
	ctx.GetRequestCtx().Response.Header.Add("Content-Encoding", "gzip")

	return nil
}

func (p *Engine) BuildTemplates() error {
	if p.Config.Directory == "" {
		return nil
	}
	return nil
}
