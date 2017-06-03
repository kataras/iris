// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package typescript provides a typescript compiler with hot-reloader
// and optionally a cloud-based editor, called 'alm-tools'.
// typescript (by microsoft) and alm-tools (by @basarat) have their own (open-source) licenses
// the tools are not used directly by this adaptor, but it's good to know where you can find
// the software.
package typescript

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/typescript/npm"
)

type (
	// Typescript contains the unique iris' typescript loader, holds all necessary fields & methods.
	Typescript struct {
		Config *Config
		// taken from framework
		log func(format string, a ...interface{})
	}
)

// New creates & returns a new instnace typescript plugin
func New() *Typescript {
	c := DefaultConfig()

	if !strings.Contains(c.Ignore, nodeModules) {
		c.Ignore += "," + nodeModules
	}

	return &Typescript{Config: &c}
}

// implementation

func (t *Typescript) start() {

	if t.hasTypescriptFiles() {
		//Can't check if permission denied returns always exists = true....

		if !npm.NodeModuleExists(t.Config.Bin) {
			t.log("installing typescript, please wait...")
			res := npm.NodeModuleInstall("typescript")
			if res.Error != nil {
				t.log(res.Error.Error())
				return
			}
			t.log(res.Message)
		}

		projects := t.getTypescriptProjects()
		if len(projects) > 0 {
			watchedProjects := 0
			//typescript project (.tsconfig) found
			for _, project := range projects {
				cmd := npm.CommandBuilder("node", t.Config.Bin, "-p", project[0:strings.LastIndex(project, npm.PathSeparator)]) //remove the /tsconfig.json)
				projectConfig, perr := FromFile(project)
				if perr != nil {
					t.log("error while trying to read tsconfig: %s", perr.Error())
					continue
				}

				if projectConfig.CompilerOptions.Watch {
					watchedProjects++
					// if has watch : true then we have to wrap the command to a goroutine (I don't want to use the .Start here)
					go func() {
						_, err := cmd.Output()
						if err != nil {
							t.log(err.Error())
							return
						}
					}()
				} else {

					_, err := cmd.Output()
					if err != nil {
						t.log(err.Error())
						return
					}

				}

			}
			// t.log("%d Typescript project(s) compiled ( %d monitored by a background file watcher", len(projects), watchedProjects)
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
							t.log(compilerMsg)
						}

					}()

				}
				// t.log("%d Typescript file(s) compiled ( %d monitored by a background file watcher )", len(files), watchedFiles)
			}

		}

	}
}

func (t *Typescript) hasTypescriptFiles() bool {
	root := t.Config.Dir
	ignoreFolders := strings.Split(t.Config.Ignore, ",")
	hasTs := false
	if !npm.Exists(root) {
		t.log("typescript error: directory '%s' couldn't be found,\nplease specify a valid path for your *.ts files", root)
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

func (t *Typescript) getTypescriptProjects() []string {
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
func (t *Typescript) getTypescriptFiles() []string {
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

// Attach attaches the typescript to one or more Iris instance(s).
func (t *Typescript) Attach(app *iris.Application) {
	t.log = app.Log
	t.start()
}
