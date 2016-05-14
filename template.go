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

/* This is not idiomatic code but I did it to help users configure the Templates without need to import other packages for template configuration */
import (
	"github.com/kataras/iris/template"
	"github.com/kataras/iris/template/engine"
	"github.com/kataras/iris/template/engine/pongo"
	"github.com/kataras/iris/template/engine/standar"
)

//[ENGINE-3]
// conversions
const (
	StandarEngine EngineType = 0
	PongoEngine   EngineType = 1
)

type (
	EngineType engine.EngineType
	// TemplateConfig template.TemplateOptions
	StandarConfig standar.StandarConfig
	PongoConfig   pongo.PongoConfig

	TemplateConfig struct {
		// contains common configs for both standar & pongo
		Engine        EngineType
		Gzip          bool
		IsDevelopment bool
		Directory     string
		Extensions    []string
		ContentType   string
		Charset       string
		Asset         func(name string) ([]byte, error)
		AssetNames    func() []string
		Layout        string
		Standar       StandarConfig // contains specific configs for standar html/template
		Pongo         PongoConfig   // contains specific configs for pongo2
	}
)

func (tc *TemplateConfig) Convert() template.TemplateOptions {
	opt := template.TemplateOptions{}
	opt.Engine = engine.EngineType(tc.Engine)
	opt.Gzip = tc.Gzip
	opt.IsDevelopment = tc.IsDevelopment
	opt.Directory = tc.Directory
	opt.Extensions = tc.Extensions
	opt.ContentType = tc.ContentType
	opt.Charset = tc.Charset
	opt.Asset = tc.Asset
	opt.AssetNames = tc.AssetNames
	opt.Layout = tc.Layout
	opt.Standar = standar.StandarConfig(tc.Standar)
	opt.Pongo = pongo.PongoConfig(tc.Pongo)
	return opt
}

/* same as
&TemplateConfig{
			Engine:  engine.Standar,
			Config:  engine.Common(),
			Standar: standar.DefaultStandarConfig(),
			Pongo:   pongo.DefaultPongoConfig(),
*/
func DefaultTemplateConfig() *TemplateConfig {
	common := engine.Common()
	defaultStandar := standar.DefaultStandarConfig()
	defaultPongo := pongo.DefaultPongoConfig()

	tc := &TemplateConfig{}
	tc.Engine = StandarEngine
	tc.Gzip = common.Gzip
	tc.IsDevelopment = common.IsDevelopment
	tc.Directory = common.Directory
	tc.Extensions = common.Extensions
	tc.ContentType = common.ContentType
	tc.Charset = common.Charset
	tc.Asset = common.Asset
	tc.AssetNames = common.AssetNames
	tc.Layout = common.Layout
	tc.Standar = StandarConfig(defaultStandar)
	tc.Pongo = PongoConfig(defaultPongo)
	return tc
}

// end
