// Copyright (c) 2016, Gerasimos Maropoulos. All rights reserved.
// Copyright (c) 2012 The Go Authors. All rights reserved.
// Copyright (c) 2012 fsnotify Authors. All rights reserved for package fsnotify/fsnotify
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
/////////////////////////////////////////////////////////////////////////////////////////////////////////
// License for package fsnotify/fsnotify
// Copyright (c) 2012 The Go Authors. All rights reserved.
// Copyright (c) 2012 fsnotify Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:

//   * Redistributions of source code must retain the above copyright
//notice, this list of conditions and the following disclaimer.
//   * Redistributions in binary form must reproduce the above
//copyright notice, this list of conditions and the following disclaimer
//in the documentation and/or other materials provided with the
//distribution.
//   * Neither the name of Google Inc. nor the names of its
//contributors may be used to endorse or promote products derived from
//this software without specific prior written permission.
//
//THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
//"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
//LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
//A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
//OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
//SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
//LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
//DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
//THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
//(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
//OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package iris

import (
	"github.com/fsnotify/fsnotify"
	"html/template"
	"os"
	"strings"
	"sync"
	"time"
)

type (
	HTMLTemplates struct {
		logger *Logger
		// Templates contains all the html templates it's type of *template.Template from standar API
		Templates *template.Template
		rootPath  string
		loaded    bool
		ext       string
		directory string
		pattern   string
		mu        *sync.Mutex
	}
)

func NewHTMLTemplates(logger *Logger) *HTMLTemplates {
	return &HTMLTemplates{logger: logger, mu: &sync.Mutex{}}
}

func (html *HTMLTemplates) Load(globPathExp string) {
	var err error
	var rootPath string
	if html.loaded == false {
		html.Templates, err = template.ParseGlob(globPathExp)

		if err != nil {
			//if err then try to load the same path but with the current directory prefix
			// and if not success again then just panic with the first error
			pwd, cerr := os.Getwd()
			if cerr != nil {
				Printf(html.logger, ErrTemplateParse, cerr.Error())
				return
			}
			html.Templates, err = template.ParseGlob(pwd + globPathExp)
			if err != nil {
				Printf(html.logger, ErrTemplateParse, err.Error())
				return

			}

			rootPath = pwd + globPathExp
		} else {
			rootPath = globPathExp
		}

		html.directory = strings.Replace(rootPath[0:strings.LastIndexByte(rootPath, SlashByte)], "/", string(os.PathSeparator), -1)
		html.ext = rootPath[strings.IndexByte(rootPath, MatchEverythingByte):]
		html.startWatch(html.directory)
		html.loaded = true
	}

}

func (html *HTMLTemplates) reload() {
	var err error
	html.Templates, err = html.Templates.ParseGlob(html.directory + string(os.PathSeparator) + html.ext) //template.ParseGlob(html.directory + string(os.PathSeparator) + html.ext)

	if err != nil {
		Printf(html.logger, ErrTemplateParse, err.Error())
	}
}

///TODO: After breakfast continue this.
//func (html *HTMLTemplates) Add(globPathExp string) {
//
//}

func (html *HTMLTemplates) startWatch(rootPath string) {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		Printf(html.logger, ErrTemplateWatch, err.Error())
		return
	}

	go func() {
		var lastChange = time.Now()
		var i = 0
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					//this is received two times, the last time is the real changed file, so
					html.mu.Lock()
					i++
					if i%2 == 0 {
						if time.Now().After(lastChange.Add(time.Duration(1) * time.Second)) {
							lastChange = time.Now()
							html.reload() //reload all html templates, no just the .html file [ for now ]
						}
					}
					html.mu.Unlock()

				}
			case err := <-watcher.Errors:
				Printf(html.logger, err)
			}
		}
	}()

	err = watcher.Add(rootPath)
	if err != nil {
		Printf(html.logger, err)
	}

}
