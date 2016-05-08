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

package iris

import (
	"html/template"

	"github.com/kataras/iris/render"
)

// these are just for conversional with the render package, in order to be used inside the Iris(ex Station) type

// Delims represents a set of Left and Right delimiters for HTML template rendering.
type Delims struct {
	// Left delimiter, defaults to {{.
	Left string
	// Right delimiter, defaults to }}.
	Right string
}

// HTMLOptions is a struct for overriding some rendering Options for specific context.HTML call.
type HTMLOptions struct {
	// Layout template name. Overrides Options.Layout.
	Layout string
}

// RenderConfig is a struct for specifying configuration options for the render.Render object.
type RenderConfig struct {
	// Directory to load templates. Default is "templates".
	Directory string
	// Asset function to use in place of directory. Defaults to nil.
	Asset func(name string) ([]byte, error)
	// AssetNames function to use in place of directory. Defaults to nil.
	AssetNames func() []string
	// Layout template name. Will not render a layout if blank (""). Defaults to blank ("").
	Layout string
	// Extensions to parse template files from. Defaults to [".tmpl"].
	Extensions []string
	// Funcs is a slice of FuncMaps to apply to the template upon compilation. This is useful for helper functions. Defaults to [].
	Funcs []template.FuncMap
	// Delims sets the action delimiters to the specified strings in the Delims struct.
	Delims Delims
	// Appends the given character set to the Content-Type header. Default is "UTF-8".
	Charset string
	// Gzip enable it if you want to render using gzip compression. Default is false
	Gzip bool
	// Outputs human readable JSON.
	IndentJSON bool
	// Outputs human readable XML. Default is false.
	IndentXML bool
	// Prefixes the JSON output with the given bytes. Default is false.
	PrefixJSON []byte
	// Prefixes the XML output with the given bytes.
	PrefixXML []byte
	// Allows changing of output to XHTML instead of HTML. Default is "text/html".
	HTMLContentType string
	// If IsDevelopment is set to true, this will recompile the templates on every request. Default is false.
	IsDevelopment bool
	// Unescape HTML characters "&<>" to their original values. Default is false.
	UnEscapeHTML bool
	// Streams JSON responses instead of marshalling prior to sending. Default is false.
	StreamingJSON bool
	// Require that all partials executed in the layout are implemented in all templates using the layout. Default is false.
	RequirePartials bool
	// Deprecated: Use the above `RequirePartials` instead of this. As of Go 1.6, blocks are built in. Default is false.
	RequireBlocks bool
	// Disables automatic rendering of http.StatusInternalServerError when an error occurs. Default is false.
	DisableHTTPErrorRendering bool
}

// newRender returns a new render.Render from iris.RenderConfig, used inside New(...)
func newRender(config *RenderConfig) *render.Render {
	if config == nil {
		config = DefaultConfig().Render //to prevent panics on nil when Render.
	}
	options := render.Options{}
	options.Directory = config.Directory
	options.Asset = config.Asset
	options.AssetNames = config.AssetNames
	options.Layout = config.Layout
	options.Extensions = config.Extensions
	options.Funcs = config.Funcs
	options.Delims = render.Delims{config.Delims.Left, config.Delims.Right}
	options.Charset = config.Charset
	options.Gzip = config.Gzip
	options.IndentJSON = config.IndentJSON
	options.IndentXML = config.IndentXML
	options.PrefixJSON = config.PrefixJSON
	options.PrefixXML = config.PrefixXML
	options.HTMLContentType = config.HTMLContentType
	options.IsDevelopment = config.IsDevelopment
	options.UnEscapeHTML = config.UnEscapeHTML
	options.StreamingJSON = config.StreamingJSON
	options.RequirePartials = config.RequirePartials
	options.RequireBlocks = config.RequireBlocks
	options.DisableHTTPErrorRendering = config.DisableHTTPErrorRendering

	if options.Charset == "" {
		options.Charset = DefaultCharset
	}

	return render.New(options)
}

// parseHTMLOptions takes an iris.HTMLOptions and convert to render.HTMLOptions
func parseHTMLOptions(options ...HTMLOptions) (opt []render.HTMLOptions) {
	if options != nil && len(options) > 0 {
		opt = []render.HTMLOptions{render.HTMLOptions{options[0].Layout}}
	}
	return
}
