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

package template

import (
	"html/template"
	"os"
	"strings"
	"sync"

	"github.com/kataras/iris/logger"
	"github.com/kataras/iris/utils"
)

type (
	// HTMLContainer are used to cache the templates and watch for file changes on these
	HTMLContainer struct {
		// Templates contains all the html templates it's type of *template.Template from standar API
		Templates   *template.Template
		rootPath    string
		loaded      bool
		ext         string
		directory   string
		pattern     string
		delimsLeft  string
		delimsRight string
		funcMap     template.FuncMap
		logger      *logger.Logger
		mu          *sync.Mutex
	}
)

// NewHTMLContainer creates and returns a new NewHTMLContainer object, using a logger
func NewHTMLContainer(logger *logger.Logger) *HTMLContainer {
	return &HTMLContainer{logger: logger, mu: &sync.Mutex{}, delimsLeft: "{{", delimsRight: "}}"}
}

// Delims set custom Delims before the Load
func (html *HTMLContainer) Delims(left string, right string) *HTMLContainer {
	html.delimsLeft = left
	html.delimsRight = right
	return html
}

// Funcs adds the elements of the argument map to the template's function map.
// It panics if a value in the map is not a function with appropriate return
// type. However, it is legal to overwrite elements of the map. The return
// value is the template, so calls can be chained.
func (html *HTMLContainer) Funcs(funcMap template.FuncMap) *HTMLContainer {
	html.funcMap = funcMap
	return html
}

// inline method for parseglob, just tries to parse the templates
func (html *HTMLContainer) parseGlob(globPathExp, namespace string) (tmp *template.Template, err error) {

	tmp, err = template.New(namespace).Delims(html.delimsLeft, html.delimsRight).Funcs(html.funcMap).ParseGlob(globPathExp)

	return tmp, err
}

// Load loads and saves/cache the templates
// accepts one parameter
// globPathExp the path which the html files are, for example .Load("./frontend/templates/*.html")
// panics if something bad happens during the loading
func (html *HTMLContainer) Load(globPathExp string, namespace ...string) {
	var err error
	var rootPath string
	var ns = ""
	if namespace != nil && len(namespace) > 0 {
		ns = namespace[0]
	}

	if html.loaded == false {

		if strings.LastIndexByte(globPathExp, '*') == len(globPathExp)-1 {
			globPathExp += ".html" // ./* -> ./*.html
		}

		html.Templates, err = html.parseGlob(globPathExp, ns)

		if err != nil {
			//if err then try to load the same path but with the current directory prefix
			// and if not success again then just panic with the first error
			pwd, cerr := os.Getwd()
			if cerr != nil {
				html.logger.Panic(ErrTemplateParse.With(err).Error() + " \nSecond try: \n" + cerr.Error())
			}
			//try with current directory + path
			html.Templates, cerr = html.parseGlob(pwd+globPathExp, ns)
			if cerr != nil {
				//this will fail if path starts with '.', so try again without the first letter
				//we do that and no html.Load again because we want to keep the first error
				html.Templates, cerr = html.parseGlob(pwd+globPathExp[1:], ns)
				if cerr != nil {
					html.logger.Panic(ErrTemplateParse.With(err).Error() + " \n and second try: \n" + cerr.Error())
				}
			}

			rootPath = pwd + globPathExp
		} else {
			rootPath = globPathExp
		}
		if strings.Contains(html.directory, "/") {
			html.directory = strings.Replace(rootPath[0:strings.LastIndexByte(rootPath, '/')], "/", string(os.PathSeparator), -1)
		}
		html.ext = rootPath[strings.IndexByte(rootPath, '*'):]
		html.startWatch(html.directory)
		html.loaded = true
	}

	if err != nil {
		html.logger.Panic(err.Error())
	}

}

// Reload reloads the templates, it just calls templates.ParseGlob again
func (html *HTMLContainer) Reload() error {
	var err error
	html.Templates, err = html.Templates.ParseGlob(html.directory + string(os.PathSeparator) + html.ext) //template.ParseGlob(html.directory + string(os.PathSeparator) + html.ext)
	return ErrTemplateParse.With(err)
}

// startWatch start watching for template-file changes and reload them if needed
func (html *HTMLContainer) startWatch(rootPath string) {
	utils.WatchDirectoryChanges(rootPath, func(fname string) {
		html.mu.Lock()
		html.Reload() //reload all html templates, no just the .html file [ for now ]
		html.mu.Unlock()
	}, html.logger)

}
