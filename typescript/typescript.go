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

	"github.com/kataras/iris/typescript/npm"
)

type (
	// Typescript contains the unique iris' typescript loader, holds all necessary fields & methods.
	Typescript struct {
		Config *Config
		log    func(format string, a ...interface{})
	}
)

// New creates & returns a new instnace typescript plugin
func New(cfg ...Config) *Typescript {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	return &Typescript{Config: &c}
}

var (
	// NoOpLogger can be used as the logger argument, it prints nothing.
	NoOpLogger = func(string, ...interface{}) {}
)

// Run starts the typescript filewatcher watcher and the typescript compiler.
func (t *Typescript) Run(logger func(format string, a ...interface{})) {
	c := t.Config
	if c.Tsconfig == nil {
		tsC := DefaultTsconfig()
		c.Tsconfig = &tsC
	}

	if c.Dir == "" {
		c.Tsconfig.CompilerOptions.OutDir = c.Dir
	}

	if c.Dir == "" {
		c.Dir = "./"
	}

	if !strings.Contains(c.Ignore, nodeModules) {
		c.Ignore += "," + nodeModules
	}

	if logger == nil {
		logger = NoOpLogger
	}

	t.log = logger

	t.start()
}

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
			// typescript project (.tsconfig) found
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
							t.log("error when 'watch' is true: %v", err)
							return
						}
					}()
				} else {
					_, err := cmd.Output()
					if err != nil {
						t.log("unexpected error from output: %v", err)
						return
					}

				}

			}
			return
		}
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
					t.log("error while trying to resolve absolute path for %s: %v", file, err)
					continue
				}

				// these will be used if no .tsconfig found.
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
			return
		}
		return
	}
	absPath, err := filepath.Abs(t.Config.Dir)
	if err != nil {
		t.log("no typescript file, the directory cannot be resolved: %v", err)
		return
	}
	t.log("no typescript files found on : %s", absPath)
}

func (t *Typescript) hasTypescriptFiles() bool {
	root := t.Config.Dir
	ignoreFolders := t.getIgnoreFolders()
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
		for _, s := range ignoreFolders {
			if strings.HasSuffix(path, s) || path == s {
				return filepath.SkipDir
			}
		}

		if strings.HasSuffix(path, ".ts") {
			hasTs = true
			return errors.New("typescript found, hope that will stop here")
		}

		return nil
	})
	return hasTs
}

func (t *Typescript) getIgnoreFolders() (folders []string) {
	ignoreFolders := strings.Split(t.Config.Ignore, ",")

	for _, s := range ignoreFolders {
		if s != "" {
			folders = append(folders, s)
		}
	}

	return folders
}

func (t *Typescript) getTypescriptProjects() []string {
	var projects []string
	ignoreFolders := t.getIgnoreFolders()

	root := t.Config.Dir
	//t.logger.Printf("\nSearching for typescript projects in %s", root)

	// ignore error
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}
		for _, s := range ignoreFolders {
			if strings.HasSuffix(path, s) || path == s {
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
	ignoreFolders := t.getIgnoreFolders()

	root := t.Config.Dir

	// ignore error
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}
		for _, s := range ignoreFolders {
			if strings.HasSuffix(path, s) || path == s {
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
