// Package typescript provides a typescript compiler with hot-reloader
// and optionally a cloud-based editor, called 'alm-tools'.
// typescript (by microsoft) and alm-tools (by basarat) have their own (open-source) licenses
// the tools are not used directly by this adaptor, but it's good to know where you can find
// the software.
package typescript

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/typescript/npm"
)

type (
	// TsAdaptor the struct of the Typescript TsAdaptor, holds all necessary fields & methods
	TsAdaptor struct {
		Config *Config
		// taken from framework
		logger func(iris.LogMode, string)
	}
)

// New creates & returns a new instnace typescript plugin
func New() *TsAdaptor {
	c := DefaultConfig()

	if !strings.Contains(c.Ignore, nodeModules) {
		c.Ignore += "," + nodeModules
	}

	return &TsAdaptor{Config: &c}
}

// Adapt addapts a TsAdaptor to the Policies via EventPolicy.
// We use that method instead of direct return EventPolicy from new because
// the user should be able to change its configuration from that public API
func (t *TsAdaptor) Adapt(frame *iris.Policies) {
	policy := iris.EventPolicy{
		Build: t.build,
	}

	policy.Adapt(frame)
}

func (t *TsAdaptor) build(s *iris.Framework) {
	t.logger = s.Log
	t.start()
}

//

// implementation

func (t *TsAdaptor) start() {

	if t.hasTypescriptFiles() {
		//Can't check if permission denied returns always exists = true....

		if !npm.NodeModuleExists(t.Config.Bin) {
			t.logger(iris.DevMode, "Installing typescript, please wait...")
			res := npm.NodeModuleInstall("typescript")
			if res.Error != nil {
				t.logger(iris.ProdMode, res.Error.Error())
				return
			}
			t.logger(iris.DevMode, res.Message)
		}

		projects := t.getTypescriptProjects()
		if len(projects) > 0 {
			watchedProjects := 0
			//typescript project (.tsconfig) found
			for _, project := range projects {
				cmd := npm.CommandBuilder("node", t.Config.Bin, "-p", project[0:strings.LastIndex(project, npm.PathSeparator)]) //remove the /tsconfig.json)
				projectConfig, perr := FromFile(project)
				if perr != nil {
					t.logger(iris.ProdMode, "error while trying to read tsconfig: "+perr.Error())
					continue
				}

				if projectConfig.CompilerOptions.Watch {
					watchedProjects++
					// if has watch : true then we have to wrap the command to a goroutine (I don't want to use the .Start here)
					go func() {
						_, err := cmd.Output()
						if err != nil {
							t.logger(iris.DevMode, err.Error())
							return
						}
					}()
				} else {

					_, err := cmd.Output()
					if err != nil {
						t.logger(iris.DevMode, err.Error())
						return
					}

				}

			}
			// t.logger(iris.DevMode, fmt.Sprintf("%d Typescript project(s) compiled ( %d monitored by a background file watcher ) ", len(projects), watchedProjects))
		} else {
			//search for standalone typescript (.ts) files and compile them
			files := t.getTypescriptFiles()
			if len(files) > 0 {
				/* watchedFiles := 0
				if t.Config.Tsconfig.CompilerOptions.Watch {
					watchedFiles = len(files)
				}*/
				//it must be always > 0 if we came here, because of if hasTypescriptFiles == true.
				for _, file := range files {
					absPath, err := filepath.Abs(file)
					if err != nil {
						continue
					}

					//these will be used if no .tsconfig found.
					// cmd := npm.CommandBuilder("node", t.Config.Bin)
					// cmd.Arguments(t.Config.Bin, t.Config.Tsconfig.CompilerArgs()...)
					// cmd.AppendArguments(absPath)
					compilerArgs := t.Config.Tsconfig.CompilerArgs()
					cmd := npm.CommandBuilder("node", t.Config.Bin)
					for _, s := range compilerArgs {
						cmd.AppendArguments(s)
					}
					cmd.AppendArguments(absPath)
					go func() {
						compilerMsgB, _ := cmd.Output()
						compilerMsg := string(compilerMsgB)
						cmd.Args = cmd.Args[0 : len(cmd.Args)-1] //remove the last, which is the file

						if strings.Contains(compilerMsg, "error") {
							t.logger(iris.DevMode, compilerMsg)
						}

					}()

				}
				// t.logger(iris.DevMode, fmt.Sprintf("%d Typescript file(s) compiled ( %d monitored by a background file watcher )", len(files), watchedFiles))
			}

		}

	}
}

func (t *TsAdaptor) hasTypescriptFiles() bool {
	root := t.Config.Dir
	ignoreFolders := strings.Split(t.Config.Ignore, ",")
	hasTs := false
	if !npm.Exists(root) {
		t.logger(iris.ProdMode, fmt.Sprintf("Typescript Adaptor Error: Directory '%s' couldn't be found,\nplease specify a valid path for your *.ts files", root))
		return false
	}
	// ignore error
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

func (t *TsAdaptor) getTypescriptProjects() []string {
	var projects []string
	ignoreFolders := strings.Split(t.Config.Ignore, ",")

	root := t.Config.Dir
	//t.logger.Printf("\nSearching for typescript projects in %s", root)

	// ignore error
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}
		for i := range ignoreFolders {
			if strings.Contains(path, ignoreFolders[i]) {
				//t.logger.Println(path + " ignored")
				return filepath.SkipDir
			}
		}

		if strings.HasSuffix(path, npm.PathSeparator+"tsconfig.json") {
			//t.logger.Printf("\nTypescript project found in %s", path)
			projects = append(projects, path)
		}

		return nil
	})
	return projects
}

// this is being called if getTypescriptProjects return 0 len, then we are searching for files using that:
func (t *TsAdaptor) getTypescriptFiles() []string {
	var files []string
	ignoreFolders := strings.Split(t.Config.Ignore, ",")

	root := t.Config.Dir

	// ignore error
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
