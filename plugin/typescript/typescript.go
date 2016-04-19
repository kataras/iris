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
package typescript

/* Notes
17 April 2016:
According to: https://github.com/Microsoft/TypeScript/issues/2375
tsc.exe --watch doesn't works on windows but node tsc.js --watch works
so instead of the tsc.exe we will use node $global_node_modules\typescript\lib\tsc.js as the host/executable/bin file
*/
///TODO: check if nodejs is installed before the check for typescript.
import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/cli/npm"
	"github.com/kataras/iris/cli/system"
)

var (
	node_modules = system.PathSeparator + "node_modules" + system.PathSeparator
	Name         = "TypescriptPlugin"
)

type (
	// Options the struct which holds the TypescriptPlugin options
	// Has four (4) fields
	//
	// 1. Bin: 	string, the typescript installation directory/bin (where the tsc or tsc.cmd are exists), if empty it will search inside global npm modules
	// 2. Dir:     string, Dir set the root, where to search for typescript files/project. Default "./"
	// 3. Ignore:  string, comma separated ignore typescript files/project from these directories. Default "" (node_modules are always ignored)
	// 4. Watch:	 boolean, watch for any changes and re-build if true/. Default true
	Options struct {
		Bin    string
		Dir    string
		Ignore string
		Watch  bool
	}
	// TypescriptPlugin the struct of the plugin, holds all necessary fields & methods
	TypescriptPlugin struct {
		options Options
		logger  *iris.Logger
	}
)

// DefaultOptions returns the default Options of the TypescriptPlugin
func DefaultOptions() Options {
	root, err := os.Getwd()
	if err != nil {
		panic("Typescript Plugin: Cannot get the Current Working Directory !!! [os.getwd()]")
	}
	opt := Options{Dir: root + system.PathSeparator, Ignore: node_modules, Watch: true}
	opt.Bin = npm.Abs("typescript/lib/tsc.js")
	return opt

}

// TypescriptPlugin

// New creates & returns a new instnace typescript plugin
func New(_opt ...Options) *TypescriptPlugin {
	var options = DefaultOptions()

	if _opt != nil && len(_opt) > 0 { //not nil always but I like this way :)
		opt := _opt[0]

		if opt.Bin != "" {
			options.Bin = opt.Bin
		}
		if opt.Dir != "" {
			options.Dir = opt.Dir
		}

		if !strings.Contains(opt.Ignore, "node_modules") {
			opt.Ignore += "," + node_modules
		}

		options.Ignore = opt.Ignore
		options.Watch = opt.Watch
	}

	return &TypescriptPlugin{options: options}
}

// implement the IPlugin & IPluginPreListen
func (t *TypescriptPlugin) Activate(container iris.IPluginContainer) error {
	return nil
}

func (t *TypescriptPlugin) GetName() string {
	return Name
}

func (t *TypescriptPlugin) GetDescription() string {
	return Name + " scans and compile typescript files with ease. \n"
}

func (t *TypescriptPlugin) PreListen(s *iris.Station) {
	t.logger = s.Logger()
	t.start()
}

//

// implementation

func (t *TypescriptPlugin) start() {
	if t.hasTypescriptFiles() {
		//Can't check if permission denied returns always exists = true....
		//typescriptModule := out + string(os.PathSeparator) + "typescript" + string(os.PathSeparator) + "bin"
		if !npm.Exists("typescript/lib/tsc.js") {
			t.logger.Println("Typescript is not installed, please wait while installing typescript")
			res := npm.Install("typescript")

			t.logger.Print(res.Output())

		}

		dirs := t.getTypescriptProjects()
		if len(dirs) > 0 {
			//typescript project (.tsconfig) found
			for _, dir := range dirs {
				_, err := system.Command("node", npm.Abs("typescript/lib/tsc.js"), "-p", dir)

				if err != nil {
					t.logger.Println(err.Error())
					return
				}

			}
			t.logger.Printf("%d Typescript project(s) compiled", len(dirs))
		} else {
			//search for standalone typescript (.ts) files and combile them
			files := t.getTypescriptFiles()
			if len(files) > 0 {
				//it must be always > 0 if we came here, because of if hasTypescriptFiles == true.
				for _, file := range files {

					_, err := system.Command("node ", npm.Abs("typescript/lib/tsc.js"), file)
					if err != nil {
						t.logger.Println(err.Error())
						return
					}

				}
				t.logger.Printf("%d Typescript file(s) compiled", len(files))
			}

		}

	}
}

func (t *TypescriptPlugin) hasTypescriptFiles() bool {
	root := t.options.Dir
	ignoreFolders := strings.Split(t.options.Ignore, ",")
	hasTs := false

	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {

		if fi.IsDir() {
			return nil
		}
		for i := range ignoreFolders {
			if strings.Contains(path, ignoreFolders[i]) {
				return nil
			}
		}
		if strings.HasSuffix(path, ".ts") {
			hasTs = true
			return errors.New("Typescript found, hope that will stop here")
		}

		return nil
	})
	return hasTs
}

func (t *TypescriptPlugin) getTypescriptProjects() []string {
	projects := make([]string, 0)
	ignoreFolders := strings.Split(t.options.Ignore, ",")

	root := t.options.Dir
	//t.logger.Printf("\nSearching for typescript projects in %s", root)

	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}
		for i := range ignoreFolders {
			if strings.Contains(path, ignoreFolders[i]) {
				//t.logger.Println(path + " ignored")
				return nil
			}
		}

		if strings.HasSuffix(path, system.PathSeparator+"tsconfig.json") {
			//t.logger.Printf("\nTypescript project found in %s", path)
			projects = append(projects, path[0:strings.LastIndex(path, system.PathSeparator)]) //remove the /tsconfig.json, it can run with it but it's better to have only the directory
		}

		return nil
	})
	return projects
}

// this is being called if getTypescriptProjects return 0 len, then we are searching for files using that:
func (t *TypescriptPlugin) getTypescriptFiles() []string {
	files := make([]string, 0)
	ignoreFolders := strings.Split(t.options.Ignore, ",")

	root := t.options.Dir
	//t.logger.Printf("\nSearching for typescript files in %s", root)

	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}
		for i := range ignoreFolders {
			if strings.Contains(path, ignoreFolders[i]) {
				//t.logger.Println(path + " ignored")
				return nil
			}
		}

		if strings.HasSuffix(path, ".ts") {
			//t.logger.Printf("\nTypescript file found in %s", path)
			files = append(files, path)
		}

		return nil
	})
	return files
}

//
//
