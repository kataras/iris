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
