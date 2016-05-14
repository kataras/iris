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
