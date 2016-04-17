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
package cli

import (
	"runtime"
	"strings"
	"time"

	"github.com/kataras/iris"
)

type (
	// Npm service for global nodejs modules
	// use: npm = &Npm{logger{
	// 		npm.Install("typescript")
	Npm struct {
		logger *iris.Logger
	}
	// NpmModule is just an easy way to run Npm's commands when this module is using many times in the code base
	//use: module := NewNpmModule("typescript", logger)
	// 	   module.Install()
	NpmModule struct {
		Name    string
		Bin     string
		service *Npm
	}
)

// maybe it's bad practise to use a logger here, we could just return a message and an error with message if error happens, we will see at the future maybe this will change.
func (npm *Npm) SetLogger(logger *iris.Logger) {
	npm.logger = logger
}

// Install installs a module
func (npm *Npm) Install(moduleName string) error {
	finish := make(chan bool)

	go func() {
		print("\n|")
		print("_")
		print("|")

		for {
			select {
			case v := <-finish:
				{
					if v {
						print("\010\010\010") //remove the loading chars
						close(finish)
						return
					}

				}
			default:
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
			}
		}

	}()
	out, err := Command("npm", "install", moduleName, "-g")
	finish <- true
	if err != nil {
		return Printf(npm.logger, ErrInstallingModule, moduleName, err.Error())
	} else {
		npm.logger.Printf("%s installed %s", moduleName, out)
		return nil
	}
}

// Unistall removes a module
func (npm *Npm) Unistall(moduleName string) error {
	out, err := Command("npm", "unistall", "-g", moduleName)
	if err != nil {
		return Printf(npm.logger, ErrUnistallingModule, moduleName, err.Error())
	} else {
		npm.logger.Printf("\n %s unistalled %s", moduleName, out)
		return nil
	}
}

// Exists returns true if a module exists
// here we have two options
//1 . search by command something like npm -ls -g --depth=x
//2.  search on files, we choosen the second
func (npm *Npm) Exists(moduleNameExecFile string) bool {
	binfile := npm.GetExecutable(moduleNameExecFile)
	if binfile == "" {
		return false
	}

	return Exists(binfile)
}

// GetExecutable returns the absolute path of a module's binary(executable) file
// it doesn't checks for errors or if the file exists, it justs returns a string which can be empty too
func (npm *Npm) GetExecutable(moduleNameExecFile string) (absPath string) {
	out := MustCommand("npm", "root", "-g")

	npmDir := out[0:strings.LastIndex(out, PathSeparator)]
	absPath = npmDir + PathSeparator + moduleNameExecFile
	if runtime.GOOS == "windows" {
		absPath += ".cmd"
	}

	return
}

// NpmModule

func NewNpmModule(name string, binName string, logger *iris.Logger) *NpmModule {
	service := &Npm{logger: logger}
	return &NpmModule{name, binName, service}
}

func (module *NpmModule) SetLogger(logger *iris.Logger) {
	module.service.SetLogger(logger)
}

func (module *NpmModule) Install() error {
	return module.service.Install(module.Name)
}

func (module *NpmModule) Unistall() error {
	return module.service.Unistall(module.Name)
}

func (module *NpmModule) Exists() bool {
	return module.service.Exists(module.Bin)
}

func (module *NpmModule) GetExecutable() string {
	return module.service.GetExecutable(module.Bin)
}
