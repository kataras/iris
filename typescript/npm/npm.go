package npm

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var (
	// nodeModulesPath is the path of the root npm modules
	// Ex: C:\\Users\\iris\\AppData\\Roaming\\npm\\node_modules
	nodeModulesPath string
)

type (
	// NodeModuleResult holds Message and Error, if error != nil then the npm command has failed
	NodeModuleResult struct {
		// Message the message (string)
		Message string
		// Error the error (if any)
		Error error
	}
)

// NodeModulesPath sets the root directory for the node_modules and returns that
func NodeModulesPath() string {
	if nodeModulesPath == "" {
		nodeModulesPath = MustCommand("npm", "root", "-g") //here it ends with \n we have to remove it
		nodeModulesPath = nodeModulesPath[0 : len(nodeModulesPath)-1]
	}
	return nodeModulesPath
}

func success(output string, a ...interface{}) NodeModuleResult {
	return NodeModuleResult{fmt.Sprintf(output, a...), nil}
}

func fail(errMsg string, a ...interface{}) NodeModuleResult {
	return NodeModuleResult{"", fmt.Errorf("\n"+errMsg, a...)}
}

// Output returns the error message if result.Error exists, otherwise returns the result.Message
func (res NodeModuleResult) Output() (out string) {
	if res.Error != nil {
		out = res.Error.Error()
	} else {
		out = res.Message
	}
	return
}

// NodeModuleInstall installs a module
func NodeModuleInstall(moduleName string) NodeModuleResult {
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
		return fail("Error installing module %s. Trace: %s", moduleName, err.Error())
	}

	return success("\n%s installed %s", moduleName, out)

}

// NodeModuleUnistall removes a module
func NodeModuleUnistall(moduleName string) NodeModuleResult {
	out, err := Command("npm", "unistall", "-g", moduleName)
	if err != nil {
		return fail("Error unstalling module %s. Trace: %s", moduleName, err.Error())
	}
	return success("\n %s unistalled %s", moduleName, out)

}

// NodeModuleAbs returns the absolute path of the global node_modules directory + relative
func NodeModuleAbs(relativePath string) string {
	return NodeModulesPath() + PathSeparator + strings.Replace(relativePath, "/", PathSeparator, -1)
}

// NodeModuleExists returns true if a module exists
// here we have two options
//1 . search by command something like npm -ls -g --depth=x
//2.  search on files, we choose the second
func NodeModuleExists(execPath string) bool {
	if !filepath.IsAbs(execPath) {
		execPath = NodeModuleAbs(execPath)
	}

	return Exists(execPath)
}
