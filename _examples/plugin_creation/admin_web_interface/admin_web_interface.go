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
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package admin_web_interface

import (
	"fmt"
	"github.com/kataras/iris"
	"html/template"
)

// This is just an example template may not working at your file system
const DefaultTemplatesPath = "../mygopath/src/github.com/kataras/iris/plugins/admin_web_interface/templates/*"

type Options struct {
	// Path the path url which the admin web interfaces lives
	Path string

	// Admins basic authendication, just username & password for now
	// if empty then the interface is open for all users
	Admins map[string]string
}

type IndexPage struct {
	Title          string
	WelcomeMessage string
}
type AdminWebInterface struct {
	pluginContainer *iris.PluginContainer //we will use it to print messages
	options         Options
	templates       *template.Template
	failed          bool
}

func Newbie() *AdminWebInterface {
	options := Options{Path: "/plugin/admin/", Admins: nil}
	return New(options)
}

func New(options Options) *AdminWebInterface {
	if options.Path == "" {
		options.Path = "/plugin/admin/" // last slash will be removed automatically from .Party func, we need the last slash to compare it with other route's path prefixes
	}
	return &AdminWebInterface{options: options, templates: template.Must(template.ParseGlob(DefaultTemplatesPath))}
}

// runs on the PreBuild state
func (w *AdminWebInterface) registerHandlers(s *iris.Station) {
	admin := s.Party(w.options.Path)
	{
		admin.Get("/", func(c *iris.Context) {
			w.templates.ExecuteTemplate(c.ResponseWriter, "index.html", IndexPage{w.GetName(), "Welcome to Iris admin panel"})
		})
	}
}

func (w *AdminWebInterface) GetName() string {
	return "Admin Web Interface"
}

func (w *AdminWebInterface) GetDescription() string {
	return "Admin Web Interface registers routes and webpages to your application to allow remote access the Iris' server"
}

func (w *AdminWebInterface) Activate(p *iris.PluginContainer) error {
	fmt.Printf("### %s is activated \n%s\n", w.GetName(), w.GetDescription())
	w.pluginContainer = p
	return nil
}

func (w *AdminWebInterface) PreHandle(method string, r *iris.Route) {}

func (w *AdminWebInterface) PostHandle(method string, r *iris.Route) {}

func (w *AdminWebInterface) PreListen(s *iris.Station) {
	if w.failed {
		w.pluginContainer.RemovePlugin(w.GetName()) //removes itself
	} else {
		w.registerHandlers(s)
	}
}

func (w *AdminWebInterface) PostListen(s *iris.Station, err error) {}

func (w *AdminWebInterface) PreClose(s *iris.Station) {}
