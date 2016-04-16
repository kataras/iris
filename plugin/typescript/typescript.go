package typescript

import (
	"strings"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/cli"
)

// Options the struct which holds the TypescriptPlugin options
// Has 3 fields
//
// 1. Dir:     string, Dir set the root, where to search for typescript files/project. Default "./"
// 2. Ignore:  string, comma separated ignore typescript files/project from these directories. Default "" (node_modules are always ignored)
// 3. Watch:	 boolean, watch for any changes and re-build if true/. Default true
type Options struct {
	Dir    string
	Ignore string
	Watch  bool
}

var node_modules = cli.ToDir("node_modules")
var Name = "TypescriptPlugin"

// TypescriptPlugin the struct of the plugin, holds all necessary fields & methods
type TypescriptPlugin struct {
	options Options
	logger  *iris.Logger
}

// defaultOptions returns the default Options of the TypescriptPlugin
func defaultOptions() Options {
	return Options{Dir: "." + cli.PathSeparator, Ignore: node_modules, Watch: true}
}

// New creates & returns a new instnace typescript plugin
func New(_opt ...Options) *TypescriptPlugin {
	var options = defaultOptions()

	if _opt != nil && len(_opt) > 0 { //not nil always but I like this way :)
		opt := _opt[0]
		options.Dir = opt.Dir
		if !strings.Contains(opt.Ignore, "node_modules") {
			opt.Ignore += "," + node_modules
		}
		options.Ignore = opt.Ignore
		options.Watch = opt.Watch
	}

	return &TypescriptPlugin{options: options}
}

// implement the IPlugin & IPluginPostListen
func (t *TypescriptPlugin) Activate(container iris.IPluginContainer) error {
	return nil
}

func (t *TypescriptPlugin) GetName() string {
	return Name
}

func (t *TypescriptPlugin) GetDescription() string {
	return Name + " is a helper for client-side typescript projects.\n"
}

func (t *TypescriptPlugin) PostListen(s *iris.Station) {
	t.logger = s.Logger()
	t.start()
}

//

// implementation

func (t *TypescriptPlugin) start() {

}

func (t *TypescriptPlugin) installTypescript() {
	finish := false

	go func() {
		i := 0
		print("\n|")
		print("_")
		print("|")

	printer:
		{
			i++

			print("\010\010-")
			time.Sleep(time.Second / 2)
			print("\010\\")
			time.Sleep(time.Second / 2)
			print("\010|")
			time.Sleep(time.Second / 2)
			print("\010/")
			time.Sleep(time.Second / 2)
			print("\010-")
			time.Sleep(time.Second / 2)
			print("|")
			if finish {
				goto ok
			}
			goto printer
		}

	ok:
	}()
	out, err := cli.Command("npm", "install", "typescript", "-g")
	finish = true
	if err != nil {
		t.logger.Printf("\nError installing typescript %s", err.Error())
	} else {
		t.logger.Printf("\nTypescript installed %s", out)
	}

}

//
