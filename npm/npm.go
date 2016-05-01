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

package npm

import (
	"fmt"
	"strings"
	"time"

	"github.com/kataras/iris/utils"
)

var (
	// NodeModules is the path of the root npm modules
	// Ex: C:\\Users\\kataras\\AppData\\Roaming\\npm\\node_modules
	NodeModules string
)

type (
	// Result holds Message and Error, if error != nil then the npm command has failed
	Result struct {
		// Message the message (string)
		Message string
		// Error the error (if any)
		Error error
	}
)

// init sets the root directory for the node_modules
func init() {
	NodeModules = utils.MustCommand("npm", "root", "-g") //here it ends with \n we have to remove it
	NodeModules = NodeModules[0 : len(NodeModules)-1]
}

func success(output string, a ...interface{}) Result {
	return Result{fmt.Sprintf(output, a...), nil}
}

func fail(errMsg string, a ...interface{}) Result {
	return Result{"", fmt.Errorf("\n"+errMsg, a...)}
}

// Output returns the error message if result.Error exists, otherwise returns the result.Message
func (res Result) Output() (out string) {
	if res.Error != nil {
		out = res.Error.Error()
	} else {
		out = res.Message
	}
	return
}

// Install installs a module
func Install(moduleName string) Result {
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
	out, err := utils.Command("npm", "install", moduleName, "-g")
	finish <- true
	if err != nil {
		return fail("Error installing module %s. Trace: %s", moduleName, err.Error())
	}

	return success("\n%s installed %s", moduleName, out)

}

// Unistall removes a module
func Unistall(moduleName string) Result {
	out, err := utils.Command("npm", "unistall", "-g", moduleName)
	if err != nil {
		return fail("Error unstalling module %s. Trace: %s", moduleName, err.Error())
	}
	return success("\n %s unistalled %s", moduleName, out)

}

// Abs returns the absolute path of the global node_modules directory + relative
func Abs(relativePath string) string {
	return NodeModules + utils.PathSeparator + strings.Replace(relativePath, "/", utils.PathSeparator, -1)
}

// Exists returns true if a module exists
// here we have two options
//1 . search by command something like npm -ls -g --depth=x
//2.  search on files, we choose the second
func Exists(executableRelativePath string) bool {
	execAbsPath := Abs(executableRelativePath)
	if execAbsPath == "" {
		return false
	}

	return utils.Exists(execAbsPath)
}
